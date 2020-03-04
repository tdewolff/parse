package js

import (
	"bytes"
	"io"
	"strconv"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/buffer"
)

// TODO: clarify usage of yield, is it a YieldExpression or an Identifier?

type Node struct {
	GrammarType
	Nodes []Node

	// filled if GrammarType == TokenGrammar
	TokenType
	Data []byte
}

func (n Node) String() string {
	if n.GrammarType == TokenGrammar {
		return string(n.Data)
	}
	s := ""
	for _, child := range n.Nodes {
		s += " " + child.String()
	}
	if 0 < len(s) {
		s = s[1:]
	}
	if n.GrammarType == ModuleGrammar {
		return s
	}
	return n.GrammarType.String() + "(" + s + ")"
}

// GrammarType determines the type of grammar.
type GrammarType uint32

// GrammarType values.
const (
	ErrorGrammar GrammarType = iota // extra token when errors occur
	TokenGrammar
	ModuleGrammar
	BindingGrammar
	ClauseGrammar
	MethodGrammar
	ParamsGrammar
	ExprGrammar
	StmtGrammar
)

// String returns the string representation of a GrammarType.
func (tt GrammarType) String() string {
	switch tt {
	case ErrorGrammar:
		return "Error"
	case TokenGrammar:
		return "Token"
	case ModuleGrammar:
		return "Module"
	case BindingGrammar:
		return "Binding"
	case ClauseGrammar:
		return "Clause"
	case MethodGrammar:
		return "Method"
	case ParamsGrammar:
		return "Params"
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

	tt                 TokenType
	data               []byte
	prevLineTerminator bool
	asyncLevel         int
	inFor              bool
}

// Parse returns a JS AST tree of.
func Parse(r io.Reader) (Node, error) {
	l := NewLexer(r)
	defer l.Restore()

	p := &Parser{l: l}

	p.tt = WhitespaceToken // trick so that next() works from the start
	p.next()

	module := p.parseModule()
	if p.err == nil {
		p.err = p.l.Err()
	}
	if p.err == io.EOF {
		p.err = nil
	}
	return module, p.err
}

////////////////////////////////////////////////////////////////

func (p *Parser) next() {
	if p.tt == ErrorToken {
		return
	}
	p.prevLineTerminator = false

	p.tt, p.data = p.l.Next()
	for p.tt == WhitespaceToken || p.tt == LineTerminatorToken || p.tt == CommentToken || p.tt == CommentLineTerminatorToken {
		if p.tt == LineTerminatorToken || p.tt == CommentLineTerminatorToken {
			p.prevLineTerminator = true
		}
		p.tt, p.data = p.l.Next()
	}
}

func (p *Parser) fail(in string, expected ...TokenType) {
	if p.err == nil {
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

		at := "'" + string(p.data) + "'"
		if p.tt == ErrorToken {
			at = p.l.Err().Error()
		}

		offset := p.l.r.Offset() - len(p.data)
		p.err = parse.NewError(buffer.NewReader(p.l.r.Bytes()), offset, "%s %s in %s", s, at, in)
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
		case ImportToken:
			nodes = append(nodes, p.parseImportStmt())
		case ExportToken:
			nodes = append(nodes, p.parseExportStmt())
		default:
			nodes = append(nodes, p.parseStmt())
		}
	}
}

