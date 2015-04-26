# XML [![GoDoc](http://godoc.org/github.com/tdewolff/parse/xml?status.svg)](http://godoc.org/github.com/tdewolff/parse/xml) [![GoCover](http://gocover.io/_badge/github.com/tdewolff/parse/xml)](http://gocover.io/github.com/tdewolff/parse/xml)

This package is an XML tokenizer written in [Go][1]. It follows the specification at [Extensible Markup Language (XML) 1.0 (Fifth Edition)](http://www.w3.org/TR/REC-xml/). The tokenizer takes an io.Reader and converts it into tokens until the EOF.

## Installation
Run the following command

	go get github.com/tdewolff/parse/xml

or add the following import and run project with `go get`

	import "github.com/tdewolff/parse/xml"

## Tokenizer
### Usage
The following initializes a new tokenizer with io.Reader `r`:
``` go
z := xml.NewTokenizer(r)
```

To tokenize until EOF an error, use:
``` go
for {
	tt, data := z.Next()
	switch tt {
	case xml.ErrorToken:
		// error or EOF set in z.Err()
		return
	case xml.StartTagToken:
		// ...
		for {
			ttAttr, dataAttr := z.Next()
			if ttAttr != xml.AttributeToken {
				// handle StartTagCloseToken/StartTagCloseVoidToken/StartTagClosePIToken
				break
			}
			// ...
		}
	case xml.EndTagToken:
		// ...
	}
}
```

All tokens:
``` go
ErrorToken TokenType = iota // extra token when errors occur
CommentToken
CDATAToken
StartTagToken
StartTagCloseToken
StartTagCloseVoidToken
StartTagClosePIToken
EndTagToken
AttributeToken
TextToken
```

### Examples
``` go
package main

import (
	"os"

	"github.com/tdewolff/parse/xml"
)

// Tokenize XML from stdin.
func main() {
	z := xml.NewTokenizer(os.Stdin)
	for {
		tt, data := z.Next()
		switch tt {
		case xml.ErrorToken:
			if z.Err() != io.EOF {
				fmt.Println("Error on line", z.Line(), ":", z.Err())
			}
			return
		case xml.StartTagToken:
			fmt.Println("Tag", string(data))
			for {
				ttAttr, dataAttr := z.Next()
				if ttAttr != xml.AttributeToken {
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
