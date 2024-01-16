package egen

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	chromaHTML "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/efreitasn/egen/internal/latex"
	"github.com/russross/blackfriday/v2"
	"gopkg.in/yaml.v2"
)

var (
	latexGenerator latexImageGenerator = &latex.ImageGenerator{}

	mdCodeBlockInfoRegExp       = regexp.MustCompile(`^((?:[a-z]|[0-9])+?)(?:{((?:\[[0-9]{1,},[0-9]{1,}\])(?:(?:,\[[0-9]{1,},[0-9]{1,}\])+)?)})?$`)
	mdCodeBlockInfoHLinesRegExp = regexp.MustCompile(`\[([0-9]{1,}),([0-9]{1,})\]`)
	postContentRegExp           = regexp.MustCompile(`(?s)^---\n(.*?)\n---(.*)`)

	nonPostAssetsRxs = []*regexp.Regexp{
		regexp.MustCompile(`content_.+\.md`),
		regexp.MustCompile(`data\.yaml`),
		// ignore all directories
		regexp.MustCompile(".*/$"),
	}
)

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

type (
	generatePostsListsInput struct {
		bc            *BuildConfig
		c             *config
		gat           *assetsTreeNode
		assetsOutPath string
	}

	generatePostsListsOutput struct {
		allPostsByLangTag, visiblePostsByLangTag, invisiblePostsByLangTag map[string][]*Post
	}
)

func generatePostsLists(input generatePostsListsInput) (*generatePostsListsOutput, error) {
	postsInPath := path.Join(input.bc.InPath, "posts")

	postsFileInfos, err := os.ReadDir(postsInPath)
	if err != nil {
		return nil, err
	}

	output := generatePostsListsOutput{
		allPostsByLangTag:       make(map[string][]*Post),
		visiblePostsByLangTag:   make(map[string][]*Post),
		invisiblePostsByLangTag: make(map[string][]*Post),
	}

	for _, postsFileInfo := range postsFileInfos {
		if !postsFileInfo.IsDir() {
			continue
		}

		postSlug := postsFileInfo.Name()
		postDirPath := path.Join(postsInPath, postSlug)

		pat, err := generateAssetsTree(postDirPath, nonPostAssetsRxs)
		if err != nil {
			return nil, fmt.Errorf("generating pat for %v post: %v", postSlug, err)
		}

		// this condition exists so that assetsPathOut is only created if the post
		// has at least one asset.
		if pat.firstChild != nil {
			assetsPathOut := path.Join(input.assetsOutPath, postSlug)

			// it's checked whether assetsPathOut already exists because it could've
			// been already created when generating the global assets tree (GAT) if
			// there's a directory in it whose name is the same as the post's slug.
			if _, err := os.Stat(assetsPathOut); err != nil {
				if os.IsNotExist(err) {
					err := os.Mkdir(assetsPathOut, os.ModeDir|os.ModePerm)
					if err != nil {
						return nil, fmt.Errorf("creating %v: %v", assetsPathOut, err)
					}
				} else {
					return nil, err
				}
			}

			if err = pat.process(assetsPathOut, false); err != nil {
				return nil, fmt.Errorf("processing pat: %v", err)
			}
		}

		// data.yaml file
		postYAMLDataFile, err := os.Open(path.Join(postDirPath, "data.yaml"))
		if err != nil {
			return nil, fmt.Errorf("opening %v data.yaml: %v", postSlug, err)
		}

		var postYAMLData postYAMLDataFileContent
		err = yaml.NewDecoder(postYAMLDataFile).Decode(&postYAMLData)
		if err != nil {
			return nil, fmt.Errorf("decoding %v data.yaml: %v", postSlug, err)
		}

		postDate, err := time.Parse(time.RFC3339, postYAMLData.Date)
		if err != nil {
			return nil, fmt.Errorf("parsing %v data.yaml date: %v", postSlug, err)
		}

		var postLastUpdateDate time.Time

		if postYAMLData.LastUpdateDate != "" {
			postLastUpdateDate, err = time.Parse(time.RFC3339, postYAMLData.LastUpdateDate)
			if err != nil {
				return nil, fmt.Errorf("parsing %v data.yaml lastUpdateDate: %v", postSlug, err)
			}
		}

		// content_*.md files
		for _, l := range input.c.Langs {
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
			postContent, err := os.ReadFile(postContentFilePath)
			if err != nil {
				if os.IsNotExist(err) {
					return nil, fmt.Errorf("%v for %v post doesn't exist", postContentFilename, postSlug)
				}

				return nil, err
			}
			if !postContentRegExp.Match(postContent) {
				return nil, fmt.Errorf("post content at %v is invalid", postContentFilePath)
			}

			matchesIndexes := postContentRegExp.FindSubmatchIndex(postContent)
			postContentYAML := postContent[matchesIndexes[2]:matchesIndexes[3]]
			postContentMD := postContent[matchesIndexes[4]:matchesIndexes[5]]

			p.generateContent(input, l, postContentMD)

			// yaml
			var yamlData postYAMLFrontMatter
			err = yaml.Unmarshal(postContentYAML, &yamlData)
			if err != nil {
				return nil, fmt.Errorf("parsing YAML content of %v: %v", postContentFilePath, err)
			}

			if yamlData.Title == "" {
				return nil, fmt.Errorf("title field in %v post frontmatter in %v cannot be empty", p.Slug, l.Tag)
			}

			if yamlData.Excerpt == "" {
				return nil, fmt.Errorf("excerpt field in %v post frontmatter in %v cannot be empty", p.Slug, l.Tag)
			}

			p.Title = yamlData.Title
			p.Excerpt = yamlData.Excerpt

			if postYAMLData.Img != "" {
				if yamlData.ImgAlt == "" {
					return nil, fmt.Errorf("img alt in %v for %v post not provided", l.Tag, p.Slug)
				}

				p.Img = &Img{
					Path: postYAMLData.Img,
					Alt:  yamlData.ImgAlt,
				}
			}

			if output.allPostsByLangTag[l.Tag] == nil {
				output.allPostsByLangTag[l.Tag] = make([]*Post, 0, 1)
			}

			output.allPostsByLangTag[l.Tag] = append(output.allPostsByLangTag[l.Tag], &p)

			if postYAMLData.Feed {
				if output.visiblePostsByLangTag[l.Tag] == nil {
					output.visiblePostsByLangTag[l.Tag] = make([]*Post, 0, 1)
				}

				output.visiblePostsByLangTag[l.Tag] = append(output.visiblePostsByLangTag[l.Tag], &p)
			} else {
				if output.invisiblePostsByLangTag[l.Tag] == nil {
					output.invisiblePostsByLangTag[l.Tag] = make([]*Post, 0, 1)
				}

				output.invisiblePostsByLangTag[l.Tag] = append(output.invisiblePostsByLangTag[l.Tag], &p)
			}
		}
	}

	return &output, nil
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
	pat *assetsTreeNode
}

