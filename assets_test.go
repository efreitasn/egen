package egen

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func printDebugNode(n *assetsTreeNode) {
	n.traverse(func(n *assetsTreeNode) (traverseStatus, error) {
		fmt.Printf("%p - %+v\n", n, n)

		return next, nil
	})
}

func TestGenerateAssetsTree(t *testing.T) {
	rootNode := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: "testdata/tree/ok/1",
	}

	fooNode := &assetsTreeNode{
		t:      FILENODE,
		name:   "foo.txt",
		path:   path.Join(rootNode.path, "foo.txt"),
		parent: rootNode,
	}
	rootNode.firstChild = fooNode

	imgsDirNode := &assetsTreeNode{
		t:        DIRNODE,
		name:     "imgs",
		path:     path.Join(rootNode.path, "imgs"),
		previous: fooNode,
		parent:   rootNode,
	}
	fooNode.next = imgsDirNode

	redImgNode := &assetsTreeNode{
		t:      IMGNODE,
		name:   "red.png",
		parent: imgsDirNode,
		path:   path.Join(imgsDirNode.path, "red.png"),
		sizes: []*assetsTreeNodeImgSize{
			{
				original: true,
				width:    1920,
			},
		},
	}
	imgsDirNode.firstChild = redImgNode

	rootNode2 := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: "testdata/tree/ok/1",
	}

	fooNode2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "foo.txt",
		path:   path.Join(rootNode2.path, "foo.txt"),
		parent: rootNode2,
	}
	rootNode2.firstChild = fooNode2

	tests := []struct {
		path          string
		err           error
		tree          *assetsTreeNode
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
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

	/*
		dir5
			dir6
				file4
			dir7
				file5
			dir8
				file6
	*/
	dir5 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir6 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir5,
		path:   path.Join(dir5.path, "dir2"),
	}
	dir5.firstChild = dir6

	file4 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir6,
		path:   path.Join(dir6.path, "file1"),
	}
	dir6.firstChild = file4

	dir7 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir5,
		path:     path.Join(dir5.path, "dir3"),
		previous: dir6,
	}
	dir6.next = dir7

	file5 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir7,
		path:   path.Join(dir7.path, "file2"),
	}
	dir7.firstChild = file5

	dir8 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir5,
		path:     path.Join(dir5.path, "dir4"),
		previous: dir7,
	}
	dir7.next = dir8

	file6 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir8,
		path:   path.Join(dir8.path, "file3"),
	}
	dir8.firstChild = file6

	/*
		dir9
			dir10
				file7
			dir11
				file8
			dir12
				file9
	*/
	dir9 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir9",
		path: "dir9",
	}

	dir10 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir10",
		parent: dir9,
		path:   path.Join(dir9.path, "dir10"),
	}
	dir9.firstChild = dir10

	file7 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file7",
		parent: dir10,
		path:   path.Join(dir10.path, "file7"),
	}
	dir10.firstChild = file7

	dir11 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir11",
		parent:   dir9,
		path:     path.Join(dir9.path, "dir11"),
		previous: dir10,
	}
	dir10.next = dir11

	file8 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file8",
		parent: dir11,
		path:   path.Join(dir11.path, "file8"),
	}
	dir11.firstChild = file8

	dir12 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir12",
		parent:   dir9,
		path:     path.Join(dir9.path, "dir12"),
		previous: dir11,
	}
	dir11.next = dir12

	file9 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file9",
		parent: dir12,
		path:   path.Join(dir12.path, "file9"),
	}
	dir12.firstChild = file9

	tests := []struct {
		a   *assetsTreeNode
		b   *assetsTreeNode
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
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

	/*
		AFTER REMOVAL:
		dir1
			dir3
				file2
			dir4
				file3
	*/
	dir1AD := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir3AD := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir3",
		parent: dir1AD,
		path:   path.Join(dir1AD.path, "dir3"),
	}
	dir1AD.firstChild = dir3AD

	file2AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3AD,
		path:   path.Join(dir3AD.path, "file2"),
	}
	dir3AD.firstChild = file2AD

	dir4AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir4"),
		previous: dir3AD,
	}
	dir3AD.next = dir4AD

	file3AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4AD,
		path:   path.Join(dir4AD.path, "file3"),
	}
	dir4AD.firstChild = file3AD

	dir2.removeFromTree()

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir2.parent != nil {
		t.Errorf("parent should be nil")
	}

	if dir2.next != nil {
		t.Errorf("next should be nil")
	}

	if dir2.previous != nil {
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
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

	/*
		AFTER REMOVAL:
		dir1
			dir2
				file1
			dir4
				file3
	*/
	dir1AD := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2AD := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1AD,
		path:   path.Join(dir1AD.path, "dir2"),
	}
	dir1AD.firstChild = dir2AD

	file1AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2AD,
		path:   path.Join(dir2AD.path, "file1"),
	}
	dir2AD.firstChild = file1AD

	dir4AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir4"),
		previous: dir2AD,
	}
	dir2AD.next = dir4AD

	file3AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4AD,
		path:   path.Join(dir4AD.path, "file3"),
	}
	dir4AD.firstChild = file3AD

	dir3.removeFromTree()

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir3.parent != nil {
		t.Errorf("parent should be nil")
	}

	if dir3.next != nil {
		t.Errorf("next should be nil")
	}

	if dir3.previous != nil {
		t.Errorf("previous should be nil")
	}
}

