package content

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestBuild_ok(t *testing.T) {
	dirs, err := ioutil.ReadDir("testdata/build/ok")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		t.Run(dir.Name(), func(t *testing.T) {
			buildOutPath := path.Join("testdata/build/ok", dir.Name(), "test_output")
			expectedBuildOutPath := path.Join("testdata/build/ok", dir.Name(), "out")

			wc, err := New(path.Join("testdata/build/ok", dir.Name(), "in"))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			// defer os.RemoveAll(buildOutPath)

			err = wc.Build(buildOutPath)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			compareDirsRec(t, expectedBuildOutPath, buildOutPath)
		})
	}
}

func compareDirsRec(t *testing.T, a, b string) {
	aFilesDirs, err := ioutil.ReadDir(a)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	for _, aFileDirInfo := range aFilesDirs {
		aFileDirPath := path.Join(a, aFileDirInfo.Name())
		bFileDirPath := path.Join(b, aFileDirInfo.Name())
		bFileDirInfo, err := os.Stat(bFileDirPath)
		if os.IsNotExist(err) {
			t.Fatalf("%v file/directory should exist", bFileDirPath)
		}

		if aFileDirInfo.IsDir() != bFileDirInfo.IsDir() {
			if aFileDirInfo.IsDir() {
				t.Fatalf("%v should be a directory", bFileDirPath)
			}

			t.Fatalf("%v should not be a directory", bFileDirPath)
		}

		if aFileDirInfo.IsDir() {
			compareDirsRec(
				t,
				path.Join(a, aFileDirInfo.Name()),
				path.Join(b, bFileDirInfo.Name()),
			)

			continue
		}

		aFileDir, err := os.Open(aFileDirPath)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		aFileDirContentBs, err := ioutil.ReadAll(aFileDir)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		bFileDir, err := os.Open(bFileDirPath)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		bFileDirContentBs, err := ioutil.ReadAll(bFileDir)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		if !reflect.DeepEqual(aFileDirContentBs, bFileDirContentBs) {
			t.Errorf("content in %v is not the expected", bFileDirPath)
		}
	}
}
