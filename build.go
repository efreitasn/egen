package egen

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	chromaHTML "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/efreitasn/egen/htmlp"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"gopkg.in/russross/blackfriday.v2"
	"gopkg.in/yaml.v2"
)

var configFilename = "egen.yaml"
var mdCodeBlockInfoRegExp = regexp.MustCompile("^((?:[a-z]|[0-9])+?)(?:{((?:\\[[0-9]{1,},[0-9]{1,}\\])(?:(?:,\\[[0-9]{1,},[0-9]{1,}\\])+)?)})?$")
var mdCodeBlockInfoHLinesRegExp = regexp.MustCompile("\\[([0-9]{1,}),([0-9]{1,})\\]")
var postContentRegExp = regexp.MustCompile("(?s)^---\n(.*?)\n---(.*)")
var htmlFilenameRegExp = regexp.MustCompile(".*\\.html")
var indexHTML = `
{{ define "index" -}}
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
		<meta property="og:image:url" content="{{ relToAbsLink (staticLink .Img.Name) }}">
		<meta property="og:image:width" content="{{ .Img.Width }}">
		<meta property="og:image:height" content="{{ .Img.Height }}">
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
  {{ template "head" . }}
	<link rel="stylesheet" href="{{ staticLink "chroma.css" }}">
</head>
<body>
  {{ template "content" . }}
</body>
</html>
{{- end }}
`

type configFileDataTextByLang struct {
	Lang string
	Text string
}

type configFileData struct {
	Title       string
	Description []configFileDataTextByLang
	ImgAlt      []configFileDataTextByLang `yaml:"imgAlt"`
	URL         string
	Img         string
	Langs       []*Lang
	Author      *Author
	Keywords    map[string]map[string]string
}

// Author represents an author.
type Author struct {
	Name, Twitter string
}

// Lang represents a language.
type Lang struct {
	Name string
	// The language tag, as in RFC 5646.
	Tag     string
	Default bool
}

