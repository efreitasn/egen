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
	okDir := path.Join("testdata", "build", "ok")
	errDir := path.Join("testdata", "build", "err")

	tests := []struct {
		bc        BuildConfig
		nonNilErr bool
		expected  string
	}{
		{
			BuildConfig{
				InPath:  path.Join(okDir, "1", "in"),
				OutPath: path.Join(okDir, "1", "test_output"),
				PreGATProc: func(gat *AssetsTreeNode) {
					aaNode := gat.AddChild(FILENODE, "aa.txt")
					aaNode.SetContent([]byte("aa"))
				},
				PrePWATProc: func(postSlug string, pwat *AssetsTreeNode) {
					if postSlug == "third" {
						zzNode := pwat.AddChild(FILENODE, "zz.txt")
						zzNode.SetContent([]byte("zz"))
					}
				},
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
			false,
			path.Join(okDir, "1", "out"),
		},
		{
			BuildConfig{
				InPath:  path.Join(okDir, "2", "in"),
				OutPath: path.Join(okDir, "2", "test_output"),
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
			false,
			path.Join(okDir, "2", "out"),
		},
		{
			BuildConfig{
				InPath:  path.Join(errDir, "1", "in"),
				OutPath: path.Join(errDir, "1", "test_output"),
				PreGATProc: func(gat *AssetsTreeNode) {
					aaNode := gat.AddChild(FILENODE, "aa.txt")
					aaNode.SetContent([]byte("aa"))
				},
				PrePWATProc: func(postSlug string, pwat *AssetsTreeNode) {
					if postSlug == "third" {
						zzNode := pwat.AddChild(FILENODE, "zz.txt")
						zzNode.SetContent([]byte("zz"))
					}
				},
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
			true,
			path.Join(errDir, "1", "out"),
		},
		{
			BuildConfig{
				InPath:  path.Join(errDir, "2", "in"),
				OutPath: path.Join(errDir, "2", "test_output"),
				PreGATProc: func(gat *AssetsTreeNode) {
					aaNode := gat.AddChild(FILENODE, "aa.txt")
					aaNode.SetContent([]byte("aa"))
				},
				PrePWATProc: func(postSlug string, pwat *AssetsTreeNode) {
					if postSlug == "third" {
						zzNode := pwat.AddChild(FILENODE, "zz.txt")
						zzNode.SetContent([]byte("zz"))
					}
				},
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
			true,
			path.Join(errDir, "2", "out"),
		},
		{
			BuildConfig{
				InPath:  path.Join(errDir, "3", "in"),
				OutPath: path.Join(errDir, "3", "test_output"),
				PreGATProc: func(gat *AssetsTreeNode) {
					aaNode := gat.AddChild(FILENODE, "aa.txt")
					aaNode.SetContent([]byte("aa"))
				},
				PrePWATProc: func(postSlug string, pwat *AssetsTreeNode) {
					if postSlug == "third" {
						zzNode := pwat.AddChild(FILENODE, "zz.txt")
						zzNode.SetContent([]byte("zz"))
					}
				},
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
			true,
			path.Join(errDir, "3", "out"),
		},
		{
			BuildConfig{
				InPath:  path.Join(errDir, "4", "in"),
				OutPath: path.Join(errDir, "4", "test_output"),
				PreGATProc: func(gat *AssetsTreeNode) {
					aaNode := gat.AddChild(FILENODE, "aa.txt")
					aaNode.SetContent([]byte("aa"))
				},
				PrePWATProc: func(postSlug string, pwat *AssetsTreeNode) {
					if postSlug == "third" {
						zzNode := pwat.AddChild(FILENODE, "zz.txt")
						zzNode.SetContent([]byte("zz"))
					}
				},
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
			true,
			path.Join(errDir, "4", "out"),
		},
	}

	for _, test := range tests {
		t.Run(test.bc.InPath, func(t *testing.T) {
			err := Build(test.bc)
			if test.nonNilErr {
				if err == nil {
					t.Fatal("expected an error")
				}

				return
			} else if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			// defer os.RemoveAll(buildOutPath)

			compareDirsRec(t, test.expected, test.bc.OutPath)
		})
	}
}

func compareDirsRec(t *testing.T, a, b string) {
	aFilesDirs, err := ioutil.ReadDir(a)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	bFilesDirs, err := ioutil.ReadDir(b)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if len(aFilesDirs) > len(bFilesDirs) {
		t.Fatalf("there are files in %v that don't exist in %v", a, b)
	} else if len(aFilesDirs) < len(bFilesDirs) {
		t.Fatalf("there are files in %v that don't exist in %v", b, a)
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
			t.Errorf("content in %v is not the same as %v", aFileDirPath, bFileDirPath)
		}
	}
}