func (p *Post) generateContent(input generatePostsListsInput, l *Lang, markdown []byte) error {
	mdProcessor := blackfriday.New(blackfriday.WithExtensions(blackfriday.CommonExtensions))
	rootNode := mdProcessor.Parse(markdown)

	latexBlockMap, inlineLatexMap := p.processContentBFTree(input, rootNode)

	err := latexGenerator.SetDirPath(input.bc.InPath)
	if err != nil {
		return fmt.Errorf("setting latex image generator dir path: %w", err)
	}

	err = p.renderContentBFTree(input, l, rootNode, latexBlockMap, inlineLatexMap)
	if err != nil {
		return err
	}

	return nil
}

func (p *Post) processContentBFTree(input generatePostsListsInput, rootNode *blackfriday.Node) (latexBlockMap, inlineLatexMap map[*blackfriday.Node]struct{}) {
	latexBlockMap = map[*blackfriday.Node]struct{}{}
	inlineLatexMap = map[*blackfriday.Node]struct{}{}

	rootNode.Walk(func(node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		switch {
		// Remove img tags from inside p tags.
		case node.Type == blackfriday.Image && entering:
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
					oldParentChildrenAfterNode = oldParentChildren[nodeOldParentIndex+1:]
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
					newParentChildrenAfterOldParent := newParentChildren[oldParentNewParentIndex+1:]

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

		case node.Type == blackfriday.Text && entering:
			if input.c.Latex {
				for i := 0; i < len(node.Literal); {
					if node.Literal[i] == '$' {
						if i != 0 && node.Literal[i-1] == '\\' {
							node.Literal = slices.Delete(node.Literal, i-1, i)
							i++
							continue
						}

						if len(node.Literal) > i+1 && node.Literal[i+1] == '$' {
							var (
								found bool

								start = i + 2
								end   = start
							)

							for ; end < len(node.Literal); end++ {
								if node.Literal[end] == '$' && node.Literal[end-1] != '\\' && len(node.Literal) > end+1 && node.Literal[end+1] == '$' {
									found = true
									break
								}
							}

							if !found {
								return blackfriday.GoToNext
							}

							content := node.Literal[start:end]

							// If it's empty (i.e. $$$$), remove it.
							if len(content) == 0 {
								node.Literal = slices.Delete(node.Literal, start-2, end+2)
								i++
								continue
							}

							// If the first $ is not on the 0th position, then the current block needs to be
							// splitted.
							if i != 0 {
								textNode := blackfriday.NewNode(blackfriday.Text)
								textNode.Literal = node.Literal[:i]

								node.InsertBefore(textNode)
							}

							// The content after the ending $$, if there's any, is the caption.
							node.Title = node.Literal[end+2:]
							node.Literal = content
							latexBlockMap[node] = struct{}{}

							return blackfriday.GoToNext
						}

						// Inline latex.
						var (
							found bool

							start = i + 1
							end   = start
						)

						for ; end < len(node.Literal); end++ {
							if node.Literal[end] == '$' && node.Literal[end-1] != '\\' {
								found = true
								break
							}
						}

						if !found {
							return blackfriday.GoToNext
						}

						content := node.Literal[start:end]

						// If it's empty (i.e. $$), remove it.
						if len(content) == 0 {
							node.Literal = slices.Delete(node.Literal, start-1, end+1)
							i++
							continue
						}

						// If the starting $ is not on the 0th position, then a text node needs to be inserted
						// before the current node.
						if i != 0 {
							textNode := blackfriday.NewNode(blackfriday.Text)
							textNode.Literal = node.Literal[:i]

							node.InsertBefore(textNode)
						}

						// If the ending $ is not on the last position, then a text node needs to be inserted
						// after the current node.
						if end != len(node.Literal)-1 {
							textNode := blackfriday.NewNode(blackfriday.Text)
							textNode.Literal = node.Literal[end+1:]

							if node.Next == nil {
								node.Parent.AppendChild(textNode)
							} else {
								node.Next.InsertBefore(textNode)
							}
						}

						node.Literal = content
						inlineLatexMap[node] = struct{}{}

						return blackfriday.GoToNext
					}

					i++
				}
			}
		}

		return blackfriday.GoToNext
	})

	return latexBlockMap, inlineLatexMap
}

