package egen

import (
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
)

// AssetsTreeNodeType is the type of a node in a tree of assets.
type AssetsTreeNodeType int

// Node types.
const (
	FILENODE AssetsTreeNodeType = iota
	DIRNODE
)

// AssetRelPath is the path of an asset relative to the global assets
// tree (GAT) or to a post-wise assets tree (PAT). The former happens
// when the path starts with "/", while the latter happens when the
// path starts with any character other than "/".
type AssetRelPath string

// TraverseStatus is a status return when traversing a tree.
type TraverseStatus int

// Status codes returned when traversing a tree.
const (
	Next         TraverseStatus = iota // Go to next node
	SkipChildren                       // Skip the current node's children
	Terminate                          // Terminate the traversal
)

// AssetsTreeNodeTraverseFn is the function executed for each one in a tree traversal.
type AssetsTreeNodeTraverseFn func(n *AssetsTreeNode) (TraverseStatus, error)

// AssetsTreeNode is a node in a tree of assets.
type AssetsTreeNode struct {
	Type AssetsTreeNodeType
	// Name is the name of the file or directory.
	// If it's the root node (parent == nil), the name is "assets".
	Name string
	// Path is the path of the node from, and including, the root's path.
	// Each path segment is separated by / regardless of the current OS.
	Path       string
	content    []byte
	Parent     *AssetsTreeNode
	FirstChild *AssetsTreeNode
	Next       *AssetsTreeNode
	Previous   *AssetsTreeNode
	// processedPath is the node's path after processing. The path doesn't necessarily starts
	// with the tree's root's path, since it starts with the outDirPath value provided when
	// processing the tree.
	processedPath string
	// processedRelPath is the node's relative path after processing. It's processedPath without the
	// outDirPath value at the beginning.
	processedRelPath string
}

var defaultIgnoreRegexps = []*regexp.Regexp{
	regexp.MustCompile("\\.gitkeep"),
}

// generateAssetsTree builds an assets tree root at assetsPath ignoring any descendant node
// whose name matches any item in ignoreRegexps or defaultIgnoreRegexps. When testing a name
// against a regexp, it ends with / if it's a directory. Note that, once a node is ignored,
// all of its descendants are automatically ignored, regardless of whether their names match
// one of the regexps.
func generateAssetsTree(assetsPath string, ignoreRegexps []*regexp.Regexp) (*AssetsTreeNode, error) {
	rootNode := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: path.Clean(assetsPath),
	}

	err := generateAssetsTreeRec(rootNode, ignoreRegexps)
	if err != nil {
		return nil, err
	}

	return rootNode, nil
}

func generateAssetsTreeRec(rootNode *AssetsTreeNode, ignoreRegexps []*regexp.Regexp) error {
	fileInfos, err := ioutil.ReadDir(rootNode.Path)
	if err != nil {
		return err
	}
	var lastNode *AssetsTreeNode

fileInfosLoop:
	for _, fileInfo := range fileInfos {
		var node *AssetsTreeNode
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

		if fileInfo.IsDir() {
			node = &AssetsTreeNode{
				Type: DIRNODE,
				Name: nodeName,
				Path: path.Join(rootNode.Path, nodeName),
			}

			err := generateAssetsTreeRec(node, []*regexp.Regexp{})
			if err != nil {
				return err
			}
		} else {
			node = &AssetsTreeNode{
				Type: FILENODE,
				Name: nodeName,
				Path: path.Join(rootNode.Path, nodeName),
			}
		}

		node.Parent = rootNode
		if lastNode == nil {
			rootNode.FirstChild = node
		} else {
			lastNode.Next = node
			node.Previous = lastNode
		}

		lastNode = node
	}

	return nil
}

// Content returns the content of n.
// If the node is not a file, it panics.
func (n *AssetsTreeNode) Content() ([]byte, error) {
	if n.Type != FILENODE {
		panic("not a file node")
	}

	if n.content != nil {
		return n.content, nil
	}

	return ioutil.ReadFile(n.Path)
}

// SetContent sets the content of n.
// If the node is not a file, it panics.
func (n *AssetsTreeNode) SetContent(content []byte) {
	if n.Type != FILENODE {
		panic("not a file node")
	}

	n.content = content
}

// RemoveFromTree removes n from the tree.
func (n *AssetsTreeNode) RemoveFromTree() {
	if n.Parent == nil {
		return
	}

	if n.Previous == nil {
		if n.Next != nil {
			n.Parent.FirstChild = n.Next
			n.Next.Previous = nil
		} else {
			n.Parent.FirstChild = nil
		}
	} else {
		n.Previous.Next = n.Next

		if n.Next != nil {
			n.Next.Previous = n.Previous
		}
	}

	if n.Type == DIRNODE {
		n.Traverse(func(n *AssetsTreeNode) (TraverseStatus, error) {
			n.Path = ""

			return Next, nil
		})
	}

	n.Parent = nil
	n.Previous = nil
	n.Next = nil
	n.Path = ""
}

// AddChild adds c as child of n.
func (n *AssetsTreeNode) AddChild(t AssetsTreeNodeType, name string) *AssetsTreeNode {
	c := &AssetsTreeNode{
		Type:   t,
		Name:   name,
		Parent: n,
		Path:   path.Join(n.Path, name),
	}

	if n.FirstChild == nil {
		n.FirstChild = c
	} else {
		lastChild := n.LastChild()
		lastChild.Next = c
		c.Previous = lastChild
	}

	c.Traverse(func(n *AssetsTreeNode) (TraverseStatus, error) {
		n.Path = path.Join(n.Parent.Path, n.Name)

		return Next, nil
	})

	return c
}

