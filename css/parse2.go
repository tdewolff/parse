package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
	"strconv"
)

////////////////////////////////////////////////////////////////

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	AtRuleGrammar
	EndAtRuleGrammar
	RulesetGrammar
	EndRulesetGrammar
	BlockGrammar
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

// ParserState values
const (
	StylesheetState ParserState = iota
	AtRuleState
	RulesetState
)

////////////////////////////////////////////////////////////////

type TokenStream interface {
	Next() (TokenType, []byte)
	Err() error
}

type Token struct {
	TokenType
	Data []byte
}

////////////////////////////////////////////////////////////////

type Parser2 struct {
	z     TokenStream
	buf   []*TokenNode
	state []ParserState
}

func NewParser2(z TokenStream) *Parser2 {
	return &Parser2{
		z,
		[]*TokenNode{},
		[]ParserState{StylesheetState},
	}
}

func (p *Parser2) Parse() (*StylesheetNode, error) {
	stylesheet := NewStylesheet()
	for {
		gt, n := p.Next()
		if gt == ErrorGrammar {
			break
		} else if gt == AtRuleGrammar || gt == RulesetGrammar {
			p.appendNodesRecursively(gt, n)
		}
		stylesheet.Nodes = append(stylesheet.Nodes, n)
	}
	err := p.z.Err()
	if err == io.EOF {
		err = nil
	}
	return stylesheet, err
}

func (p *Parser2) appendNodesRecursively(gt GrammarType, n Node) {
	if gt == AtRuleGrammar {
		atRule := n.(*AtRuleNode)
		for {
			gt2, n2 := p.Next()
			if gt2 == ErrorGrammar || gt2 == EndAtRuleGrammar {
				break
			}
			if gt2 == AtRuleGrammar || gt2 == RulesetGrammar {
				p.appendNodesRecursively(gt2, n2)
			}
			atRule.Rules = append(atRule.Rules, n2)
		}
	} else if gt == RulesetGrammar {
		ruleset := n.(*RulesetNode)
		for {
			gt2, n2 := p.Next()
			if gt2 == ErrorGrammar || gt2 == EndRulesetGrammar {
				break
			}
			if m, ok := n2.(*DeclarationNode); ok {
				ruleset.Decls = append(ruleset.Decls, m)
			}
		}
	}
}

func (p *Parser2) Next() (GrammarType, Node) {
	if p.at(ErrorToken) {
		return ErrorGrammar, nil
	}

	// return End types
	if p.State() == AtRuleState && (p.at(RightBraceToken) || p.at(SemicolonToken)) {
		p.state = p.state[:len(p.state)-1]
		t := p.shift()
		p.skipWhile(SemicolonToken)
		return EndAtRuleGrammar, t
	} else if p.State() == RulesetState && p.at(RightBraceToken) {
		p.state = p.state[:len(p.state)-1]
		return EndRulesetGrammar, p.shift()
	}

	if cn := p.parseAtRule(); cn != nil {
		return AtRuleGrammar, cn
	} else if cn := p.parseRuleset(); cn != nil {
		return RulesetGrammar, cn
	} else if cn := p.parseDeclaration(); cn != nil {
		return DeclarationGrammar, cn
	}
	return TokenGrammar, p.shift()
}

func (p *Parser2) State() ParserState {
	return p.state[len(p.state)-1]
}

func (p *Parser2) parseAtRule() *AtRuleNode {
	if !p.at(AtKeywordToken) {
		return nil
	}
	n := NewAtRule(p.shift())
	for !p.at(SemicolonToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		n.Nodes = append(n.Nodes, p.shiftComponent())
	}
	if p.at(LeftBraceToken) {
		p.shift()
	}
	p.state = append(p.state, AtRuleState)
	return n
}

func (p *Parser2) parseRuleset() *RulesetNode {
	// check if left brace appears, which is the only check whether this is a valid ruleset
	i := 0
	for p.index(i) != LeftBraceToken {
		if p.index(i) == SemicolonToken || p.index(i) == ErrorToken {
			return nil
		}
		i++
	}
	n := NewRuleset()
	for {
		if cn := p.parseSelectorsGroup(); cn != nil {
			n.SelGroups = append(n.SelGroups, cn)
		} else {
			break
		}
	}
	if !p.at(LeftBraceToken) {
		return nil
	}
	p.shift()
	p.state = append(p.state, RulesetState)
	return n
}

