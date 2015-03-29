[![GoDoc](http://godoc.org/github.com/tdewolff/parse?status.svg)](http://godoc.org/github.com/tdewolff/parse) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse)](http://gocover.io/github.com/tdewolff/parse)

# Parse
This package contains several tokenizers and parsers written in [Go][1]. All subpackages are built to be streaming, high performance and to be in accordance with the official (latest) specifications.

The tokenizers are implemented using `ShiftBuffer` and the parsers work on top of the tokenizers. Some subpackages have hashes defined (using [Hasher](https://github.com/tdewolff/hasher)) that speed up common byte-slice comparisons.

## HTML
An HTML5 tokenizer. [See README here](https://github.com/tdewolff/parse/tree/master/html).

## CSS
A CSS3 tokenizer and parser. [See README here](https://github.com/tdewolff/parse/tree/master/css).

## JS
An ECMAScript5.1 tokenizer. [See README here](https://github.com/tdewolff/parse/tree/master/js).

## License
Released under the [MIT license](LICENSE.md).

[1]: http://golang.org/ "Go Language"
