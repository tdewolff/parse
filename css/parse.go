/*
Package css is a CSS3 tokenizer and parser written in Go. The tokenizer is implemented using the specifications at http://www.w3.org/TR/css-syntax-3/
The parser is not, because documentation is lacking.

Tokenizer using example:

	package main

	import (
		"fmt"
		"io"
		"os"

		"github.com/tdewolff/css"
	)

	// Tokenize CSS3 from stdin.
	func main() {
		z := css.NewTokenizer(os.Stdin)
		for {
			tt, data := z.Next()
			switch tt {
			case css.ErrorToken:
				if z.Err() != io.EOF {
					fmt.Println("Error on line", z.Line(), ":", z.Err())
				}
				return
			case css.IdentToken:
				fmt.Println("Identifier", data)
			case css.NumberToken:
				fmt.Println("Number", data)
			// ...
			}
		}
	}

Parser using example:

	package main

	import (
		"fmt"
		"os"

		"github.com/tdewolff/css"
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
*/
package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
	"io/ioutil"
)

////////////////////////////////////////////////////////////////

type parser struct {
	z   *Tokenizer
	buf []*TokenNode
}

// Parse parses a CSS3 source from a Reader. It uses the package tokenizer and returns a tree of nodes to represent the CSS document.
// The returned StylesheetNode is the root node. All leaf nodes are TokenNode's.
func Parse(r io.Reader) (*StylesheetNode, error) {
	// TODO: make parser streaming
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	p := &parser{
		NewTokenizer(bytes.NewBuffer(b)),
		make([]*TokenNode, 0, 20),
	}

	err = p.z.Err()
	if err == io.EOF {
		err = nil
	}
	return p.parseStylesheet(), err
}

////////////////////////////////////////////////////////////////

func (p *parser) index(i int) TokenType {
	for j := len(p.buf); j <= i; j++ {
		tt, text := p.z.Next()
		if tt == ErrorToken {
			return ErrorToken
		}
		p.buf = append(p.buf, NewToken(tt, text))
	}
	return p.buf[i].TokenType
}

func (p *parser) at(tts ...TokenType) bool {
	i := 0
	for _, tt := range tts {
		for p.index(i) == WhitespaceToken || p.index(i) == CommentToken {
			i++
		}
		if p.index(i) != tt {
			return false
		}
		if p.index(i) == ErrorToken {
			return tt == ErrorToken
		}
		i++
	}
	return true
}

func (p *parser) shift() *TokenNode {
	p.skipWhitespace()
	if len(p.buf) > 0 {
		token := p.buf[0]
		p.buf = p.buf[1:]
		return token
	}
	return nil
}

func (p *parser) skipWhitespace() {
	for p.index(0) == WhitespaceToken || p.index(0) == CommentToken {
		p.buf = p.buf[1:]
	}
}

func (p *parser) skipWhile(tt TokenType) {
	for p.index(0) == tt || p.index(0) == WhitespaceToken || p.index(0) == CommentToken {
		p.buf = p.buf[1:]
	}
}

func (p *parser) skipUntil(tt TokenType) {
	for p.index(0) != tt && p.index(0) != ErrorToken {
		p.buf = p.buf[1:]
	}
}

////////////////////////////////////////////////////////////////

