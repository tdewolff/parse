/*
A CSS3 parser written in Go. Implemented using the specifications at http://www.w3.org/TR/css-syntax-3/
*/
package css

import (
	"io"
)

////////////////////////////////////////////////////////////////

type StateFunc func(*Parser)

type Parser struct {
	z *Tokenizer
	buf []*NodeToken
}

func Parse(r io.Reader) (*NodeStylesheet, error) {
	p := &Parser{
		z: NewTokenizer(r),
		buf: []*NodeToken{},
	}
	return p.parseStylesheet(), p.z.Err()
}

////////////////////////////////////////////////////////////////

func (p *Parser) index(i int) TokenType {
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

func (p *Parser) shift() *NodeToken {
	if len(p.buf) == 0 {
		p.index(0) // reads in atleast one token
	}
	token := p.buf[0]
	p.buf = p.buf[1:]
	return token
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseStylesheet() *NodeStylesheet {
	n := newStylesheet()
	for {
		switch p.index(0) {
		case ErrorToken:
			return n
		case CDOToken, CDCToken:
			n.Nodes = append(n.Nodes, p.shift())
		default:
			p.shift()
		}
	}
}

// func (p *Parser) atDeclarationList() {
// 	return p.atDeclaration()
// }

// func (p *Parser) parseDeclarationList() {
// 	for {
// 		p.parseDeclaration()
// 		if
// 	}
// }

// func (p *Parser) atDeclaration() {
// 	if p.index(0) != IdentToken || p.index(1) != ColonToken {
// 		return
// 	}
// }

// func (n DeclarationList) Next() *Node {
// 	switch n.parser.index(0) {
// 	case ErrorToken:
// 		return nil
// 	case WhitespaceToken, CommentToken:
// 		n.parser.shift()
// 		return n.Next()
// 	case CDOToken, CDCToken:
// 		return n.parser.shift()
// 	case IdentToken:
// 		if c := newDeclarationList; c != nil {
// 			return c
// 		}
// 	}
// 	return n.parser.shift()
// }
