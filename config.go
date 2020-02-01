package egen

import (
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

// Map of internationalized versions of a string.
// Example: pt-BR -> foobar
type i18nStrings map[string]string

type configFileData struct {
	Title       string
	Description i18nStrings
	ImgAlt      i18nStrings `yaml:"imgAlt"`
	URL         string
	Img         string
	Langs       []*Lang
	Author      *Author
	Keywords    map[string]i18nStrings
}

type config struct {
	configFileData

	defaultLang         *Lang
	defaultImgByLangTag map[string]*Img
}

func readConfigFile(inPath string) (*config, error) {
	cFilePath := path.Join(inPath, configFilename)
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

	var c config

	// default lang
	for _, lang := range cFileData.Langs {
		if lang.Default {
			c.defaultLang = lang
			break
		}
	}

	// default img
	c.configFileData = cFileData
	c.defaultImgByLangTag = make(map[string]*Img, len(cFileData.ImgAlt))

	if cFileData.Img != "" {
		defaultImgWidth, defaultImgHeight, err := imgDimensions(path.Join(inPath, "static", cFileData.Img))
		if err != nil {
			return nil, err
		}

		for langTag, alt := range cFileData.ImgAlt {
			c.defaultImgByLangTag[langTag] = &Img{
				Name:   cFileData.Img,
				Alt:    alt,
				Width:  defaultImgWidth,
				Height: defaultImgHeight,
			}
		}
	}

	return &c, nil
}
