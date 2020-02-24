package egen

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"

	"github.com/alecthomas/chroma"
	chromaHTML "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/styles"
)

// BuildConfig is the config used to build a blog.
type BuildConfig struct {
	InPath, OutPath string
	TemplateFuncs   template.FuncMap
	ChromaStyle     *chroma.Style
}

// Build builds the blog.
func Build(bc BuildConfig) error {
	if bc.InPath == "" {
		return errors.New("inPath not provided")
	}

	if bc.OutPath == "" {
		return errors.New("outPath not provided")
	}

	// deletes bc.OutPath if it already exists
	if _, err := os.Stat(bc.OutPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		err := os.RemoveAll(bc.OutPath)
		if err != nil {
			return fmt.Errorf("removing %v and its contents: %v", bc.OutPath, err)
		}
	}

	// creates bc.OutPath
	err := os.Mkdir(bc.OutPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	// config file
	c, err := readConfigFile(bc.InPath)
	if err != nil {
		return err
	}

	// assets in
	assetsPath := path.Join(bc.InPath, "assets")
	gat, err := generateAssetsTree(assetsPath, nil)
	if err != nil {
		return fmt.Errorf("reading %v: %v", assetsPath, err)
	}

	// chroma styles
	var chromaStylesBuff bytes.Buffer

	chromaStyle := bc.ChromaStyle
	if chromaStyle == nil {
		chromaStyle = styles.Get("swapoff")
	}

	if err := chromaHTML.New().WriteCSS(&chromaStylesBuff, chromaStyle); err != nil {
		return err
	}

	chromaNode := gat.AddChild(FILENODE, "chroma.css")
	chromaNode.SetContent(chromaStylesBuff.Bytes())

	// assets out
	assetsOutPath := path.Join(bc.OutPath, "assets")

	err = os.Mkdir(assetsOutPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating %v: %v", assetsOutPath, err)
	}

	// process gat
	err = bundleCSSFilesInAT(gat)
	if err != nil {
		return err
	}

	err = processAT(gat, assetsOutPath, false)
	if err != nil {
		return err
	}

	// posts
	allPostsByLangTag, visiblePostsByLangTag, invisiblePostsByLangTag, err := generatePostsLists(
		gat,
		path.Join(bc.InPath, "posts"),
		c.Langs,
		assetsOutPath,
		chromaStyle,
		c.PostImgMediaQuery,
		c.PostImgSizes,
	)
	if err != nil {
		return err
	}

	// base template
	baseTemplate, err := createBaseTemplateWithIncludes(
		bc.TemplateFuncs,
		path.Join(bc.InPath, "includes"),
		invisiblePostsByLangTag,
		gat,
		c.URL,
	)
	if err != nil {
		return err
	}

	pagesInPath := path.Join(bc.InPath, "pages")

	// home page
	homePageTemplate, err := createPageTemplate(pagesInPath, baseTemplate, "home")
	if err != nil {
		return err
	}

	// post page
	postPageTemplate, err := createPageTemplate(pagesInPath, baseTemplate, "post")
	if err != nil {
		return err
	}

	// executing templates per lang
	for _, l := range c.Langs {
		data := TemplateData{
			Posts:  visiblePostsByLangTag[l.Tag],
			Lang:   l,
			Author: c.Author,
			Color:  c.Color,
		}

		langOutPath := bc.OutPath
		if !l.Default {
			langOutPath = path.Join(langOutPath, l.Tag)
			if err := os.Mkdir(langOutPath, os.ModeDir|os.ModePerm); err != nil {
				return err
			}
		}

		// home page
		data.Title = c.Title
		data.Description = c.Description[l.Tag]
		data.Page = "home"
		data.Img = c.defaultImgByLangTag[l.Tag]

		if l.Default {
			data.URL = "/"
		} else {
			data.URL = "/" + l.Tag
		}

		// alternate links
		data.AlternateLinks = generateAlternateLinks(nil, nil, c.Langs)

		err := executePrettifyAndWriteTemplate(homePageTemplate, data, path.Join(langOutPath, "index.html"))
		if err != nil {
			return err
		}

		// post page
		if len(visiblePostsByLangTag) > 0 || len(invisiblePostsByLangTag) > 0 {
			postsDirOutPath := path.Join(langOutPath, "posts")
			err = os.Mkdir(postsDirOutPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return err
			}

			for _, p := range allPostsByLangTag[l.Tag] {
				postDirPath := path.Join(postsDirOutPath, p.Slug)
				err := os.Mkdir(postDirPath, os.ModeDir|os.ModePerm)
				if err != nil {
					return err
				}

				data := TemplateData{
					Title:       fmt.Sprintf("%v - %v", p.Title, c.Title),
					Description: p.Excerpt,
					Page:        "post",
					Color:       c.Color,
					Post:        p,
					Lang:        l,
					Author:      c.Author,
				}

				if l.Default {
					data.URL = "/posts/" + p.Slug
				} else {
					data.URL = "/" + l.Tag + "/posts/" + p.Slug
				}

				if p.Img != nil {
					data.Img = p.Img
				} else {
					data.Img = c.defaultImgByLangTag[l.Tag]
				}

				// alternate links
				data.AlternateLinks = generateAlternateLinks(nil, []string{"posts", p.Slug}, c.Langs)

				postPageTemplate.Funcs(map[string]interface{}{
					"assetsLink": generateAssetsLinkFn(gat, p.pat, p.Slug),
					"hasAsset":   generateHasAsset(gat, p.pat, p.Slug),
				})

				err = executePrettifyAndWriteTemplate(postPageTemplate, data, path.Join(postDirPath, "index.html"))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
