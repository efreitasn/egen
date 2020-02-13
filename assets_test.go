package egen

import (
	"errors"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGenerateAssetsTree(t *testing.T) {
	rootNode := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: "testdata/tree/ok/1",
	}

	fooNode := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "foo.txt",
		Path:   path.Join(rootNode.Path, "foo.txt"),
		Parent: rootNode,
	}
	rootNode.FirstChild = fooNode

	imgsDirNode := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "imgs",
		Path:     path.Join(rootNode.Path, "imgs"),
		Previous: fooNode,
		Parent:   rootNode,
	}
	fooNode.Next = imgsDirNode

	redNode := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "red.png",
		Parent: imgsDirNode,
		Path:   path.Join(imgsDirNode.Path, "red.png"),
	}
	imgsDirNode.FirstChild = redNode

	rootNode2 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: "testdata/tree/ok/1",
	}

	fooNode2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "foo.txt",
		Path:   path.Join(rootNode2.Path, "foo.txt"),
		Parent: rootNode2,
	}
	rootNode2.FirstChild = fooNode2

	tests := []struct {
		path          string
		err           error
		tree          *AssetsTreeNode
		ignoreRegexps []*regexp.Regexp
	}{
		{
			"./testdata/tree/ok/1",
			nil,
			rootNode,
			nil,
		},
		{
			"./testdata/tree/ok/1",
			nil,
			rootNode2,
			[]*regexp.Regexp{regexp.MustCompile(".*imgs.*")},
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			tree, err := generateAssetsTree(test.path, test.ignoreRegexps)

			if err != test.err {
				t.Errorf("got %v, want %v", err, test.err)
			}

			if !reflect.DeepEqual(tree, test.tree) {
				t.Error("trees are not equal")
			}
		})
	}
}

func TestCompareAssetsTrees(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1,
		Path:   path.Join(dir1.Path, "dir2"),
	}
	dir1.FirstChild = dir2

	file1 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2,
		Path:   path.Join(dir2.Path, "file1"),
	}
	dir2.FirstChild = file1

	dir3 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir3"),
		Previous: dir2,
	}
	dir2.Next = dir3

	file2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3,
		Path:   path.Join(dir3.Path, "file2"),
	}
	dir3.FirstChild = file2

	dir4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir4"),
		Previous: dir3,
	}
	dir3.Next = dir4

	file3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4,
		Path:   path.Join(dir4.Path, "file3"),
	}
	dir4.FirstChild = file3

	/*
		dir5
			dir6
				file4
			dir7
				file5
			dir8
				file6
	*/
	dir5 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir6 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir5,
		Path:   path.Join(dir5.Path, "dir2"),
	}
	dir5.FirstChild = dir6

	file4 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir6,
		Path:   path.Join(dir6.Path, "file1"),
	}
	dir6.FirstChild = file4

	dir7 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir5,
		Path:     path.Join(dir5.Path, "dir3"),
		Previous: dir6,
	}
	dir6.Next = dir7

	file5 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir7,
		Path:   path.Join(dir7.Path, "file2"),
	}
	dir7.FirstChild = file5

	dir8 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir5,
		Path:     path.Join(dir5.Path, "dir4"),
		Previous: dir7,
	}
	dir7.Next = dir8

	file6 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir8,
		Path:   path.Join(dir8.Path, "file3"),
	}
	dir8.FirstChild = file6

	/*
		dir9
			dir10
				file7
			dir11
				file8
			dir12
				file9
	*/
	dir9 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir9",
		Path: "dir9",
	}

	dir10 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir10",
		Parent: dir9,
		Path:   path.Join(dir9.Path, "dir10"),
	}
	dir9.FirstChild = dir10

	file7 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file7",
		Parent: dir10,
		Path:   path.Join(dir10.Path, "file7"),
	}
	dir10.FirstChild = file7

	dir11 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir11",
		Parent:   dir9,
		Path:     path.Join(dir9.Path, "dir11"),
		Previous: dir10,
	}
	dir10.Next = dir11

	file8 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file8",
		Parent: dir11,
		Path:   path.Join(dir11.Path, "file8"),
	}
	dir11.FirstChild = file8

	dir12 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir12",
		Parent:   dir9,
		Path:     path.Join(dir9.Path, "dir12"),
		Previous: dir11,
	}
	dir11.Next = dir12

	file9 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file9",
		Parent: dir12,
		Path:   path.Join(dir12.Path, "file9"),
	}
	dir12.FirstChild = file9

	tests := []struct {
		a   *AssetsTreeNode
		b   *AssetsTreeNode
		res bool
	}{
		{
			dir1,
			dir5,
			true,
		},
		{
			dir1,
			dir9,
			false,
		},
		{
			dir1,
			nil,
			false,
		},
		{
			nil,
			dir1,
			false,
		},
		{
			nil,
			nil,
			true,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := reflect.DeepEqual(test.a, test.b)

			if res != test.res {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}

func TestRemoveFromTree(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1,
		Path:   path.Join(dir1.Path, "dir2"),
	}
	dir1.FirstChild = dir2

	file1 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2,
		Path:   path.Join(dir2.Path, "file1"),
	}
	dir2.FirstChild = file1

	dir3 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir3"),
		Previous: dir2,
	}
	dir2.Next = dir3

	file2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3,
		Path:   path.Join(dir3.Path, "file2"),
	}
	dir3.FirstChild = file2

	dir4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir4"),
		Previous: dir3,
	}
	dir3.Next = dir4

	file3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4,
		Path:   path.Join(dir4.Path, "file3"),
	}
	dir4.FirstChild = file3

	/*
		AFTER REMOVAL:
		dir1
			dir3
				file2
			dir4
				file3
	*/
	dir1AD := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir3AD := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir3",
		Parent: dir1AD,
		Path:   path.Join(dir1AD.Path, "dir3"),
	}
	dir1AD.FirstChild = dir3AD

	file2AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3AD,
		Path:   path.Join(dir3AD.Path, "file2"),
	}
	dir3AD.FirstChild = file2AD

	dir4AD := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1AD,
		Path:     path.Join(dir1AD.Path, "dir4"),
		Previous: dir3AD,
	}
	dir3AD.Next = dir4AD

	file3AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4AD,
		Path:   path.Join(dir4AD.Path, "file3"),
	}
	dir4AD.FirstChild = file3AD

	dir2.RemoveFromTree()

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir2.Parent != nil {
		t.Errorf("parent should be nil")
	}

	if dir2.Next != nil {
		t.Errorf("next should be nil")
	}

	if dir2.Previous != nil {
		t.Errorf("previous should be nil")
	}
}

