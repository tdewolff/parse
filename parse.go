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
			panic(err)
		}

		for _, node := range stylesheet.Nodes {
			switch node.Type() {
			case css.TokenNode:
				fmt.Println("Token", node.String())
			case css.DeclarationNode:
				fmt.Println("Declaration", node.String())
			case css.RulesetNode:
				ruleset := node.(*css.NodeRuleset)
				fmt.Println("Ruleset with", len(ruleset.Decls), "declarations")
				fmt.Println("Ruleset", node.String())
			case css.AtRuleNode:
				fmt.Println("AtRule", node.String())
			}
		}
	}
*/
package css

import (
	"io"
)

////////////////////////////////////////////////////////////////

type parser struct {
	z   *Tokenizer
	buf []*NodeToken
}

// Parse parses a CSS3 source from a Reader. It uses the package tokenizer and returns a tree of nodes to represent the CSS document.
// The returned NodeStylesheet is the root node. All leaf nodes are NodeToken's.
func Parse(r io.Reader) (*NodeStylesheet, error) {
	p := &parser{
		z:   NewTokenizer(r),
		buf: []*NodeToken{},
	}

	err := p.z.Err()
	if err == io.EOF {
		err = nil
	}
	return p.parseStylesheet(), err
}

////////////////////////////////////////////////////////////////

func (p *parser) index(i int) TokenType {
	if i >= len(p.buf) {
		for j := len(p.buf); j <= i; j++ {
			tt, text := p.z.Next()
			p.buf = append(p.buf, NewToken(tt, string(text)))
			if tt == ErrorToken {
				return ErrorToken
			}
		}
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

func (p *parser) shift() *NodeToken {
	p.skipWhitespace()
	token := p.buf[0]
	p.buf = p.buf[1:]
	return token
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

func (p *parser) parseStylesheet() *NodeStylesheet {
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
		} else if cn := p.parseDeclaration(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else if cn := p.parseRuleset(); cn != nil {
			n.Nodes = append(n.Nodes, cn)
		} else {
			//n.Nodes = append(n.Nodes, p.shift())
			p.shift()
		}
	}
}

func (p *parser) parseRuleset() *NodeRuleset {
	// check if left brace appears, which is the only check if this is a valid ruleset
	i := 0
	for p.index(i) != LeftBraceToken {
		if p.index(i) == ErrorToken || p.index(i) == AtKeywordToken || p.index(i) == FunctionToken || p.index(i) == SemicolonToken {
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
		if cn := p.parseDeclaration(); cn != nil {
			n.Decls = append(n.Decls, cn)
		} else if p.at(RightBraceToken) {
			break
		}
	}
	p.skipUntil(RightBraceToken)
	p.shift()
	if len(n.Decls) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseSelectorGroup() *NodeSelectorGroup {
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

func (p *parser) parseSelector() *NodeSelector {
	n := NewSelector()
	for p.index(0) != WhitespaceToken && p.index(0) != CommaToken && p.index(0) != LeftBraceToken && p.index(0) != ErrorToken {
		if p.index(0) == CommentToken {
			p.buf = p.buf[1:]
			continue
		}
		n.Nodes = append(n.Nodes, p.shift())
	}
	p.skipWhitespace()
	if len(n.Nodes) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseDeclaration() *NodeDeclaration {
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

func (p *parser) parseFunction() *NodeFunction {
	if !p.at(FunctionToken) {
		return nil
	}
	n := NewFunction(p.shift())
	for !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			continue
		}
		n.Args = append(n.Args, p.shift())
	}
	p.skipUntil(RightParenthesisToken)
	p.shift()
	return n
}

func (p *parser) parseAtRule() *NodeAtRule {
	if !p.at(AtKeywordToken) {
		return nil
	}
	n := NewAtRule(p.shift())
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		n.Nodes = append(n.Nodes, p.shift())
	}
	if p.at(LeftBraceToken) {
		p.shift()
		for {
			if p.at(RightBraceToken) {
				break
			} else if cn := p.parseDeclaration(); cn != nil {
				n.Block = append(n.Block, cn)
			} else if cn := p.parseRuleset(); cn != nil {
				n.Block = append(n.Block, cn)
			} else {
				p.shift()
			}
		}
		p.skipUntil(RightBraceToken)
		p.shift()
	}
	p.skipWhile(SemicolonToken)
	return n
}
