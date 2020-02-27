package js

import (
	"bytes"
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
	ClauseGrammar
	MethodGrammar
	ParamGrammar
	ExprGrammar
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
	case ClauseGrammar:
		return "Clause"
	case MethodGrammar:
		return "Method"
	case ParamGrammar:
		return "Param"
	case ExprGrammar:
		return "Expr"
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

	tt     TokenType
	data   []byte
	regexp bool
}

// Parse returns a JS AST tree of.
func Parse(r io.Reader) (Node, error) {
	l := NewLexer(r)
	defer l.Restore()

	p := &Parser{l: l}
	p.next()
	return p.parseModule(), p.err
}

////////////////////////////////////////////////////////////////

func (p *Parser) next() {
	if p.err != nil {
		return
	}

	p.tt, p.data = p.l.Next()
	if p.tt == WhitespaceToken {
		p.tt, p.data = p.l.Next()
	} else if p.tt == ErrorToken {
		p.err = p.l.Err()
	}
}

func (p *Parser) fail(in string, expected ...TokenType) {
	if p.tt != ErrorToken {
		s := "unexpected"
		if 0 < len(expected) {
			s = "expected"
			for i, tt := range expected[:len(expected)-1] {
				if 0 < i {
					s += ","
				}
				s += " '" + tt.String() + "'"
			}
			if 2 < len(expected) {
				s += ", or"
			} else if 1 < len(expected) {
				s += " or"
			}
			s += " '" + expected[len(expected)-1].String() + "' instead of"
		}
		p.err = fmt.Errorf("%s '%v' in %s", s, string(p.data), in)
		p.tt = ErrorToken
		p.data = nil
	}
}

func (p *Parser) consume(in string, tt TokenType) bool {
	if p.tt != tt {
		p.fail(in, tt)
		return false
	}
	p.next()
	return true
}

func (p *Parser) parseModule() Node {
	nodes := []Node{}
	for {
		switch p.tt {
		case ErrorToken:
			return Node{ModuleGrammar, nodes, 0, nil}
		case ImportToken, ExportToken:
			panic("import and export statements not implemented") // TODO
		default:
			nodes = append(nodes, p.parseStmt())
		}
	}
}

