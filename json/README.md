# JSON [![GoDoc](http://godoc.org/github.com/tdewolff/parse/json?status.svg)](http://godoc.org/github.com/tdewolff/parse/json) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/json)](http://gocover.io/github.com/tdewolff/parse/json)

This package is a JSON tokenizer (ECMA-404) written in [Go][1]. It follows the specification at [JSON](http://json.org/). The tokenizer takes an io.Reader and converts it into tokens until the EOF.

## Installation
Run the following command

	go get github.com/tdewolff/parse/json

or add the following import and run project with `go get`

	import "github.com/tdewolff/parse/json"

## Tokenizer
### Usage
The following initializes a new tokenizer with io.Reader `r`:
``` go
z := json.NewTokenizer(r)
```

To tokenize until EOF an error, use:
``` go
for {
	tt, text := z.Next()
	switch tt {
	case json.ErrorToken:
		// error or EOF set in z.Err()
		return
	// ...
	}
}
```

All tokens:
``` go
ErrorToken          TokenType = iota // extra token when errors occur
UnknownToken                         // extra token when no token can be matched
WhitespaceToken                      // space \t \r \n
LiteralToken                         // null true false
PunctuatorToken                      // { } [ ] . :
NumberToken
StringToken
```

### Examples
``` go
package main

import (
	"os"

	"github.com/tdewolff/parse/json"
)

// Tokenize JSON from stdin.
func main() {
	z := json.NewTokenizer(os.Stdin)
	for {
		tt, text := z.Next()
		switch tt {
		case json.ErrorToken:
			if z.Err() != io.EOF {
				fmt.Println("Error on line", z.Line(), ":", z.Err())
			}
			return
		case json.LiteralToken:
			fmt.Println("Literal", string(text))
		case json.NumberToken:
			fmt.Println("Number", string(text))
		// ...
		}
	}
}
```

## License
Released under the [MIT license](https://github.com/tdewolff/parse/blob/master/LICENSE.md).

[1]: http://golang.org/ "Go Language"
