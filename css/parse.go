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
	BeginAtRuleGrammar
	AtRuleBlockGrammar
	EndAtRuleGrammar
	BeginRulesetGrammar
	RulesetBlockGrammar
	EndRulesetGrammar
	PropertyGrammar
	BeginValuesGrammar
	EndValuesGrammar
	BeginFunctionGrammar
	EndFunctionGrammar
	BeginBlockGrammar
	EndBlockGrammar
	TokenGrammar
	WhitespaceGrammar
)

// String returns the string representation of a GrammarType.
func (tt GrammarType) String() string {
	switch tt {
	case ErrorGrammar:
		return "Error"
	case BeginAtRuleGrammar:
		return "BeginAtRule"
	case AtRuleBlockGrammar:
		return "AtRuleBlock"
	case EndAtRuleGrammar:
		return "EndAtRule"
	case BeginRulesetGrammar:
		return "BeginRuleset"
	case RulesetBlockGrammar:
		return "RulesetBlock"
	case EndRulesetGrammar:
		return "EndRuleset"
	case PropertyGrammar:
		return "Property"
	case BeginValuesGrammar:
		return "BeginValues"
	case EndValuesGrammar:
		return "EndValues"
	case BeginFunctionGrammar:
		return "BeginFunction"
	case EndFunctionGrammar:
		return "EndFunction"
	case BeginBlockGrammar:
		return "BeginBlock"
	case EndBlockGrammar:
		return "EndBlock"
	case TokenGrammar:
		return "Token"
	case WhitespaceGrammar:
		return "Whitespace"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

type State func() GrammarType

type Token struct {
	tt   TokenType
	data []byte
}

type Parser struct {
	z      *Tokenizer
	state  []State
	atRule Hash
	err    error

	tt        TokenType
	data      []byte
	prevWS    bool
	reconsume bool

	forward    []Token
	forwardPos int
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
	if !p.reconsume {
		p.nextToken()
	} else {
		p.reconsume = false
		p.prevWS = false
	}
	gt := p.state[len(p.state)-1]()
	if gt == WhitespaceGrammar {
		return gt, WhitespaceToken, wsBytes
	} else if p.reconsume {
		return gt, EmptyToken, emptyBytes
	}
	return gt, p.tt, p.data
}

func (p *Parser) peekToken(i int) (TokenType, []byte) {
	if len(p.forward) == 0 {
		p.data = parse.Copy(p.data)
		tt, data := p.z.Next()
		p.forward = append(p.forward, Token{tt, data})
	}
	for len(p.forward) <= p.forwardPos+i {
		p.forward[len(p.forward)-1].data = parse.Copy(p.forward[len(p.forward)-1].data)
		tt, data := p.z.Next()
		p.forward = append(p.forward, Token{tt, data})
	}
	return p.forward[p.forwardPos+i].tt, p.forward[p.forwardPos+i].data
}

func (p *Parser) shiftToken() {
	if len(p.forward) > p.forwardPos {
		p.tt, p.data = p.forward[p.forwardPos].tt, p.forward[p.forwardPos].data
		p.forwardPos++
		if len(p.forward) == p.forwardPos {
			p.forward = p.forward[:0]
			p.forwardPos = 0
		}
	} else {
		p.tt, p.data = p.z.Next()
	}
}

func (p *Parser) nextToken() {
	p.prevWS = false
	p.shiftToken()
	for p.tt == WhitespaceToken || p.tt == CommentToken {
		if p.tt == WhitespaceToken {
			p.prevWS = true
		}
		p.shiftToken()
	}
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseStylesheet() GrammarType {
	if p.tt == CDOToken || p.tt == CDCToken {
		return TokenGrammar
	} else if p.tt == AtKeywordToken {
		p.state = append(p.state, p.parseAtRule)
		p.atRule = ToHash(p.data[1:])
		return BeginAtRuleGrammar
	} else if p.tt == ErrorToken {
		return ErrorGrammar
	} else {
		p.state = append(p.state, p.parseQualifiedRule)
		p.reconsume = true
		return BeginRulesetGrammar
	}
}

func (p *Parser) parseRuleList() GrammarType {
	if p.tt == AtKeywordToken {
		p.state = append(p.state, p.parseAtRule)
		p.atRule = ToHash(p.data[1:])
		return BeginAtRuleGrammar
	} else if p.tt == ErrorToken {
		return ErrorGrammar
	} else {
		p.state = append(p.state, p.parseQualifiedRule)
		p.reconsume = true
		return BeginRulesetGrammar
	}
}

func (p *Parser) parseDeclarationList() GrammarType {
	for p.tt == SemicolonToken {
		p.nextToken()
	}
	if p.tt == ErrorToken {
		return ErrorGrammar
	} else if p.tt == AtKeywordToken {
		p.state = append(p.state, p.parseAtRule)
		p.atRule = ToHash(p.data[1:])
		return BeginAtRuleGrammar
	} else if p.tt == IdentToken {
		p.state = append(p.state, p.parseDeclaration)
		return PropertyGrammar
	}
	// parse error
	for p.tt != SemicolonToken && p.tt != ErrorToken {
		p.nextToken()
	}
	return p.state[len(p.state)-1]()
}

////////////////////////////////////////////////////////////////

func (p *Parser) parseAtRule() GrammarType {
	if p.tt == LeftBraceToken {
		p.state[len(p.state)-1] = p.parseAtRuleBlockSkipWhitespace
		return AtRuleBlockGrammar
	} else if p.tt == SemicolonToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseAtRuleBlockSkipWhitespace() GrammarType {
	p.prevWS = false
	if p.atRule == Font_Face || p.atRule == Page {
		p.state[len(p.state)-1] = p.parseAtRuleDeclarationList
	} else if p.atRule == Document || p.atRule == Keyframes || p.atRule == Media || p.atRule == Supports {
		p.state[len(p.state)-1] = p.parseAtRuleRuleList
	} else {
		p.state[len(p.state)-1] = p.parseAtRuleComponents
	}
	return p.state[len(p.state)-1]()
}

func (p *Parser) parseAtRuleComponents() GrammarType {
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	return p.parseComponent()
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
		p.nextToken()
	}
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndAtRuleGrammar
	}
	return p.parseDeclarationList()
}

func (p *Parser) parseQualifiedRule() GrammarType {
	if p.tt == LeftBraceToken {
		p.state[len(p.state)-1] = p.parseQualifiedRuleDeclarationList
		return RulesetBlockGrammar
	} else if p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		p.reconsume = true
		return EndRulesetGrammar
	} else if p.prevWS {
		p.reconsume = true
		return WhitespaceGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseQualifiedRuleDeclarationList() GrammarType {
	for p.tt == SemicolonToken {
		p.nextToken()
	}
	if p.tt == RightBraceToken || p.tt == ErrorToken {
		p.state = p.state[:len(p.state)-1]
		return EndRulesetGrammar
	}
	return p.parseDeclarationList()
}

func (p *Parser) parseDeclaration() GrammarType {
	if p.tt != ColonToken {
		p.err = errors.New("unexpected token for declaration colon: " + p.tt.String())
		return ErrorGrammar
	}
	p.state[len(p.state)-1] = p.parseDeclarationValuesSkipWhitespace
	return BeginValuesGrammar
}

func (p *Parser) parseDeclarationValuesSkipWhitespace() GrammarType {
	p.prevWS = false
	p.state[len(p.state)-1] = p.parseDeclarationValues
	return p.parseDeclarationValues()
}

func (p *Parser) parseDeclarationValues() GrammarType {
	if p.tt == SemicolonToken {
		p.state = p.state[:len(p.state)-1]
		return EndValuesGrammar
	} else if p.tt == RightBraceToken {
		p.state = p.state[:len(p.state)-1]
		p.reconsume = true
		return EndValuesGrammar
	} else if p.tt == DelimToken && p.data[0] == '!' {
		p.state[len(p.state)-1] = p.parseDeclarationValuesImportant
	} else if p.prevWS {
		p.reconsume = true
		return WhitespaceGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseDeclarationValuesImportant() GrammarType {
	p.state[len(p.state)-1] = p.parseDeclarationValues
	if p.tt == IdentToken && ToHash(p.data) == Important {
		return TokenGrammar
	}
	return p.parseDeclarationValues()
}

func (p *Parser) parseComponent() GrammarType {
	if p.prevWS {
		p.reconsume = true
		return WhitespaceGrammar
	} else if p.tt == LeftParenthesisToken {
		p.state = append(p.state, p.parseParenthesisBlock)
		return BeginBlockGrammar
	} else if p.tt == LeftBraceToken {
		p.state = append(p.state, p.parseBraceBlock)
		return BeginBlockGrammar
	} else if p.tt == LeftBracketToken {
		p.state = append(p.state, p.parseBracketBlock)
		return BeginBlockGrammar
	} else if p.tt == FunctionToken {
		p.state = append(p.state, p.parseFunction)
		return BeginFunctionGrammar
	} else if p.tt == ErrorToken {
		return ErrorGrammar
	} else {
		return TokenGrammar
	}
}

func (p *Parser) parseParenthesisBlock() GrammarType {
	if !p.prevWS && p.tt == RightParenthesisToken {
		p.state = p.state[:len(p.state)-1]
		return EndBlockGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseBraceBlock() GrammarType {
	if !p.prevWS && p.tt == RightBraceToken {
		p.state = p.state[:len(p.state)-1]
		return EndBlockGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseBracketBlock() GrammarType {
	if !p.prevWS && p.tt == RightBracketToken {
		p.state = p.state[:len(p.state)-1]
		return EndBlockGrammar
	}
	return p.parseComponent()
}

func (p *Parser) parseFunction() GrammarType {
	if !p.prevWS && p.tt == RightParenthesisToken {
		p.state = p.state[:len(p.state)-1]
		return EndFunctionGrammar
	}
	return p.parseComponent()
}