func (p *Parser) parseStmt() Node {
	nodes := []Node{}
	switch p.tt {
	case OpenBraceToken:
		return p.parseBlockStmt("block statement")
	case LetToken, ConstToken, VarToken:
		nodes = p.parseVarDecl(nodes)
	case ContinueToken, BreakToken:
		nodes = append(nodes, p.parseToken())
		if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			nodes = append(nodes, p.parseToken())
		}
	case ReturnToken:
		nodes = append(nodes, p.parseToken())
		if p.tt != SemicolonToken && p.tt != LineTerminatorToken && p.tt != ErrorToken {
			nodes = append(nodes, p.parseExpr(RegularExpr))
		}
	case IfToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("if statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(RegularExpr))
		if !p.consume("if statement", CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
		if p.tt == ElseToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseStmt())
		}
	case WithToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("with statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(RegularExpr))
		if !p.consume("with statement", CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case DoToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseStmt())
		if p.tt != WhileToken {
			p.fail("do statement", WhileToken)
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
		if !p.consume("do statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(RegularExpr))
		if !p.consume("do statement", CloseParenToken) {
			return Node{}
		}
	case WhileToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("while statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(RegularExpr))
		if !p.consume("while statement", CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case ForToken:
		nodes = append(nodes, p.parseToken())
		if p.tt == AwaitToken {
			nodes = append(nodes, p.parseToken())
		}
		if !p.consume("for statement", OpenParenToken) {
			return Node{}
		}
		if p.tt == VarToken || p.tt == LetToken || p.tt == ConstToken {
			declNodes := []Node{}
			declNodes = p.parseVarDecl(declNodes)
			nodes = append(nodes, Node{StmtGrammar, declNodes, 0, nil})
		} else {
			nodes = append(nodes, p.parseExpr(LeftHandSideExpr))
		}
		if p.tt == SemicolonToken {
			p.next()
			nodes = append(nodes, p.parseExpr(RegularExpr))
			if !p.consume("for statement", SemicolonToken) {
				return Node{}
			}
			nodes = append(nodes, p.parseExpr(RegularExpr))
		} else if p.tt == InToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr(RegularExpr))
		} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("of")) {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr(AssignmentExpr))
		} else {
			p.fail("for statement", InToken, OfToken, SemicolonToken)
			return Node{}
		}
		if !p.consume("for statement", CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case IdentifierToken, YieldToken, AwaitToken:
		ident := p.parseToken()
		if p.tt == ColonToken {
			nodes = append(nodes, ident)
			p.next()
			nodes = append(nodes, p.parseStmt())
		} else {
			expr := p.parseExpr(RegularExpr)
			expr.nodes = append([]Node{ident}, expr.nodes...)
			nodes = append(nodes, expr)
		}
	case SwitchToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("switch statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr(RegularExpr))
		if !p.consume("switch statement", CloseParenToken) {
			return Node{}
		}

		// case block
		if !p.consume("switch statement", OpenBraceToken) {
			return Node{}
		}
		for p.tt != ErrorToken {
			if p.tt == CloseBraceToken {
				p.next()
				break
			}

			clauseNodes := []Node{}
			if p.tt == CaseToken {
				clauseNodes = append(clauseNodes, p.parseToken())
				clauseNodes = append(clauseNodes, p.parseExpr(RegularExpr))
			} else if p.tt == DefaultToken {
				clauseNodes = append(clauseNodes, p.parseToken())
			} else {
				p.fail("switch statement", CaseToken, DefaultToken)
				return Node{}
			}
			if !p.consume("switch statement", ColonToken) {
				return Node{}
			}
			for p.tt != CaseToken && p.tt != DefaultToken && p.tt != CloseBraceToken && p.tt != ErrorToken {
				clauseNodes = append(clauseNodes, p.parseStmt())
			}
			nodes = append(nodes, Node{ClauseGrammar, clauseNodes, 0, nil})
		}
	case FunctionToken:
		nodes = p.parseFuncDecl(nodes)
	case AsyncToken: // async function
		nodes = append(nodes, p.parseToken())
		if p.tt != FunctionToken {
			p.fail("async function statement", FunctionToken)
			return Node{}
		}
		nodes = p.parseFuncDecl(nodes)
	case ClassToken:
		nodes = p.parseClassDecl(nodes)
	case ThrowToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseExpr(RegularExpr))
	case TryToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseBlockStmt("try statement"))

		if p.tt == CatchToken {
			nodes = append(nodes, p.parseToken())
			if p.tt == OpenParenToken {
				nodes = append(nodes, p.parseBinding())
			}
			nodes = append(nodes, p.parseBlockStmt("catch statement"))
		}
		if p.tt == FinallyToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseBlockStmt("finally statement"))
		}
	case DebuggerToken:
		nodes = append(nodes, p.parseToken())
	case SemicolonToken, LineTerminatorToken:
		// empty
	case ErrorToken:
		return Node{}
	default:
		nodes = append(nodes, p.parseExpr(RegularExpr))
	}
	if p.tt == SemicolonToken || p.tt == LineTerminatorToken {
		p.next()
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseVarDecl(nodes []Node) []Node {
	// assume we're at var, let or const
	nodes = append(nodes, p.parseToken())
	for {
		nodes = append(nodes, p.parseBinding())
		if p.tt == EqToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr(AssignmentExpr))
		}
		if p.tt != CommaToken {
			break
		}
		nodes = append(nodes, p.parseToken())
	}
	return nodes
}

func (p *Parser) parseFuncDecl(nodes []Node) []Node {
	// assume we're at function
	nodes = append(nodes, p.parseToken())
	if p.tt == MulToken {
		nodes = append(nodes, p.parseToken())
	}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseToken())
	}
	nodes = p.parseFuncParams("function declaration", nodes)
	nodes = append(nodes, p.parseBlockStmt("function declaration"))
	return nodes
}

func (p *Parser) parseFuncParams(in string, nodes []Node) []Node {
	if !p.consume(in, OpenParenToken) {
		return nil
	}

	for p.tt != CloseParenToken {
		param := []Node{}
		// binding rest element
		if p.tt == EllipsisToken {
			param = append(param, p.parseToken())
			param = append(param, p.parseBinding())
			nodes = append(nodes, Node{ParamGrammar, param, 0, nil})
			break
		}

		// binding element
		param = append(param, p.parseBinding())
		if p.tt == EqToken {
			param = append(param, p.parseToken())
			param = append(param, p.parseExpr(AssignmentExpr))
		}
		nodes = append(nodes, Node{ParamGrammar, param, 0, nil})

		if p.tt == CommaToken {
			p.next()
		} else if p.tt == CloseParenToken {
			break
		} else {
			p.fail(in, CommaToken, CloseParenToken)
			return nil
		}
	}
	if !p.consume(in, CloseParenToken) {
		return nil
	}
	return nodes
}

