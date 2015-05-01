/*
Package css is a CSS3 tokenizer and parser written in Go. Both are implemented using the specifications at http://www.w3.org/TR/css-syntax-3/
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
	"errors"
	"io"
	"strconv"

	"github.com/tdewolff/parse"
)

var wsBytes = []byte(" ")
var emptyBytes = []byte("")

////////////////////////////////////////////////////////////////

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values.
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	AtRuleGrammar
	BeginAtRuleGrammar
	EndAtRuleGrammar
	BeginRulesetGrammar
	EndRulesetGrammar
	DeclarationGrammar
	TokenGrammar
)

// String returns the string representation of a GrammarType.
func (tt GrammarType) String() string {
	switch tt {
	case ErrorGrammar:
		return "Error"
	case AtRuleGrammar:
		return "AtRule"
	case BeginAtRuleGrammar:
		return "BeginAtRule"
	case EndAtRuleGrammar:
		return "EndAtRule"
	case BeginRulesetGrammar:
		return "BeginRuleset"
	case EndRulesetGrammar:
		return "EndRuleset"
	case DeclarationGrammar:
		return "Declaration"
	case TokenGrammar:
		return "Token"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

type State func() GrammarType

type Token struct {
	TokenType
	Data []byte
}

func String(ts []Token) string {
	var data []byte
	for _, t := range ts {
		data = append(data, t.Data...)
	}
	return string(data)
}

type Parser struct {
	z     *Tokenizer
	state []State
	err   error

	buf []Token

	tt     TokenType
	data   []byte
	prevWS bool
	level  int
}

func NewParser(r io.Reader, isStylesheet bool) *Parser {
	z := NewTokenizer(r)
	p := &Parser{
		z: z,
	}
	if isStylesheet {
		p.state = []State{p.parseStylesheet}
	} else {
		p.state = []State{p.parseDeclarationList}
	}
	return p
}

func (p *Parser) Err() error {
	if p.err != nil {
		return p.err
	}
	return p.z.Err()
}

func (p *Parser) Next() (GrammarType, TokenType, []byte) {
	p.tt, p.data = p.popToken()
	gt := p.state[len(p.state)-1]()
	return gt, p.tt, p.data
}

func (p *Parser) Values() []Token {
	return p.buf
}

func (p *Parser) popToken() (TokenType, []byte) {
	p.prevWS = false
	tt, data := p.z.Next()
	for tt == WhitespaceToken || tt == CommentToken {
		if tt == WhitespaceToken {
			p.prevWS = true
		}
		tt, data = p.z.Next()
	}
	return tt, data
}

func (p *Parser) initBuf() {
	if !p.z.IsEOF() {
		p.data = parse.Copy(p.data)
	}
	p.buf = p.buf[:0]
}

func (p *Parser) pushBuf(tt TokenType, data []byte) {
	if p.z.IsEOF() {
		p.buf = append(p.buf, Token{tt, data})
	} else {
		p.buf = append(p.buf, Token{tt, parse.Copy(data)})
	}
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseStylesheet() GrammarType {
	if p.tt == CDOToken || p.tt == CDCToken {
		return TokenGrammar
	} else if p.tt == AtKeywordToken {
		return p.parseAtRule()
	} else if p.tt == ErrorToken {
		return ErrorGrammar
	} else {
		return p.parseQualifiedRule()
	}
}

func (p *Parser) parseRuleList() GrammarType {
	if p.tt == AtKeywordToken {
		return p.parseAtRule()
	} else if p.tt == ErrorToken {
		return ErrorGrammar
	} else {
		return p.parseQualifiedRule()
	}
}

func (p *Parser) parseDeclarationList() GrammarType {
	for p.tt == SemicolonToken {
		p.tt, p.data = p.popToken()
	}
	if p.tt == ErrorToken {
		return ErrorGrammar
	} else if p.tt == AtKeywordToken {
		return p.parseAtRule()
	} else if p.tt == IdentToken {
		return p.parseDeclaration()
	} else if p.tt == DelimToken && p.data[0] == '*' { // CSS hack
		p.tt, p.data = p.popToken()
		if p.tt == IdentToken {
			p.data = append([]byte("*"), p.data...)
			return p.parseDeclaration()
		}
		return p.state[len(p.state)-1]()
	}
	// parse error
	for p.tt != SemicolonToken && p.tt != ErrorToken {
		p.tt, p.data = p.popToken()
	}
	return p.state[len(p.state)-1]()
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseAtRule() GrammarType {
	p.initBuf()
	parse.ToLower(p.data)
	if len(p.data) > 0 && p.data[1] == '-' {
		if i := bytes.IndexByte(p.data[2:], '-'); i != -1 {
			p.data = p.data[i+3:] // skip vendor specific prefix
		}
	}
	atRule := ToHash(p.data[1:])
	first := true
	skipWS := false
	for {
		tt, data := p.popToken()
		if tt == LeftBraceToken && p.level == 0 {
			if atRule == Font_Face || atRule == Page {
				p.state = append(p.state, p.parseAtRuleDeclarationList)
			} else if atRule == Document || atRule == Keyframes || atRule == Media || atRule == Supports {
				p.state = append(p.state, p.parseAtRuleRuleList)
			} else {
				p.state = append(p.state, p.parseAtRuleUnknown)
			}
			return BeginAtRuleGrammar
		} else if tt == SemicolonToken && p.level == 0 || tt == ErrorToken {
			return AtRuleGrammar
		} else if tt == LeftParenthesisToken || tt == LeftBraceToken || tt == LeftBracketToken || tt == FunctionToken {
			p.level++
		} else if tt == RightParenthesisToken || tt == RightBraceToken || tt == RightBracketToken {
			p.level--
		}
		if first {
			if tt == LeftParenthesisToken || tt == LeftBracketToken {
				p.prevWS = false
			}
			first = false
		}
		if len(data) == 1 && (data[0] == ',' || data[0] == ':') {
			skipWS = true
		} else if p.prevWS && !skipWS && tt != RightParenthesisToken {
			p.pushBuf(WhitespaceToken, wsBytes)
		} else {
			skipWS = false
		}
		if tt == LeftParenthesisToken {
			skipWS = true
		}
		p.pushBuf(tt, data)
	}
}

func (p *Parser) parseAtRuleRuleList() GrammarType {
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	return p.parseRuleList()
}

func (p *Parser) parseAtRuleDeclarationList() GrammarType {
	for p.tt == SemicolonToken {
		p.tt, p.data = p.popToken()
	}
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	return p.parseDeclarationList()
}

func (p *Parser) parseAtRuleUnknown() GrammarType {
	if p.level == 0 && p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	if p.tt == LeftParenthesisToken || p.tt == LeftBraceToken || p.tt == LeftBracketToken || p.tt == FunctionToken {
		p.level++
	} else if p.tt == RightParenthesisToken || p.tt == RightBraceToken || p.tt == RightBracketToken {
		p.level--
	}
	return TokenGrammar
}

func (p *Parser) parseQualifiedRule() GrammarType {
	p.initBuf()
	first := true
	inAttrSel := false
	skipWS := true
	var tt TokenType
	var data []byte
	for {
		if first {
			tt, data = p.tt, p.data
			p.tt = WhitespaceToken
			p.data = emptyBytes
			first = false
		} else {
			tt, data = p.popToken()
		}
		if tt == LeftBraceToken && p.level == 0 {
			p.state = append(p.state, p.parseQualifiedRuleDeclarationList)
			return BeginRulesetGrammar
		} else if tt == ErrorToken {
			p.err = errors.New("unexpected error in qualified rule '" + String(p.buf) + "' before encountering '{'")
			return ErrorGrammar
		} else if tt == LeftParenthesisToken || tt == LeftBraceToken || tt == LeftBracketToken || tt == FunctionToken {
			p.level++
		} else if tt == RightParenthesisToken || tt == RightBraceToken || tt == RightBracketToken {
			p.level--
		}
		if len(data) == 1 && (data[0] == ',' || data[0] == '>' || data[0] == '+' || data[0] == '~') {
			skipWS = true
		} else if p.prevWS && !skipWS && !inAttrSel {
			p.pushBuf(WhitespaceToken, wsBytes)
		} else {
			skipWS = false
		}
		if tt == LeftBracketToken {
			inAttrSel = true
		} else if tt == RightBracketToken {
			inAttrSel = false
		}
		p.pushBuf(tt, data)
	}
}

func (p *Parser) parseQualifiedRuleDeclarationList() GrammarType {
	for p.tt == SemicolonToken {
		p.tt, p.data = p.popToken()
	}
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndRulesetGrammar
	}
	return p.parseDeclarationList()
}

func (p *Parser) parseDeclaration() GrammarType {
	p.initBuf()
	parse.ToLower(p.data)
	if tt, _ := p.popToken(); tt != ColonToken {
		p.err = errors.New("unexpected token for declaration colon: " + p.tt.String())
		return ErrorGrammar
	}
	skipWS := true
	for {
		tt, data := p.popToken()
		if (tt == SemicolonToken || tt == RightBraceToken) && p.level == 0 {
			return DeclarationGrammar
		} else if tt == ErrorToken {
			return DeclarationGrammar
		} else if tt == LeftParenthesisToken || tt == LeftBraceToken || tt == LeftBracketToken || tt == FunctionToken {
			p.level++
		} else if tt == RightParenthesisToken || tt == RightBraceToken || tt == RightBracketToken {
			p.level--
		}
		if len(data) == 1 && (data[0] == ',' || data[0] == '/' || data[0] == ':' || data[0] == '!' || data[0] == '=') {
			skipWS = true
		} else if p.prevWS && !skipWS {
			p.pushBuf(WhitespaceToken, wsBytes)
		} else {
			skipWS = false
		}
		p.pushBuf(tt, data)
	}
}
