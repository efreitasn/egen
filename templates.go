package egen

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/efreitasn/egen/htmlp"
)

var htmlFilenameRegExp = regexp.MustCompile(".*\\.html")
var indexHTML = `
<!DOCTYPE html>
<html lang="{{ .Lang.Tag }}">
<head>
  <meta charset="utf-8">
	<title>{{ .Title }}</title>
	{{ if .Description }}
		<meta property="description" content="{{ .Description }}">
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
		<meta property="og:image:url" content="{{ relToAbsLink (assetsLink .Img.Path) }}">
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
	<meta property="twitter:site" content="@{{ .Author.Twitter }}">
	{{ if eq .Page "post" }}
	<meta property="twitter:creator" content="@{{ .Author.Twitter }}">
	{{ end }}
	{{ range .AlternateLinks -}}
  	<link rel="alternate" hreflang="{{ .Lang.Tag }}" href="{{ relToAbsLink .URL }}">
	{{- end }}
	{{ with $cssFile := assetsLink "/style.css" }}
		{{ if $cssFile }}
			<link rel="stylesheet" href="{{ $cssFile }}">
		{{ end }}
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
	// relative
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
	// Posts is a list of posts that are visible (feed: true)
	Posts []*Post
	// it's equal to nil unless it's the post page
	Post *Post
	Lang *Lang
	// relative
	URL string
	// AlternateLinks is a list of alternate links to be used in meta tags.
	// It also includes the current link.
	AlternateLinks []*AlternateLink
}

func createBaseTemplateWithIncludes(bd buildData) (*template.Template, error) {
	// funcs
	funcs := bd.bc.Funcs

	if funcs == nil {
		funcs = make(template.FuncMap, 4)
	}

	funcs["dateISO"] = func(d time.Time) string {
		return d.Format(time.RFC3339)
	}

	funcs["assetsLink"] = generateAssetsLinkFn(bd.gat, nil, "")

	funcs["postLinkBySlugAndLang"] = func(slug string, l *Lang) string {
		return fmt.Sprintf("/%v/posts/%v", l.Tag, slug)
	}

	funcs["relToAbsLink"] = func(link string) string {
		if link == "/" {
			return bd.c.URL
		}

		return bd.c.URL + link
	}

	baseTemplate := template.Must(template.New("base").Funcs(funcs).Parse(indexHTML))

	// includes
	includesPath := path.Join(bd.bc.InPath, "includes")
	includesFileInfos, err := ioutil.ReadDir(includesPath)
	if err != nil {
		return nil, err
	}

	for _, includesFileInfo := range includesFileInfos {
		if includesFileInfo.IsDir() || !htmlFilenameRegExp.MatchString(includesFileInfo.Name()) {
			continue
		}

		includeFileContent, err := ioutil.ReadFile(path.Join(includesPath, includesFileInfo.Name()))
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

	// templates
	return baseTemplate, nil
}

func createPageTemplate(bd buildData, baseTemplate *template.Template, pageName string) (*template.Template, error) {
	pageContent, err := ioutil.ReadFile(path.Join(
		bd.bc.InPath,
		"pages",
		fmt.Sprintf("%v.html", pageName),
	))
	if err != nil {
		return nil, err
	}

	return template.Must(
		template.Must(baseTemplate.Clone()).Parse(`{{ define "content" }}` + string(pageContent) + `{{ end }}`),
	), nil
}

func executePrettifyAndWriteTemplate(t *template.Template, tData TemplateData, outFilePath string) error {
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

	htmlPretty, err := htmlp.Pretty(buff.Bytes())
	if err != nil {
		return err
	}
	if _, err = outFile.Write(htmlPretty); err != nil {
		return err
	}

	return nil
}

func executePostTemplateForEachPost(bd buildData, postsDirOutPath string, postPageT *template.Template, currentLang *Lang, posts []*Post) error {
	for _, p := range posts {
		postDirPath := path.Join(postsDirOutPath, p.Slug)
		err := os.Mkdir(postDirPath, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}

		data := TemplateData{
			Title:       fmt.Sprintf("%v - %v", p.Title, bd.c.Title),
			Description: p.Excerpt,
			Page:        "post",
			Post:        p,
			Lang:        currentLang,
			Author:      bd.c.Author,
		}

		if currentLang.Default {
			data.URL = "/posts/" + p.Slug
		} else {
			data.URL = "/" + currentLang.Tag + "/posts/" + p.Slug
		}

		if p.Img != nil {
			data.Img = p.Img
		} else {
			data.Img = bd.c.defaultImgByLangTag[currentLang.Tag]
		}

		// alternate links
		data.AlternateLinks = generateAlternateLinks(nil, []string{"posts", p.Slug}, bd.c.Langs)

		postPageT.Funcs(map[string]interface{}{
			"assetsLink": generateAssetsLinkFn(bd.gat, p.pwat, p.Slug),
		})

		err = executePrettifyAndWriteTemplate(postPageT, data, path.Join(postDirPath, "index.html"))
		if err != nil {
			return err
		}
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

func generateAssetsLinkFn(gat, pwat *AssetsTreeNode, postSlug string) func(assetPath AssetRelPath) (string, error) {
	return func(assetPath AssetRelPath) (string, error) {
		if n, searchedInPWAT := findByRelPathInGATOrPWAT(gat, pwat, assetPath); n != nil {
			if searchedInPWAT {
				return path.Join("/assets", postSlug, strings.TrimPrefix(n.processedRelPath, pwat.Path+"/")), nil
			}

			return path.Join("/assets", strings.TrimPrefix(n.processedRelPath, gat.Path+"/")), nil
		}

		return "", fmt.Errorf("%v not found in either GAT or PWAT", assetPath)
	}
}
