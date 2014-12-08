/*
A CSS3 parser written in Go. Implemented using the specifications at http://www.w3.org/TR/css-syntax-3/
*/
package css

import (
	"io"
)

////////////////////////////////////////////////////////////////

type parser struct {
	z *Tokenizer
	buf []*NodeToken
}

func Parse(r io.Reader) (*NodeStylesheet, error) {
	p := &parser{
		z: NewTokenizer(r),
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
			if tt == WhitespaceToken || tt == CommentToken {
				j--
				continue
			}
			p.buf = append(p.buf, newToken(tt, string(text)))
			if tt == ErrorToken {
				return ErrorToken
			}
		}
	}
	return p.buf[i].tt
}

func (p *parser) at(i int) string {
	if i >= len(p.buf) {
		p.index(i)
	}
	return p.buf[i].data
}

func (p *parser) shift() *NodeToken {
	if len(p.buf) == 0 {
		p.index(0) // reads in atleast one token
	}
	token := p.buf[0]
	p.buf = p.buf[1:]
	return token
}

////////////////////////////////////////////////////////////////

func (p *parser) parseStylesheet() *NodeStylesheet {
	n := newStylesheet()
	for {
		switch p.index(0) {
		case ErrorToken:
			return n
		case CDOToken, CDCToken:
			n.Nodes = append(n.Nodes, p.shift())
		default:
			if cn := p.parseAtRule(); cn != nil {
				n.Nodes = append(n.Nodes, cn)
			} else if cn := p.parseDeclaration(); cn != nil {
				n.Nodes = append(n.Nodes, cn)
			} else if cn := p.parseRuleset(); cn != nil {
				n.Nodes = append(n.Nodes, cn)
			} else {
				n.Nodes = append(n.Nodes, p.shift())
			}
		}
	}
}

func (p *parser) parseRuleset() *NodeRuleset {
	n := newRuleset()
	for {
		if cn := p.parseSelector(); cn != nil {
			n.Selectors = append(n.Selectors, cn)
		} else {
			break
		}
	}
	if len(n.Selectors) == 0 {
		return nil
	}
	if cn := p.parseDeclarationBlock(); cn != nil {
		n.Decl = cn
	}
	return n
}

func (p *parser) parseSelector() *NodeSelector {
	n := newSelector()
	for p.index(0) != CommaToken && p.index(0) != LeftBraceToken && p.index(0) != ErrorToken {
		n.Selector = append(n.Selector, p.shift())
	}
	for p.index(0) == CommaToken {
		p.shift()
	}
	if len(n.Selector) == 0 {
		return nil
	}
	return n
}

func (p *parser) parseDeclarationBlock() *NodeDeclarationBlock {
	if p.index(0) != LeftBraceToken {
		return nil
	}
	p.shift()
	n := newDeclarationBlock()
	for {
		if cn := p.parseDeclaration(); cn != nil {
			n.Decls = append(n.Decls, cn)
		} else {
			break
		}
	}
	for p.index(0) != RightBraceToken {
		p.shift()
	}
	p.shift()
	return n
}

func (p *parser) parseDeclaration() *NodeDeclaration {
	if p.index(0) != IdentToken || p.index(1) != ColonToken {
		return nil
	}
	n := newDeclaration(p.shift())
	p.shift() // colon
	for p.index(0) != SemicolonToken && p.index(0) != RightBraceToken && p.index(0) != ErrorToken {
		if cn := p.parseFunction(); cn != nil {
			n.Val = append(n.Val, cn)
		} else {
			n.Val = append(n.Val, p.shift())
		}
	}
	for p.index(0) == SemicolonToken {
		p.shift()
	}
	return n
}

func (p *parser) parseFunction() *NodeFunction {
	if p.index(0) != FunctionToken {
		return nil
	}
	n := newFunction(p.shift())
	for p.index(0) != RightParenthesisToken && p.index(0) != ErrorToken {
		if p.index(0) == CommaToken {
			continue
		}
		n.Arg = append(n.Arg, p.shift())
	}
	if p.index(0) == RightParenthesisToken {
		p.shift()
	}
	return n
}

func (p *parser) parseAtRule() *NodeAtRule {
	if p.index(0) != AtKeywordToken {
		return nil
	}
	n := newAtRule(p.shift())
	for p.index(0) != SemicolonToken && p.index(0) != LeftBraceToken && p.index(0) != ErrorToken {
		n.Nodes = append(n.Nodes, p.shift())
	}
	if p.index(0) == LeftBraceToken {
		p.shift()
		for {
			if p.index(0) == RightBraceToken {
				break
			} else if cn := p.parseDeclaration(); cn != nil {
				n.Block = append(n.Block, cn)
			} else if cn := p.parseRuleset(); cn != nil {
				n.Block = append(n.Block, cn)
			} else {
				break
			}
		}
		for p.index(0) != RightBraceToken {
			p.shift()
		}
		p.shift()
	}
	for p.index(0) == SemicolonToken {
		p.shift()
	}
	return n
}
