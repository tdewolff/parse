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

// MinBuf is the initial internal token buffer size.
var MinBuf = 16

// Parser is the state for the parser.
type Parser struct {
	z     *Tokenizer
	state []ParserState

	buf  []TokenNode
	pos  int
	copy bool
}

// NewParser returns a new Parser for a io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		z:     NewTokenizer(r),
		state: []ParserState{StylesheetState},
		buf:   make([]TokenNode, 0, MinBuf),
	}
}

// Parse parses the entire CSS file and returns the root StylesheetNode.
func Parse(r io.Reader) (*StylesheetNode, error) {
	p := NewParser(r)
	p.EnableLookback()
	var err error
	stylesheet := &StylesheetNode{}
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
func (p *Parser) Err() error {
	return p.z.Err()
}

// Next returns the next grammar unit from the CSS file.
// This is a lower-level function than Parse and is used to stream parse CSS instead of loading it fully into memory.
// Returned nodes of AtRule and Ruleset do not contain entries for Decls and Rules respectively. These are returned consecutively with calls to Next.
func (p *Parser) Next() (GrammarType, Node) {
	if p.at(ErrorToken) {
		return ErrorGrammar, nil
	}
	p.reset()
	p.skipWhitespace()

	state := p.State()
	if p.at(RightBraceToken) && (state == AtRuleState || state == RulesetState) || p.at(SemicolonToken) && state == AtRuleState {
		// return End types
		token := p.shift()
		p.skipWhile(SemicolonToken)
		p.state = p.state[:len(p.state)-1]
		if state == AtRuleState {
			return EndAtRuleGrammar, token
		} else {
			return EndRulesetGrammar, token
		}
	} else if p.at(CDOToken) || p.at(CDCToken) {
		return TokenGrammar, p.shift()
	}

	// find out whether this is a declaration or ruleset, because we don't know if we have a stylesheet or a style attribute
	// second objective is to visit each TokenNodes before taking pointers to them, so that buffer reallocations won't invalidate those pointers
	i := 0
	hasSemicolon := false
	for p.peek(i).TokenType != LeftBraceToken {
		if p.peek(i).TokenType == SemicolonToken || p.peek(i).TokenType == ErrorToken {
			hasSemicolon = true
			break
		}
		i++
	}
	if atrule := p.parseAtRule(); atrule != nil {
		return AtRuleGrammar, atrule
	} else if hasSemicolon {
		if decl := p.parseDeclaration(); decl != nil {
			return DeclarationGrammar, decl
		}
	} else if ruleset := p.parseRuleset(); ruleset != nil {
		return RulesetGrammar, ruleset
	}
	return TokenGrammar, p.shift()
}

func (p *Parser) EnableLookback() {
	p.copy = true
}

func (p *Parser) State() ParserState {
	return p.state[len(p.state)-1]
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseAtRule() *AtRuleNode {
	if !p.at(AtKeywordToken) {
		return nil
	}
	atrule := &AtRuleNode{}
	atrule.Name = p.shift()
	p.skipWhitespace()
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		atrule.Nodes = append(atrule.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if p.at(LeftBraceToken) {
		p.shift()
	}
	p.state = append(p.state, AtRuleState)
	return atrule
}

func (p *Parser) parseRuleset() *RulesetNode {
	ruleset := &RulesetNode{}
	for !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			p.skipWhitespace()
			continue
		}
		if sel := p.parseSelector(); sel != nil {
			ruleset.Selectors = append(ruleset.Selectors, sel)
		}
		p.skipWhitespace()
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.shift()
	p.state = append(p.state, RulesetState)
	return ruleset
}

func (p *Parser) parseSelector() *SelectorNode {
	sel := &SelectorNode{}
	var ws *TokenNode
	for !p.at(CommaToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && (p.data()[0] == '>' || p.data()[0] == '+' || p.data()[0] == '~') {
			sel.Elems = append(sel.Elems, p.shift())
			p.skipWhitespace()
		} else if p.at(LeftBracketToken) {
			for !p.at(RightBracketToken) && !p.at(ErrorToken) {
				sel.Elems = append(sel.Elems, p.shift())
				p.skipWhitespace()
			}
			if p.at(RightBracketToken) {
				sel.Elems = append(sel.Elems, p.shift())
			}
		} else {
			if ws != nil {
				sel.Elems = append(sel.Elems, ws)
			}
			sel.Elems = append(sel.Elems, p.shift())
		}

		if p.at(WhitespaceToken) {
			ws = p.shift()
		} else {
			ws = nil
		}
	}
	if len(sel.Elems) == 0 {
		return nil
	}
	return sel
}

func (p *Parser) parseDeclaration() *DeclarationNode {
	if !p.at(IdentToken) {
		return nil
	}
	decl := &DeclarationNode{}
	decl.Prop = p.shift()
	decl.Prop.Data = bytes.ToLower(decl.Prop.Data)
	p.skipWhitespace()
	if !p.at(ColonToken) {
		return nil
	}
	p.shift() // colon
	p.skipWhitespace()
	for !p.at(SemicolonToken) && !p.at(RightBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && p.data()[0] == '!' {
			exclamation := p.shift()
			p.skipWhitespace()
			if p.at(IdentToken) && ToHash(bytes.ToLower(p.data())) == Important {
				decl.Important = true
				p.shift()
			} else {
				decl.Vals = append(decl.Vals, exclamation)
			}
		} else if fun := p.parseFunction(); fun != nil {
			decl.Vals = append(decl.Vals, fun)
		} else {
			decl.Vals = append(decl.Vals, p.shift())
		}
		p.skipWhitespace()
	}
	p.skipWhile(SemicolonToken)
	return decl
}

func (p *Parser) parseFunction() *FunctionNode {
	fun := &FunctionNode{}
	if !p.at(FunctionToken) {
		return nil
	}
	fun.Name = p.shift()
	fun.Name.Data = fun.Name.Data[:len(fun.Name.Data)-1]
	p.skipWhitespace()
	for !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.shift()
			p.skipWhitespace()
			continue
		}
		fun.Args = append(fun.Args, p.parseArgument())
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.shift()
	return fun
}

func (p *Parser) parseArgument() *ArgumentNode {
	arg := &ArgumentNode{}
	for !p.at(CommaToken) && !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		arg.Vals = append(arg.Vals, p.shiftComponent())
		p.skipWhitespace()
	}
	return arg
}

