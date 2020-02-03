package egen

import "gopkg.in/russross/blackfriday.v2"

func findBFNodeIndex(node *blackfriday.Node, parent *blackfriday.Node) int {
	children := getBFNodeChildren(parent)
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

func getBFNodeChildren(n *blackfriday.Node) []*blackfriday.Node {
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