func TestAddChild_1(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
			dir6
				file5
	*/
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

	dir6 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir6",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir6"),
		previous: dir4,
	}
	dir4.next = dir6

	file5 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file5",
		parent: dir6,
		path:   path.Join(dir6.path, "file5"),
	}
	dir6.firstChild = file5

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
			dir6
				file5
	*/
	dir1AD := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2AD := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1AD,
		path:   path.Join(dir1AD.path, "dir2"),
	}
	dir1AD.firstChild = dir2AD

	file1AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2AD,
		path:   path.Join(dir2AD.path, "file1"),
	}
	dir2AD.firstChild = file1AD

	dir3AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir3"),
		previous: dir2AD,
	}
	dir2AD.next = dir3AD

	file2AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3AD,
		path:   path.Join(dir3AD.path, "file2"),
	}
	dir3AD.firstChild = file2AD

	dir4AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir4"),
		previous: dir3AD,
	}
	dir3AD.next = dir4AD

	file3AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4AD,
		path:   path.Join(dir4AD.path, "file3"),
	}
	dir4AD.firstChild = file3AD

	dir5Name := "dir5"
	dir5AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     dir5Name,
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, dir5Name),
		previous: dir4AD,
	}
	dir4AD.next = dir5AD

	file4Name := "file4"
	file4AD := &assetsTreeNode{
		t:      FILENODE,
		name:   file4Name,
		parent: dir5AD,
		path:   path.Join(dir5AD.path, file4Name),
	}
	dir5AD.firstChild = file4AD

	dir6AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir6",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir6"),
		previous: dir5AD,
	}
	dir5AD.next = dir6AD

	file5AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file5",
		parent: dir6AD,
		path:   path.Join(dir6AD.path, "file5"),
	}
	dir6AD.firstChild = file5AD

	dir5 := dir1.addChild(DIRNODE, dir5Name)
	file4 := dir5.addChild(FILENODE, file4Name)

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir5.parent != dir1 {
		t.Errorf("parent of %v should be %v", dir5, dir1)
	}

	if file4.parent != dir5 {
		t.Errorf("parent of %v should be %v", file4, dir5)
	}

	expectedDir5Path := path.Join(dir1.path, dir5Name)
	if dir5.path != expectedDir5Path {
		t.Errorf("got %v, want %v", dir5.path, expectedDir5Path)
	}

	expectedFile4Path := path.Join(dir5.path, file4Name)
	if file4.path != expectedFile4Path {
		t.Errorf("got %v, want %v", file4.path, expectedFile4Path)
	}
}

