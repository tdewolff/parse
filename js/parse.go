package js

import (
	"fmt"
	"io"
	"strconv"
)

type Node struct {
	gt    GrammarType
	nodes []Node

	// filled if gt == TokenGrammar
	tt   TokenType
	data []byte
}

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values.
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	ModuleGrammar
	TokenGrammar
	CommentGrammar
	BindingGrammar
	ExprGrammar
	DeclGrammar
	StmtGrammar
)

// String returns the string representation of a GrammarType.
func (tt GrammarType) String() string {
	switch tt {
	case ErrorGrammar:
		return "Error"
	case ModuleGrammar:
		return "Module"
	case TokenGrammar:
		return "Token"
	case CommentGrammar:
		return "Comment"
	case BindingGrammar:
		return "Binding"
	case ExprGrammar:
		return "Expr"
	case DeclGrammar:
		return "Decl"
	case StmtGrammar:
		return "Stmt"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Parser is the state for the parser.
type Parser struct {
	l   *Lexer
	err error

	tt   TokenType
	data []byte
}

// Parse returns a JS AST tree of.
func Parse(r io.Reader) (Node, error) {
	l := NewLexer(r)
	defer l.Restore()

	p := &Parser{l: l}
	p.next()

	nodes := []Node{}
	for {
		if p.err != nil {
			break
		}

		switch p.tt {
		case ImportToken, ExportToken:
			// TODO
			nodes = append(nodes, p.parseToken())
		default:
			nodes = append(nodes, p.parseStmt())
		}
	}
	return Node{ModuleGrammar, nodes, 0, nil}, p.err
}

////////////////////////////////////////////////////////////////

func (p *Parser) next() {
	if p.err != nil {
		p.tt = ErrorToken
		p.data = nil
		return
	}

	p.tt, p.data = p.l.Next()
	if p.tt == WhitespaceToken {
		p.tt, p.data = p.l.Next()
	} else if p.tt == ErrorToken {
		p.err = p.l.Err()
	}
}

func (p *Parser) consume(tt TokenType) bool {
	if p.tt != tt {
		if p.tt != ErrorToken {
			p.err = fmt.Errorf("expected '%v' instead of '%v' in if statement", tt.String(), string(p.data))
		}
		return false
	}
	p.next()
	return true
}

func (p *Parser) parseStmt() Node {
	nodes := []Node{}
	switch p.tt {
	case LetToken, ConstToken, VarToken:
		nodes = append(nodes, p.parseDecl())
	case OpenBraceToken:
		nodes = append(nodes, p.parseToken())
		for p.tt != CloseBraceToken && p.tt != ErrorToken {
			nodes = append(nodes, p.parseStmt())
		}
		if p.tt == CloseBraceToken {
			nodes = append(nodes, p.parseToken())
		}
	case ContinueToken, BreakToken:
		nodes = append(nodes, p.parseToken())
		if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			nodes = append(nodes, p.parseToken())
		}
	case ReturnToken:
		nodes = append(nodes, p.parseToken())
		if p.tt != SemicolonToken && p.tt != LineTerminatorToken && p.tt != ErrorToken {
			nodes = append(nodes, p.parseExpr(true))
		}
	case IfToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume(OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(true))
		if !p.consume(CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
		if p.tt == ElseToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseStmt())
		}
	case WithToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume(OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(true))
		if !p.consume(CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case DoToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseStmt())
		if p.tt != WhileToken {
			if p.tt != ErrorToken {
				p.err = fmt.Errorf("expected 'while' instead of '%v' in do statement", string(p.data))
			}
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
		if !p.consume(OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(true))
		if !p.consume(CloseParenToken) {
			return Node{}
		}
	case WhileToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume(OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(true))
		if !p.consume(CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case ForToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume(OpenParenToken) {
			return Node{}
		}
		if p.tt == VarToken || p.tt == LetToken || p.tt == ConstToken {
			nodes = append(nodes, p.parseDecl())
			if p.tt == InToken {
				nodes = append(nodes, p.parseToken())
			} else if p.tt == SemicolonToken {
				p.next()
				nodes = append(nodes, p.parseExpr(true))
				if !p.consume(SemicolonToken) {
					return Node{}
				}
				nodes = append(nodes, p.parseExpr(true))
			} else {
				if p.tt != ErrorToken {
					p.err = fmt.Errorf("expected 'in' or ';' instead of '%v' in for statement", string(p.data))
				}
				return Node{}
			}
		}
		if !p.consume(CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case FunctionToken:
	case AsyncToken: // async function
	case ClassToken:
	case SemicolonToken, LineTerminatorToken, ErrorToken:
		// empty
	default:
		nodes = append(nodes, p.parseExpr(true))
	}
	if p.tt == SemicolonToken || p.tt == LineTerminatorToken {
		p.next()
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseDecl() Node {
	nodes := []Node{}
	if p.tt == VarToken || p.tt == LetToken || p.tt == ConstToken {
		nodes = append(nodes, p.parseToken())
		for {
			nodes = append(nodes, p.parseBinding())
			if p.tt != CommaToken {
				break
			}
			nodes = append(nodes, p.parseToken())
		}
	} else {
		if p.tt != ErrorToken {
			p.err = fmt.Errorf("unexpected token in declaration: %v", string(p.data))
		}
		return Node{}
	}
	return Node{DeclGrammar, nodes, 0, nil}
}

func (p *Parser) parseBinding() Node {
	nodes := []Node{}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseToken())
		if p.tt == EqToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr(false))
		}
	} else if p.tt == OpenBracketToken {
	} else if p.tt == OpenBraceToken {
	} else {
		if p.tt != ErrorToken {
			p.err = fmt.Errorf("unexpected token in binding: %v", string(p.data))
		}
		return Node{}
	}
	return Node{BindingGrammar, nodes, 0, nil}
}

func (p *Parser) parseExpr(allowComma bool) Node {
	level := 0
	nodes := []Node{}
	for {
		switch p.tt {
		case IdentifierToken, FalseToken, TrueToken, NullToken, ThisToken, StringToken, NumericToken, TemplateToken, TypeofToken, VoidToken, DeleteToken, AwaitToken, InstanceofToken, InToken, NewToken, SuperToken, AddToken, SubToken, MulToken, ExpToken, DivToken, ModToken, IncrToken, DecrToken, LtLtToken, GtGtToken, GtGtGtToken, LtToken, GtToken, LtEqToken, GtEqToken, EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken, BitAndToken, BitXorToken, BitOrToken, AndToken, OrToken, EqToken, AddEqToken, SubEqToken, MulEqToken, ExpEqToken, DivEqToken, ModEqToken, LtLtEqToken, GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitXorEqToken, BitOrEqToken:
			nodes = append(nodes, p.parseToken())
		case CommaToken:
			if !allowComma {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			nodes = append(nodes, p.parseToken())
		case OpenParenToken, OpenBraceToken, OpenBracketToken:
			level++
			nodes = append(nodes, p.parseToken())
		case CloseParenToken, CloseBraceToken, CloseBracketToken:
			if level == 0 {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			level--
			nodes = append(nodes, p.parseToken())
		case DotToken, QuestionToken, ColonToken:
			nodes = append(nodes, p.parseToken())
		case SemicolonToken, ErrorToken:
			return Node{ExprGrammar, nodes, 0, nil}
		default:
			return Node{ExprGrammar, nodes, 0, nil}
		}
	}
}

func (p *Parser) parseToken() Node {
	node := Node{TokenGrammar, nil, p.tt, p.data}
	p.next()
	return node
}
