egen is an opinionated blog generator. It was created mainly to be used in [https://efreitasn.dev](https://efreitasn.dev). Because of that, some of its features are:

* Uses go templates.
* Every CSS file present in the `<inPath>/assets` directory becomes one single minified CSS file called `style.css` stored in `<outPath>/assets`.
* Every file stored in `<outPath>/assets` is renamed to `<filename_base>-<md5sum(file_content).<filename_ext>`.
* Every post must have a version in each language provided in the config file (`<inPath>/egen.yaml`).

There are some examples in the `testdata` directory, such as [this one](testdata/build/ok/1/in).