func (p *Parser) parseBlockStmt(in string) Node {
	if p.tt != OpenBraceToken {
		p.fail(in, OpenBraceToken)
		return Node{}
	}
	nodes := []Node{}
	nodes = append(nodes, p.parseToken())
	for p.tt != ErrorToken {
		if p.tt == CloseBraceToken {
			nodes = append(nodes, p.parseToken())
			break
		}
		nodes = append(nodes, p.parseStmt())
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseClassDecl(nodes []Node) []Node {
	// assume we're at class
	nodes = append(nodes, p.parseToken())
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseToken())
	}
	if p.tt == ExtendsToken {
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseExpr(ClassLeftHandSideExpr))
	}

	if !p.consume("class statement", OpenBraceToken) {
		return nil
	}
	for p.tt != ErrorToken {
		if p.tt == SemicolonToken {
			p.next()
			continue
		} else if p.tt == CloseBraceToken {
			break
		}

		var methodDef Node
		if p.tt == StaticToken {
			static := p.parseToken()
			methodDef = p.parseMethodDef()
			methodDef.nodes = append([]Node{static}, methodDef.nodes...)
		} else {
			methodDef = p.parseMethodDef()
		}
		nodes = append(nodes, methodDef)
	}
	if !p.consume("class statement", CloseBraceToken) {
		return nil
	}
	return nodes
}

func (p *Parser) parseMethodDefStart(in string, nodes []Node) []Node {
	for {
		if p.tt == MulToken || p.tt == AsyncToken || p.tt == IdentifierToken || p.tt == StringToken || p.tt == NumericToken || p.tt == IdentifierToken && (bytes.Equal(p.data, []byte("get")) || bytes.Equal(p.data, []byte("set"))) {
			nodes = append(nodes, p.parseToken())
		} else if p.tt == OpenBracketToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr(LeftHandSideExpr))
			if p.tt != CloseBracketToken {
				p.fail(in, CloseBracketToken)
				return nil
			}
			nodes = append(nodes, p.parseToken())
		} else {
			if len(nodes) == 0 {
				p.fail(in, MulToken, GetToken, SetToken, AsyncToken, IdentifierToken, StringToken, NumericToken, OpenBracketToken)
				return nil
			}
			return nodes
		}
	}
}

func (p *Parser) parseMethodDef() Node {
	nodes := []Node{}
	nodes = p.parseMethodDefStart("method definition", nodes)
	nodes = p.parseFuncParams("method definition", nodes)
	nodes = append(nodes, p.parseBlockStmt("method definition"))
	return Node{MethodGrammar, nodes, 0, nil}
}

func (p *Parser) parseBinding() Node {
	// binding identifier or binding pattern
	nodes := []Node{}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseToken())
	} else if p.tt == OpenBracketToken {
		nodes = append(nodes, p.parseToken())
		for {
			// elision
			for p.tt == CommaToken {
				nodes = append(nodes, p.parseToken())
			}
			// binding rest element
			if p.tt == EllipsisToken {
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseBinding())
				if p.tt != CloseBracketToken {
					p.fail("array binding pattern", CloseBracketToken)
					return Node{}
				}
				break
			}

			// binding element
			nodes = append(nodes, p.parseBinding())
			if p.tt == EqToken {
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseExpr(AssignmentExpr))
			}

			if p.tt == CloseBracketToken {
				break
			} else if p.tt != CommaToken {
				p.fail("array binding pattern", CommaToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
		}
		nodes = append(nodes, p.parseToken())
	} else if p.tt == OpenBraceToken {
		nodes = append(nodes, p.parseToken())
		for {
			// binding rest property
			if p.tt == EllipsisToken {
				nodes = append(nodes, p.parseToken())
				if p.tt != IdentifierToken && p.tt != YieldToken && p.tt != AwaitToken {
					p.fail("object binding pattern", IdentifierToken, YieldToken, AwaitToken)
				}
				nodes = append(nodes, p.parseToken())
				if p.tt != CloseBraceToken {
					p.fail("object binding pattern", CloseBraceToken)
					return Node{}
				}
				break
			}

			// binding property
			ttPrev, dataPrev := p.tt, p.data
			p.next()
			nodes = append(nodes, Node{TokenGrammar, nil, ttPrev, dataPrev})
			if p.tt == ColonToken {
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseBinding())
				if p.tt == EqToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseExpr(AssignmentExpr))
				}
			}

			if p.tt == CloseBraceToken {
				break
			} else if p.tt != CommaToken {
				p.fail("object binding pattern", CommaToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
		}
		nodes = append(nodes, p.parseToken())
	} else {
		p.fail("binding")
		return Node{}
	}
	return Node{BindingGrammar, nodes, 0, nil}
}

