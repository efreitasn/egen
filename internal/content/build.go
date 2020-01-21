package content

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/yosssi/gohtml"
	"gopkg.in/yaml.v2"
)

var postContentRegExp = regexp.MustCompile("(?s)^---\n(.*?)\n---(.*)")
var htmlFilenameRegExp = regexp.MustCompile(".*\\.html")
var indexHTML = `
{{ define "index" -}}
<!DOCTYPE html>
<html lang="{{ .Lang.Tag }}">
<head>
  <meta charset="utf-8">
	<title>{{ .Title }}</title>
	{{ range .AlternateLinks -}}
  	<link rel="alternate" hreflang="{{ .Lang.Tag }}" href="{{ relToAbsLink .URL }}">
	{{- end }}
  {{ template "head" . }}
</head>
<body>
  {{ template "content" . }}
</body>
</html>
{{- end }}
`

type templateData struct {
	Title string
	// Posts is a list of posts that are visible (feed: true)
	Posts []*post
	// it's equal to nil unless it's the post page
	Post *post
	Lang *lang
	// relative
	URL string
	// AlternateLinks is a list of alternate links to be used in meta tags.
	// It also includes the current link.
	AlternateLinks []*alternateLink
}

type post struct {
	Title          string
	Content        template.HTML
	Slug           string
	Excerpt        string
	Keywords       []string
	Date           time.Time
	LastUpdateDate time.Time
	Lang           *lang
	// relative
	URL string
}

type postYAMLFrontMatter struct {
	Title   string `yaml:"title"`
	Excerpt string `yaml:"excerpt"`
}

type postYAMLDataFileContent struct {
	Keywords       string `yaml:"keywords"`
	Feed           bool   `yaml:"feed"`
	Date           string `yaml:"date"`
	LastUpdateDate string `yaml:"lastUpdateDate"`
}

type alternateLink struct {
	// relative
	URL  string
	Lang *lang
}

