package egen

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

// assetsTreeNodeType is the type of a node in a tree of assets.
type assetsTreeNodeType int

// Node types.
const (
	FILENODE assetsTreeNodeType = iota
	DIRNODE
	IMGNODE
)

var imgNodeNameRegExp = regexp.MustCompile(`.+\.(jpg|jpeg|png)`)
var cssFilenameRegExp = regexp.MustCompile(`^.*\.css$`)

// AssetRelPath is the path of an asset relative to the global assets
// tree (GAT) or to a post assets tree (PAT). The former happens
// when the path starts with "/", while the latter happens when the
// path starts with any character other than "/".
type AssetRelPath string

// traverseStatus is a status return when traversing a tree.
type traverseStatus int

// Status codes returned when traversing a tree.
const (
	next         traverseStatus = iota // Go to next node
	skipChildren                       // Skip the current node's children
	terminate                          // Terminate the traversal
)

type assetsTreeNodeImgSize struct {
	original  bool
	width     int
	processed bool
}

// assetsTreeNodeTraverseFn is the function executed for each one in a tree traversal.
type assetsTreeNodeTraverseFn func(n *assetsTreeNode) (traverseStatus, error)

// assetsTreeNode is a node in a tree of assets.
type assetsTreeNode struct {
	// t is the type of the node.
	t assetsTreeNodeType
	// name is the name of the file, img or directory.
	// If it's the root node (parent == nil), the name is "assets".
	name string
	// path is the path of the node from, and including, the root's path.
	// Each path segment is separated by / regardless of the current OS.
	path       string
	content    []byte
	parent     *assetsTreeNode
	firstChild *assetsTreeNode
	next       *assetsTreeNode
	previous   *assetsTreeNode
	sizes      []*assetsTreeNodeImgSize
	// processedPath is the node's path after processing. The path doesn't necessarily starts
	// with the tree's root's path, since it starts with the outDirPath value provided when
	// processing the tree.
	processedPath string
	// processedRelPath is the node's relative path after processing. It's processedPath without the
	// outDirPath value at the beginning.
	processedRelPath string
}

var defaultIgnoreRegexps = []*regexp.Regexp{
	regexp.MustCompile(`\.gitkeep`),
}

// generateAssetsTree builds an assets tree root at assetsPath ignoring any descendant node
// whose name matches any item in ignoreRegexps or defaultIgnoreRegexps. When testing a name
// against a regexp, it ends with / if it's a directory. Note that, once a node is ignored,
// all of its descendants are automatically ignored, regardless of whether their names match
// one of the regexps. The returned tree is sorted alphabetically by node name in ascending order.
func generateAssetsTree(assetsPath string, ignoreRegexps []*regexp.Regexp) (*assetsTreeNode, error) {
	rootNode := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: path.Clean(assetsPath),
	}

	err := generateAssetsTreeRec(rootNode, ignoreRegexps)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return rootNode, nil
}

func generateAssetsTreeRec(rootNode *assetsTreeNode, ignoreRegexps []*regexp.Regexp) error {
	fileInfos, err := os.ReadDir(rootNode.path)
	if err != nil {
		return err
	}
	var lastNode *assetsTreeNode

fileInfosLoop:
	for _, fileInfo := range fileInfos {
		var node *assetsTreeNode
		nodeName := fileInfo.Name()

		nodeNameToMatch := nodeName
		if fileInfo.IsDir() {
			nodeNameToMatch += "/"
		}

		for _, rx := range defaultIgnoreRegexps {
			if rx.MatchString(nodeNameToMatch) {
				continue fileInfosLoop
			}
		}

		for _, rx := range ignoreRegexps {
			if rx.MatchString(nodeNameToMatch) {
				continue fileInfosLoop
			}
		}

		switch {
		case imgNodeNameRegExp.MatchString(nodeName):
			nodePath := path.Join(rootNode.path, nodeName)

			width, _, err := imgDimensions(nodePath)
			if err != nil {
				return err
			}

			node = &assetsTreeNode{
				t:    IMGNODE,
				name: nodeName,
				path: nodePath,
				sizes: []*assetsTreeNodeImgSize{
					{
						original: true,
						width:    width,
					},
				},
			}
		case fileInfo.IsDir():
			node = &assetsTreeNode{
				t:    DIRNODE,
				name: nodeName,
				path: path.Join(rootNode.path, nodeName),
			}

			err := generateAssetsTreeRec(node, ignoreRegexps)
			if err != nil {
				return err
			}
		default:
			node = &assetsTreeNode{
				t:    FILENODE,
				name: nodeName,
				path: path.Join(rootNode.path, nodeName),
			}
		}

		node.parent = rootNode
		if lastNode == nil {
			rootNode.firstChild = node
		} else {
			lastNode.next = node
			node.previous = lastNode
		}

		lastNode = node
	}

	return nil
}