func (p *Parser) parseStmt() Node {
	nodes := []Node{}
	switch p.tt {
	case OpenBraceToken:
		block := p.parseBlockStmt("block statement")
		if len(block.Nodes) == 2 {
			nodes = append(nodes, Node{TokenGrammar, nil, SemicolonToken, []byte(";")})
		} else {
			return block
		}
	case LetToken, ConstToken, VarToken:
		nodes = p.parseVarDecl(nodes)
	case ContinueToken, BreakToken:
		nodes = append(nodes, p.parseToken())
		if !p.prevLineTerminator && (p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken && p.asyncLevel == 0) {
			nodes = append(nodes, p.parseTokenAs(IdentifierToken))
		}
	case ReturnToken:
		nodes = append(nodes, p.parseToken())
		if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != LineTerminatorToken && p.tt != ErrorToken {
			nodes = append(nodes, p.parseExpr())
		}
	case IfToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("if statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr())
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
		nodes = append(nodes, p.parseExpr())
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
		nodes = append(nodes, p.parseExpr())
		if !p.consume("do statement", CloseParenToken) {
			return Node{}
		}
	case WhileToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("while statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr())
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
		p.inFor = true
		if p.tt == VarToken || p.tt == LetToken || p.tt == ConstToken {
			declNodes := []Node{}
			declNodes = p.parseVarDecl(declNodes)
			nodes = append(nodes, Node{StmtGrammar, declNodes, 0, nil})
		} else if p.tt != SemicolonToken {
			nodes = append(nodes, p.parseExpr())
		}
		p.inFor = false

		if p.tt == SemicolonToken {
			nodes = append(nodes, p.parseToken())
			if p.tt != SemicolonToken && p.tt != CloseParenToken {
				nodes = append(nodes, p.parseExpr())
			}
			if p.tt == SemicolonToken {
				nodes = append(nodes, p.parseToken())
				if p.tt != CloseParenToken {
					nodes = append(nodes, p.parseExpr())
				}
			}
		} else if p.tt == InToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseExpr())
		} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("of")) {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseAssignmentExpr())
		} else {
			p.fail("for statement", InToken, OfToken, SemicolonToken)
			return Node{}
		}
		if !p.consume("for statement", CloseParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseStmt())
	case IdentifierToken, YieldToken, AwaitToken:
		// could be expression or labelled statement, try expression first and convert to labelled statement if possible
		expr := p.parseExpr()
		if p.tt == ColonToken && len(expr.Nodes) == 1 && (p.tt != AwaitToken || p.asyncLevel == 0) {
			expr.Nodes[0].TokenType = IdentifierToken
			nodes = append(nodes, expr.Nodes[0])
			p.next()
			nodes = append(nodes, p.parseStmt())
		} else {
			nodes = append(nodes, expr)
		}
	case SwitchToken:
		nodes = append(nodes, p.parseToken())
		if !p.consume("switch statement", OpenParenToken) {
			return Node{}
		}
		nodes = append(nodes, p.parseExpr())
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
				clauseNodes = append(clauseNodes, p.parseExpr())
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
		p.asyncLevel++
		nodes = p.parseFuncDecl(nodes)
		p.asyncLevel--
	case ClassToken:
		nodes = p.parseClassDecl(nodes)
	case ThrowToken:
		nodes = append(nodes, p.parseToken())
		if !p.prevLineTerminator {
			nodes = append(nodes, p.parseExpr())
		}
	case TryToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseBlockStmt("try statement"))
		if p.tt == CatchToken {
			nodes = append(nodes, p.parseToken())
			if p.tt == OpenParenToken {
				p.next()
				nodes = append(nodes, p.parseBinding())
				if p.tt != CloseParenToken {
					p.fail("try statement", CloseParenToken)
					return Node{}
				}
				p.next()
			}
			nodes = append(nodes, p.parseBlockStmt("try statement"))
		}
		if p.tt == FinallyToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseBlockStmt("try statement"))
		}
	case DebuggerToken:
		nodes = append(nodes, p.parseToken())
	case SemicolonToken, LineTerminatorToken, ErrorToken:
		nodes = append(nodes, Node{TokenGrammar, nil, SemicolonToken, []byte(";")})
	default:
		nodes = append(nodes, p.parseExpr())
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseVarDecl(nodes []Node) []Node {
	// assume we're at var, let or const
	nodes = append(nodes, p.parseToken())
	for {
		nodes = append(nodes, p.parseBindingElement())
		if p.tt == CommaToken {
			p.next()
		} else {
			break
		}
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

func (p *Parser) parseImportStmt() Node {
	// assume we're at import
	nodes := []Node{}
	nodes = append(nodes, p.parseToken())
	if p.tt == StringToken {
		nodes = append(nodes, p.parseToken())
	} else {
		if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			nodes = append(nodes, p.parseTokenAs(IdentifierToken))
			if p.tt == CommaToken {
				nodes = append(nodes, p.parseToken())
			}
		}
		if p.tt == MulToken {
			nodes = append(nodes, p.parseToken())
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("as")) {
				p.fail("import statement", AsToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
			if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
				nodes = append(nodes, p.parseTokenAs(IdentifierToken))
			} else {
				p.fail("import statement", IdentifierToken)
				return Node{}
			}
		} else if p.tt == OpenBraceToken {
			nodes = append(nodes, p.parseToken())
			for IsIdentifier(p.tt) {
				nodes = append(nodes, p.parseTokenAs(IdentifierToken))
				if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
					nodes = append(nodes, p.parseToken())
					if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
						nodes = append(nodes, p.parseTokenAs(IdentifierToken))
					} else {
						p.fail("import statement", IdentifierToken)
						return Node{}
					}
				}
				if p.tt == CommaToken {
					nodes = append(nodes, p.parseToken())
				}
			}
			if nodes[len(nodes)-1].TokenType == CommaToken {
				nodes = nodes[:len(nodes)-1]
			}
			if p.tt != CloseBraceToken {
				p.fail("import statement", CloseBraceToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
		}
		if len(nodes) == 1 {
			p.fail("import statement", StringToken, IdentifierToken, MulToken, OpenBraceToken)
			return Node{}
		}

		if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("from")) {
			p.fail("import statement", FromToken)
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
		if p.tt != StringToken {
			p.fail("import statement", StringToken)
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseExportStmt() Node {
	// assume we're at export
	nodes := []Node{}
	nodes = append(nodes, p.parseToken())
	if p.tt == MulToken || p.tt == OpenBraceToken {
		if p.tt == MulToken {
			nodes = append(nodes, p.parseToken())
			if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
				nodes = append(nodes, p.parseTokenAs(AsToken))
				if !IsIdentifier(p.tt) {
					p.fail("export statement", IdentifierToken)
					return Node{}
				}
				nodes = append(nodes, p.parseTokenAs(IdentifierToken))
			}
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("from")) {
				p.fail("export statement", FromToken)
				return Node{}
			}
		} else {
			nodes = append(nodes, p.parseToken())
			for IsIdentifier(p.tt) {
				nodes = append(nodes, p.parseTokenAs(IdentifierToken))
				if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
					nodes = append(nodes, p.parseTokenAs(AsToken))
					if !IsIdentifier(p.tt) {
						p.fail("export statement", IdentifierToken)
						return Node{}
					}
					nodes = append(nodes, p.parseTokenAs(IdentifierToken))
				}
				if p.tt == CommaToken {
					nodes = append(nodes, p.parseToken())
				}
			}
			if nodes[len(nodes)-1].TokenType == CommaToken {
				nodes = nodes[:len(nodes)-1]
			}
			if p.tt != CloseBraceToken {
				p.fail("export statement", CloseBraceToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
		}
		if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("from")) {
			nodes = append(nodes, p.parseTokenAs(FromToken))
			if p.tt != StringToken {
				p.fail("export statement", StringToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
		}
	} else if p.tt == VarToken || p.tt == ConstToken || p.tt == LetToken {
		nodes = append(nodes, Node{StmtGrammar, p.parseVarDecl(nil), 0, nil})
	} else if p.tt == FunctionToken {
		nodes = append(nodes, Node{StmtGrammar, p.parseFuncDecl(nil), 0, nil})
	} else if p.tt == AsyncToken { // async function
		stmt := []Node{}
		stmt = append(stmt, p.parseToken())
		if p.tt != FunctionToken {
			p.fail("export statement", FunctionToken)
			return Node{}
		}
		p.asyncLevel++
		nodes = append(nodes, Node{StmtGrammar, p.parseFuncDecl(stmt), 0, nil})
		p.asyncLevel--
	} else if p.tt == ClassToken {
		nodes = append(nodes, Node{StmtGrammar, p.parseClassDecl(nil), 0, nil})
	} else if p.tt == DefaultToken {
		nodes = append(nodes, p.parseToken())
		if p.tt == FunctionToken {
			nodes = append(nodes, Node{StmtGrammar, p.parseFuncDecl(nil), 0, nil})
		} else if p.tt == AsyncToken { // async function
			stmt := []Node{}
			stmt = append(stmt, p.parseToken())
			if p.tt != FunctionToken {
				p.fail("export statement", FunctionToken)
				return Node{}
			}
			p.asyncLevel++
			nodes = append(nodes, Node{StmtGrammar, p.parseFuncDecl(stmt), 0, nil})
			p.asyncLevel--
		} else if p.tt == ClassToken {
			nodes = append(nodes, Node{StmtGrammar, p.parseClassDecl(nil), 0, nil})
		} else {
			nodes = append(nodes, p.parseAssignmentExpr())
		}
	} else {
		p.fail("export statement", MulToken, OpenBraceToken, VarToken, LetToken, ConstToken, FunctionToken, AsyncToken, ClassToken, DefaultToken)
		return Node{}
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return Node{StmtGrammar, nodes, 0, nil}
}

func (p *Parser) parseFuncDecl(nodes []Node) []Node {
	// assume we're at function
	nodes = append(nodes, p.parseToken())
	if p.tt == MulToken {
		nodes = append(nodes, p.parseToken())
	}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	}
	nodes = append(nodes, p.parseFuncParams("function declaration"))
	nodes = append(nodes, p.parseBlockStmt("function declaration"))
	return nodes
}

func (p *Parser) parseFuncParams(in string) Node {
	if !p.consume(in, OpenParenToken) {
		return Node{}
	}

	nodes := []Node{}
	for p.tt != CloseParenToken {
		// binding rest element
		if p.tt == EllipsisToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseBinding())
			break
		}

		nodes = append(nodes, p.parseBindingElement())

		if p.tt == CommaToken {
			p.next()
		} else if p.tt == CloseParenToken {
			break
		} else {
			p.fail(in, CommaToken, CloseParenToken)
			return Node{}
		}
	}
	if !p.consume(in, CloseParenToken) {
		return Node{}
	}
	return Node{ParamsGrammar, nodes, 0, nil}
}

func (p *Parser) parseClassDecl(nodes []Node) []Node {
	// assume we're at class
	nodes = append(nodes, p.parseToken())
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	}
	if p.tt == ExtendsToken {
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, Node{ExprGrammar, p.parseLeftHandSideExpr(nil), 0, nil})
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
		nodes = append(nodes, p.parseMethodDef())
	}
	if !p.consume("class statement", CloseBraceToken) {
		return nil
	}
	return nodes
}