func TestRemoveFromTree_2(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1,
		Path:   path.Join(dir1.Path, "dir2"),
	}
	dir1.FirstChild = dir2

	file1 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2,
		Path:   path.Join(dir2.Path, "file1"),
	}
	dir2.FirstChild = file1

	dir3 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir3"),
		Previous: dir2,
	}
	dir2.Next = dir3

	file2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3,
		Path:   path.Join(dir3.Path, "file2"),
	}
	dir3.FirstChild = file2

	dir4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir4"),
		Previous: dir3,
	}
	dir3.Next = dir4

	file3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4,
		Path:   path.Join(dir4.Path, "file3"),
	}
	dir4.FirstChild = file3

	/*
		AFTER REMOVAL:
		dir1
			dir2
				file1
			dir4
				file3
	*/
	dir1AD := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2AD := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1AD,
		Path:   path.Join(dir1AD.Path, "dir2"),
	}
	dir1AD.FirstChild = dir2AD

	file1AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2AD,
		Path:   path.Join(dir2AD.Path, "file1"),
	}
	dir2AD.FirstChild = file1AD

	dir4AD := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1AD,
		Path:     path.Join(dir1AD.Path, "dir4"),
		Previous: dir2AD,
	}
	dir2AD.Next = dir4AD

	file3AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4AD,
		Path:   path.Join(dir4AD.Path, "file3"),
	}
	dir4AD.FirstChild = file3AD

	dir3.RemoveFromTree()

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir3.Parent != nil {
		t.Errorf("parent should be nil")
	}

	if dir3.Next != nil {
		t.Errorf("next should be nil")
	}

	if dir3.Previous != nil {
		t.Errorf("previous should be nil")
	}
}

