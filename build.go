package egen

import (
	"bytes"
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
	Funcs           template.FuncMap
	ChromaStyles    *chroma.Style
	PreGATProc      func(gat *AssetsTreeNode)
	PrePWATProc     func(postSlug string, pwat *AssetsTreeNode)
}

// buildData is the data used by functions called by build.
type buildData struct {
	bc          *BuildConfig
	c           *config
	gat         *AssetsTreeNode
	chromaStyle *chroma.Style
}

// Build builds the blog.
func Build(bc BuildConfig) error {
	// config file
	c, err := readConfigFile(bc.InPath)
	if err != nil {
		return err
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
	err = os.Mkdir(bc.OutPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	// assets
	assetsPath := path.Join(bc.InPath, "assets")
	assetsPathOut := path.Join(bc.OutPath, "assets")

	err = os.Mkdir(assetsPathOut, os.ModeDir|os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating %v: %v", assetsPathOut, err)
	}

	gat, err := generateAssetsTree(assetsPath, nil)
	if err != nil {
		return fmt.Errorf("reading %v: %v", assetsPath, err)
	}

	// chroma styles
	var chromaStylesBuff bytes.Buffer

	chromaStyle := bc.ChromaStyles
	if chromaStyle == nil {
		chromaStyle = styles.Get("swapoff")
	}

	if err := chromaHTML.New().WriteCSS(&chromaStylesBuff, chromaStyle); err != nil {
		return err
	}

	chromaNode := gat.AddChild(FILENODE, "chroma.css")
	chromaNode.SetContent(chromaStylesBuff.Bytes())

	// build data
	bd := buildData{
		bc:          &bc,
		c:           c,
		gat:         gat,
		chromaStyle: chromaStyle,
	}

	// process assets
	if bc.PreGATProc != nil {
		bc.PreGATProc(gat)
	}

	err = processAT(gat, assetsPathOut)
	if err != nil {
		return err
	}

	// base template
	baseTemplate, err := createBaseTemplateWithIncludes(bd)
	if err != nil {
		return err
	}

	// home page
	homePageTemplate, err := createPageTemplate(bd, baseTemplate, "home")
	if err != nil {
		return err
	}

	// post page
	postPageTemplate, err := createPageTemplate(bd, baseTemplate, "post")
	if err != nil {
		return err
	}

	// posts
	visiblePostsByLangTag, invisiblePostsByLangTag, err := generatePostsLists(bd)
	if err != nil {
		return err
	}

	// executing templates per lang
	for _, l := range c.Langs {
		data := TemplateData{
			Posts:  visiblePostsByLangTag[l.Tag],
			Lang:   l,
			Author: c.Author,
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

			err := executePostTemplateForEachPost(bd, postsDirOutPath, postPageTemplate, l, visiblePostsByLangTag[l.Tag])
			if err != nil {
				return err
			}

			err = executePostTemplateForEachPost(bd, postsDirOutPath, postPageTemplate, l, invisiblePostsByLangTag[l.Tag])
			if err != nil {
				return err
			}
		}
	}

	return nil
}
