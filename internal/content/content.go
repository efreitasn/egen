// Package content provides an API to interact with a website's content managed by ecms.
package content

import (
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

var configFilename = "ecms.yaml"

// WebsiteContent is the content of a website managed by ecms.
type WebsiteContent struct {
	path        string
	title       string
	url         string
	defaultLang *lang
	langs       map[string]*lang
}

type configFileData struct {
	Title string
	URL   string
	Langs []*lang
}

type lang struct {
	Name    string
	Tag     string
	Default bool
}

// New creates a WebsiteContent.
func New(websiteDirPath string) (*WebsiteContent, error) {
	wc := WebsiteContent{
		path:  websiteDirPath,
		langs: make(map[string]*lang),
	}

	// config file
	cFile, err := os.Open(path.Join(websiteDirPath, configFilename))
	if err != nil {
		return nil, err
	}

	var cFileData configFileData

	err = yaml.NewDecoder(cFile).Decode(&cFileData)
	if err != nil {
		return nil, err
	}

	for _, lang := range cFileData.Langs {
		if lang.Default {
			wc.defaultLang = lang
		}

		wc.langs[lang.Tag] = lang
	}

	wc.title = cFileData.Title
	wc.url = cFileData.URL

	return &wc, nil
}
