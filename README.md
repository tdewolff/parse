[![GoDoc](http://godoc.org/github.com/tdewolff/parse?status.svg)](http://godoc.org/github.com/tdewolff/parse) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse)](http://gocover.io/github.com/tdewolff/parse)

# Parse
This package contains tokenizers and parsers for several content types written in [Go][1]. All subpackages are built to be streaming, high performance and to conform with the specifications.

The tokenizers are implemented using `ShiftBuffer` and the parsers work on top of the tokenizers. Some content types have hashes defined (using [Hasher](https://github.com/tdewolff/hasher)) that speed up byte-slice comparisons.

## CSS
[See README here](https://github.com/tdewolff/parse/blob/master/css/README.md).

## JS
[See README here](https://github.com/tdewolff/parse/blob/master/js/README.md).

## Installation
Run the following commands

	go get github.com/tdewolff/parse/css
	go get github.com/tdewolff/parse/js

or add the following imports and run project with `go get`

	import "github.com/tdewolff/parse/css"
	import "github.com/tdewolff/parse/js"

## License
Released under the [MIT license](LICENSE.md).

[1]: http://golang.org/ "Go Language"