// TemplateDataImg represents the image of the current page/post.
type TemplateDataImg struct {
	Name   string
	Alt    string
	Width  int
	Height int
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
	Img         *TemplateDataImg
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

// Post is a post received by a template.
type Post struct {
	Title          string
	Content        template.HTML
	Slug           string
	Excerpt        string
	Img            *TemplateDataImg
	Keywords       []string
	Date           time.Time
	LastUpdateDate time.Time
	Lang           *Lang
	// relative
	URL string
}

type postYAMLFrontMatter struct {
	Title   string `yaml:"title"`
	Excerpt string `yaml:"excerpt"`
	ImgAlt  string `yaml:"imgAlt"`
}

type postYAMLDataFileContent struct {
	Keywords       []string `yaml:"keywords"`
	Feed           bool     `yaml:"feed"`
	Date           string   `yaml:"date"`
	LastUpdateDate string   `yaml:"lastUpdateDate"`
	Img            string
}

// AlternateLink is a link to a version of the current page in another language.
type AlternateLink struct {
	// relative
	URL  string
	Lang *Lang
}

// BuildConfig is the config used to build a blog.
type BuildConfig struct {
	InPath, OutPath string
	Funcs           template.FuncMap
}

// Build builds the blog.
func Build(bc BuildConfig) error {
	// config file
	cFile, err := os.Open(path.Join(bc.InPath, configFilename))
	if err != nil {
		return err
	}

	var cFileData configFileData

	err = yaml.NewDecoder(cFile).Decode(&cFileData)
	if err != nil {
		return err
	}

	var defaultLang *Lang

	for _, lang := range cFileData.Langs {
		if lang.Default {
			defaultLang = lang
			break
		}
	}

	descriptionByLangTag := make(map[string]string, len(cFileData.Description))

	for _, d := range cFileData.Description {
		descriptionByLangTag[d.Lang] = d.Text
	}

	// deletes bc.OutPath if it already exists
	if _, err := os.Stat(bc.OutPath); !os.IsNotExist(err) {
		err := os.RemoveAll(bc.OutPath)
		if err != nil {
			return err
		}
	}

	err = os.Mkdir(bc.OutPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	// static

	staticPath := path.Join(bc.InPath, "static")
	staticPathOut := path.Join(bc.OutPath, "static")

	chromaStyleFilePath := path.Join(staticPath, "chroma.css")
	chromaStyleFile, err := os.Create(chromaStyleFilePath)
	if err != nil {
		return err
	}

	chromaStyle := styles.Get("swapoff")
	f := chromaHTML.New()
	err = f.WriteCSS(chromaStyleFile, chromaStyle)
	if err != nil {
		return err
	}

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

	err = os.Remove(chromaStyleFilePath)
	if err != nil {
		return err
	}

	// img
	defaultImgByLangTag := make(map[string]*TemplateDataImg, len(cFileData.ImgAlt))

	if cFileData.Img != "" {
		defaultImgWidth, defaultImgHeight, err := imgDimensions(path.Join(staticPath, cFileData.Img))
		if err != nil {
			return err
		}

		for _, ia := range cFileData.ImgAlt {
			defaultImgByLangTag[ia.Lang] = &TemplateDataImg{
				Name:   cFileData.Img,
				Alt:    ia.Text,
				Width:  defaultImgWidth,
				Height: defaultImgHeight,
			}
		}
	}

	// posts
	postsPath := path.Join(bc.InPath, "posts")
	postsFileInfos, err := ioutil.ReadDir(postsPath)
	if err != nil {
		return err
	}
	visiblePostsByLangTag := make(map[string][]*Post)
	invisiblePostsByLangTag := make(map[string][]*Post)

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

		var postImg *TemplateDataImg

		if postYAMLData.Img != "" {
			postImgWidth, postImgHeight, err := imgDimensions(path.Join(staticPath, postYAMLData.Img))
			if err != nil {
				return err
			}

			postImg = &TemplateDataImg{
				Name:   postYAMLData.Img,
				Width:  postImgWidth,
				Height: postImgHeight,
			}
		}

		// content_*.md files
		for _, l := range cFileData.Langs {
			var postURL string

			if l.Default {
				postURL = fmt.Sprintf("/posts/%v", postSlug)
			} else {
				postURL = fmt.Sprintf("/%v/posts/%v", l.Tag, postSlug)
			}

			keywords := make([]string, len(postYAMLData.Keywords))
			copy(keywords, postYAMLData.Keywords)

			for i, keyword := range keywords {
				kLangs, ok := cFileData.Keywords[keyword]
				if !ok {
					continue
				}

				k, ok := kLangs[l.Tag]
				if !ok {
					continue
				}

				keywords[i] = k
			}

			p := Post{
				Slug:           postSlug,
				Keywords:       keywords,
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

			if postImg != nil {
				p.Img = &TemplateDataImg{
					Name:   postImg.Name,
					Width:  postImg.Width,
					Height: postImg.Height,
					Alt:    yamlData.ImgAlt,
				}
			}

			// TODO extensions
			mdProcessor := blackfriday.New(blackfriday.WithExtensions(blackfriday.CommonExtensions))
			rootNode := mdProcessor.Parse(postContentMD)

			rootNode.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
				if node.Type == blackfriday.Image && entering {
					oldParent := node.Parent

					if oldParent.Type == blackfriday.Paragraph {
						newParent := oldParent.Parent

						// this should never happen
						if newParent.Type == blackfriday.Paragraph {
							node.Unlink()

							return blackfriday.GoToNext
						}

						oldParentChildren := getChildren(oldParent)
						nodeOldParentIndex := findIndex(node, oldParent)

						var oldParentChildrenAfterNode []*blackfriday.Node
						if nodeOldParentIndex+1 < len(oldParentChildren) {
							oldParentChildrenAfterNode = oldParentChildren[nodeOldParentIndex+1 : len(oldParentChildren)]
						}

						if oldParent.Next == nil {
							newParent.AppendChild(node)

							if oldParentChildrenAfterNode != nil {
								pNode := blackfriday.NewNode(blackfriday.Paragraph)

								for _, c := range oldParentChildrenAfterNode {
									pNode.AppendChild(c)
								}
								newParent.AppendChild(pNode)
							}
						} else {
							oldParentNewParentIndex := findIndex(oldParent, newParent)
							newParentChildren := getChildren(newParent)
							newParentChildrenAfterOldParent := newParentChildren[oldParentNewParentIndex+1 : len(newParentChildren)]

							newParent.AppendChild(node)

							if oldParentChildrenAfterNode != nil {
								pNode := blackfriday.NewNode(blackfriday.Paragraph)

								for _, c := range oldParentChildrenAfterNode {
									pNode.AppendChild(c)
								}

								newParent.AppendChild(pNode)
							}

							for _, c := range newParentChildrenAfterOldParent {
								newParent.AppendChild(c)
							}
						}
					}
				}

				return blackfriday.GoToNext
			})

			var htmlBuff bytes.Buffer

			// TODO flags
			r := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{})
			rootNode.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
				switch {
				case node.Type == blackfriday.CodeBlock && entering:
					if !mdCodeBlockInfoRegExp.Match(node.Info) {
						return blackfriday.GoToNext
					}

					cbInfoMatches := mdCodeBlockInfoRegExp.FindStringSubmatch(string(node.Info))
					lang := cbInfoMatches[1]

					hLines := make([][2]int, 0)

					if cbInfoMatches[2] != "" {
						hLinesMatches := mdCodeBlockInfoHLinesRegExp.FindAllStringSubmatch(cbInfoMatches[2], -1)

						for _, hLinesMatch := range hLinesMatches {
							startLine, err := strconv.Atoi(hLinesMatch[1])
							if err != nil {
								return blackfriday.GoToNext
							}

							endLine, err := strconv.Atoi(hLinesMatch[2])
							if err != nil {
								return blackfriday.GoToNext
							}

							hLines = append(hLines, [2]int{
								startLine,
								endLine,
							})
						}
					}

					lexer := lexers.Get(lang)
					if lexer == nil {
						return blackfriday.GoToNext
					}

					iterator, _ := lexer.Tokenise(nil, string(node.Literal))
					formatter := chromaHTML.New(
						chromaHTML.WithClasses(true),
						chromaHTML.WithLineNumbers(true),
						chromaHTML.HighlightLines(hLines),
					)

					err := formatter.Format(&htmlBuff, chromaStyle, iterator)
					if err != nil {
						return blackfriday.GoToNext
					}

					return blackfriday.GoToNext
				case node.Type == blackfriday.Image && entering:
					// the image element is only added if its alt
					// attribute has been set.
					if node.FirstChild != nil {
						title := string(node.Title)
						alt := string(node.FirstChild.Literal)

						src := string(node.LinkData.Destination)
						if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
							src = "/static/" + staticFilePaths[src]
						}

						img := fmt.Sprintf(`<img src="%v" alt="%v">`, src, alt)
						figcaption := ""

						if title != "" {
							figcaption = fmt.Sprintf("<figcaption>%v</figcaption>", title)
						}

						htmlBuff.WriteString(
							fmt.Sprintf("<figure>%v%v</figure>", img, figcaption),
						)
					}

					return blackfriday.SkipChildren
				default:
					return r.RenderNode(&htmlBuff, node, entering)
				}
			})

			// markdown
			p.Content = template.HTML(
				string(
					bytes.ReplaceAll(
						htmlBuff.Bytes(),
						[]byte(`<pre`),
						[]byte(`<pre data-htmlp-ignore`),
					),
				),
			)

			if postYAMLData.Feed {
				if visiblePostsByLangTag[l.Tag] == nil {
					visiblePostsByLangTag[l.Tag] = make([]*Post, 0, 1)
				}

				visiblePostsByLangTag[l.Tag] = append(visiblePostsByLangTag[l.Tag], &p)
			} else {
				if invisiblePostsByLangTag[l.Tag] == nil {
					invisiblePostsByLangTag[l.Tag] = make([]*Post, 0, 1)
				}

				invisiblePostsByLangTag[l.Tag] = append(invisiblePostsByLangTag[l.Tag], &p)
			}
		}
	}

	// funcs
	var funcs template.FuncMap

	if bc.Funcs != nil {
		funcs = bc.Funcs
	} else {
		funcs = make(template.FuncMap, 3)
	}

	funcs["dateISO"] = func(d time.Time) string {
		return d.Format(time.RFC3339)
	}

	funcs["staticLink"] = func(filepath string) string {
		if newFilePath, ok := staticFilePaths[filepath]; ok {
			return "/static/" + newFilePath
		}

		return ""
	}

	funcs["postLinkBySlugAndLang"] = func(slug string, l *Lang) string {
		return fmt.Sprintf("/%v/posts/%v", l.Tag, slug)
	}

	funcs["relToAbsLink"] = func(link string) string {
		if link == "/" {
			return cFileData.URL
		}

		return cFileData.URL + link
	}

	// templates
	baseTemplate := template.Must(template.New("base").Funcs(funcs).Parse(indexHTML))

	// includes
	includesPath := path.Join(bc.InPath, "includes")
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
	pagesPath := path.Join(bc.InPath, "pages")

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
	for _, l := range cFileData.Langs {
		data := TemplateData{
			Posts:  visiblePostsByLangTag[l.Tag],
			Lang:   l,
			Author: cFileData.Author,
		}

		langOutPath := bc.OutPath
		if !l.Default {
			langOutPath = path.Join(bc.OutPath, l.Tag)
			err := os.Mkdir(langOutPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return err
			}
		}

		// home page
		data.Title = cFileData.Title
		data.Description = descriptionByLangTag[l.Tag]
		data.Page = "home"
		data.Img = defaultImgByLangTag[l.Tag]

		if l.Default {
			data.URL = "/"
		} else {
			data.URL = "/" + l.Tag
		}

		// alternate links
		data.AlternateLinks = make([]*AlternateLink, 0, len(cFileData.Langs))

		// default lang is always the first
		data.AlternateLinks = append(data.AlternateLinks, &AlternateLink{
			URL:  "/",
			Lang: defaultLang,
		})

		for _, l2 := range cFileData.Langs {
			if l2.Default {
				continue
			}

			data.AlternateLinks = append(data.AlternateLinks, &AlternateLink{
				Lang: l2,
				URL:  "/" + l2.Tag,
			})
		}

		homePageOutPathFile, err := os.Create(path.Join(langOutPath, "index.html"))
		if err != nil {
			return err
		}

		var buff bytes.Buffer
		err = homePageTemplate.ExecuteTemplate(&buff, "index", data)
		if err != nil {
			homePageOutPathFile.Close()
			return err
		}

		htmlPretty, err := htmlp.Pretty(buff.Bytes())
		if err != nil {
			return err
		}
		homePageOutPathFile.Write(htmlPretty)

		homePageOutPathFile.Close()

		postsDirOutPath := path.Join(langOutPath, "posts")
		err = os.Mkdir(postsDirOutPath, os.ModeDir|os.ModePerm)
		if err != nil {
			return err
		}

		doPost := func(p *Post) error {
			postDirPath := path.Join(postsDirOutPath, p.Slug)
			err = os.Mkdir(postDirPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return err
			}

			data.Title = fmt.Sprintf("%v - %v", p.Title, cFileData.Title)
			data.Description = p.Excerpt
			data.Page = "post"
			data.Post = p

			if l.Default {
				data.URL = "/posts/" + p.Slug
			} else {
				data.URL = "/" + l.Tag + "/posts/" + p.Slug
			}

			if p.Img != nil {
				data.Img = p.Img
			} else {
				data.Img = defaultImgByLangTag[l.Tag]
			}

			// alternate links
			data.AlternateLinks = make([]*AlternateLink, 0, len(cFileData.Langs)-1)

			// default lang is always the first
			data.AlternateLinks = append(data.AlternateLinks, &AlternateLink{
				URL:  "/posts/" + p.Slug,
				Lang: defaultLang,
			})

			for _, l2 := range cFileData.Langs {
				if l2.Default {
					continue
				}

				data.AlternateLinks = append(data.AlternateLinks, &AlternateLink{
					Lang: l2,
					URL:  fmt.Sprintf("/%v/posts/%v", l2.Tag, p.Slug),
				})
			}

			postPageOutPathFile, err := os.Create(path.Join(postDirPath, "index.html"))
			if err != nil {
				return err
			}

			var buff bytes.Buffer
			err = postPageTemplate.ExecuteTemplate(&buff, "index", data)
			if err != nil {
				postPageOutPathFile.Close()
				return err
			}

			htmlPretty, err := htmlp.Pretty(buff.Bytes())
			if err != nil {
				return err
			}
			postPageOutPathFile.Write(htmlPretty)

			postPageOutPathFile.Close()

			return nil
		}

		// post page
		for _, p := range visiblePostsByLangTag[l.Tag] {
			err := doPost(p)
			if err != nil {
				return err
			}
		}

		for _, p := range invisiblePostsByLangTag[l.Tag] {
			err := doPost(p)
			if err != nil {
				return err
			}
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

func findIndex(node *blackfriday.Node, parent *blackfriday.Node) int {
	children := getChildren(parent)
	if len(children) == 0 {
		return -1
	}

	for i, c := range children {
		if c == node {
			return i
		}
	}

	return -1
}

func getChildren(n *blackfriday.Node) []*blackfriday.Node {
	if n == nil || n.FirstChild == nil {
		return nil
	}

	children := make([]*blackfriday.Node, 0)

	c := n.FirstChild
	for c != nil {
		children = append(children, c)
		c = c.Next
	}

	return children
}