func (p *Parser) parseBlock() *BlockNode {
	if !p.at(LeftParenthesisToken) && !p.at(LeftBraceToken) && !p.at(LeftBracketToken) {
		return nil
	}
	block := &BlockNode{}
	block.Open = p.shift()
	p.skipWhitespace()
	for {
		if p.at(RightBraceToken) || p.at(RightParenthesisToken) || p.at(RightBracketToken) || p.at(ErrorToken) {
			break
		}
		block.Nodes = append(block.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if !p.at(ErrorToken) {
		block.Close = p.shift()
	}
	return block
}

func (p *Parser) shiftComponent() Node {
	if block := p.parseBlock(); block != nil {
		return block
	} else if fun := p.parseFunction(); fun != nil {
		return fun
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

func (p *Parser) copyTokens() {
	dst := make([]TokenNode, len(p.buf))
	copy(dst, p.buf)
}

func (p *Parser) read() TokenNode {
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
	return TokenNode{
		tt,
		data,
	}
}

func (p *Parser) peek(i int) *TokenNode {
	if p.pos+i >= len(p.buf) {
		c := cap(p.buf)
		if p.pos+i >= c {
			buf1 := make([]TokenNode, len(p.buf), 2*c)
			copy(buf1, p.buf)
			p.buf = buf1
		}
		for j := len(p.buf); j <= p.pos+i; j++ {
			p.buf = append(p.buf, p.read())
		}
	}
	return &p.buf[p.pos+i]
}

func (p *Parser) shift() *TokenNode {
	shifted := p.peek(0)
	p.pos++
	return shifted
}

func (p *Parser) reset() {
	var buf1 []TokenNode
	if p.copy {
		buf1 = make([]TokenNode, len(p.buf[p.pos:]), MinBuf)
	} else {
		buf1 = p.buf[:len(p.buf[p.pos:])]
	}
	copy(buf1, p.buf[p.pos:])
	p.buf = buf1
	p.pos = 0
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