func (p *Parser) parseMethodDef() Node {
	nodes := []Node{}
	if p.tt == StaticToken {
		nodes = append(nodes, p.parseToken())
	}
	async := false
	if p.tt == AsyncToken || p.tt == MulToken {
		if p.tt == AsyncToken {
			nodes = append(nodes, p.parseToken())
			async = true
		}
		if p.tt == MulToken {
			nodes = append(nodes, p.parseToken())
		}
	} else if p.tt == IdentifierToken && (bytes.Equal(p.data, []byte("get")) || bytes.Equal(p.data, []byte("set"))) {
		nodes = append(nodes, p.parseToken())
	}

	if IsIdentifier(p.tt) {
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	} else if p.tt == StringToken || IsNumeric(p.tt) {
	} else if p.tt == OpenBracketToken {
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseAssignmentExpr())
		if p.tt != CloseBracketToken {
			p.fail("method definition", CloseBracketToken)
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
	} else {
		p.fail("method definition", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
		return Node{}
	}
	if async {
		p.asyncLevel++
	}
	nodes = append(nodes, p.parseFuncParams("method definition"))
	nodes = append(nodes, p.parseBlockStmt("method definition"))
	if async {
		p.asyncLevel--
	}
	return Node{MethodGrammar, nodes, 0, nil}
}

func (p *Parser) parseBindingElement() Node {
	// binding element
	binding := p.parseBinding()
	if p.tt == EqToken {
		binding.Nodes = append(binding.Nodes, p.parseToken())
		binding.Nodes = append(binding.Nodes, p.parseAssignmentExpr())
	}
	return binding
}

func (p *Parser) parseBinding() Node {
	// binding identifier or binding pattern
	nodes := []Node{}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	} else if p.tt == OpenBracketToken {
		nodes = append(nodes, p.parseToken())
		for p.tt != CloseBracketToken {
			// elision
			for p.tt == CommaToken {
				p.next()
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

			nodes = append(nodes, p.parseBindingElement())

			if p.tt == CommaToken {
				for p.tt == CommaToken {
					p.next()
				}
			} else if p.tt != CloseBracketToken {
				p.fail("array binding pattern", CommaToken, CloseBracketToken)
				return Node{}
			}
		}
		nodes = append(nodes, p.parseToken())
	} else if p.tt == OpenBraceToken {
		nodes = append(nodes, p.parseToken())
		for p.tt != CloseBraceToken {
			// binding rest property
			if p.tt == EllipsisToken {
				nodes = append(nodes, p.parseToken())
				if p.tt != IdentifierToken && p.tt != YieldToken && p.tt != AwaitToken {
					p.fail("object binding pattern", IdentifierToken)
				}
				nodes = append(nodes, Node{BindingGrammar, []Node{p.parseTokenAs(IdentifierToken)}, 0, nil})
				if p.tt != CloseBraceToken {
					p.fail("object binding pattern", CloseBraceToken)
					return Node{}
				}
				break
			}

			if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
				// single name binding
				ident := p.parseTokenAs(IdentifierToken)
				if p.tt == ColonToken {
					// property name + : + binding element
					nodes = append(nodes, ident)
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseBindingElement())
				} else {
					binding := []Node{ident}
					if p.tt == EqToken {
						binding = append(binding, p.parseToken())
						binding = append(binding, p.parseAssignmentExpr())
					}
					nodes = append(nodes, Node{BindingGrammar, binding, 0, nil})
				}
			} else if IsIdentifier(p.tt) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
				// property name + : + binding element
				if p.tt == OpenBracketToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseAssignmentExpr())
					if p.tt != CloseBracketToken {
						p.fail("object binding pattern", CloseBracketToken)
						return Node{}
					}
					nodes = append(nodes, p.parseToken())
				} else if IsIdentifier(p.tt) {
					nodes = append(nodes, p.parseTokenAs(IdentifierToken))
				} else {
					nodes = append(nodes, p.parseToken())
				}
				if p.tt != ColonToken {
					p.fail("object binding pattern", ColonToken)
					return Node{}
				}
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseBindingElement())
			} else {
				p.fail("object binding pattern", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
				return Node{}
			}

			if p.tt == CommaToken {
				p.next()
			} else if p.tt != CloseBraceToken {
				p.fail("object binding pattern", CommaToken, CloseBraceToken)
				return Node{}
			}
		}
		nodes = append(nodes, p.parseToken())
	} else {
		p.fail("binding")
		return Node{}
	}
	return Node{BindingGrammar, nodes, 0, nil}
}

