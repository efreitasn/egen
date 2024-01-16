package egen

import (
	"errors"
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

var configFilename = "egen.yaml"

// Author represents an author.
type Author struct {
	Name, Twitter string
}

// Img represents the image of the current page/post.
type Img struct {
	Path AssetRelPath
	Alt  string
}

// Map of internationalized versions of a string.
// Example: pt-BR -> foobar
type i18nStrings map[string]string

type configFileData struct {
	Title                     string
	Description               i18nStrings
	ImgAlt                    i18nStrings `yaml:"imgAlt"`
	URL                       string
	Color                     string
	Img                       AssetRelPath
	Langs                     []*Lang
	Author                    *Author
	ResponsiveImgSizes        []int  `yaml:"responsiveImgSizes"`
	ResponsiveImgMediaQueries string `yaml:"responsiveImgMediaQueries"`
	Latex                     bool
}

type config struct {
	configFileData

	defaultLang         *Lang
	defaultImgByLangTag map[string]*Img
}

func readConfigFile(InPath string) (*config, error) {
	cFilePath := path.Join(InPath, configFilename)
	cFile, err := os.Open(cFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %v", cFilePath)
		}

		return nil, err
	}

	var cFileData configFileData

	err = yaml.NewDecoder(cFile).Decode(&cFileData)
	if err != nil {
		return nil, fmt.Errorf("error while parsing the contents of config file: %v", err)
	}

	// checkings
	switch {
	case cFileData.Title == "":
		return nil, errors.New("title field in config file cannot be empty")
	case cFileData.URL == "":
		return nil, errors.New("url field in config file cannot be empty")
	case len(cFileData.Langs) == 0:
		return nil, errors.New("langs field in config file cannot be empty")
	case cFileData.Author == nil:
		return nil, errors.New("author field in config file cannot be empty")
	case cFileData.Author.Name == "":
		return nil, errors.New("author.name field in config file cannot be empty")
	}

	var c config

	// default img
	c.configFileData = cFileData
	c.defaultImgByLangTag = make(map[string]*Img, len(cFileData.ImgAlt))

	// default lang
	for _, lang := range cFileData.Langs {
		if cFileData.Description[lang.Tag] == "" {
			return nil, fmt.Errorf("description in %v in config file not provided", lang.Tag)
		}

		if cFileData.Img != "" {
			if cFileData.ImgAlt[lang.Tag] == "" {
				return nil, fmt.Errorf("alt for default image in %v in config file not provided", lang.Tag)
			}

			c.defaultImgByLangTag[lang.Tag] = &Img{
				Path: cFileData.Img,
				Alt:  cFileData.ImgAlt[lang.Tag],
			}
		}

		if lang.Default {
			c.defaultLang = lang
		}
	}

	if c.defaultLang == nil {
		return nil, errors.New("there must a default lang in the config file")
	}

	return &c, nil
}
