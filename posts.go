package egen

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	chromaHTML "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"gopkg.in/russross/blackfriday.v2"
	"gopkg.in/yaml.v2"
)

var mdCodeBlockInfoRegExp = regexp.MustCompile("^((?:[a-z]|[0-9])+?)(?:{((?:\\[[0-9]{1,},[0-9]{1,}\\])(?:(?:,\\[[0-9]{1,},[0-9]{1,}\\])+)?)})?$")
var mdCodeBlockInfoHLinesRegExp = regexp.MustCompile("\\[([0-9]{1,}),([0-9]{1,})\\]")
var postContentRegExp = regexp.MustCompile("(?s)^---\n(.*?)\n---(.*)")

var nonPostAssetsRxs = []*regexp.Regexp{
	regexp.MustCompile("content_.+\\.md"),
	regexp.MustCompile("data.yaml"),
	// ignore all directories
	regexp.MustCompile(".*/$"),
}

// Post is a post received by a template.
type Post struct {
	Title          string
	Content        template.HTML
	Slug           string
	Excerpt        string
	Img            *Img
	Date           time.Time
	LastUpdateDate time.Time
	Lang           *Lang
	// relative
	URL string
	// pat is a tree composed of any files in the post's path
	// whose name doesn't match any item in nonPostAssetsRxs.
	pat *AssetsTreeNode
}

type postYAMLFrontMatter struct {
	Title   string `yaml:"title"`
	Excerpt string `yaml:"excerpt"`
	ImgAlt  string `yaml:"imgAlt"`
}

type postYAMLDataFileContent struct {
	Feed           bool   `yaml:"feed"`
	Date           string `yaml:"date"`
	LastUpdateDate string `yaml:"lastUpdateDate"`
	Img            AssetRelPath
}