func (n *assetsTreeNode) getContent() ([]byte, error) {
	if n.t != FILENODE && n.t != IMGNODE {
		panic("not a file or img node")
	}

	if n.content != nil {
		return n.content, nil
	}

	return os.ReadFile(n.path)
}

func (n *assetsTreeNode) setContent(content []byte) {
	if n.t != FILENODE {
		panic("not a file node")
	}

	n.content = content
}

func (n *assetsTreeNode) removeFromTree() {
	if n.parent == nil {
		return
	}

	if n.previous == nil {
		if n.next != nil {
			n.parent.firstChild = n.next
			n.next.previous = nil
		} else {
			n.parent.firstChild = nil
		}
	} else {
		n.previous.next = n.next

		if n.next != nil {
			n.next.previous = n.previous
		}
	}

	if n.t == DIRNODE {
		n.traverse(func(n *assetsTreeNode) (traverseStatus, error) {
			n.path = ""

			return next, nil
		})
	}

	n.parent = nil
	n.previous = nil
	n.next = nil
	n.path = ""
}

// addChild adds c as child of n in a position that keeps n's children sorted alphabetically by name in ascending order.
func (n *assetsTreeNode) addChild(t assetsTreeNodeType, name string) *assetsTreeNode {
	c := &assetsTreeNode{
		t:      t,
		name:   name,
		parent: n,
		path:   path.Join(n.path, name),
	}

	if n.firstChild == nil {
		n.firstChild = c
	} else {
		var previousNode *assetsTreeNode

		n.traverse(func(n2 *assetsTreeNode) (traverseStatus, error) {
			if n2 == n {
				return next, nil
			}

			if sort.StringSlice([]string{c.name, n2.name}).Less(0, 1) {
				return terminate, nil
			}

			previousNode = n2

			if n2.t == DIRNODE {
				return skipChildren, nil
			}

			return next, nil
		})

		if previousNode == nil {
			n.firstChild.previous = c
			c.next = n.firstChild
			n.firstChild = c
		} else {
			c.previous = previousNode
			c.next = previousNode.next
			if previousNode.next != nil {
				previousNode.next.previous = c
			}

			previousNode.next = c
		}
	}

	c.traverse(func(n *assetsTreeNode) (traverseStatus, error) {
		n.path = path.Join(n.parent.path, n.name)

		return next, nil
	})

	return c
}

func (n *assetsTreeNode) lastChild() *assetsTreeNode {
	if n.firstChild == nil {
		return nil
	}

	lastChild := n.firstChild

	for lastChild.next != nil {
		lastChild = lastChild.next
	}

	return lastChild
}

/* sizes */

func (n *assetsTreeNode) addSizes(widths ...int) {
	originalSize := n.findOriginalSize()

	for _, width := range widths {
		if originalSize.width < width {
			return
		}

		for _, size := range n.sizes {
			if size.width == width {
				return
			}
		}

		n.sizes = append(n.sizes, &assetsTreeNodeImgSize{
			width: width,
		})
	}
}

func (n *assetsTreeNode) findSize(width int) *assetsTreeNodeImgSize {
	for _, size := range n.sizes {
		if size.width == width {
			return size
		}
	}

	return nil
}

func (n *assetsTreeNode) findOriginalSize() *assetsTreeNodeImgSize {
	if n.t != IMGNODE {
		panic("not an img node")
	}

	for _, size := range n.sizes {
		if size.original {
			return size
		}
	}

	return nil
}

func (n *assetsTreeNode) generateSizeProcessedPath(rel bool, size *assetsTreeNodeImgSize) string {
	if n.t != IMGNODE {
		panic("not an img node")
	}

	ext := filepath.Ext(n.name)

	if rel {
		return path.Join(n.processedRelPath, strconv.Itoa(size.width)+ext)
	}

	return path.Join(n.processedPath, strconv.Itoa(size.width)+ext)
}