// Build builds the website.
func (wc *WebsiteContent) Build(outPath string) error {
	// deletes outPath if it doesn't already exist
	if _, err := os.Stat(outPath); !os.IsNotExist(err) {
		err := os.RemoveAll(outPath)
		if err != nil {
			return err
		}
	}

	err := os.Mkdir(outPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	// static
	staticPath := path.Join(wc.path, "static")
	staticPathOut := path.Join(outPath, "static")
	var staticFilePaths map[string]string

	if _, err := os.Stat(staticPath); !os.IsNotExist(err) {
		err := os.Mkdir(staticPathOut, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}

		staticFilePaths, err = processFilesToDirRec(staticPath, staticPathOut)
		if err != nil {
			return err
		}
	}

	// posts
	postsPath := path.Join(wc.path, "posts")
	postsFileInfos, err := ioutil.ReadDir(postsPath)
	if err != nil {
		return err
	}
	visiblePostsByLangTag := make(map[string][]*post)
	invisiblePostsByLangTag := make(map[string][]*post)

	for _, postsFileInfo := range postsFileInfos {
		if !postsFileInfo.IsDir() {
			continue
		}

		postSlug := postsFileInfo.Name()
		postDirPath := path.Join(postsPath, postSlug)

		// data.yaml file
		postYAMLDataFile, err := os.Open(path.Join(postDirPath, "data.yaml"))
		if err != nil {
			return fmt.Errorf("opening %v data.yaml: %v", postSlug, err)
		}

		var postYAMLData postYAMLDataFileContent
		err = yaml.NewDecoder(postYAMLDataFile).Decode(&postYAMLData)
		if err != nil {
			return fmt.Errorf("decoding %v data.yaml: %v", postSlug, err)
		}

		postDate, err := time.Parse(time.RFC3339, postYAMLData.Date)
		if err != nil {
			return fmt.Errorf("parsing %v data.yaml date: %v", postSlug, err)
		}

		var postLastUpdateDate time.Time

		if postYAMLData.LastUpdateDate != "" {
			postLastUpdateDate, err = time.Parse(time.RFC3339, postYAMLData.LastUpdateDate)
			if err != nil {
				return fmt.Errorf("parsing %v data.yaml lastUpdateDate: %v", postSlug, err)
			}
		}

		postKeywords := strings.Split(postYAMLData.Keywords, ", ")

		// content_*.md files
		for _, l := range wc.langs {
			var postURL string

			if l.Default {
				postURL = fmt.Sprintf("/posts/%v", postSlug)
			} else {
				postURL = fmt.Sprintf("/%v/posts/%v", l.Tag, postSlug)
			}

			p := post{
				Slug:           postSlug,
				Keywords:       postKeywords,
				Date:           postDate,
				LastUpdateDate: postLastUpdateDate,
				Lang:           l,
				URL:            postURL,
			}

			postContentFilename := "content_" + l.Tag + ".md"
			postContentFilePath := path.Join(postDirPath, postContentFilename)
			postContent, err := ioutil.ReadFile(postContentFilePath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("%v for %v doesn't exist", postContentFilename, postSlug)
				}

				return err
			}
			if !postContentRegExp.Match(postContent) {
				return fmt.Errorf("post content at %v is invalid", postContentFilePath)
			}

			matchesIndexes := postContentRegExp.FindSubmatchIndex(postContent)
			postContentYAML := postContent[matchesIndexes[2]:matchesIndexes[3]]
			postContentMD := postContent[matchesIndexes[4]:matchesIndexes[5]]

			// yaml
			var yamlData postYAMLFrontMatter
			err = yaml.Unmarshal(postContentYAML, &yamlData)
			if err != nil {
				return fmt.Errorf("parsing YAML content of %v: %v", postContentFilePath, err)
			}

			p.Title = yamlData.Title
			p.Excerpt = yamlData.Excerpt

			// markdown
			mdParser := parser.New()
			p.Content = template.HTML(string(markdown.ToHTML(postContentMD, mdParser, nil)))

			if postYAMLData.Feed {
				if visiblePostsByLangTag[l.Tag] == nil {
					visiblePostsByLangTag[l.Tag] = make([]*post, 0, 1)
				}

				visiblePostsByLangTag[l.Tag] = append(visiblePostsByLangTag[l.Tag], &p)
			} else {
				if invisiblePostsByLangTag[l.Tag] == nil {
					invisiblePostsByLangTag[l.Tag] = make([]*post, 0, 1)
				}

				invisiblePostsByLangTag[l.Tag] = append(invisiblePostsByLangTag[l.Tag], &p)
			}
		}
	}

	// funcs
	funcs := template.FuncMap{
		"staticLink": func(filepath string) string {
			if newFilePath, ok := staticFilePaths[filepath]; ok {
				return "/static/" + newFilePath
			}

			return ""
		},
		"postLinkBySlugAndLang": func(slug string, l *lang) string {
			return fmt.Sprintf("/%v/posts/%v", l.Tag, slug)
		},
		"relToAbsLink": func(link string) string {
			if link == "/" {
				return wc.url
			}

			return wc.url + link
		},
	}

	// templates
	baseTemplate := template.Must(template.New("base").Funcs(funcs).Parse(indexHTML))

	// includes
	includesPath := path.Join(wc.path, "includes")
	includesFileInfos, err := ioutil.ReadDir(includesPath)
	if err != nil {
		return err
	}

	for _, includeFileInfo := range includesFileInfos {
		if includeFileInfo.IsDir() || !htmlFilenameRegExp.MatchString(includeFileInfo.Name()) {
			continue
		}

		includeFileContent, err := ioutil.ReadFile(path.Join(includesPath, includeFileInfo.Name()))
		if err != nil {
			return err
		}

		baseTemplate, err = baseTemplate.Parse(
			fmt.Sprintf(
				`{{ define "%v" }}%v{{ end }}`,
				strings.TrimRight(includeFileInfo.Name(), ".html"),
				string(includeFileContent),
			),
		)
		if err != nil {
			return err
		}
	}

	// creates a head template if one wasn't present in includes
	if t := baseTemplate.Lookup("head"); t == nil {
		baseTemplate = template.Must(baseTemplate.Parse(`{{ define "head" }}{{ end }}`))
	}

	// pages
	pagesPath := path.Join(wc.path, "pages")

	// home page
	homePageContent, err := ioutil.ReadFile(path.Join(pagesPath, "home.html"))
	if err != nil {
		return err
	}

	homePageTemplate := template.Must(
		template.Must(baseTemplate.Clone()).Parse(`{{ define "content" }}` + string(homePageContent) + `{{ end }}`),
	)

	// post page
	postPageContent, err := ioutil.ReadFile(path.Join(pagesPath, "post.html"))
	if err != nil {
		return err
	}

	postPageTemplate := template.Must(
		template.Must(baseTemplate.Clone()).Parse(`{{ define "content" }}` + string(postPageContent) + `{{ end }}`),
	)

	// executing templates per lang
	for _, l := range wc.langs {
		data := templateData{
			Posts: visiblePostsByLangTag[l.Tag],
			Lang:  l,
		}

		langOutPath := outPath
		if !l.Default {
			langOutPath = path.Join(outPath, l.Tag)
			err := os.Mkdir(langOutPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return err
			}
		}

		// home page
		data.Title = wc.title
		data.URL = "/"

		// alternate links
		data.AlternateLinks = make([]*alternateLink, 0, len(wc.langs))

		// default lang is always the first
		data.AlternateLinks = append(data.AlternateLinks, &alternateLink{
			URL:  "/",
			Lang: wc.defaultLang,
		})

		for _, l2 := range wc.langs {
			if l2.Default {
				continue
			}

			data.AlternateLinks = append(data.AlternateLinks, &alternateLink{
				Lang: l2,
				URL:  "/" + l2.Tag,
			})
		}

		homePageOutPathFile, err := os.Create(path.Join(langOutPath, "index.html"))
		if err != nil {
			return err
		}

		err = homePageTemplate.ExecuteTemplate(gohtml.NewWriter(homePageOutPathFile), "index", data)
		if err != nil {
			homePageOutPathFile.Close()
			return err
		}

		homePageOutPathFile.Close()

		postsDirOutPath := path.Join(langOutPath, "posts")
		err = os.Mkdir(postsDirOutPath, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}

		// post page
		for _, p := range visiblePostsByLangTag[l.Tag] {
			postDirPath := path.Join(postsDirOutPath, p.Slug)
			err = os.Mkdir(postDirPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return err
			}

			data.Title = fmt.Sprintf("%v - %v", p.Title, wc.title)
			data.URL = "/posts/" + p.Slug

			// alternate links
			data.AlternateLinks = make([]*alternateLink, 0, len(wc.langs)-1)

			// default lang is always the first
			data.AlternateLinks = append(data.AlternateLinks, &alternateLink{
				URL:  "/posts/" + p.Slug,
				Lang: wc.defaultLang,
			})

			for _, l2 := range wc.langs {
				if l2.Default {
					continue
				}

				data.AlternateLinks = append(data.AlternateLinks, &alternateLink{
					Lang: l2,
					URL:  fmt.Sprintf("/%v/posts/%v", l2.Tag, p.Slug),
				})
			}

			data.Post = p

			postPageOutPathFile, err := os.Create(path.Join(postDirPath, "index.html"))
			if err != nil {
				return err
			}

			err = postPageTemplate.ExecuteTemplate(gohtml.NewWriter(postPageOutPathFile), "index", data)
			if err != nil {
				postPageOutPathFile.Close()
				return err
			}

			postPageOutPathFile.Close()
		}
	}

	return nil
}