func (p *Parser) parseObjectLiteral(nodes []Node) []Node {
	// assume we're on {
	nodes = append(nodes, p.parseToken())
	for p.tt != CloseBraceToken && p.tt != ErrorToken {
		if p.tt == EllipsisToken {
			nodes = append(nodes, p.parseToken())
			nodes = append(nodes, p.parseAssignmentExpr())
		} else if p.tt == CommaToken {
			nodes = append(nodes, p.parseToken())
		} else {
			async := false
			property := []Node{}
			for p.tt == MulToken || p.tt == AsyncToken || IsIdentifier(p.tt) {
				if p.tt == AsyncToken {
					async = true
					property = append(property, p.parseToken())
				} else if IsIdentifier(p.tt) {
					property = append(property, p.parseTokenAs(IdentifierToken))
				} else {
					property = append(property, p.parseToken())
				}
			}

			if (p.tt == EqToken || p.tt == CommaToken || p.tt == CloseBraceToken) && len(property) == 1 && (property[0].TokenType == IdentifierToken || property[0].TokenType == YieldToken || property[0].TokenType == AwaitToken && p.asyncLevel == 0) {
				property[0].TokenType = IdentifierToken
				nodes = append(nodes, property[0])
				if p.tt == EqToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseAssignmentExpr())
				}
			} else if 0 < len(property) && IsIdentifier(property[len(property)-1].TokenType) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
				if p.tt == StringToken || IsNumeric(p.tt) {
					property = append(property, p.parseToken())
				} else if p.tt == OpenBracketToken {
					property = append(property, p.parseToken())
					property = append(property, p.parseAssignmentExpr())
					if p.tt != CloseBracketToken {
						p.fail("object literal", CloseBracketToken)
						return nil
					}
					property = append(property, p.parseToken())
				}

				if p.tt == ColonToken {
					nodes = append(nodes, property...)
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseAssignmentExpr())
				} else if p.tt == OpenParenToken {
					if async {
						p.asyncLevel++
					}
					property = append(property, p.parseFuncParams("method definition"))
					property = append(property, p.parseBlockStmt("method definition"))
					if async {
						p.asyncLevel--
					}
					nodes = append(nodes, Node{MethodGrammar, property, 0, nil})
				} else {
					p.fail("object literal", ColonToken, OpenParenToken)
					return nil
				}
			} else {
				p.fail("object literal", EqToken, CommaToken, CloseBraceToken, EllipsisToken, IdentifierToken, StringToken, NumericToken, OpenBracketToken)
				return nil
			}
		}
	}
	if p.tt == CloseBraceToken {
		nodes = append(nodes, p.parseToken())
	}
	return nodes
}

