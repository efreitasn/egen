// Package htmlp provides an HTML prettifier.
package htmlp

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var tagsWhoseContentDoesntNeedIndentation = regexp.MustCompile("^(?:pre|p|h1|h2|h3|h4|h5|h6)$")
var ingoreAttrRx = regexp.MustCompile(".* data-htmlp-ignore.*")

// Pretty prettifies the given HTML.
func Pretty(data []byte) ([]byte, error) {
	r := bytes.NewReader(data)

	t := html.NewTokenizer(r)

	var buff bytes.Buffer

	err := renderToken(t, &buff, 0)
	if err != nil {
		return nil, err
	}

	// f, _ := os.Create("ff.html")
	// buff.WriteTo(f)

	return buff.Bytes(), nil
}

func renderToken(t *html.Tokenizer, w *bytes.Buffer, depth int) error {
	tt := t.Next()

	tagBs, _ := t.TagName()
	tag := string(tagBs)
	void := isVoid(tag)

	switch tt {
	case html.ErrorToken:
		return nil
	case html.DoctypeToken:
		w.WriteString("<!DOCTYPE html>")
		w.WriteString("\n")
	case html.StartTagToken:
		r := t.Raw()

		w.WriteString(strings.Repeat("  ", depth))

		if containsIgnoreAttr(r) || tagsWhoseContentDoesntNeedIndentation.MatchString(tag) {
			r = removeIgnoreAttr(r)
			w.Write(r)

			if !void {
				for {
					tt := t.Next()
					childTag, _ := t.TagName()

					if tt == html.ErrorToken {
						break
					}

					w.Write(t.Raw())

					if tt == html.EndTagToken && tag == string(childTag) {
						w.WriteString("\n")

						break
					}
				}
			}
		} else {
			w.Write(r)
			w.WriteString("\n")

			if !void {
				depth++
			}
		}
	case html.EndTagToken:
		depth--
		w.WriteString(strings.Repeat("  ", depth))
		w.Write(t.Raw())
		w.WriteString("\n")
	case html.SelfClosingTagToken:
		w.WriteString(strings.Repeat("  ", depth))
		w.Write(t.Raw())
		w.WriteString("\n")
	case html.TextToken:
		r := bytes.Trim(t.Raw(), " \n\t")

		if len(r) > 0 {
			w.WriteString(strings.Repeat("  ", depth))
			w.Write(r)
			w.WriteString("\n")
		}
	}

	return renderToken(t, w, depth)
}

// https://html.spec.whatwg.org/multipage/syntax.html#void-elements
func isVoid(tag string) bool {
	return (tag == "area" ||
		tag == "base" ||
		tag == "br" ||
		tag == "col" ||
		tag == "embed" ||
		tag == "hr" ||
		tag == "img" ||
		tag == "input" ||
		tag == "link" ||
		tag == "meta" ||
		tag == "param" ||
		tag == "source" ||
		tag == "track" ||
		tag == "wbr")
}

func containsIgnoreAttr(bs []byte) bool {
	return ingoreAttrRx.Match(bs)
}

func removeIgnoreAttr(bs []byte) []byte {
	return bytes.Replace(bs, []byte(" data-htmlp-ignore"), []byte{}, 1)
}
