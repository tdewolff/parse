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
	"io"
	"strconv"
)

////////////////////////////////////////////////////////////////

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values.
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	AtRuleGrammar
	EndAtRuleGrammar
	RulesetGrammar
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
	case EndAtRuleGrammar:
		return "EndAtRule"
	case RulesetGrammar:
		return "Ruleset"
	case EndRulesetGrammar:
		return "EndRuleset"
	case DeclarationGrammar:
		return "Declaration"
	case TokenGrammar:
		return "Token"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

// ParserState denotes the state of the parser.
type ParserState uint32

// ParserState values.
const (
	StylesheetState ParserState = iota
	AtRuleState
	RulesetState
)

// String returns the string representation of a ParserState.
func (tt ParserState) String() string {
	switch tt {
	case StylesheetState:
		return "Stylesheet"
	case AtRuleState:
		return "AtRule"
	case RulesetState:
		return "Ruleset"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Parser is the state for the parser.
type Parser struct {
	z     *Tokenizer
	state []ParserState

	buf []*TokenNode
	pos int
}

// NewParser returns a new Parser for a Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		NewTokenizer(r),
		[]ParserState{StylesheetState},
		make([]*TokenNode, 0, 16),
		0,
	}
}

// Parse parses the entire CSS file and returns the root StylesheetNode.
func Parse(r io.Reader) (*StylesheetNode, error) {
	p := NewParser(r)
	var err error
	stylesheet := NewStylesheet()
	for {
		gt, n := p.Next()
		if gt == ErrorGrammar {
			err = p.z.Err()
			break
		}
		stylesheet.Nodes = append(stylesheet.Nodes, n)
		if err = p.parseRecursively(gt, n); err != nil {
			break
		}
	}
	if err != io.EOF {
		return stylesheet, err
	}
	return stylesheet, nil
}

func (p *Parser) parseRecursively(rootGt GrammarType, n Node) error {
	if rootGt == AtRuleGrammar {
		atRule := n.(*AtRuleNode)
		for {
			gt, m := p.Next()
			if gt == ErrorGrammar {
				return p.z.Err()
			} else if gt == EndAtRuleGrammar {
				break
			}
			atRule.Rules = append(atRule.Rules, m)
			if err := p.parseRecursively(gt, m); err != nil {
				return err
			}
		}
	} else if rootGt == RulesetGrammar {
		ruleset := n.(*RulesetNode)
		for {
			gt, m := p.Next()
			if gt == ErrorGrammar {
				return p.z.Err()
			} else if gt == EndRulesetGrammar {
				break
			}
			if decl, ok := m.(*DeclarationNode); ok {
				ruleset.Decls = append(ruleset.Decls, decl)
			}
		}
	}
	return nil
}

// Err returns the error encountered during parsing, this is often io.EOF but also other errors can be returned.
func (p Parser) Err() error {
	return p.z.Err()
}

// Next returns the next grammar unit from the CSS file.
// This is a lower-level function than Parse and is used to stream parse CSS instead of loading it fully into memory.
// Returned nodes of AtRule and Ruleset do not contain entries for Decls and Rules respectively. These are returned consecutively with calls to Next.
func (p *Parser) Next() (GrammarType, Node) {
	if p.at(ErrorToken) {
		return ErrorGrammar, nil
	}
	p.skipWhitespace()

	// return End types
	state := p.State()
	if p.at(RightBraceToken) && (state == AtRuleState || state == RulesetState) || p.at(SemicolonToken) && state == AtRuleState {
		n := p.shift()
		p.skipWhile(SemicolonToken)

		p.state = p.state[:len(p.state)-1]
		if state == AtRuleState {

			return EndAtRuleGrammar, n
		}
		return EndRulesetGrammar, n
	}

	if p.at(CDOToken) || p.at(CDCToken) {
		return TokenGrammar, p.shift()
	} else if cn := p.parseAtRule(); cn != nil {
		return AtRuleGrammar, cn
	} else if cn := p.parseRuleset(); cn != nil {
		return RulesetGrammar, cn
	} else if cn := p.parseDeclaration(); cn != nil {
		return DeclarationGrammar, cn
	}
	return TokenGrammar, p.shift()
}

func (p *Parser) State() ParserState {
	return p.state[len(p.state)-1]
}

func (p *Parser) parseAtRule() *AtRuleNode {
	if !p.at(AtKeywordToken) {
		return nil
	}
	n := NewAtRule(p.shift())
	p.skipWhitespace()
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		n.Nodes = append(n.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if p.at(LeftBraceToken) {
		p.shift()
	}
	p.state = append(p.state, AtRuleState)
	return n
}

func (p *Parser) parseRuleset() *RulesetNode {
	// check if left brace appears, which is the only check if this is a valid ruleset
	i := 0
	for p.peek(i).TokenType != LeftBraceToken {
		if p.peek(i).TokenType == SemicolonToken || p.peek(i).TokenType == ErrorToken {
			return nil
		}
		i++
	}
	n := NewRuleset()
	for !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			p.skipWhitespace()
			continue
		}
		if cn := p.parseSelector(); cn != nil {
			n.Selectors = append(n.Selectors, cn)
		}
		p.skipWhitespace()
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.shift()
	p.state = append(p.state, RulesetState)
	return n
}