func (p *Parser2) parseSelectorsGroup() *SelectorsGroupNode {
	p.skipWhitespace()
	n := NewSelectorsGroup()
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

func (p *Parser2) parseSelector() *SelectorNode {
	n := NewSelector()
	for !p.at(CommaToken) && !p.at(LeftBraceToken) && !p.at(ErrorToken) {
		if p.index(0) == CommentToken {
			p.buf = p.buf[1:]
			continue
		}
		if p.at(DelimToken) && (p.data()[0] == '>' || p.data()[0] == '+' || p.data()[0] == '~') {
			n.Elems = append(n.Elems, p.shift())
			p.skipWhitespace()
		} else if p.index(0) == WhitespaceToken {
			n.Elems = append(n.Elems, p.buf[0])
			p.buf = p.buf[1:]
			p.skipWhitespace()
		} else if p.at(LeftBracketToken) {
			for !p.at(RightBracketToken) {
				n.Elems = append(n.Elems, p.shift())
			}
			n.Elems = append(n.Elems, p.shift())
		} else {
			n.Elems = append(n.Elems, p.shift())
		}
	}
	if len(n.Elems) == 0 {
		return nil
	}
	return n
}

func (p *Parser2) parseDeclaration() *DeclarationNode {
	if !p.at(IdentToken) {
		return nil
	}
	ident := p.shift()
	if !p.at(ColonToken) {
		return nil
	}
	p.shift() // colon
	n := NewDeclaration(ident)
	for !p.at(SemicolonToken) && !p.at(RightBraceToken) && !p.at(ErrorToken) {
		if p.at(DelimToken) && p.data()[0] == '!' {
			exclamation := p.shift()
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
	}
	p.skipWhile(SemicolonToken)
	return n
}

func (p *Parser2) parseFunction() *FunctionNode {
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

func (p *Parser2) parseArgument() *ArgumentNode {
	n := NewArgument()
	for !p.at(CommaToken) && !p.at(RightParenthesisToken) && !p.at(ErrorToken) {
		n.Vals = append(n.Vals, p.shiftComponent())
	}
	return n
}

func (p *Parser2) parseBlock() *BlockNode {
	if !p.at(LeftParenthesisToken) && !p.at(LeftBraceToken) && !p.at(LeftBracketToken) {
		return nil
	}
	n := NewBlock(p.shift())
	for {
		if p.at(RightBraceToken) || p.at(RightParenthesisToken) || p.at(RightBracketToken) || p.at(ErrorToken) {
			break
		}
		n.Nodes = append(n.Nodes, p.shiftComponent())
	}
	if !p.at(ErrorToken) {
		n.Close = p.shift()
	}
	return n
}

func (p *Parser2) shiftComponent() Node {
	if cn := p.parseBlock(); cn != nil {
		return cn
	} else if cn := p.parseFunction(); cn != nil {
		return cn
	} else {
		return p.shift()
	}
}

////////////////////////////////////////////////////////////////

// copyBytes copies bytes to the same position.
// This is required because the referenced slices from the tokenizer might be overwritten on subsequent Next calls.
func copyBytes(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (p *Parser2) index(i int) TokenType {
	for j := len(p.buf); j <= i; j++ {
		tt, text := p.z.Next()
		if tt == ErrorToken {
			return ErrorToken
		}
		p.buf = append(p.buf, NewToken(tt, copyBytes(text)))
	}
	return p.buf[i].TokenType
}

func (p *Parser2) at(tt TokenType) bool {
	i := 0
	for p.index(i) == WhitespaceToken || p.index(i) == CommentToken {
		i++
	}
	if p.index(i) != tt {
		return false
	}
	return true
}

func (p *Parser2) data() []byte {
	i := 0
	for p.index(i) == WhitespaceToken || p.index(i) == CommentToken {
		i++
	}
	return p.buf[i].Data
}

func (p *Parser2) shift() *TokenNode {
	p.skipWhitespace()
	if len(p.buf) > 0 {
		token := p.buf[0]
		p.buf = p.buf[1:]
		return token
	}
	return nil
}

func (p *Parser2) skipWhitespace() {
	for p.index(0) == WhitespaceToken || p.index(0) == CommentToken {
		p.buf = p.buf[1:]
	}
}

func (p *Parser2) skipWhile(tt TokenType) {
	for p.index(0) == tt || p.index(0) == WhitespaceToken || p.index(0) == CommentToken {
		p.buf = p.buf[1:]
	}
}

func (p *Parser2) skipUntil(tt TokenType) {
	for p.index(0) != tt && p.index(0) != ErrorToken {
		p.buf = p.buf[1:]
	}
}
