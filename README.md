# Parse [![GoDoc](http://godoc.org/github.com/tdewolff/parse?status.svg)](http://godoc.org/github.com/tdewolff/parse) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse)](http://gocover.io/github.com/tdewolff/parse)

This package contains several lexers and parsers. All subpackages are built to be streaming, high performance and to be in accordance with the official (latest) specifications.

The lexers are implemented using `Shifter` in https://github.com/tdewolff/buffer and the parsers work on top of the lexers. Some subpackages have hashes defined (using [Hasher](https://github.com/tdewolff/hasher)) that speed up common byte-slice comparisons.

## CSS [![GoDoc](http://godoc.org/github.com/tdewolff/parse/css?status.svg)](http://godoc.org/github.com/tdewolff/parse/css) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/css)](http://gocover.io/github.com/tdewolff/parse/css)

This package is a CSS3 lexer and parser. Both follow the specification at [CSS Syntax Module Level 3](http://www.w3.org/TR/css-syntax-3/). The lexer takes an io.Reader and converts it into tokens until the EOF. The parser returns a parse tree of the full io.Reader input stream, but the low-level `Next` function can be used for stream parsing to returns grammar units until the EOF.

[See README here](https://github.com/tdewolff/parse/tree/master/css).

## HTML [![GoDoc](http://godoc.org/github.com/tdewolff/parse/html?status.svg)](http://godoc.org/github.com/tdewolff/parse/html) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/html)](http://gocover.io/github.com/tdewolff/parse/html)

This package is an HTML5 lexer. It follows the specification at [The HTML syntax](http://www.w3.org/TR/html5/syntax.html). The lexer takes an io.Reader and converts it into tokens until the EOF.

[See README here](https://github.com/tdewolff/parse/tree/master/html).

## JS [![GoDoc](http://godoc.org/github.com/tdewolff/parse/js?status.svg)](http://godoc.org/github.com/tdewolff/parse/js) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/js)](http://gocover.io/github.com/tdewolff/parse/js)

This package is a JS lexer (ECMA-262, edition 5.1). It follows the specification at [ECMAScript Language Specification](http://www.ecma-international.org/ecma-262/5.1/). The lexer takes an io.Reader and converts it into tokens until the EOF.

[See README here](https://github.com/tdewolff/parse/tree/master/js).

## JSON [![GoDoc](http://godoc.org/github.com/tdewolff/parse/json?status.svg)](http://godoc.org/github.com/tdewolff/parse/json) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/json)](http://gocover.io/github.com/tdewolff/parse/json)

This package is a JSON parser (ECMA-404). It follows the specification at [JSON](http://json.org/). The parser takes an io.Reader and converts it into tokens until the EOF.

[See README here](https://github.com/tdewolff/parse/tree/master/json).

## SVG [![GoDoc](http://godoc.org/github.com/tdewolff/parse/svg?status.svg)](http://godoc.org/github.com/tdewolff/parse/svg) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/svg)](http://gocover.io/github.com/tdewolff/parse/svg)

This package contains common hashes for SVG1.1 tags and attributes.

## XML [![GoDoc](http://godoc.org/github.com/tdewolff/parse/xml?status.svg)](http://godoc.org/github.com/tdewolff/parse/xml) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/xml)](http://gocover.io/github.com/tdewolff/parse/xml)

This package is an XML1.0 lexer. It follows the specification at [Extensible Markup Language (XML) 1.0 (Fifth Edition)](http://www.w3.org/TR/xml/). The lexer takes an io.Reader and converts it into tokens until the EOF.

[See README here](https://github.com/tdewolff/parse/tree/master/xml).

## License
Released under the [MIT license](LICENSE.md).

[1]: http://golang.org/ "Go Language"
