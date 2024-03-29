package egen

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
)

var htmlFilenameRegExp = regexp.MustCompile(`.*\.html`)
var indexHTML = `
<!DOCTYPE html>
<html lang="{{ .Lang.Tag }}">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	{{ if .Color }}
		<meta name="theme-color" content="{{ .Color }}">
	{{ end }}
	<title>{{ .Title }}</title>
	{{ if .Description }}
		<meta name="description" content="{{ .Description }}">
	{{ end }}
	{{ if eq .Page "home" }}
		<meta property="og:type" content="website">
	{{ else if eq .Page "post" }}
		<meta property="og:type" content="article">
	{{ end }}
	<meta property="og:url" content="{{ relToAbsLink .URL }}">
	<meta property="og:title" content="{{ .Title }}">
	{{ if .Description }}
		<meta property="og:description" content="{{ .Description }}">
	{{ end }}
	{{ if .Img }}
		<meta property="og:image:url" content="{{ relToAbsLink (assetLink .Img.Path) }}">
		<meta property="og:image:alt" content="{{ .Img.Alt }}">
	{{ end }}
	{{ if eq .Page "post" }}
		<meta property="article:published_time" content="{{ dateISO .Post.Date }}">
		{{ if not .Post.LastUpdateDate.IsZero }}
			<meta property="article:modified_time" content="{{ dateISO .Post.LastUpdateDate }}">
		{{ end }}
	{{ end }}
	{{ if .Img }}
		<meta property="twitter:image:alt" content="{{ .Img.Alt }}">
	{{ end }}
	{{ if .Author.Twitter }}
		<meta property="twitter:site" content="@{{ .Author.Twitter }}">
		{{ if eq .Page "post" }}
			<meta property="twitter:creator" content="@{{ .Author.Twitter }}">
		{{ end }}
	{{ end }}
	{{ if hasAsset "/icon.png" }}
		<link rel="icon" href="{{ assetLink "/icon.png" }}">
		<link rel="apple-touch-icon" href="{{ assetLink "/icon.png" }}">
	{{ end }}
	{{ range .AlternateLinks -}}
  	<link rel="alternate" hreflang="{{ .Lang.Tag }}" href="{{ relToAbsLink .URL }}">
	{{- end }}
	{{ if hasAsset "/style.css" }}
		<link rel="stylesheet" href="{{ assetLink "/style.css" }}">
	{{ end }}
	{{ template "head" . }}
</head>
<body>
  {{ template "content" . }}
</body>
</html>
`

// Lang represents a language.
type Lang struct {
	Name string
	// The language tag, as in RFC 5646.
	Tag     string
	Default bool
}

// AlternateLink is a link to a version of the current page in another language.
type AlternateLink struct {
	// URL is a relative URL.
	URL  string
	Lang *Lang
}

// TemplateData is the data passed to a template.
type TemplateData struct {
	// Page is an identifier for the current page.
	// Home page -> home
	// Posts page -> posts
	Page        string
	Title       string
	Description string
	Author      *Author
	Img         *Img
	Color       string
	// Posts is a list of posts that are visible (feed: true)
	Posts []*Post
	// Post is equal to nil unless page == 'post'
	Post *Post
	Lang *Lang
	// URL is a relative URL.
	URL string
	// AlternateLinks is a list of alternate links to be used in meta tags.
	// It also includes the a link for the current page in the current language.
	AlternateLinks            []*AlternateLink
	ResponsiveImgMediaQueries string
}

