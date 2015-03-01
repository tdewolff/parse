[![GoDoc](http://godoc.org/github.com/tdewolff/parse?status.svg)](http://godoc.org/github.com/tdewolff/parse)

# CSS
This package is a CSS3 tokenizer and parser written in [Go][1]. The tokenizer follows the specification at [CSS Syntax Module Level 3](http://www.w3.org/TR/css-syntax-3/). It takes an io.Reader and converts it into tokens until the EOF. The parser does not follow the CSS3 specifications because the documentation is subpar or lacking. The parser returns a parse tree of the full io.Reader input stream.

## Installation
Run the following command

	go get github.com/tdewolff/parse

or add the following import and run project with `go get`

	import "github.com/tdewolff/parse/css"

## Tokenizer
### Usage
The following initializes a new tokenizer with io.Reader `r`:
``` go
z := css.NewTokenizer(r)
```

The following takes a `[]byte`:
``` go
z := css.NewTokenizerBytes(b)
```

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
UrlToken			// url(
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
``` go
package main

import (
	"os"

	"github.com/tdewolff/parse/css"
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

## Parser
### Usage
The following parses until EOF with io.Reader `r`:
``` go
stylesheet, err := css.Parse(r)
if err != nil {
	fmt.Println("Error", err)
	return
}
```

To iterate over the stylesheet, use:
``` go
for _, node := range stylesheet.Nodes {
	switch node.(type) {
	case *css.TokenNode:
		// ...
	}
}
```

Grammer:

	StylesheetNode.Nodes := (RulesetNode | DeclarationNode | AtRuleNode | TokenNode)*
	RulesetNode.SelGroups := SelectorGroupNode*
	RulesetNode.Decls := DeclarationNode*

	SelectorGroupNode.Selectors := SelectorNode*
	SelectorNode.Nodes := (TokenNode | AttributeSelectorNode)*
	AttributeSelectorNode.Vals := TokenNode*

	DeclarationNode.Prop := TokenNode
	DeclarationNode.Vals := (FunctionNode | TokenNode)*

	FunctionNode.Func := TokenNode
	FunctionNode.Args := ArgumentNode*
	ArgumentNode.Key := TokenNode | nil
	ArgumentNode.Val := TokenNode

	AtRuleNode.At := TokenNode
	AtRuleNode.Nodes := TokenNode*
	AtRuleNode.Block := BlockNode | nil

	BlockNode.Open := TokenNode
	BlockNode.Nodes := (RulesetNode | DeclarationNode | AtRuleNode | TokenNode)*
	BlockNode.Close := TokenNode

	TokenNode.TokenType := TokenType
	TokenNode.Data := string

### Examples
``` go
package main

import (
	"fmt"
	"os"

	"github.com/tdewolff/parse/css"
)

// Parse CSS3 from stdin.
func main() {
	stylesheet, err := css.Parse(os.Stdin)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, node := range stylesheet.Nodes {
		switch m := node.(type) {
		case *css.TokenNode:
			fmt.Println("Token", string(m.Data))
		case *css.DeclarationNode:
			fmt.Println("Declaration for property", string(m.Prop.Data))
		case *css.RulesetNode:
			fmt.Println("Ruleset with", len(m.Decls), "declarations")
		case *css.AtRuleNode:
			fmt.Println("AtRule", string(m.At.Data))
		}
	}
}
```

## License

Released under the [MIT license](LICENSE.md).

[1]: http://golang.org/ "Go Language"