func generatePostsLists(
	gat *AssetsTreeNode,
	postsInPath string,
	langs []*Lang,
	assetsOutPath string,
	chromaStyle *chroma.Style,
) (allPostsByLangTag, visiblePostsByLangTag, invisiblePostsByLangTag map[string][]*Post, err error) {
	postsFileInfos, err := ioutil.ReadDir(postsInPath)
	if err != nil {
		return nil, nil, nil, err
	}

	allPostsByLangTag = make(map[string][]*Post)
	visiblePostsByLangTag = make(map[string][]*Post)
	invisiblePostsByLangTag = make(map[string][]*Post)

	for _, postsFileInfo := range postsFileInfos {
		if !postsFileInfo.IsDir() {
			continue
		}

		postSlug := postsFileInfo.Name()
		postDirPath := path.Join(postsInPath, postSlug)

		pat, err := generateAssetsTree(postDirPath, nonPostAssetsRxs)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("generating pat for %v: %v", postSlug, err)
		}

		// this condition exists so that assetsPathOut is only created if the post
		// has at least one asset.
		if pat.FirstChild != nil {
			assetsPathOut := path.Join(assetsOutPath, postSlug)

			// it's checked whether assetsPathOut already exists because it could've
			// been already created when generating the global assets tree (GAT) if
			// there's a directory in it whose name is the same as the post's slug.
			if _, err := os.Stat(assetsPathOut); err != nil {
				if os.IsNotExist(err) {
					err := os.Mkdir(assetsPathOut, os.ModeDir|os.ModePerm)
					if err != nil {
						return nil, nil, nil, fmt.Errorf("creating %v: %v", assetsPathOut, err)
					}
				} else {
					return nil, nil, nil, err
				}
			}

			if err = processAT(pat, assetsPathOut, false); err != nil {
				return nil, nil, nil, fmt.Errorf("processing pat: %v", err)
			}
		}

		// data.yaml file
		postYAMLDataFile, err := os.Open(path.Join(postDirPath, "data.yaml"))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("opening %v data.yaml: %v", postSlug, err)
		}

		var postYAMLData postYAMLDataFileContent
		err = yaml.NewDecoder(postYAMLDataFile).Decode(&postYAMLData)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("decoding %v data.yaml: %v", postSlug, err)
		}

		postDate, err := time.Parse(time.RFC3339, postYAMLData.Date)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parsing %v data.yaml date: %v", postSlug, err)
		}

		var postLastUpdateDate time.Time

		if postYAMLData.LastUpdateDate != "" {
			postLastUpdateDate, err = time.Parse(time.RFC3339, postYAMLData.LastUpdateDate)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("parsing %v data.yaml lastUpdateDate: %v", postSlug, err)
			}
		}

		// content_*.md files
		for _, l := range langs {
			var postURL string

			if l.Default {
				postURL = fmt.Sprintf("/posts/%v", postSlug)
			} else {
				postURL = fmt.Sprintf("/%v/posts/%v", l.Tag, postSlug)
			}

			p := Post{
				Slug:           postSlug,
				Date:           postDate,
				LastUpdateDate: postLastUpdateDate,
				Lang:           l,
				URL:            postURL,
				pat:            pat,
			}

			postContentFilename := "content_" + l.Tag + ".md"
			postContentFilePath := path.Join(postDirPath, postContentFilename)
			postContent, err := ioutil.ReadFile(postContentFilePath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, nil, nil, fmt.Errorf("%v for %v post doesn't exist", postContentFilename, postSlug)
				}

				return nil, nil, nil, err
			}
			if !postContentRegExp.Match(postContent) {
				return nil, nil, nil, fmt.Errorf("post content at %v is invalid", postContentFilePath)
			}

			matchesIndexes := postContentRegExp.FindSubmatchIndex(postContent)
			postContentYAML := postContent[matchesIndexes[2]:matchesIndexes[3]]
			postContentMD := postContent[matchesIndexes[4]:matchesIndexes[5]]

			// yaml
			var yamlData postYAMLFrontMatter
			err = yaml.Unmarshal(postContentYAML, &yamlData)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("parsing YAML content of %v: %v", postContentFilePath, err)
			}

			p.Title = yamlData.Title
			p.Excerpt = yamlData.Excerpt

			if postYAMLData.Img != "" {
				if yamlData.ImgAlt == "" {
					return nil, nil, nil, fmt.Errorf("img alt in %v for %v post not provided", l.Tag, p.Slug)
				}

				p.Img = &Img{
					Path: postYAMLData.Img,
					Alt:  yamlData.ImgAlt,
				}
			}

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

						oldParentChildren := getBFNodeChildren(oldParent)
						nodeOldParentIndex := findBFNodeIndex(node, oldParent)

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
							oldParentNewParentIndex := findBFNodeIndex(oldParent, newParent)
							newParentChildren := getBFNodeChildren(newParent)
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

						if len(oldParentChildren) == 1 {
							oldParent.Unlink()
						}
					}
				}

				return blackfriday.GoToNext
			})

			var htmlBuff bytes.Buffer

			r := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{})
			var bfTraverseErr error
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
						bfTraverseErr = fmt.Errorf("no lexer found for %v code in %v post (%v)", lang, p.Slug, l.Tag)

						return blackfriday.Terminate
					}

					iterator, _ := lexer.Tokenise(nil, string(node.Literal))
					formatter := chromaHTML.New(
						chromaHTML.WithClasses(true),
						chromaHTML.HighlightLines(hLines),
					)

					var formattedCode bytes.Buffer
					err := formatter.Format(&formattedCode, chromaStyle, iterator)
					if err != nil {
						bfTraverseErr = err

						return blackfriday.Terminate
					}

					if _, err = htmlBuff.Write(bytes.ReplaceAll(
						formattedCode.Bytes(),
						[]byte(`<pre`),
						[]byte(`<pre data-htmlp-ignore`),
					)); err != nil {
						bfTraverseErr = err

						return blackfriday.Terminate
					}

					return blackfriday.GoToNext
				case node.Type == blackfriday.Image && entering:
					// the image element is only added if its alt
					// attribute has been set.
					if node.FirstChild != nil {
						title := string(node.Title)
						alt := string(node.FirstChild.Literal)

						src, err := generateAssetsLinkFn(gat, p.pat, p.Slug)(AssetRelPath(node.LinkData.Destination))
						if err != nil {
							bfTraverseErr = fmt.Errorf(
								"%v post: %v",
								p.Slug,
								err,
							)

							return blackfriday.Terminate
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
				case node.Type == blackfriday.Paragraph:
					if len(strings.Trim(string(node.FirstChild.Literal), "\n\t ")) == 0 {
						return blackfriday.GoToNext
					}

					return r.RenderNode(&htmlBuff, node, entering)
				default:
					return r.RenderNode(&htmlBuff, node, entering)
				}
			})
			if bfTraverseErr != nil {
				return nil, nil, nil, bfTraverseErr
			}

			// markdown
			p.Content = template.HTML(htmlBuff.Bytes())

			if allPostsByLangTag[l.Tag] == nil {
				allPostsByLangTag[l.Tag] = make([]*Post, 0, 1)
			}

			allPostsByLangTag[l.Tag] = append(allPostsByLangTag[l.Tag], &p)

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

	return allPostsByLangTag, visiblePostsByLangTag, invisiblePostsByLangTag, nil
}
