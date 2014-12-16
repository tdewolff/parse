/*
A CSS3 parser written in Go. Implemented using the specifications at http://www.w3.org/TR/css-syntax-3/
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
			n.Nodes = append(n.Nodes, p.shift())
		}
	}
}

func (p *parser) parseRuleset() *NodeRuleset {
	n := NewRuleset()
	for {
		if cn := p.parseSelectorGroup(); cn != nil {
			n.SelGroups = append(n.SelGroups, cn)
		} else {
			break
		}
	}
	if len(n.SelGroups) == 0 {
		return nil
	}
	if cn := p.parseDeclarationList(); cn != nil {
		n.DeclList = cn
	} else {
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
			p.shift()
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

func (p *parser) parseDeclarationList() *NodeDeclarationList {
	if !p.at(LeftBraceToken) {
		return nil
	}
	p.shift()
	n := NewDeclarationList()
	for {
		if cn := p.parseDeclaration(); cn != nil {
			n.Decls = append(n.Decls, cn)
		} else {
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
				break
			}
		}
		p.skipUntil(RightBraceToken)
		p.shift()
	}
	p.skipWhile(SemicolonToken)
	return n
}