func (p *Post) renderContentBFTree(input generatePostsListsInput, l *Lang, rootNode *blackfriday.Node, latexBlockMap, inlineLatexMap map[*blackfriday.Node]struct{}) error {
	var (
		traverseErr error
		htmlBuff    bytes.Buffer
	)

	r := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{
		Flags: blackfriday.HrefTargetBlank | blackfriday.NoreferrerLinks,
	})

	rootNode.Walk(func(bfNode *blackfriday.Node, entering bool) blackfriday.WalkStatus {
		switch {
		case bfNode.Type == blackfriday.CodeBlock && entering:
			if !mdCodeBlockInfoRegExp.Match(bfNode.Info) {
				return blackfriday.GoToNext
			}

			cbInfoMatches := mdCodeBlockInfoRegExp.FindStringSubmatch(string(bfNode.Info))
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
				traverseErr = fmt.Errorf("no lexer found for %v code in %v post (%v)", lang, p.Slug, l.Tag)

				return blackfriday.Terminate
			}

			iterator, _ := lexer.Tokenise(nil, string(bfNode.Literal))
			formatter := chromaHTML.New(
				chromaHTML.WithClasses(true),
				chromaHTML.HighlightLines(hLines),
			)

			var formattedCode bytes.Buffer
			err := formatter.Format(&formattedCode, input.bc.ChromaStyle, iterator)
			if err != nil {
				traverseErr = err

				return blackfriday.Terminate
			}

			if _, err = htmlBuff.Write(formattedCode.Bytes()); err != nil {
				traverseErr = err

				return blackfriday.Terminate
			}

			return blackfriday.GoToNext

		case bfNode.Type == blackfriday.Image && entering:
			if bfNode.FirstChild == nil || string(bfNode.FirstChild.Literal) == "" {
				traverseErr = fmt.Errorf("%v img in %v post in %v must have an alt attribute", string(bfNode.LinkData.Destination), p.Slug, l.Tag)

				return blackfriday.Terminate
			}

			title := string(bfNode.Title)
			alt := string(bfNode.FirstChild.Literal)

			node, searchedInPAT := findByRelPathInGATOrPAT(input.gat, p.pat, AssetRelPath(bfNode.LinkData.Destination))
			if node == nil {
				traverseErr = fmt.Errorf(
					"%v img not found in %v post",
					string(bfNode.LinkData.Destination),
					p.Slug,
				)

				return blackfriday.Terminate
			}

			node.addSizes(input.c.ResponsiveImgSizes...)

			if err := node.processSizes(); err != nil {
				traverseErr = fmt.Errorf("while processing sizes for %v img: %v", node.path, err)

				return blackfriday.Terminate
			}

			var figcaption string
			if title != "" {
				figcaption = fmt.Sprintf("<figcaption>%v</figcaption>", title)
			}

			var src string
			if searchedInPAT {
				src = node.assetLink(p.Slug, node.findOriginalSize())
			} else {
				src = node.assetLink("", node.findOriginalSize())
			}

			var img string
			if input.c.ResponsiveImgMediaQueries != "" {
				var srcset string
				if searchedInPAT {
					srcset = node.generateSrcSetValue(p.Slug)
				} else {
					srcset = node.generateSrcSetValue("")
				}

				img = fmt.Sprintf(`<img srcset="%v" sizes="%v" src="%v" alt="%v">`, srcset, input.c.ResponsiveImgMediaQueries, src, alt)
			} else {
				img = fmt.Sprintf(`<img src="%v" alt="%v">`, src, alt)
			}

			htmlBuff.WriteString(
				fmt.Sprintf(`<figure><a href="%v">%v</a>%v</figure>`, src, img, figcaption),
			)

			return blackfriday.SkipChildren

		case bfNode.Type == blackfriday.Text && mapContains(latexBlockMap, bfNode):
			if !entering {
				return blackfriday.GoToNext
			}

			svgBs, err := latexGenerator.SVGBlock(bfNode.Literal)
			if err != nil {
				traverseErr = fmt.Errorf("generating latex block in %v post: %w", p.Slug, err)

				return blackfriday.Terminate
			}

			var figCaption string
			if len(bfNode.Title) > 0 {
				figCaption = fmt.Sprintf("<figcaption>%s</figcaption>", bfNode.Title)
			}

			fmt.Fprintf(
				&htmlBuff,
				`<figure><div style="text-align: center; font-size: 2rem">%s</div>%s</figure>`,
				svgBs,
				figCaption,
			)

			return blackfriday.GoToNext

		case bfNode.Type == blackfriday.Text && mapContains(inlineLatexMap, bfNode):
			if !entering {
				return blackfriday.GoToNext
			}

			svgBs, err := latexGenerator.SVGInline(bfNode.Literal)
			if err != nil {
				traverseErr = fmt.Errorf("generating inline latex in %v post: %w", p.Slug, err)

				return blackfriday.Terminate
			}

			fmt.Fprintf(&htmlBuff, `<span>%s</span>`, svgBs)

			return blackfriday.GoToNext

		case bfNode.Type == blackfriday.Paragraph:
			firstChildIsEmpty := bfNode.FirstChild == nil || len(strings.Trim(string(bfNode.FirstChild.Literal), "\n\t ")) == 0
			onlyChildIsLatexBlock := bfNode.FirstChild != nil && mapContains(latexBlockMap, bfNode.FirstChild) && bfNode.FirstChild.Next == nil

			if firstChildIsEmpty || onlyChildIsLatexBlock {
				return blackfriday.GoToNext
			}

			bfNode.FirstChild.Literal = bytes.TrimLeft(bfNode.FirstChild.Literal, "\n\t ")
			bfNode.LastChild.Literal = bytes.TrimRight(bfNode.LastChild.Literal, "\n\t ")

			return r.RenderNode(&htmlBuff, bfNode, entering)

		default:
			return r.RenderNode(&htmlBuff, bfNode, entering)
		}
	})
	if traverseErr != nil {
		return traverseErr
	}

	p.Content = template.HTML(htmlBuff.Bytes())

	return nil
}
