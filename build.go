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
		return errors.New("InPath not provided")
	}

	if bc.OutPath == "" {
		return errors.New("OutPath not provided")
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

	chromaNode := gat.addChild(FILENODE, "chroma.css")
	chromaNode.setContent(chromaStylesBuff.Bytes())

	// assets out
	assetsOutPath := path.Join(bc.OutPath, "assets")

	err = os.Mkdir(assetsOutPath, os.ModeDir|os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating %v: %v", assetsOutPath, err)
	}

	// process gat
	err = gat.processCSSFileNodes()
	if err != nil {
		return err
	}

	err = gat.process(assetsOutPath, false)
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
		c.ResponsiveImgMediaQueries,
		c.ResponsiveImgSizes,
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
		c.ResponsiveImgSizes,
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

	// 404 page
	notFoundPageTemplate, err := createPageTemplate(pagesInPath, baseTemplate, "404")
	if err != nil {
		return err
	}

	// executing templates per lang
	for _, l := range c.Langs {
		langOutPath := bc.OutPath
		if !l.Default {
			langOutPath = path.Join(langOutPath, l.Tag)
			if err := os.Mkdir(langOutPath, os.ModeDir|os.ModePerm); err != nil {
				return err
			}
		}

		// home page
		homePageTemplateData := TemplateData{
			Posts:                     visiblePostsByLangTag[l.Tag],
			Lang:                      l,
			Author:                    c.Author,
			Color:                     c.Color,
			ResponsiveImgMediaQueries: c.ResponsiveImgMediaQueries,
			Title:                     c.Title,
			Description:               c.Description[l.Tag],
			Page:                      "home",
			Img:                       c.defaultImgByLangTag[l.Tag],
		}

		homePageTemplateData.AlternateLinks = generateAlternateLinks(nil, nil, c.Langs)

		if l.Default {
			homePageTemplateData.URL = "/"
		} else {
			homePageTemplateData.URL = "/" + l.Tag
		}

		err := executeMinifyAndWriteTemplate(homePageTemplate, homePageTemplateData, path.Join(langOutPath, "index.html"))
		if err != nil {
			return err
		}

		// 404 page
		// only execute the 404 page's template if it's the default language.
		if l.Default {
			notFoundPageTemplateData := TemplateData{
				Color:                     c.Color,
				Author:                    c.Author,
				Description:               c.Description[l.Tag],
				Img:                       c.defaultImgByLangTag[l.Tag],
				Lang:                      l,
				Page:                      "404",
				Posts:                     visiblePostsByLangTag[l.Tag],
				Title:                     fmt.Sprintf("Not found - %v", c.Title),
				ResponsiveImgMediaQueries: c.ResponsiveImgMediaQueries,
				URL:                       "/404.html",
			}

			err := executeMinifyAndWriteTemplate(notFoundPageTemplate, notFoundPageTemplateData, path.Join(langOutPath, "404.html"))
			if err != nil {
				return err
			}
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

				postPageTemplateData := TemplateData{
					Title:                     fmt.Sprintf("%v - %v", p.Title, c.Title),
					Description:               p.Excerpt,
					Page:                      "post",
					Color:                     c.Color,
					Post:                      p,
					Lang:                      l,
					Author:                    c.Author,
					ResponsiveImgMediaQueries: c.ResponsiveImgMediaQueries,
					Posts:                     visiblePostsByLangTag[l.Tag],
				}

				postPageTemplateData.AlternateLinks = generateAlternateLinks(nil, []string{"posts", p.Slug}, c.Langs)

				if l.Default {
					postPageTemplateData.URL = "/posts/" + p.Slug
				} else {
					postPageTemplateData.URL = "/" + l.Tag + "/posts/" + p.Slug
				}

				if p.Img != nil {
					postPageTemplateData.Img = p.Img
				} else {
					postPageTemplateData.Img = c.defaultImgByLangTag[l.Tag]
				}

				postPageTemplate.Funcs(map[string]interface{}{
					"assetLink":   generateAssetsLinkFn(gat, p.pat, p.Slug),
					"srcSetValue": generateSrcSetValueFn(gat, p.pat, p.Slug, c.ResponsiveImgSizes),
					"hasAsset":    generateHasAsset(gat, p.pat, p.Slug),
				})

				err = executeMinifyAndWriteTemplate(postPageTemplate, postPageTemplateData, path.Join(postDirPath, "index.html"))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