func (n *assetsTreeNode) generateSrcSetValue(postSlug string) string {
	var srcsetStrB strings.Builder

	// sort sizes
	nodeSizesSorted := make([]*assetsTreeNodeImgSize, len(n.sizes))
	copy(nodeSizesSorted, n.sizes)
	sort.Slice(nodeSizesSorted, func(i, j int) bool {
		return nodeSizesSorted[i].width < nodeSizesSorted[j].width
	})

	for _, size := range nodeSizesSorted {
		if !size.processed {
			continue
		}

		if srcsetStrB.Len() != 0 {
			srcsetStrB.WriteString(", ")
		}

		assetLink := n.assetLink(postSlug, size)

		srcsetStrB.WriteString(
			fmt.Sprintf("%v %vw", assetLink, size.width),
		)
	}

	return srcsetStrB.String()
}

/* traversing */

// traverse performs a depth-first pre-order traversal in the tree rooted at n.
// If fn returns an error, the traversing is terminated, regardless of the status,
// and the error is returned.
func (n *assetsTreeNode) traverse(fn assetsTreeNodeTraverseFn) error {
	_, err := traverseRec(n, fn)
	if err != nil {
		return err
	}

	return nil
}

func traverseRec(n *assetsTreeNode, fn assetsTreeNodeTraverseFn) (traverseStatus, error) {
	status, err := fn(n)
	if err != nil || status == terminate {
		return terminate, err
	}
	if status == skipChildren {
		return skipChildren, nil
	}

	c := n.firstChild

	for c != nil {
		cNext := c.next

		switch c.t {
		case DIRNODE:
			status, err := traverseRec(c, fn)
			if err != nil || status == terminate {
				return terminate, err
			}
		case IMGNODE:
			fallthrough
		case FILENODE:
			status, err := fn(c)
			if err != nil {
				return terminate, err
			}

			switch status {
			case skipChildren:
				return skipChildren, nil
			case terminate:
				return terminate, nil
			}
		}

		c = cNext
	}

	return next, nil
}

/* processing */

// process processes each node of a tree of assets rooted at n and places the output
// in outDirPath. Each processed node has its processedRelPath and processedPath properties
// set.
func (n *assetsTreeNode) process(outDirPath string, processRoot bool) error {
	err := n.traverse(func(n2 *assetsTreeNode) (traverseStatus, error) {
		if n2 == n && !processRoot {
			return next, nil
		}

		pathWithoutRoot := strings.TrimPrefix(n2.path, n.path+"/")

		switch n2.t {
		case IMGNODE:
			nodeContent, err := n2.getContent()
			if err != nil {
				return terminate, err
			}

			md5HashBs := md5.Sum(nodeContent)
			md5Hash := hex.EncodeToString(md5HashBs[:])
			pathWithoutRootProcessed := path.Join(pathWithoutRoot, "..", md5Hash)
			processedPath := path.Join(outDirPath, pathWithoutRootProcessed)

			if err := os.Mkdir(processedPath, os.ModePerm|os.ModeDir); err != nil {
				return terminate, fmt.Errorf("while creating %v directory: %v", processedPath, err)
			}

			n2.processedRelPath = pathWithoutRootProcessed
			n2.processedPath = path.Join(outDirPath, pathWithoutRootProcessed)

			if err := n2.processSizes(); err != nil {
				return terminate, err
			}
		case FILENODE:
			ext := filepath.Ext(pathWithoutRoot)
			pathWithoutRootWithoutExt := strings.TrimSuffix(pathWithoutRoot, ext)

			// md5 hash
			nodeContent, err := n2.getContent()
			if err != nil {
				return terminate, err
			}

			md5HashBs := md5.Sum(nodeContent)
			md5Hash := hex.EncodeToString(md5HashBs[:])
			pathWithoutRootProcessed := pathWithoutRootWithoutExt + "-" + string(md5Hash[:]) + ext

			fileOutPath := path.Join(outDirPath, pathWithoutRootProcessed)
			fileOut, err := os.Create(fileOutPath)
			if err != nil {
				return terminate, err
			}

			// writing to new file
			_, err = fileOut.Write(nodeContent)
			if err != nil {
				fileOut.Close()
				return terminate, err
			}

			fileOut.Close()

			n2.processedRelPath = pathWithoutRootProcessed
			n2.processedPath = fileOutPath
		case DIRNODE:
			processedPath := path.Join(outDirPath, pathWithoutRoot)
			err := os.Mkdir(processedPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return terminate, err
			}

			n2.processedRelPath = pathWithoutRoot
			n2.processedPath = processedPath
		}

		return next, nil
	})
	if err != nil {
		return err
	}

	return nil
}