// processFilesToDirRec takes every file from inDirPath (recursively), do any
// processing related to the file (e.g. adding a hash to the file's name,
// minifying etc) and copies it to outDirPath. It returns a map of old filename
// path to new filename path. Both paths are relative to inDirPath and outDirPath,
// respectively.
func processFilesToDirRec(inDirPath, outDirPath string) (map[string]string, error) {
	filePaths := make(map[string]string)
	fileInfos, err := ioutil.ReadDir(inDirPath)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			dirFilePaths, err := processFilesToDirRec(
				path.Join(inDirPath, fileInfo.Name()),
				path.Join(outDirPath, fileInfo.Name()),
			)
			if err != nil {
				return nil, err
			}

			for oldFilePath, newFilePath := range dirFilePaths {
				filePaths[path.Join(fileInfo.Name(), oldFilePath)] = path.Join(fileInfo.Name(), newFilePath)
			}
		}

		file, err := os.Open(path.Join(inDirPath, fileInfo.Name()))
		if err != nil {
			return nil, err
		}

		fileContent, err := ioutil.ReadAll(file)
		if err != nil {
			file.Close()
			return nil, err
		}

		file.Close()

		ext := filepath.Ext(fileInfo.Name())
		filenameWithoutExt := strings.TrimSuffix(fileInfo.Name(), ext)

		// minifying
		m := minify.New()
		m.AddFunc("text/css", css.Minify)

		fileContentOut := fileContent

		switch ext {
		case ".css":
			fileContentOut, err = m.Bytes("text/css", fileContent)
			if err != nil {
				return nil, err
			}
		}

		// md5 hash
		md5HashBs := md5.Sum(fileContentOut)
		md5Hash := hex.EncodeToString(md5HashBs[:])

		newFilename := filenameWithoutExt + "-" + string(md5Hash[:]) + ext
		fileOut, err := os.Create(path.Join(outDirPath, newFilename))
		if err != nil {
			return nil, err
		}

		// writing to new file
		_, err = fileOut.Write(fileContentOut)
		if err != nil {
			fileOut.Close()
			return nil, err
		}

		fileOut.Close()

		filePaths[fileInfo.Name()] = newFilename
	}

	return filePaths, nil
}