func (p *parser) parseStylesheet() *StylesheetNode {
	n := NewStylesheet()
	for {
		p.skipWhitespace()
		if p.at(ErrorToken) {
			return n
		}
		if p.at(CDOToken) || p.at(CDCToken) {
			n.Nodes = append(n.Nodes, p.shift())
		} else if cn := p.parseAtRule(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if cn := p.parseRuleset(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if cn := p.parseDeclaration(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if !p.at(ErrorToken) {
			n.Nodes = append(n.Nodes, p.shift())
		}
	}
}

func (p *parser) parseRuleset() *RulesetNode {
	// check if left brace appears, which is the only check if this is a valid ruleset
	i := 0
	for p.index(i) != LeftBraceToken {
		if p.index(i) == SemicolonToken || p.index(i) == ErrorToken {
			return nil
		}
		i++
	}
	if i == 0 {
		return nil
	}

	n := NewRuleset()
	for {
		if cn := p.parseSelectorGroup(); cn != nil {
			n.SelGroups = append(n.SelGroups, cn)
		} else {
			break
		}
	}

	// declarations
	if !p.at(LeftBraceToken) {
		return nil
	}
	p.shift()
	for {
		if p.at(IdentToken, ColonToken) {
			if cn := p.parseDeclaration(); cn != nil {
				n.Decls = append(n.Decls, cn)
			}
		} else if p.at(RightBraceToken) || p.at(ErrorToken) {
			break
		} else {
			p.skipUntil(SemicolonToken)
			p.shift()
		}
	}
	p.skipUntil(RightBraceToken)
	p.shift()
	if len(n.Decls) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseSelectorGroup() *SelectorGroupNode {
	n := NewSelectorGroup()
	for !p.at(CommaToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if cn := p.parseSelector(); cn != nil {
			n.Selectors = append(n.Selectors, cn)
		} else {
			break
		}
	}
	p.skipWhile(CommaToken)
	if len(n.Selectors) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseSelector() *SelectorNode {
	n := NewSelector()
	for p.index(0) != WhitespaceToken && p.index(0) != CommaToken && p.index(0) != LeftBraceToken && p.index(0) != ErrorToken {
		if p.index(0) == CommentToken {
			p.buf = p.buf[1:]
			continue
		} else if p.index(0) == LeftParenthesisToken {
			for p.index(0) != RightParenthesisToken && p.index(0) != ErrorToken {
				n.Nodes = append(n.Nodes, p.shift())
			}
			n.Nodes = append(n.Nodes, p.shift())
		} else if p.index(0) == LeftBracketToken {
			p.shift()
			if attr := p.parseAttributeSelector(); attr != nil {
				n.Nodes = append(n.Nodes, attr)
			} else {
				for p.index(0) != RightBracketToken && p.index(0) != ErrorToken {
					n.Nodes = append(n.Nodes, p.shift())
				}
				n.Nodes = append(n.Nodes, p.shift())
			}
		} else {
			n.Nodes = append(n.Nodes, p.shift())
		}
	}
	p.skipWhitespace()
	if len(n.Nodes) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseAttributeSelector() *AttributeSelectorNode {
	if !p.at(IdentToken) && p.index(1) != RightBracketToken && p.index(1) != DelimToken && p.index(1) != IncludeMatchToken && p.index(1) != DashMatchToken && p.index(1) != PrefixMatchToken && p.index(1) != SuffixMatchToken && p.index(1) != SubstringMatchToken {
		return nil
	}
	n := NewAttributeSelector(p.shift())
	if p.index(0) != RightBracketToken {
		n.Op = p.shift()
		for p.index(0) != RightBracketToken {
			n.Vals = append(n.Vals, p.shift())
		}
	}
	p.shift() // right bracket
	return n
}

func (p *parser) parseDeclaration() *DeclarationNode {
	if !p.at(IdentToken, ColonToken) {
		return nil
	}
	n := NewDeclaration(p.shift())
	p.shift() // colon
	for !p.at(SemicolonToken) && !p.at(RightBraceToken) && !p.at(ErrorToken) {
		if cn := p.parseFunction(); cn != nil {
			n.Vals = append(n.Vals, cn)
		} else {
			n.Vals = append(n.Vals, p.shift())
		}
	}
	if len(n.Vals) == 0 {
		p.skipWhile(SemicolonToken)
		return nil
	}
	p.skipWhile(SemicolonToken)
	return n
}

func (p *parser) parseArgument() *ArgumentNode {
	first := p.shift()
	n := NewArgument(first)
	if p.at(DelimToken) && len(p.buf[0].Data) == 1 && p.buf[0].Data[0] == '=' {
		p.shift()
		n.Key = n.Vals[0]
		n.Vals[0] = p.shift()
	}
	for !p.at(CommaToken) && !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		p.skipWhitespace()
		n.Vals = append(n.Vals, p.shift())
	}
	return n
}

func (p *parser) parseFunction() *FunctionNode {
	if !p.at(FunctionToken) {
		return nil
	}
	n := NewFunction(p.shift())
	for !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			continue
		}
		n.Args = append(n.Args, p.parseArgument())
	}
	p.skipUntil(RightParenthesisToken)
	p.shift()
	return n
}

func (p *parser) parseBlock() *BlockNode {
	if !p.at(LeftBraceToken) && !p.at(LeftParenthesisToken) && !p.at(LeftBracketToken) {
		return nil
	}
	n := NewBlock(p.shift())
	for {
		p.skipWhitespace()
		if p.at(RightBraceToken) || p.at(RightParenthesisToken) || p.at(RightBracketToken) || p.at(ErrorToken) {
			break
		} else if cn := p.parseAtRule(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if cn := p.parseRuleset(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if cn := p.parseDeclaration(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if !p.at(ErrorToken) {
			n.Nodes = append(n.Nodes, p.shift())
		}
	}
	if !p.at(ErrorToken) {
		n.Close = p.shift()
	}
	return n
}

func (p *parser) parseAtRule() *AtRuleNode {
	if !p.at(AtKeywordToken) {
		return nil
	}
	n := NewAtRule(p.shift())
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		n.Nodes = append(n.Nodes, p.shift())
	}
	if p.at(LeftBraceToken) {
		if cn := p.parseBlock(); cn != nil {
			n.Block = cn
		} else {
			p.skipUntil(RightBraceToken)
			p.shift()
		}
	}
	p.skipWhile(SemicolonToken)
	return n
}
