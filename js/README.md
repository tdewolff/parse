# JS [![GoDoc](http://godoc.org/github.com/tdewolff/parse/js?status.svg)](http://godoc.org/github.com/tdewolff/parse/js)

This package is a JS lexer (ECMAScript 2020) written in [Go][1]. It follows the specification at [ECMAScript 2020 Language Specification](https://tc39.es/ecma262/). The lexer takes an io.Reader and converts it into tokens until the EOF.

## Installation
Run the following command

	go get -u github.com/tdewolff/parse/v2/js

or add the following import and run project with `go get`

	import "github.com/tdewolff/parse/v2/js"

## Lexer
### Usage
The following initializes a new Lexer with io.Reader `r`:
``` go
l := js.NewLexer(r)
```

To tokenize until EOF an error, use:
``` go
for {
	tt, text := l.Next()
	switch tt {
	case js.ErrorToken:
		// error or EOF set in l.Err()
		return
	// ...
	}
}
```

### Regular Expressions
The ECMAScript specification for `PunctuatorToken` (of which the `/` and `/=` symbols) and `RegExpToken` depend on a parser state to differentiate between the two. The lexer will always parse the first token as `/` or `/=` operator, upon which the parser can rescan that token to scan a regular expression using `RegExp()`.

### Examples
``` go
package main

import (
	"os"

	"github.com/tdewolff/parse/v2/js"
)

// Tokenize JS from stdin.
func main() {
	l := js.NewLexer(os.Stdin)
	for {
		tt, text := l.Next()
		switch tt {
		case js.ErrorToken:
			if l.Err() != io.EOF {
				fmt.Println("Error on line", l.Line(), ":", l.Err())
			}
			return
		case js.IdentifierToken:
			fmt.Println("Identifier", string(text))
		case js.NumericToken:
			fmt.Println("Numeric", string(text))
		// ...
		}
	}
}
```

## License
Released under the [MIT license](https://github.com/tdewolff/parse/blob/master/LICENSE.md).

[1]: http://golang.org/ "Go Language"
