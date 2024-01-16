egen is an opinionated blog generator. It was created mainly to be used in [https://efreitasn.dev](https://efreitasn.dev/posts/a-new-version/). Some of its features are (when the word "must" appears, it means that the blog won't build if the specified condition isn't met):

* Uses Go templates.
* Every CSS file present in the `<inPath>/assets` directory becomes one single minified CSS file called `style.css` stored in `<outPath>/assets`. The order of concatenation is alphabetically, which means the content of a file named `1.css` will come first in the resulting `style.css` than the content of a file named `a.css`, for example.
* Every file stored in `<outPath>/assets` is renamed to `<filename_base>-<md5sum(file_content)>.<filename_ext>`, except JPEG and PNG images.
* Every JPEG and PNG image file present in the `<inPath>/assets` directory become a directory in `<outPath>/assets` whose name is the md5sum of the file. The files in this directory are named `<width>.<png|jpg|jpeg>`.
* Every post must have a version for each language provided in the config file.
* Every image used in a post must have an alt attribute.
* The icon of the blog is a file located at `<inPath>/assets/icon.png`.
* Supports responsive images by the `responsiveImgSizes` and `responsiveImgMediaQueries` fields present in the config file. The former is used to generate the `srcset` attribute and the latter is used as the `sizes` attribute. From that, `egen` handles the creation of resized images. All of this behaviour is automatic to any image encountered in a post, but responsive images can also be used outside of a post. This is achieved through the `srcSetValue` template function and the `TemplateData.ResponsiveImgMediaQueries` value.

## Terms
There are some terms used in `egen` that need some clarification.

* **TemplateData**: a struct received by a template. To see its fields, check [this page](https://pkg.go.dev/github.com/efreitasn/egen?tab=doc#TemplateData).
* **GAT**: short for global assets tree. It's a tree generated from the `<inPath>/assets` directory.
* **PAT**: short for post assets tree. It's a tree generated for each post from the `<inPath>/posts/<post_slug>` directory. It's composed of any file whose name doesn't match `/(^content_.+\.md$)|(^data\.yaml$)|(^.*/$)/` (when buidling the tree, directory names end with a `/` when matching against a regular expression).
* **Invisible post**: a name for posts whose config file's `feed` field is set to `false`. These posts are not present in the list provided in `TemplateData` and can only be "found" through the `getInvisiblePost` template function. This type of post serves the purpose of a page in a blog.
* **AssetRelPath**: the path of an asset relative to a GAT or a PAT. If the path starts with a `/`, it's relative to the former, while any other character at the beginning of the string makes it relative to the latter.
* **inPath**: the path used as input when building. It's the path that contains the config file.
* **outPath**: the path used as output when building.

## Config file
The config file is located at `<inPath>/egen.yaml`. An example of a config file is:
```yaml
title: foobar
description:
  en: foobar in english
  pt-BR: foobar em português
url: https://foo.bar
color: "#000000"
author:
  name: John Doe
  twitter: jjjjjdoee
langs:
  - tag: en
    name: English
    default: true
  - tag: pt-BR
    name: Português do Brasil
responsiveImgSizes:
  - 425
  - 640
  - 960
  - 1280
responsiveImgMediaQueries: "(max-width: 26.5625em) 100vw, (max-width: 64em) 65vw, 50vw"
latex: true
```

## Functions
These are the functions that can be used in a template:

* **dateISO(d time.Time) string**: transforms a `time.Time` into an ISO 8601 string.
* **getInvisiblePost(l \*Lang, slug string) \*Post**: returns an invisible post (`feed: false`) given a `Lang` and the post's slug.
* **assetLink(assetPath AssetRelPath) (string, error)**: returns the link of an asset given an `AssetRelPath`.
* **srcSetValue(assetPath AssetRelPath) (string, error)**: given an `AssetRelPath`, adds the sizes provided in the config file to the asset and returns a string to be used as the `srcset` attribute's value.
* **hasAsset(assetPath AssetRelPath) bool**: returns whether there's a node in the GAT or the current PAT that has a path equal to `assetPath`.
* **postLinkBySlugAndLang(slug string, l \*Lang) string**: given the post's slug and a `Lang`, returns a link to the post.
* **homeLinkByLang(l \*Lang) string**: given a `Lang`, returns a link to the home of the blog.
* **relToAbsLink(link string) string**: given a relative link, returns its absolute version.
* **sortPostsByDateDesc(posts []\*Post) []\*Post**: given a list of posts, returns the list sorted by post creation date in descending order.

## Posts
A post is located at `<inPath>/posts/<post_slug>`. The slug is like an ID, i.e. it's a unique string that each post has. Inside this directory, there's a file called `data.yaml` with the following structure:

```yaml
feed: true
date: "2019-07-07T21:43:00Z"
lastUpdateDate: "2020-02-19T01:04:33.663Z"
img: /foo.png
```

`img` and `lastUpdateDate` fields are optional.

This directory also contains one or more files named `content_<lang_tag>.md`. The number of files matching this pattern must be equal to the number of languages provided in the config file. In other words, as said in the beginning, a post must have a version for each specified language. The content file has the following structure:

```markdown
---
title: First post
excerpt: The first
imgAlt: some
---
content in markdown.
```

It starts with a YAML frontmatter followed by the post's content in Markdown. The `title` and `excerpt` fields are required, while the `imgAlt` is only required if the `img` field in the post's `data.yaml` was specified.

## Templates
There are three templates that are required and they're located at: `<inPath>/pages/404.html`, `<inPath>/pages/home.html` and `<inPath>/pages/post.html`. Besides the required templates, there are also arbitrary templates. They are created by placing a file named `<template_name>.html` at `<inPath>/includes`. This file shouldn't start with `{{ define }}` and end with `{{ end }}`, since the template name is just the file's name and there shouldn't be more than one template per file. As a special case, if there's a template located at `<inPath>/includes/head.html`, this template is rendered right before the end of the head tag automatically.

## `<inPath>` structure
`<inPath>` must have the following structure:
```
assets
includes
  <template_name>.html
pages
  404.html
  post.html
  home.html
posts
  <post_slug>
    content_<lang_tag>.md
    data.yaml
egen.yaml
```

## Code blocks
Code blocks are automatically highlighted using [chroma](https://github.com/alecthomas/chroma). By default, the style used is the swapoff style. This can be changed by providing a chroma style when calling the `Build` function.

## Examples
```go
package main

import (
	"fmt"
	"os"

	"github.com/efreitasn/egen"
)

func main() {
	err = egen.Build(egen.BuildConfig{
		InPath:  "./content",
		OutPath: "./dist",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
```

There are some examples in the `testdata` directory, such as [this one](testdata/build/ok/1/in). The [efreitasn.dev's repository](https://github.com/efreitasn/efreitasn.dev) is also a good example.

## Latex
Latex can be enabled by setting `latex` to `true` in the config file. Note that Node.js `>= v20.11.0` is required for generating latex images.