func createBaseTemplateWithIncludes(
	templateFuncs template.FuncMap,
	includesInPath string,
	invisiblePostsByLangTag map[string][]*Post,
	gat *assetsTreeNode,
	url string,
	responsiveImgSizes []int,
) (*template.Template, error) {
	// funcs
	defaultTemplateFuncs := template.FuncMap{
		"dateISO": func(d time.Time) string {
			return d.Format(time.RFC3339)
		},
		"getInvisiblePost": func(l *Lang, slug string) *Post {
			if posts := invisiblePostsByLangTag[l.Tag]; posts != nil {
				for _, p := range posts {
					if p.Slug == slug {
						return p
					}
				}
			}

			return nil
		},
		"assetLink":   generateAssetsLinkFn(gat, nil, ""),
		"srcSetValue": generateSrcSetValueFn(gat, nil, "", responsiveImgSizes),
		"hasAsset":    generateHasAsset(gat, nil, ""),
		"postLinkBySlugAndLang": func(slug string, l *Lang) string {
			if l.Default {
				return fmt.Sprintf("/posts/%v", slug)
			}

			return fmt.Sprintf("/%v/posts/%v", l.Tag, slug)
		},
		"homeLinkByLang": func(l *Lang) string {
			if l.Default {
				return "/"
			}

			return fmt.Sprintf("/%v", l.Tag)
		},
		"relToAbsLink": func(link string) string {
			if link == "/" {
				return url
			}

			return url + link
		},
		"sortPostsByDateDesc": func(posts []*Post) []*Post {
			sorted := make([]*Post, len(posts))
			copy(sorted, posts)

			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Date.After(sorted[j].Date)
			})

			return sorted
		},
	}

	baseTemplate := template.Must(
		template.New("base").Funcs(templateFuncs).Funcs(defaultTemplateFuncs).Parse(indexHTML),
	)

	// includes
	includesFileInfos, err := os.ReadDir(includesInPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	for _, includesFileInfo := range includesFileInfos {
		if includesFileInfo.IsDir() || !htmlFilenameRegExp.MatchString(includesFileInfo.Name()) {
			continue
		}

		includeFileContent, err := os.ReadFile(path.Join(includesInPath, includesFileInfo.Name()))
		if err != nil {
			return nil, err
		}

		baseTemplate, err = baseTemplate.Parse(
			fmt.Sprintf(
				`{{ define "%v" }}%v{{ end }}`,
				strings.TrimRight(includesFileInfo.Name(), ".html"),
				string(includeFileContent),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	// creates a head template if one wasn't present in includes
	if t := baseTemplate.Lookup("head"); t == nil {
		baseTemplate = template.Must(baseTemplate.Parse(`{{ define "head" }}{{ end }}`))
	}

	return baseTemplate, nil
}

func createPageTemplate(pagesInPath string, baseTemplate *template.Template, pageName string) (*template.Template, error) {
	pageContent, err := os.ReadFile(path.Join(
		pagesInPath,
		fmt.Sprintf("%v.html", pageName),
	))
	if err != nil {
		return nil, err
	}

	return template.Must(
		template.Must(baseTemplate.Clone()).Parse(`{{ define "content" }}` + string(pageContent) + `{{ end }}`),
	), nil
}

func executeMinifyAndWriteTemplate(t *template.Template, tData TemplateData, outFilePath string) error {
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var buff bytes.Buffer
	err = t.Execute(&buff, tData)
	if err != nil {
		return err
	}

	m := minify.New()
	m.Add("text/html", &html.Minifier{
		KeepDocumentTags: true,
		KeepQuotes:       true,
		KeepEndTags:      true,
		KeepWhitespace:   true,
	})

	htmlMinified, err := m.Bytes("text/html", buff.Bytes())
	if err != nil {
		return err
	}
	if _, err = outFile.Write(htmlMinified); err != nil {
		return err
	}

	return nil
}

func generateAlternateLinks(preLangSegments, postLangSegments []string, langs []*Lang) []*AlternateLink {
	links := make([]*AlternateLink, 0, len(langs))

	for i, l := range langs {
		var segments []string

		if l.Default {
			segments = make([]string, 0, 1+len(preLangSegments)+len(postLangSegments))
			segments = append(segments, "/")
			segments = append(segments, preLangSegments...)
			segments = append(segments, postLangSegments...)

			// default lang is always the first
			if i != 0 {
				newLinks := make([]*AlternateLink, 0, len(langs))
				newLinks = append(newLinks, &AlternateLink{
					Lang: l,
					URL:  path.Join(segments...),
				})
				links = append(newLinks, links...)

				continue
			}
		} else {
			segments = make([]string, 0, 1+len(preLangSegments)+1+len(postLangSegments))
			segments = append(segments, "/")
			segments = append(segments, preLangSegments...)
			segments = append(segments, l.Tag)
			segments = append(segments, postLangSegments...)
		}

		links = append(links, &AlternateLink{
			Lang: l,
			URL:  path.Join(segments...),
		})
	}

	return links
}

/* dynamic template funcs */

func generateAssetsLinkFn(gat, pat *assetsTreeNode, postSlug string) func(assetPath AssetRelPath) (string, error) {
	return func(assetPath AssetRelPath) (string, error) {
		if n, searchedInPAT := findByRelPathInGATOrPAT(gat, pat, assetPath); n != nil {
			if searchedInPAT {
				return n.assetLink(postSlug, nil), nil
			}

			return n.assetLink("", nil), nil
		}

		return "", fmt.Errorf("%v not found in either GAT or PAT", assetPath)
	}
}

func generateHasAsset(gat, pat *assetsTreeNode, postSlug string) func(assetPath AssetRelPath) bool {
	return func(assetPath AssetRelPath) bool {
		n, _ := findByRelPathInGATOrPAT(gat, pat, assetPath)

		return n != nil
	}
}

func generateSrcSetValueFn(gat, pat *assetsTreeNode, postSlug string, widths []int) func(assetPath AssetRelPath) (string, error) {
	return func(assetPath AssetRelPath) (string, error) {
		if n, searchedInPAT := findByRelPathInGATOrPAT(gat, pat, assetPath); n != nil {
			n.addSizes(widths...)
			err := n.processSizes()
			if err != nil {
				return "", fmt.Errorf("processing sizes: %w", err)
			}

			if searchedInPAT {
				return n.generateSrcSetValue(postSlug), nil
			}

			return n.generateSrcSetValue(""), nil
		}

		return "", fmt.Errorf("%v not found in either GAT or PAT", assetPath)
	}
}
