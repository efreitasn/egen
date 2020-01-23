package egen

import (
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
	"time"
)

func TestBuild_ok(t *testing.T) {
	baseDir := path.Join("testdata", "build", "ok")

	tests := []struct {
		bc       BuildConfig
		expected string
	}{
		{
			BuildConfig{
				InPath:  path.Join(baseDir, "1", "in"),
				OutPath: path.Join(baseDir, "1", "test_output"),
				Funcs: template.FuncMap{
					"formatDateByLang": func(date time.Time, l *Lang) string {
						switch l.Tag {
						case "en":
							return date.Format("01/02/2006")
						case "pt-BR":
							return date.Format("02/01/2006")
						default:
							return ""
						}
					},
				},
			},
			path.Join(baseDir, "1", "out"),
		},
	}

	for _, test := range tests {
		t.Run(test.bc.InPath, func(t *testing.T) {
			err := Build(test.bc)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			// defer os.RemoveAll(buildOutPath)

			compareDirsRec(t, test.bc.OutPath, test.expected)
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