// LastChild returns the last child of n.
func (n *AssetsTreeNode) LastChild() *AssetsTreeNode {
	if n.FirstChild == nil {
		return nil
	}

	lastChild := n.FirstChild

	for lastChild.Next != nil {
		lastChild = lastChild.Next
	}

	return lastChild
}

// Traverse performs a depth-first pre-order traversal in the tree rooted at n.
// If fn returns an error, the traversing is terminated, regardless of the status,
// and the error is returned.
func (n *AssetsTreeNode) Traverse(fn AssetsTreeNodeTraverseFn) error {
	_, err := traverseRec(n, fn)
	if err != nil {
		return err
	}

	return nil
}

// FindByRelPath searchs for a node whose path, by trimming n's path from the start, is equal to relPath.
func (n *AssetsTreeNode) FindByRelPath(relPath string) *AssetsTreeNode {
	segments := strings.Split(relPath, "/")

	n2 := n.FirstChild
	i := 0

	for {
		if i+1 > len(segments) || n2 == nil {
			break
		}

		if n2.Name == segments[i] {
			if i+1 == len(segments) {
				return n2
			}

			n2 = n2.FirstChild
			i++
			continue
		} else {
			n2 = n2.Next
		}
	}

	return nil
}

func traverseRec(n *AssetsTreeNode, fn AssetsTreeNodeTraverseFn) (TraverseStatus, error) {
	status, err := fn(n)
	if err != nil || status == Terminate {
		return Terminate, err
	}
	if status == SkipChildren {
		return SkipChildren, nil
	}

	c := n.FirstChild

	for c != nil {
		cNext := c.Next

		switch c.Type {
		case DIRNODE:
			status, err := traverseRec(c, fn)
			if err != nil || status == Terminate {
				return Terminate, err
			}
		case FILENODE:
			status, err := fn(c)
			if err != nil {
				return Terminate, err
			}

			switch status {
			case SkipChildren:
				return SkipChildren, nil
			case Terminate:
				return Terminate, nil
			}
		}

		c = cNext
	}

	return Next, nil
}

// findByRelPathInGATOrPAT searchs for a node whose path related to the root of the GAT or to
// the root of the PAT is equal to path. If path starts with /, it searchs in the GAT, otherwise
// it'll search in the PAT.
func findByRelPathInGATOrPAT(gat, pat *AssetsTreeNode, relPath AssetRelPath) (n *AssetsTreeNode, searchedInPAT bool) {
	if len(relPath) == 0 {
		return nil, false
	}

	if relPath[0] == '/' {
		if gat == nil {
			return nil, false
		}

		return gat.FindByRelPath(strings.TrimPrefix(string(relPath), "/")), false
	}

	if pat == nil {
		return nil, true
	}

	return pat.FindByRelPath(string(relPath)), true
}

var cssFilenameRegExp = regexp.MustCompile("^.*\\.css$")

func bundleCSSFilesInAT(rootNode *AssetsTreeNode) error {
	cssContent := make([]byte, 0)

	err := rootNode.Traverse(func(n *AssetsTreeNode) (TraverseStatus, error) {
		if n == rootNode {
			return Next, nil
		}

		// only nodes whose depth = 1
		if n.Type == DIRNODE {
			return SkipChildren, nil
		}

		if cssFilenameRegExp.MatchString(n.Name) {
			cssFileContent, err := n.Content()
			if err != nil {
				return Terminate, err
			}

			cssContent = append(cssContent, cssFileContent...)

			n.RemoveFromTree()
		}

		return Next, nil
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

	n := rootNode.AddChild(FILENODE, "style.css")
	n.SetContent(cssContentMinified)

	return nil
}

// processAT process each node of a tree of assets rooted at rootNode and places the output
// in outDirPath. Each processed node has its processedRelPath and processedPath properties
// set.
func processAT(rootNode *AssetsTreeNode, outDirPath string) error {
	err := rootNode.Traverse(func(n *AssetsTreeNode) (TraverseStatus, error) {
		if n == rootNode {
			return Next, nil
		}

		pathWithoutRoot := strings.TrimPrefix(n.Path, rootNode.Path+"/")

		switch n.Type {
		case FILENODE:
			ext := filepath.Ext(pathWithoutRoot)
			pathWithoutRootWithoutExt := strings.TrimSuffix(pathWithoutRoot, ext)

			// md5 hash
			nodeContent, err := n.Content()
			if err != nil {
				return Terminate, err
			}

			md5HashBs := md5.Sum(nodeContent)
			md5Hash := hex.EncodeToString(md5HashBs[:])
			pathWithoutRootProcessed := pathWithoutRootWithoutExt + "-" + string(md5Hash[:]) + ext

			fileOutPath := path.Join(outDirPath, pathWithoutRootProcessed)
			fileOut, err := os.Create(fileOutPath)
			if err != nil {
				return Terminate, err
			}

			// writing to new file
			_, err = fileOut.Write(nodeContent)
			if err != nil {
				fileOut.Close()
				return Terminate, err
			}

			fileOut.Close()

			n.processedRelPath = pathWithoutRootProcessed
			n.processedPath = fileOutPath
		case DIRNODE:
			processedPath := path.Join(outDirPath, pathWithoutRoot)
			err := os.Mkdir(processedPath, os.ModeDir|os.ModePerm)
			if err != nil {
				return Terminate, err
			}

			n.processedRelPath = pathWithoutRoot
			n.processedPath = processedPath
		}

		return Next, nil
	})
	if err != nil {
		return err
	}

	return nil
}