func (p *Parser) parseTemplateLiteral(nodes []Node) []Node {
	// assume we're on 'Template' or 'TemplateStart'
	for p.tt == TemplateStartToken || p.tt == TemplateMiddleToken {
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseExpr())
		if p.tt == TemplateEndToken {
			nodes = append(nodes, p.parseToken())
			return nodes
		} else {
			p.fail("template literal", TemplateToken)
			return nil
		}
	}
	nodes = append(nodes, p.parseToken())
	return nodes
}

func (p *Parser) parsePrimaryExpr(nodes []Node) []Node {
	// reparse input if we have / or /= as the beginning of a new expression, this should be a regular expression!
	if p.tt == DivToken || p.tt == DivEqToken {
		p.tt, p.data = p.l.RegExp()
	}

	switch p.tt {
	case ThisToken, IdentifierToken, NullToken, TrueToken, FalseToken, StringToken, RegExpToken:
		nodes = append(nodes, p.parseToken())
	case YieldToken:
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	case AwaitToken:
		if p.asyncLevel != 0 {
			p.fail("expression")
			return nil
		}
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	case TemplateToken, TemplateStartToken:
		nodes = p.parseTemplateLiteral(nodes)
	case OpenBracketToken:
		// array literal and [expression]
		nodes = append(nodes, p.parseToken())
		for p.tt != CloseBracketToken && p.tt != ErrorToken {
			if p.tt == EllipsisToken || p.tt == CommaToken {
				nodes = append(nodes, p.parseToken())
			} else {
				nodes = append(nodes, p.parseAssignmentExpr())
			}
		}
		nodes = append(nodes, p.parseToken())
	case OpenBraceToken:
		nodes = p.parseObjectLiteral(nodes)
	case OpenParenToken:
		// parenthesized expression and arrow parameter list
		nodes = append(nodes, p.parseToken())
		for p.tt != CloseParenToken && p.tt != ErrorToken {
			if p.tt == EllipsisToken {
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseBinding())
			} else if p.tt == CommaToken {
				nodes = append(nodes, p.parseToken())
			} else {
				nodes = append(nodes, p.parseAssignmentExpr())
			}
		}
		nodes = append(nodes, p.parseToken())
	case ClassToken:
		nodes = p.parseClassDecl(nodes)
	case FunctionToken:
		nodes = p.parseFuncDecl(nodes)
	case AsyncToken:
		// async function
		nodes = append(nodes, p.parseToken())
		if !p.prevLineTerminator {
			if p.tt == FunctionToken {
				p.asyncLevel++
				nodes = p.parseFuncDecl(nodes)
				p.asyncLevel--
			} else {
				p.fail("async function expression", FunctionToken)
				return nil
			}
		}
	default:
		if IsNumeric(p.tt) {
			nodes = append(nodes, p.parseToken())
		} else {
			p.fail("expression")
			return nil
		}
	}
	return nodes
}