func TestAddChild_2(t *testing.T) {
	/*
		dir1
			dir2
				file1
			dir3
				file2
			dir4
				file3
	*/
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

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
	dir1AD := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2AD := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1AD,
		path:   path.Join(dir1AD.path, "dir2"),
	}
	dir1AD.firstChild = dir2AD

	file1AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2AD,
		path:   path.Join(dir2AD.path, "file1"),
	}
	dir2AD.firstChild = file1AD

	dir3AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir3"),
		previous: dir2AD,
	}
	dir2AD.next = dir3AD

	file2AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3AD,
		path:   path.Join(dir3AD.path, "file2"),
	}
	dir3AD.firstChild = file2AD

	dir4AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, "dir4"),
		previous: dir3AD,
	}
	dir3AD.next = dir4AD

	file3AD := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4AD,
		path:   path.Join(dir4AD.path, "file3"),
	}
	dir4AD.firstChild = file3AD

	dir5Name := "dir5"
	dir5AD := &assetsTreeNode{
		t:        DIRNODE,
		name:     dir5Name,
		parent:   dir1AD,
		path:     path.Join(dir1AD.path, dir5Name),
		previous: dir4AD,
	}
	dir4AD.next = dir5AD

	file4Name := "file4"
	file4AD := &assetsTreeNode{
		t:      FILENODE,
		name:   file4Name,
		parent: dir5AD,
		path:   path.Join(dir5AD.path, file4Name),
	}
	dir5AD.firstChild = file4AD

	dir5 := dir1.addChild(DIRNODE, dir5Name)
	file4 := dir5.addChild(FILENODE, file4Name)

	if !reflect.DeepEqual(dir1, dir1AD) {
		t.Error("trees are not equal")
	}

	if dir5.parent != dir1 {
		t.Errorf("parent of %v should be %v", dir5, dir1)
	}

	if file4.parent != dir5 {
		t.Errorf("parent of %v should be %v", file4, dir5)
	}

	expectedDir5Path := path.Join(dir1.path, dir5Name)
	if dir5.path != expectedDir5Path {
		t.Errorf("got %v, want %v", dir5.path, expectedDir5Path)
	}

	expectedFile4Path := path.Join(dir5.path, file4Name)
	if file4.path != expectedFile4Path {
		t.Errorf("got %v, want %v", file4.path, expectedFile4Path)
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
	dir1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "dir1",
		path: "dir1",
	}

	dir2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir2",
		parent: dir1,
		path:   path.Join(dir1.path, "dir2"),
	}
	dir1.firstChild = dir2

	file1 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		parent: dir2,
		path:   path.Join(dir2.path, "file1"),
	}
	dir2.firstChild = file1

	dir3 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir3",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir3"),
		previous: dir2,
	}
	dir2.next = dir3

	file2 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file2",
		parent: dir3,
		path:   path.Join(dir3.path, "file2"),
	}
	dir3.firstChild = file2

	dir4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "dir4",
		parent:   dir1,
		path:     path.Join(dir1.path, "dir4"),
		previous: dir3,
	}
	dir3.next = dir4

	file3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file3",
		parent: dir4,
		path:   path.Join(dir4.path, "file3"),
	}
	dir4.firstChild = file3

	var res []string
	someErr := errors.New("some")

	tests := []struct {
		tree *assetsTreeNode
		fn   assetsTreeNodeTraverseFn
		err  error
		res  []string
	}{
		{
			dir1,
			func(n *assetsTreeNode) (traverseStatus, error) {
				res = append(res, n.name)

				return next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "file2", "dir4", "file3"},
		},
		{
			dir1,
			func(n *assetsTreeNode) (traverseStatus, error) {
				res = append(res, n.name)

				if n.name == "file2" {
					return terminate, nil
				}

				return next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "file2"},
		},
		{
			dir1,
			func(n *assetsTreeNode) (traverseStatus, error) {
				res = append(res, n.name)

				if n.name == "dir3" {
					return skipChildren, nil
				}

				return next, nil
			},
			nil,
			[]string{"dir1", "dir2", "file1", "dir3", "dir4", "file3"},
		},
		{
			dir1,
			func(n *assetsTreeNode) (traverseStatus, error) {
				res = append(res, n.name)

				if n.name == "dir2" {
					return next, someErr
				}

				return next, nil
			},
			someErr,
			[]string{"dir1", "dir2"},
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res = make([]string, 0)

			err := test.tree.traverse(test.fn)

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
	node1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: "foobar",
	}

	node2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir1",
		path:   path.Join(node1.path, "dir1"),
		parent: node1,
	}
	node1.firstChild = node2

	node3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		path:   path.Join(node2.path, "file1"),
		parent: node2,
	}
	node2.firstChild = node3

	node4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "file2",
		path:     path.Join(node1.path, "file2"),
		parent:   node1,
		previous: node2,
	}
	node2.next = node4

	tests := []struct {
		tree *assetsTreeNode
		path string
		res  *assetsTreeNode
	}{
		{
			node1,
			strings.TrimPrefix(node3.path, node1.path+"/"),
			node3,
		},
		{
			node1,
			path.Join(node2.path, "foo", node3.path),
			nil,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res := test.tree.findByRelPath(test.path)

			if res != test.res {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}

func TestFindByRelPathInGATOrPAT(t *testing.T) {
	/*
		node1
			node2
				node3
			node4
	*/
	node1 := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: "foobar",
	}

	node2 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir1",
		path:   path.Join(node1.path, "dir1"),
		parent: node1,
	}
	node1.firstChild = node2

	node3 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file1",
		path:   path.Join(node2.path, "file1"),
		parent: node2,
	}
	node2.firstChild = node3

	node4 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "file2",
		path:     path.Join(node1.path, "file2"),
		parent:   node1,
		previous: node2,
	}
	node2.next = node4

	/*
		node5
			node6
				node7
			node8
	*/
	node5 := &assetsTreeNode{
		t:    DIRNODE,
		name: "assets",
		path: "foobar",
	}

	node6 := &assetsTreeNode{
		t:      DIRNODE,
		name:   "dir3",
		path:   path.Join(node5.path, "dir3"),
		parent: node5,
	}
	node5.firstChild = node6

	node7 := &assetsTreeNode{
		t:      FILENODE,
		name:   "file4",
		path:   path.Join(node6.path, "file4"),
		parent: node6,
	}
	node6.firstChild = node7

	node8 := &assetsTreeNode{
		t:        DIRNODE,
		name:     "file5",
		path:     path.Join(node7.path, "file5"),
		parent:   node5,
		previous: node6,
	}
	node6.next = node8

	tests := []struct {
		path          AssetRelPath
		gat, pat, res *assetsTreeNode
		searchedInPAT bool
	}{
		{
			AssetRelPath("/" + strings.TrimPrefix(node2.path, node1.path+"/")),
			node1,
			node5,
			node2,
			false,
		},
		{
			AssetRelPath(strings.TrimPrefix(node2.path, node1.path+"/")),
			node1,
			node5,
			nil,
			true,
		},
		{
			AssetRelPath(strings.TrimPrefix(node6.path, node5.path+"/")),
			node1,
			node5,
			node6,
			true,
		},
		{
			AssetRelPath("/" + strings.TrimPrefix(node6.path, node5.path+"/")),
			node1,
			node5,
			nil,
			false,
		},
	}

	for i, test := range tests {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			res, searchedInPAT := findByRelPathInGATOrPAT(test.gat, test.pat, test.path)

			if res != test.res {
				t.Errorf("got %v, want %v", res, test.res)
			}

			if searchedInPAT != test.searchedInPAT {
				t.Errorf("got %v, want %v", res, test.res)
			}
		})
	}
}