type ExprType int

const (
	RegularExpr           ExprType = iota
	AssignmentExpr                 // same as regular, but without commas
	LeftHandSideExpr               // subset of assignment, mostly forbids operators
	ClassLeftHandSideExpr          // LHS without objects
)

func (p *Parser) parseExpr(et ExprType) Node {
	nodes := []Node{}

	// reparse input if we have / or /= as the beginning of a new expression, this could be a regular expression!
	if p.tt == DivToken || p.tt == DivEqToken {
		p.tt, p.data = p.l.RegExp()
	}

	for {
		switch p.tt {
		case OrToken, AndToken, BitOrToken, BitXorToken, BitAndToken, EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken, LtToken, GtToken, LtEqToken, GtEqToken, LtLtToken, GtGtToken, GtGtGtToken, AddToken, SubToken, MulToken, DivToken, ModToken, ExpToken, NotToken, BitNotToken, IncrToken, DecrToken, EqToken, MulEqToken, DivEqToken, ModEqToken, ExpEqToken, AddEqToken, SubEqToken, LtLtEqToken, GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitXorEqToken, BitOrEqToken, InstanceofToken, InToken, TypeofToken, VoidToken, DeleteToken:
			if et >= LeftHandSideExpr {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			nodes = append(nodes, p.parseToken())
		case CommaToken:
			if et >= AssignmentExpr {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			nodes = append(nodes, p.parseToken())
		case QuestionToken:
			if et >= LeftHandSideExpr {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			panic("not implemented") // TODO
		case ArrowToken:
			if et >= LeftHandSideExpr {
				return Node{ExprGrammar, nodes, 0, nil}
			}
			panic("not implemented") // TODO
		case NewToken, DotToken, SuperToken, ThisToken, NullToken, TrueToken, FalseToken, NumericToken, StringToken, TemplateToken, RegExpToken, AwaitToken, YieldToken, IdentifierToken:
			nodes = append(nodes, p.parseToken())
		case OpenBracketToken:
			// array literal and [expression]
			nodes = append(nodes, p.parseToken())
			for p.tt != CloseBracketToken && p.tt != ErrorToken {
				if p.tt == EllipsisToken || p.tt == CommaToken {
					nodes = append(nodes, p.parseToken())
				} else {
					nodes = append(nodes, p.parseExpr(AssignmentExpr))
				}
			}
			nodes = append(nodes, p.parseToken())
		case OpenBraceToken:
			if et == ClassLeftHandSideExpr {
				return Node{ExprGrammar, nodes, 0, nil}
			}

			// object literal
			nodes = append(nodes, p.parseToken())
			for p.tt != CloseBraceToken && p.tt != ErrorToken {
				if p.tt == EllipsisToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseExpr(AssignmentExpr))
				} else if p.tt == CommaToken {
					nodes = append(nodes, p.parseToken())
				} else {
					methodDef := []Node{}
					methodDef = p.parseMethodDefStart("object literal", methodDef)
					if p.tt == EqToken || p.tt == ColonToken {
						nodes = append(nodes, methodDef...)
						nodes = append(nodes, p.parseToken())
						nodes = append(nodes, p.parseExpr(AssignmentExpr))
					} else if p.tt != CommaToken {
						methodDef = p.parseFuncParams("method definition", methodDef)
						methodDef = append(methodDef, p.parseBlockStmt("method definition"))
						nodes = append(nodes, Node{MethodGrammar, methodDef, 0, nil})
					}
				}
			}
			nodes = append(nodes, p.parseToken())
		case OpenParenToken:
			// arguments, parenthesized expression and arrow parameter list
			nodes = append(nodes, p.parseToken())
			for p.tt != CloseParenToken && p.tt != ErrorToken {
				if p.tt == EllipsisToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseBinding())
				} else if p.tt == CommaToken {
					nodes = append(nodes, p.parseToken())
				} else {
					nodes = append(nodes, p.parseExpr(AssignmentExpr))
				}
			}
			nodes = append(nodes, p.parseToken())
		case FunctionToken:
			nodes = p.parseFuncDecl(nodes)
		case AsyncToken: // async function
			nodes = append(nodes, p.parseToken())
			if p.tt != FunctionToken {
				p.fail("async function statement", FunctionToken)
				return Node{}
			}
			nodes = p.parseFuncDecl(nodes)
		case ClassToken:
			nodes = p.parseClassDecl(nodes)
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