func (p *Parser) parseLeftHandSideExprEnd(nodes []Node) []Node {
	// parse arguments, [expression], .identifier, template
	if p.tt == OpenParenToken {
		nodes = append(nodes, p.parseToken())
		for {
			if p.tt == ErrorToken || p.tt == CloseParenToken {
				break
			} else if p.tt == CommaToken {
				nodes = append(nodes, p.parseToken())
			} else if p.tt == EllipsisToken {
				nodes = append(nodes, p.parseToken())
				nodes = append(nodes, p.parseAssignmentExpr())
				break
			} else {
				nodes = append(nodes, p.parseAssignmentExpr())
			}
		}
		if p.tt == CommaToken {
			p.next()
		}
		if p.tt != CloseParenToken {
			p.fail("left hand side expression", CloseParenToken)
			return nil
		}
		nodes = append(nodes, p.parseToken())
	} else if p.tt == OpenBracketToken {
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseExpr())
		if p.tt != CloseBracketToken {
			p.fail("left hand side expression", CloseBracketToken)
			return nil
		}
		nodes = append(nodes, p.parseToken())
	} else if p.tt == DotToken {
		nodes = append(nodes, p.parseToken())
		if !IsIdentifier(p.tt) {
			p.fail("left hand side expression", IdentifierToken)
			return nil
		}
		nodes = append(nodes, p.parseTokenAs(IdentifierToken))
	} else if p.tt == TemplateToken || p.tt == TemplateStartToken {
		nodes = p.parseTemplateLiteral(nodes)
	} else {
		p.fail("left hand side expression", OpenParenToken, OpenBracketToken, DotToken, TemplateToken)
		return nil
	}
	return nodes
}