func (p *Parser) parseSelector() *SelectorNode {
	n := NewSelector()
	var ws *TokenNode
	for !p.at(CommaToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && (p.data()[0] == '>' || p.data()[0] == '+' || p.data()[0] == '~') {
			n.Elems = append(n.Elems, p.shift())
			p.skipWhitespace()
		} else if p.at(LeftBracketToken) {
			for !p.at(RightBracketToken) && !p.at(ErrorToken) {
				n.Elems = append(n.Elems, p.shift())
				p.skipWhitespace()
			}
			if p.at(RightBracketToken) {
				n.Elems = append(n.Elems, p.shift())
			}
		} else {
			if ws != nil {
				n.Elems = append(n.Elems, ws)
			}
			n.Elems = append(n.Elems, p.shift())
		}

		if p.at(WhitespaceToken) {
			ws = p.shift()
		} else {
			ws = nil
		}
	}
	if len(n.Elems) == 0 {
		return nil
	}
	return n
}

func (p *Parser) parseDeclaration() *DeclarationNode {
	if !p.at(IdentToken) {
		return nil
	}
	ident := p.shift()
	p.skipWhitespace()
	if !p.at(ColonToken) {
		return nil
	}
	p.shift() // colon
	p.skipWhitespace()
	n := NewDeclaration(ident)
	for !p.at(SemicolonToken) && !p.at(RightBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && p.data()[0] == '!' {
			exclamation := p.shift()
			p.skipWhitespace()
			if p.at(IdentToken) && bytes.Equal(bytes.ToLower(p.data()), []byte("important")) {
				n.Important = true
				p.shift()
			} else {
				n.Vals = append(n.Vals, exclamation)
			}
		} else if cn := p.parseFunction(); cn != nil {
			n.Vals = append(n.Vals, cn)
		} else {
			n.Vals = append(n.Vals, p.shift())
		}
		p.skipWhitespace()
	}
	p.skipWhile(SemicolonToken)
	return n
}

func (p *Parser) parseFunction() *FunctionNode {
	if !p.at(FunctionToken) {
		return nil
	}
	n := NewFunction(p.shift())
	p.skipWhitespace()
	for !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			p.skipWhitespace()
			continue
		}
		n.Args = append(n.Args, p.parseArgument())
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.shift()
	return n
}

func (p *Parser) parseArgument() *ArgumentNode {
	n := NewArgument()
	for !p.at(CommaToken) && !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		n.Vals = append(n.Vals, p.shiftComponent())
		p.skipWhitespace()
	}
	return n
}

func (p *Parser) parseBlock() *BlockNode {
	if !p.at(LeftParenthesisToken) && !p.at(LeftBraceToken) && !p.at(LeftBracketToken) {
		return nil
	}
	n := NewBlock(p.shift())
	p.skipWhitespace()
	for {
		if p.at(RightBraceToken) || p.at(RightParenthesisToken) || p.at(RightBracketToken) || p.at(ErrorToken) {
			break
		}
		n.Nodes = append(n.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if !p.at(ErrorToken) {
		n.Close = p.shift()
	}
	return n
}

func (p *Parser) shiftComponent() Node {
	if cn := p.parseBlock(); cn != nil {
		return cn
	} else if cn := p.parseFunction(); cn != nil {
		return cn
	} else {
		return p.shift()
	}
}

////////////////////////////////////////////////////////////////

func copyBytes(src []byte) (dst []byte) {
	dst = make([]byte, len(src))
	copy(dst, src)
	return
}

func (p *Parser) read() *TokenNode {
	tt, data := p.z.Next()
	// ignore comments and multiple whitespace
	for tt == CommentToken || tt == WhitespaceToken && len(p.buf) > 0 && p.buf[len(p.buf)-1].TokenType == WhitespaceToken {
		tt, data = p.z.Next()
	}
	// copy necessary for whenever the tokenizer overwrites its buffer
	// checking if buffer has EOF optimizes for small files and files already in memory
	if !p.z.IsEOF() {
		data = copyBytes(data)
	}
	return &TokenNode{
		tt,
		data,
	}
}

func (p *Parser) peek(i int) *TokenNode {
	if p.pos+i >= len(p.buf) {
		c := cap(p.buf)
		l := len(p.buf) - p.pos
		if p.pos+i >= c {
			// expand buffer when len is bigger than half the cap
			if 2*l > c {
				buf1 := make([]*TokenNode, l, 2*c)
				copy(buf1, p.buf[p.pos:])
				p.buf = buf1
			} else {
				copy(p.buf, p.buf[p.pos:])
				p.buf = p.buf[:l]
			}
			p.pos = 0
			if i >= cap(p.buf) {
				return &TokenNode{
					ErrorToken,
					[]byte("looking too far ahead"),
				}
			}
		}
		for j := len(p.buf); j <= p.pos+i; j++ {
			p.buf = append(p.buf, p.read())
		}
	}
	return p.buf[p.pos+i]
}

func (p *Parser) shift() *TokenNode {
	shifted := p.peek(0)
	p.pos++
	return shifted
}

func (p *Parser) at(tt TokenType) bool {
	return p.peek(0).TokenType == tt
}

func (p *Parser) data() []byte {
	return p.peek(0).Data
}

func (p *Parser) skipWhitespace() {
	if p.at(WhitespaceToken) {
		p.shift()
	}
}

func (p *Parser) skipWhile(tt TokenType) {
	for p.at(tt) || p.at(WhitespaceToken) {
		p.shift()
	}
}

func (p *Parser) skipUntil(tt TokenType) {
	for !p.at(tt) && !p.at(ErrorToken) {
		p.shift()
	}
}