func TestAddChild(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1,
		Path:   path.Join(dir1.Path, "dir2"),
	}
	dir1.FirstChild = dir2

	file1 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2,
		Path:   path.Join(dir2.Path, "file1"),
	}
	dir2.FirstChild = file1

	dir3 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir3"),
		Previous: dir2,
	}
	dir2.Next = dir3

	file2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3,
		Path:   path.Join(dir3.Path, "file2"),
	}
	dir3.FirstChild = file2

	dir4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir4"),
		Previous: dir3,
	}
	dir3.Next = dir4

	file3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4,
		Path:   path.Join(dir4.Path, "file3"),
	}
	dir4.FirstChild = file3

	/*
		AFTER ADDING:
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
			dir5
				file4
	*/
	dir1AD := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2AD := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1AD,
		Path:   path.Join(dir1AD.Path, "dir2"),
	}
	dir1AD.FirstChild = dir2AD

	file1AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2AD,
		Path:   path.Join(dir2AD.Path, "file1"),
	}
	dir2AD.FirstChild = file1AD

	dir3AD := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1AD,
		Path:     path.Join(dir1AD.Path, "dir3"),
		Previous: dir2AD,
	}
	dir2AD.Next = dir3AD

	file2AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3AD,
		Path:   path.Join(dir3AD.Path, "file2"),
	}
	dir3AD.FirstChild = file2AD

	dir4AD := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1AD,
		Path:     path.Join(dir1AD.Path, "dir4"),
		Previous: dir3AD,
	}
	dir3AD.Next = dir4AD

	file3AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4AD,
		Path:   path.Join(dir4AD.Path, "file3"),
	}
	dir4AD.FirstChild = file3AD

	dir5Name := "dir5"
	dir5AD := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     dir5Name,
		Parent:   dir1AD,
		Path:     path.Join(dir1AD.Path, dir5Name),
		Previous: dir4AD,
	}
	dir4AD.Next = dir5AD

	file4Name := "file4"
	file4AD := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   file4Name,
		Parent: dir5AD,
		Path:   path.Join(dir5AD.Path, file4Name),
	}
	dir5AD.FirstChild = file4AD

	dir5 := dir1.AddChild(DIRNODE, dir5Name)
	file4 := dir5.AddChild(FILENODE, file4Name)

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir5.Parent != dir1 {
		t.Errorf("parent of %v should be %v", dir5, dir1)
	}

	if file4.Parent != dir5 {
		t.Errorf("parent of %v should be %v", file4, dir5)
	}

	expectedDir5Path := path.Join(dir1.Path, dir5Name)
	if dir5.Path != expectedDir5Path {
		t.Errorf("got %v, want %v", dir5.Path, expectedDir5Path)
	}

	expectedFile4Path := path.Join(dir5.Path, file4Name)
	if file4.Path != expectedFile4Path {
		t.Errorf("got %v, want %v", file4.Path, expectedFile4Path)
	}
}