func (p *Parser) parseLeftHandSideExpr(nodes []Node) []Node {
	for p.tt == NewToken {
		nodes = append(nodes, p.parseToken())
		if p.tt == DotToken {
			nodes = append(nodes, p.parseToken())
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("target")) {
				p.fail("left hand side expression", TargetToken)
				return nil
			}
			nodes = append(nodes, p.parseToken())
			goto LHSEND
		}
	}

	if p.tt == SuperToken {
		nodes = append(nodes, p.parseToken())
		if p.tt == TemplateToken || p.tt == TemplateStartToken {
			p.fail("left hand side expression")
		}
		nodes = p.parseLeftHandSideExprEnd(nodes)
	} else if p.tt == ImportToken {
		nodes = append(nodes, p.parseToken())
		if p.tt != OpenParenToken {
			p.fail("left hand side expression", OpenParenToken)
			return nil
		}
		p.next()
		nodes = append(nodes, p.parseExpr())
		if p.tt != CloseParenToken {
			p.fail("left hand side expression", CloseParenToken)
			return nil
		}
		p.next()
	} else {
		nodes = p.parsePrimaryExpr(nodes)
	}

LHSEND:
	// parse arguments, [expression], .identifier, template at the end of member expressions and call expressions
	for p.tt == OpenParenToken || p.tt == OpenBracketToken || p.tt == DotToken || p.tt == TemplateToken || p.tt == TemplateStartToken {
		nodes = p.parseLeftHandSideExprEnd(nodes)
	}

	// parse optional chaining at the end of left hand expressions
	for p.tt == OptChainToken {
		nodes = append(nodes, p.parseToken())
		if IsIdentifier(p.tt) {
			nodes = append(nodes, p.parseTokenAs(IdentifierToken))
		} else if p.tt == OpenParenToken || p.tt == OpenBracketToken || p.tt == TemplateToken || p.tt == TemplateStartToken {
			nodes = p.parseLeftHandSideExprEnd(nodes)
		} else {
			p.fail("left hand side expression", IdentifierToken, OpenParenToken, OpenBracketToken, TemplateToken)
			return nil
		}
		for p.tt == OpenParenToken || p.tt == OpenBracketToken || p.tt == DotToken || p.tt == TemplateToken || p.tt == TemplateStartToken {
			nodes = p.parseLeftHandSideExprEnd(nodes)
		}
	}
	return nodes
}

func (p *Parser) parseAssignmentExpr() Node {
	nodes := []Node{}
	if p.tt == YieldToken {
		yield := p.parseToken()
		if p.tt == ArrowToken {
			nodes = append(nodes, Node{ParamsGrammar, []Node{Node{BindingGrammar, []Node{yield}, 0, nil}}, 0, nil})
			nodes = append(nodes, p.parseToken())
			if p.tt == OpenBraceToken {
				nodes = append(nodes, p.parseBlockStmt("arrow function expression"))
			} else {
				nodes = append(nodes, p.parseAssignmentExpr())
			}
		} else {
			// YieldExpression
			nodes = append(nodes, yield)
			if !p.prevLineTerminator {
				if p.tt == MulToken {
					nodes = append(nodes, p.parseToken())
				}
				nodes = append(nodes, p.parseAssignmentExpr())
			}
		}
		return Node{ExprGrammar, nodes, 0, nil}
	} else if p.tt == AsyncToken {
		nodes = append(nodes, p.parseToken())
		if p.prevLineTerminator {
			p.fail("async function expression")
			return Node{}
		}
		if p.tt == FunctionToken {
			// primary expression
			p.asyncLevel++
			nodes = p.parseFuncDecl(nodes)
			p.asyncLevel--
		} else if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			nodes = append(nodes, Node{ParamsGrammar, []Node{Node{BindingGrammar, []Node{p.parseTokenAs(IdentifierToken)}, 0, nil}}, 0, nil})
			if p.tt != ArrowToken {
				p.fail("async arrow function expression", ArrowToken)
				return Node{}
			}
			nodes = append(nodes, p.parseToken())
			p.asyncLevel++
			if p.tt == OpenBraceToken {
				nodes = append(nodes, p.parseBlockStmt("async arrow function expression"))
			} else {
				nodes = append(nodes, p.parseAssignmentExpr())
			}
			p.asyncLevel--
		} else {
			p.fail("async function expression", FunctionToken, IdentifierToken)
			return Node{}
		}
		return Node{ExprGrammar, nodes, 0, nil}
	}

