# HTML [![GoDoc](http://godoc.org/github.com/tdewolff/parse/html?status.svg)](http://godoc.org/github.com/tdewolff/parse/html) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/html)](http://gocover.io/github.com/tdewolff/parse/html)

This package is an HTML5 tokenizer written in [Go][1]. It follows the specification at [The HTML syntax](http://www.w3.org/TR/html5/syntax.html). The tokenizer takes an io.Reader and converts it into tokens until the EOF.

## Installation
Run the following command

	go get github.com/tdewolff/parse/html

or add the following import and run project with `go get`

	import "github.com/tdewolff/parse/html"

## Tokenizer
### Usage
The following initializes a new tokenizer with io.Reader `r`:
``` go
z := html.NewTokenizer(r)
```

To tokenize until EOF an error, use:
``` go
for {
	tt, data := z.Next()
	switch tt {
	case html.ErrorToken:
		// error or EOF set in z.Err()
		return
	case html.StartTagToken:
		// ...
		for {
			ttAttr, dataAttr := z.Next()
			if ttAttr != html.AttributeToken {
				break
			}
			// ...
		}
	// ...
	}
}
```

All tokens:
``` go
ErrorToken TokenType = iota // extra token when errors occur
CommentToken
DoctypeToken
StartTagToken
StartTagCloseToken
StartTagVoidToken
EndTagToken
AttributeToken
TextToken
```

### Examples
``` go
package main

import (
	"os"

	"github.com/tdewolff/parse/html"
)

// Tokenize HTML from stdin.
func main() {
	z := html.NewTokenizer(os.Stdin)
	for {
		tt, data := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() != io.EOF {
				fmt.Println("Error on line", z.Line(), ":", z.Err())
			}
			return
		case html.StartTagToken:
			fmt.Println("Tag", string(data))
			for {
				ttAttr, dataAttr := z.Next()
				if ttAttr != html.AttributeToken {
					break
				}

				key := dataAttr
				val := z.AttrVal()
				fmt.Println("Attribute", string(key), "=", string(val))
			}
		// ...
		}
	}
}
```

## License
Released under the [MIT license](https://github.com/tdewolff/parse/blob/master/LICENSE.md).

[1]: http://golang.org/ "Go Language"