// processSizes processes the sizes of an img node.
func (n *assetsTreeNode) processSizes() error {
	if n.t != IMGNODE {
		panic("not an img node")
	}

	if n.processedPath == "" {
		panic("node hasn't been processed")
	}

	nodeContent, err := n.getContent()
	if err != nil {
		return fmt.Errorf("while retrieving %v content: %v", n.path, err)
	}

	for _, size := range n.sizes {
		if size.processed {
			continue
		}

		sizeFilePath := n.generateSizeProcessedPath(false, size)
		sizeFileContent := nodeContent
		sizeFile, err := os.Create(sizeFilePath)
		if err != nil {
			return fmt.Errorf("while creating %v file", sizeFilePath)
		}

		if !size.original {
			sizeFileContent, err = resizeImg(size.width, n.path)
			if err != nil {
				return fmt.Errorf("while resizing %v image", n.path)
			}
		}

		if _, err := sizeFile.Write(sizeFileContent); err != nil {
			return fmt.Errorf("while writing to %v file", sizeFilePath)
		}

		size.processed = true
	}

	return nil
}

// processCSSFileNodes processed CSS file nodes with depth = 1.
func (n *assetsTreeNode) processCSSFileNodes() error {
	cssContent := make([]byte, 0)

	err := n.traverse(func(n2 *assetsTreeNode) (traverseStatus, error) {
		if n2 == n {
			return next, nil
		}

		// only nodes whose depth = 1
		if n2.t == DIRNODE {
			return skipChildren, nil
		}

		if cssFilenameRegExp.MatchString(n2.name) {
			cssFileContent, err := n2.getContent()
			if err != nil {
				return terminate, err
			}

			cssContent = append(cssContent, cssFileContent...)

			n2.removeFromTree()
		}

		return next, nil
	})
	if err != nil {
		return err
	}

	// minifying
	m := minify.New()
	m.AddFunc("text/css", css.Minify)

	cssContentMinified, err := m.Bytes("text/css", cssContent)
	if err != nil {
		return err
	}

	n2 := n.addChild(FILENODE, "style.css")
	n2.setContent(cssContentMinified)

	return nil
}

/* asset link */

func (n *assetsTreeNode) assetLink(postSlug string, size *assetsTreeNodeImgSize) string {
	pathSegments := []string{"/assets"}

	if postSlug != "" {
		pathSegments = append(pathSegments, postSlug)
	}

	switch {
	case size != nil:
		pathSegments = append(pathSegments, n.generateSizeProcessedPath(true, size))
	case n.t == IMGNODE:
		pathSegments = append(pathSegments, n.generateSizeProcessedPath(true, n.findOriginalSize()))
	default:
		pathSegments = append(pathSegments, n.processedRelPath)
	}

	return path.Join(pathSegments...)
}

/* finding a node */

// findNodeByName returns the first node whose name is equal to the given name encountered while traversing n.
func (n *assetsTreeNode) findNodeByName(name string) *assetsTreeNode {
	var res *assetsTreeNode

	n.traverse(func(n *assetsTreeNode) (traverseStatus, error) {
		if n.name == name {
			res = n

			return terminate, nil
		}

		return next, nil
	})

	return res
}

// findByRelPath searchs for a node whose path, by trimming n's path from the start, is equal to relPath.
func (n *assetsTreeNode) findByRelPath(relPath string) *assetsTreeNode {
	segments := strings.Split(relPath, "/")

	n2 := n.firstChild
	i := 0

	for {
		if i+1 > len(segments) || n2 == nil {
			break
		}

		if n2.name == segments[i] {
			if i+1 == len(segments) {
				return n2
			}

			n2 = n2.firstChild
			i++
			continue
		} else {
			n2 = n2.next
		}
	}

	return nil
}

// findByRelPathInGATOrPAT searchs for a node whose path relative to the root of the GAT or to
// the root of the PAT is equal to path. If path starts with /, it searchs in the GAT, otherwise
// it'll search in the PAT.
func findByRelPathInGATOrPAT(gat, pat *assetsTreeNode, relPath AssetRelPath) (n *assetsTreeNode, searchedInPAT bool) {
	if len(relPath) == 0 {
		return nil, false
	}

	if relPath[0] == '/' {
		if gat == nil {
			return nil, false
		}

		return gat.findByRelPath(strings.TrimPrefix(string(relPath), "/")), false
	}

	if pat == nil {
		return nil, true
	}

	return pat.findByRelPath(string(relPath)), true
}
