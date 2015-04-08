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
	"io"
	"strconv"

	"github.com/tdewolff/parse"
)

////////////////////////////////////////////////////////////////

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values.
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	AtRuleGrammar
	StartAtRuleGrammar
	EndAtRuleGrammar
	StartRulesetGrammar
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
	case StartAtRuleGrammar:
		return "StartAtRule"
	case EndAtRuleGrammar:
		return "EndAtRule"
	case StartRulesetGrammar:
		return "StartRuleset"
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

	tb   *tokenBuffer
	copy bool
	tBuf []*TokenNode
}

// NewParser returns a new Parser for a io.Reader.
func NewParser(r io.Reader) *Parser {
	z := NewTokenizer(r)
	return &Parser{
		z:     z,
		state: []ParserState{StylesheetState},
		tb:    newTokenBuffer(z),
	}
}

// Parse parses the entire CSS file and returns the root StylesheetNode.
func Parse(r io.Reader) (*StylesheetNode, error) {
	p := NewParser(r)
	p.copy = true
	p.tb.EnableLookback()

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
	if rootGt == StartAtRuleGrammar {
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
	} else if rootGt == StartRulesetGrammar {
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
	for p.at(WhitespaceToken) || p.at(SemicolonToken) {
		p.tb.Shift()
	}

	state := p.state[len(p.state)-1]
	if p.at(RightBraceToken) && (state == AtRuleState || state == RulesetState) {
		// return End types
		token := p.tb.Shift()
		p.state = p.state[:len(p.state)-1]
		if state == AtRuleState {
			return EndAtRuleGrammar, token
		} else {
			return EndRulesetGrammar, token
		}
	} else if p.at(CDOToken) || p.at(CDCToken) {
		return TokenGrammar, p.tb.Shift()
	}

	// find out whether this is a declaration or ruleset, because we don't know if we have a stylesheet or a style attribute
	// second objective is to visit each TokenNodes before taking pointers to them, so that buffer reallocations won't invalidate those pointers
	i := 0
	hasSemicolon := false
	for p.tb.Peek(i).TokenType != LeftBraceToken {
		if p.tb.Peek(i).TokenType == SemicolonToken || p.tb.Peek(i).TokenType == ErrorToken {
			hasSemicolon = true
			break
		}
		i++
	}

	if grammar, atrule := p.parseAtRule(); atrule != nil {
		return grammar, atrule
	} else if hasSemicolon {
		if decl := p.parseDeclaration(); decl != nil {
			return DeclarationGrammar, decl
		}
	} else if ruleset := p.parseRuleset(); ruleset != nil {
		return StartRulesetGrammar, ruleset
	}
	return TokenGrammar, p.tb.Shift()
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseAtRule() (GrammarType, *AtRuleNode) {
	if !p.at(AtKeywordToken) {
		return ErrorGrammar, nil
	}
	atrule := &AtRuleNode{}
	atrule.Name = p.tb.Shift()
	p.skipWhitespace()
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		atrule.Nodes = append(atrule.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if p.at(LeftBraceToken) {
		p.tb.Shift()
		p.state = append(p.state, AtRuleState)
		return StartAtRuleGrammar, atrule
	} else if p.at(SemicolonToken) {
		p.tb.Shift()
	}
	return AtRuleGrammar, atrule
}

func (p *Parser) parseRuleset() *RulesetNode {
	p.tBuf = p.tBuf[:0]

	ruleset := &RulesetNode{}
	for !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.tb.Shift()
			p.skipWhitespace()
			continue
		}
		if sel, ok := p.parseSelector(); ok {
			ruleset.Selectors = append(ruleset.Selectors, sel)
		}
		p.skipWhitespace()
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.tb.Shift()
	p.state = append(p.state, RulesetState)
	return ruleset
}

func (p *Parser) parseSelector() (SelectorNode, bool) {
	nElems := len(p.tBuf)
	var ws *TokenNode
	for !p.at(CommaToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && (p.data()[0] == '>' || p.data()[0] == '+' || p.data()[0] == '~') {
			p.tBuf = append(p.tBuf, p.tb.Shift())
			p.skipWhitespace()
		} else {
			if ws != nil {
				p.tBuf = append(p.tBuf, ws)
			}
			if p.at(LeftBracketToken) {
				for !p.at(RightBracketToken) && !p.at(ErrorToken) {
					p.tBuf = append(p.tBuf, p.tb.Shift())
					p.skipWhitespace()
				}
				if p.at(RightBracketToken) {
					p.tBuf = append(p.tBuf, p.tb.Shift())
				}
			} else {
				p.tBuf = append(p.tBuf, p.tb.Shift())
			}
		}
		if p.at(WhitespaceToken) {
			ws = p.tb.Shift()
		} else {
			ws = nil
		}
	}
	if len(p.tBuf) == nElems {
		return SelectorNode{}, false
	}
	elems := p.tBuf[nElems:]
	if p.copy {
		elems = make([]*TokenNode, len(p.tBuf)-nElems)
		copy(elems, p.tBuf[nElems:])
	}
	return SelectorNode{elems}, true
}

func (p *Parser) parseDeclaration() *DeclarationNode {
	if !p.at(IdentToken) {
		return nil
	}
	decl := &DeclarationNode{}
	decl.Prop = p.tb.Shift()
	parse.ToLower(decl.Prop.Data)
	p.skipWhitespace()
	if !p.at(ColonToken) {
		return nil
	}
	p.tb.Shift() // colon
	p.skipWhitespace()
	for !p.at(SemicolonToken) && !p.at(RightBraceToken) && !p.at(ErrorToken) {
		if fun := p.parseFunction(); fun != nil {
			decl.Vals = append(decl.Vals, fun)
		} else {
			decl.Vals = append(decl.Vals, p.tb.Shift())
		}
		p.skipWhitespace()
	}
	if p.at(SemicolonToken) {
		p.tb.Shift()
	}
	return decl
}

func (p *Parser) parseFunction() *FunctionNode {
	fun := &FunctionNode{}
	if !p.at(FunctionToken) {
		return nil
	}
	fun.Name = p.tb.Shift()
	fun.Name.Data = fun.Name.Data[:len(fun.Name.Data)-1]
	p.skipWhitespace()
	for !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		if p.at(CommaToken) {
			p.tb.Shift()
			p.skipWhitespace()
			continue
		}
		fun.Args = append(fun.Args, p.parseArgument())
	}
	if p.at(ErrorToken) {
		return nil
	}
	p.tb.Shift() // right parenthesis
	return fun
}

func (p *Parser) parseArgument() ArgumentNode {
	arg := ArgumentNode{}
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
	block.Open = p.tb.Shift()
	p.skipWhitespace()
	for {
		if p.at(RightBraceToken) || p.at(RightParenthesisToken) || p.at(RightBracketToken) || p.at(ErrorToken) {
			break
		}
		block.Nodes = append(block.Nodes, p.shiftComponent())
		p.skipWhitespace()
	}
	if !p.at(ErrorToken) {
		block.Close = p.tb.Shift()
	}
	return block
}

func (p *Parser) shiftComponent() Node {
	if block := p.parseBlock(); block != nil {
		return block
	} else if fun := p.parseFunction(); fun != nil {
		return fun
	} else {
		return p.tb.Shift()
	}
}

func (p *Parser) skipWhitespace() {
	if p.at(WhitespaceToken) {
		p.tb.Shift()
	}
}

////////////////////////////////////////////////////////////////

func (p *Parser) at(tt TokenType) bool {
	return p.tb.Peek(0).TokenType == tt
}

func (p *Parser) data() []byte {
	return p.tb.Peek(0).Data
}
