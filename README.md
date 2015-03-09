[![GoDoc](http://godoc.org/github.com/tdewolff/parse?status.svg)](http://godoc.org/github.com/tdewolff/parse) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse)](http://gocover.io/github.com/tdewolff/parse)

# Parse
This package contains several tokenizers and parsers written in [Go][1]. All subpackages are built to be streaming, high performance and to be in accordance with the official (latest) specifications.

The tokenizers are implemented using `ShiftBuffer` and the parsers work on top of the tokenizers. Some subpackages have hashes defined (using [Hasher](https://github.com/tdewolff/hasher)) that speed up common byte-slice comparisons.

## HTML
HTML hashes.

## CSS
CSS tokenizer, parser and hashes.

[See README here](https://github.com/tdewolff/parse/blob/master/css/README.md).

## JS
JS tokenizer and hashes.

[See README here](https://github.com/tdewolff/parse/blob/master/js/README.md).

## License
Released under the [MIT license](LICENSE.md).

[1]: http://golang.org/ "Go Language"
