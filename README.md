[![GoDoc](http://godoc.org/github.com/tdewolff/css?status.svg)](http://godoc.org/github.com/tdewolff/css)

~90% test coverage

# CSS

This package is a CSS3 tokenizer written in [Go][1] and follows the specification at [CSS Syntax Module Level 3](http://www.w3.org/TR/css-syntax-3/). It takes an io.Reader and converts it into tokens until the EOF.

## Installation

Run the following command

	go get github.com/tdewolff/css

or add the following import and run project with `go get`

	import "github.com/tdewolff/css"

## Usage
The following initializes a new tokenizer with io.Reader `r`:

	z := css.NewTokenizer(r)

To tokenize until EOF an error, use:
``` go
for {
	tt, text := z.Next()
	switch tt {
	case css.ErrorToken:
		// error or EOF set in z.Err()
		return
	// ...
	}
}
```

All tokens (see [CSS Syntax Module Level 3](http://www.w3.org/TR/css3-syntax/)):
``` go
ErrorToken			// non-official token, returned when errors occur
IdentToken
FunctionToken		// rgb( rgba( ...
AtKeywordToken		// @abc
HashToken			// #abc
StringToken
BadStringToken
UrlToken
BadUrlToken
DelimToken			// any unmatched character
NumberToken			// 5
PercentageToken		// 5%
DimensionToken		// 5em
UnicodeRangeToken
IncludeMatchToken	// ~=
DashMatchToken		// |=
PrefixMatchToken	// ^=
SuffixMatchToken	// $=
SubstringMatchToken // *=
ColumnToken			// ||
WhitespaceToken
CDOToken 			// <!--
CDCToken 			// -->
ColonToken
SemicolonToken
CommaToken
BracketToken 		// ( ) [ ] { }, all bracket tokens use this, Data() can distinguish between the brackets
CommentToken		// non-official token
```

### Examples
Basic example:
``` go
package main

import (
	"os"

	"github.com/tdewolff/css"
)

// Tokenize CSS3 from stdin.
func main() {
	z := css.NewTokenizer(os.Stdin)
	for {
		tt, text := z.Next()
		switch tt {
		case css.ErrorToken:
			if z.Err() != io.EOF {
				fmt.Println("Error on line", z.Line(), ":", z.Err())
			}
			return
		case css.IdentToken:
			fmt.Println("Identifier", string(text))
		case css.NumberToken:
			fmt.Println("Number", string(text))
		// ...
		}
	}
}
```

[1]: http://golang.org/ "Go Language"