func TestTraverse(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "dir1",
		Path: "dir1",
	}

	dir2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir2",
		Parent: dir1,
		Path:   path.Join(dir1.Path, "dir2"),
	}
	dir1.FirstChild = dir2

	file1 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Parent: dir2,
		Path:   path.Join(dir2.Path, "file1"),
	}
	dir2.FirstChild = file1

	dir3 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir3",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir3"),
		Previous: dir2,
	}
	dir2.Next = dir3

	file2 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file2",
		Parent: dir3,
		Path:   path.Join(dir3.Path, "file2"),
	}
	dir3.FirstChild = file2

	dir4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "dir4",
		Parent:   dir1,
		Path:     path.Join(dir1.Path, "dir4"),
		Previous: dir3,
	}
	dir3.Next = dir4

	file3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file3",
		Parent: dir4,
		Path:   path.Join(dir4.Path, "file3"),
	}
	dir4.FirstChild = file3

	var res []string
	someErr := errors.New("some")

	tests := []struct {
		tree *AssetsTreeNode
		fn   AssetsTreeNodeTraverseFn
		err  error
		res  []string
	}{
		{
			dir1,
			func(n *AssetsTreeNode) (TraverseStatus, error) {
				res = append(res, n.Name)

				return Next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "file2", "dir4", "file3"},
		},
		{
			dir1,
			func(n *AssetsTreeNode) (TraverseStatus, error) {
				res = append(res, n.Name)

				if n.Name == "file2" {
					return Terminate, nil
				}

				return Next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "file2"},
		},
		{
			dir1,
			func(n *AssetsTreeNode) (TraverseStatus, error) {
				res = append(res, n.Name)

				if n.Name == "dir3" {
					return SkipChildren, nil
				}

				return Next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "dir4", "file3"},
		},
		{
			dir1,
			func(n *AssetsTreeNode) (TraverseStatus, error) {
				res = append(res, n.Name)

				if n.Name == "dir2" {
					return Next, someErr
				}

				return Next, nil
			},
			someErr,
			[]string{"dir1", "dir2"},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res = make([]string, 0)

			err := test.tree.Traverse(test.fn)

			if err != test.err {
				t.Errorf("got %v, want %v", err, test.err)
			}

			if !reflect.DeepEqual(res, test.res) {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}

func TestFindByRelPath(t *testing.T) {
	/*
		node1
			node2
				node3
			node4
	*/
	node1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: "foobar",
	}

	node2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir1",
		Path:   path.Join(node1.Path, "dir1"),
		Parent: node1,
	}
	node1.FirstChild = node2

	node3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Path:   path.Join(node2.Path, "file1"),
		Parent: node2,
	}
	node2.FirstChild = node3

	node4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "file2",
		Path:     path.Join(node1.Path, "file2"),
		Parent:   node1,
		Previous: node2,
	}
	node2.Next = node4

	tests := []struct {
		tree *AssetsTreeNode
		path string
		res  *AssetsTreeNode
	}{
		{
			node1,
			strings.TrimPrefix(node3.Path, node1.Path+"/"),
			node3,
		},
		{
			node1,
			path.Join(node2.Path, "foo", node3.Path),
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := test.tree.FindByRelPath(test.path)

			if res != test.res {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}

func TestFindByRelPathInGATOrPWAT(t *testing.T) {
	/*
		node1
			node2
				node3
			node4
	*/
	node1 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: "foobar",
	}

	node2 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir1",
		Path:   path.Join(node1.Path, "dir1"),
		Parent: node1,
	}
	node1.FirstChild = node2

	node3 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file1",
		Path:   path.Join(node2.Path, "file1"),
		Parent: node2,
	}
	node2.FirstChild = node3

	node4 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "file2",
		Path:     path.Join(node1.Path, "file2"),
		Parent:   node1,
		Previous: node2,
	}
	node2.Next = node4

	/*
		node5
			node6
				node7
			node8
	*/
	node5 := &AssetsTreeNode{
		Type: DIRNODE,
		Name: "assets",
		Path: "foobar",
	}

	node6 := &AssetsTreeNode{
		Type:   DIRNODE,
		Name:   "dir3",
		Path:   path.Join(node5.Path, "dir3"),
		Parent: node5,
	}
	node5.FirstChild = node6

	node7 := &AssetsTreeNode{
		Type:   FILENODE,
		Name:   "file4",
		Path:   path.Join(node6.Path, "file4"),
		Parent: node6,
	}
	node6.FirstChild = node7

	node8 := &AssetsTreeNode{
		Type:     DIRNODE,
		Name:     "file5",
		Path:     path.Join(node7.Path, "file5"),
		Parent:   node5,
		Previous: node6,
	}
	node6.Next = node8

	tests := []struct {
		path           AssetRelPath
		gat, pwat, res *AssetsTreeNode
		searchedInPWAT bool
	}{
		{
			AssetRelPath("/" + strings.TrimPrefix(node2.Path, node1.Path+"/")),
			node1,
			node5,
			node2,
			false,
		},
		{
			AssetRelPath(strings.TrimPrefix(node2.Path, node1.Path+"/")),
			node1,
			node5,
			nil,
			true,
		},
		{
			AssetRelPath(strings.TrimPrefix(node6.Path, node5.Path+"/")),
			node1,
			node5,
			node6,
			true,
		},
		{
			AssetRelPath("/" + strings.TrimPrefix(node6.Path, node5.Path+"/")),
			node1,
			node5,
			nil,
			false,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, searchedInPWAT := findByRelPathInGATOrPWAT(test.gat, test.pwat, test.path)

			if res != test.res {
				t.Errorf("got %v, want %v", res, test.res)
			}

			if searchedInPWAT != test.searchedInPWAT {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}
