// Package latex provides latex image generation for egen.
package latex

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	dirName             = ".egen-latex"
	initializedFileName = ".initialized"

	packageFileName    = "package.json"
	packageFileContent = `
	{
		"type": "module",
		"dependencies": {
			"mathjax-node": "^2.1.1"
		}
	}`

	scriptFileName    = "index.js"
	scriptFileContent = `
	import mj from 'mathjax-node';

	mj.start();
	mj.typeset(
		{
			math: process.argv[3],
			format: process.argv[2] === '--inline' ? 'inline-TeX' : 'TeX',
			svg: true,
		},
		(data) => {
			if (data.errors) {
				console.error(data.errors);
				process.exit(1);
			}

			console.log(data.svg);
		}
	);`
)

// ImageGenerator is a latex image generator.
type ImageGenerator struct {
	dirPath     string
	initiliazed bool
}

// NewImageGenerator creates a new latex image generator.
func NewImageGenerator(dirPath string) *ImageGenerator {
	return &ImageGenerator{
		dirPath: filepath.Join(dirPath, dirName),
	}
}

// SetDirPath sets the path of the directory that will be used by the generator.
func (g *ImageGenerator) SetDirPath(dirPath string) error {
	g.dirPath = filepath.Join(dirPath, dirName)

	_, err := os.Stat(filepath.Join(g.dirPath, initializedFileName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat %s file: %w", initializedFileName, err)
	}

	g.initiliazed = err == nil

	return nil
}

// SVGBlock generates a latex block svg image from math.
func (g *ImageGenerator) SVGBlock(math []byte) ([]byte, error) {
	return g.svg(math, false)
}

// SVGBlock generates an inline latex svg image from math.
func (g *ImageGenerator) SVGInline(math []byte) ([]byte, error) {
	return g.svg(math, true)
}

func (g *ImageGenerator) initDir() error {
	if g.initiliazed {
		return nil
	}

	err := os.Mkdir(g.dirPath, os.ModeDir|0755)
	if err != nil {
		return fmt.Errorf("creating %s directory: %w", dirName, err)
	}

	err = os.WriteFile(filepath.Join(g.dirPath, packageFileName), []byte(packageFileContent), 0644)
	if err != nil {
		return fmt.Errorf("writing %s file: %w", packageFileName, err)
	}

	err = os.WriteFile(filepath.Join(g.dirPath, scriptFileName), []byte(scriptFileContent), 0644)
	if err != nil {
		return fmt.Errorf("writing %s file: %w", scriptFileName, err)
	}

	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)

	cmd := exec.Command("npm", "install")

	cmd.Dir = g.dirPath
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("running npm install: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	err = os.WriteFile(filepath.Join(g.dirPath, initializedFileName), nil, 0644)
	if err != nil {
		return fmt.Errorf("creating %s file: %w", initializedFileName, err)
	}

	return nil
}

func (g *ImageGenerator) svg(math []byte, inline bool) ([]byte, error) {
	err := g.initDir()
	if err != nil {
		return nil, fmt.Errorf("init latex directory: %w", err)
	}

	stdout, stderr := bytes.NewBuffer(nil), bytes.NewBuffer(nil)

	args := []string{
		scriptFileName,
		"",
		string(math),
	}
	if inline {
		args[1] = "--inline"
	} else {
		args[1] = "--block"
	}

	cmd := exec.Command("node", args...)

	cmd.Dir = g.dirPath
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("running %s: %w\nstderr: %s", scriptFileName, err, stderr.String())
	}

	return stdout.Bytes(), nil
}