ASSIGNLOOP:
	if p.tt == DeleteToken || p.tt == VoidToken || p.tt == TypeofToken || p.tt == BitNotToken || p.tt == NotToken {
		nodes = append(nodes, p.parseToken())
		goto ASSIGNLOOP
	} else if p.tt == IncrToken {
		nodes = append(nodes, p.parseTokenAs(PreIncrToken))
		goto ASSIGNLOOP
	} else if p.tt == DecrToken {
		nodes = append(nodes, p.parseTokenAs(PreDecrToken))
		goto ASSIGNLOOP
	} else if p.tt == AddToken {
		nodes = append(nodes, p.parseTokenAs(PosToken))
		goto ASSIGNLOOP
	} else if p.tt == SubToken {
		nodes = append(nodes, p.parseTokenAs(NegToken))
		goto ASSIGNLOOP
	} else if p.tt == AwaitToken && p.asyncLevel != 0 {
		// handle AwaitExpression, otherwise await is handled by primary expression
		nodes = append(nodes, p.parseToken())
		if p.tt == ArrowToken {
			nodes[len(nodes)-1].TokenType = IdentifierToken
			goto ASSIGNSWITCH
		}
		goto ASSIGNLOOP
	}
	nodes = p.parseLeftHandSideExpr(nodes)
	if !p.prevLineTerminator {
		if p.tt == IncrToken {
			nodes = append(nodes, p.parseTokenAs(PostIncrToken))
		} else if p.tt == DecrToken {
			nodes = append(nodes, p.parseTokenAs(PostDecrToken))
		}
	}

ASSIGNSWITCH:
	switch p.tt {
	case NullishToken, OrToken, AndToken, BitOrToken, BitXorToken, BitAndToken, EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken, LtToken, GtToken, LtEqToken, GtEqToken, LtLtToken, GtGtToken, GtGtGtToken, AddToken, SubToken, MulToken, DivToken, ModToken, ExpToken, InstanceofToken:
		nodes = append(nodes, p.parseToken())
		goto ASSIGNLOOP
	case InToken:
		if p.inFor {
			break
		}
		nodes = append(nodes, p.parseToken())
		goto ASSIGNLOOP
	case EqToken, MulEqToken, DivEqToken, ModEqToken, ExpEqToken, AddEqToken, SubEqToken, LtLtEqToken, GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitXorEqToken, BitOrEqToken:
		// we allow the left-hand-side to be a full assignment expression instead of a left-hand-side expression, but that's fine
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseAssignmentExpr())
	case QuestionToken:
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseAssignmentExpr())
		if p.tt != ColonToken {
			p.fail("conditional expression", ColonToken)
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseAssignmentExpr())
	case ArrowToken:
		// we allow the start of an arrow function expressions to be anything in a left-hand-side expression, but that should be fine
		if p.prevLineTerminator {
			p.fail("expression")
			return Node{}
		}
		// previous token should be identifier, yield, await, or arrow parameter list (end with CloseParenToken)
		if len(nodes) == 1 && nodes[len(nodes)-1].TokenType == IdentifierToken || nodes[len(nodes)-1].TokenType == YieldToken || nodes[len(nodes)-1].TokenType == AwaitToken {
			nodes[len(nodes)-1].TokenType = IdentifierToken
			nodes[len(nodes)-1] = Node{ParamsGrammar, []Node{Node{BindingGrammar, []Node{nodes[len(nodes)-1]}, 0, nil}}, 0, nil}
		} else if nodes[len(nodes)-1].TokenType == CloseParenToken {
			i := len(nodes) - 2
			for nodes[i].TokenType != OpenParenToken {
				if nodes[i].GrammarType == ExprGrammar {
					nodes[i].GrammarType = BindingGrammar
				} else if nodes[i].TokenType == CommaToken {
					nodes = append(nodes[:i], nodes[i+1:]...)
				}
				i--
			}
			nodes = append(nodes[:i:i], Node{ParamsGrammar, nodes[i+1 : len(nodes)-1], 0, nil})
		} else {
			p.fail("arrow function expression")
			return Node{}
		}
		nodes = append(nodes, p.parseToken())
		if p.tt == OpenBraceToken {
			nodes = append(nodes, p.parseBlockStmt("arrow function expression"))
		} else {
			nodes = append(nodes, p.parseAssignmentExpr())
		}
	}
	return Node{ExprGrammar, nodes, 0, nil}
}

func (p *Parser) parseExpr() Node {
	node := p.parseAssignmentExpr()
	for p.tt == CommaToken {
		node.Nodes = append(node.Nodes, p.parseToken())
		node.Nodes = append(node.Nodes, p.parseAssignmentExpr().Nodes...)
	}
	return node
}

func (p *Parser) parseToken() Node {
	node := Node{TokenGrammar, nil, p.tt, p.data}
	p.next()
	return node
}

func (p *Parser) parseTokenAs(tt TokenType) Node {
	node := p.parseToken()
	node.TokenType = tt
	return node
}
