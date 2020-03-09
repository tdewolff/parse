package js

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/buffer"
)

// TODO: clarify usage of yield, is it a YieldExpression or an Identifier?

type Nodes []interface{ String() string }

type Node struct {
	GrammarType
	Nodes Nodes

	TokenType
	Data []byte
}

var ErrorNode = Node{}

func GrammarNode(gt GrammarType, nodes Nodes) Node {
	return Node{GrammarType: gt, Nodes: nodes}
}

func TokenNode(tt TokenType, data []byte) Node {
	return Node{GrammarType: TokenGrammar, TokenType: tt, Data: data}
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
type GrammarType uint16

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
func Parse(r io.Reader) (AST, error) {
	l := NewLexer(r)
	defer l.Restore()

	p := &Parser{
		l:  l,
		tt: WhitespaceToken, // trick so that next() works
	}

	p.next()
	ast := p.parseModule()

	if p.err == nil {
		p.err = p.l.Err()
	}
	if p.err == io.EOF {
		p.err = nil
	}
	return ast, p.err
}

////////////////////////////////////////////////////////////////

//func (p *Parser) push(node Node) {
//	p.buf = append(p.buf, node)
//}

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

func (p *Parser) parseModule() (ast AST) {
	for {
		switch p.tt {
		case ErrorToken:
			return
		case ImportToken:
			importStmt := p.parseImportStmt()
			ast.List = append(ast.List, &importStmt)
		case ExportToken:
			exportStmt := p.parseExportStmt()
			ast.List = append(ast.List, &exportStmt)
		default:
			ast.List = append(ast.List, p.parseStmt())
		}
	}
}

func (p *Parser) parseStmt() (stmt IStmt) {
	switch p.tt {
	case OpenBraceToken:
		blockStmt := p.parseBlockStmt("block statement")
		stmt = &blockStmt
	case LetToken, ConstToken, VarToken:
		varDecl := p.parseVarDecl()
		stmt = &varDecl
	case ContinueToken, BreakToken:
		tt := p.tt
		p.next()
		var name *Token
		if !p.prevLineTerminator && (p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken && p.asyncLevel == 0) {
			name = &Token{IdentifierToken, p.data}
			p.next()
		}
		stmt = &BranchStmt{tt, name}
	case ReturnToken:
		p.next()
		var value Expr
		if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != LineTerminatorToken && p.tt != ErrorToken {
			value = p.parseExpr()
		}
		stmt = &ReturnStmt{value}
	case IfToken:
		p.next()
		if !p.consume("if statement", OpenParenToken) {
			return
		}
		cond := p.parseExpr()
		if !p.consume("if statement", CloseParenToken) {
			return
		}
		body := p.parseStmt()

		var elseBody IStmt
		if p.tt == ElseToken {
			p.next()
			elseBody = p.parseStmt()
		}
		stmt = &IfStmt{cond, body, elseBody}
	case WithToken:
		p.next()
		if !p.consume("with statement", OpenParenToken) {
			return
		}
		cond := p.parseExpr()
		if !p.consume("with statement", CloseParenToken) {
			return
		}
		stmt = &WithStmt{cond, p.parseStmt()}
	case DoToken:
		stmt = &DoWhileStmt{}
		p.next()
		body := p.parseStmt()
		if p.tt != WhileToken {
			p.fail("do statement", WhileToken)
			return
		}
		p.next()
		if !p.consume("do statement", OpenParenToken) {
			return
		}
		stmt = &DoWhileStmt{p.parseExpr(), body}
		if !p.consume("do statement", CloseParenToken) {
			return
		}
	case WhileToken:
		p.next()
		if !p.consume("while statement", OpenParenToken) {
			return
		}
		cond := p.parseExpr()
		if !p.consume("while statement", CloseParenToken) {
			return
		}
		stmt = &WhileStmt{cond, p.parseStmt()}
	case ForToken:
		p.next()
		await := p.tt == AwaitToken
		if await {
			p.next()
		}
		if !p.consume("for statement", OpenParenToken) {
			return
		}

		var init IExpr
		p.inFor = true
		if p.tt == VarToken || p.tt == LetToken || p.tt == ConstToken {
			varDecl := p.parseVarDecl()
			init = &varDecl
		} else if p.tt != SemicolonToken {
			init = p.parseExpr()
		}
		p.inFor = false

		if p.tt == SemicolonToken {
			var cond, post Expr
			if await {
				p.fail("for statement", OfToken)
				return
			}
			p.next()
			if p.tt != SemicolonToken {
				cond = p.parseExpr()
			}
			if !p.consume("for statement", SemicolonToken) {
				return
			}
			if p.tt != CloseParenToken {
				post = p.parseExpr()
			}
			if !p.consume("for statement", CloseParenToken) {
				return
			}
			stmt = &ForStmt{init, cond, post, p.parseStmt()}
		} else if p.tt == InToken {
			if init == nil || await {
				p.fail("for statement", OfToken)
				return
			}
			p.next()
			value := p.parseExpr()
			if !p.consume("for statement", CloseParenToken) {
				return
			}
			stmt = &ForInStmt{init, value, p.parseStmt()}
		} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("of")) {
			if init == nil {
				p.fail("for statement", OfToken)
				return
			}
			p.next()
			value := p.parseAssignmentExpr()
			if !p.consume("for statement", CloseParenToken) {
				return
			}
			stmt = &ForOfStmt{await, init, value, p.parseStmt()}
		} else {
			p.fail("for statement", InToken, OfToken, SemicolonToken)
			return
		}
	case IdentifierToken, YieldToken, AwaitToken:
		// could be expression or labelled statement, try expression first and convert to labelled statement if possible
		expr := p.parseExpr()
		stmt = &ExprStmt{expr}
		if p.tt == ColonToken && len(expr.List) == 1 && len(expr.List[0].Nodes) == 1 {
			if node, ok := expr.List[0].Nodes[0].(Node); ok && (node.TokenType != AwaitToken || p.asyncLevel == 0) {
				p.next() // colon
				stmt = &LabelledStmt{Token{IdentifierToken, node.Data}, p.parseStmt()}
			}
		}
	case SwitchToken:
		p.next()
		if !p.consume("switch statement", OpenParenToken) {
			return
		}
		init := p.parseExpr()
		if !p.consume("switch statement", CloseParenToken) {
			return
		}

		// case block
		if !p.consume("switch statement", OpenBraceToken) {
			return
		}

		clauses := []CaseClause{}
		for p.tt != ErrorToken {
			if p.tt == CloseBraceToken {
				p.next()
				break
			}

			tt := p.tt
			var list Expr
			if p.tt == CaseToken {
				p.next()
				list = p.parseExpr()
			} else if p.tt == DefaultToken {
				p.next()
			} else {
				p.fail("switch statement", CaseToken, DefaultToken)
				return
			}
			if !p.consume("switch statement", ColonToken) {
				return
			}

			var stmts []IStmt
			for p.tt != CaseToken && p.tt != DefaultToken && p.tt != CloseBraceToken && p.tt != ErrorToken {
				stmts = append(stmts, p.parseStmt())
			}
			clauses = append(clauses, CaseClause{tt, list, stmts})
		}
		stmt = &SwitchStmt{init, clauses}
	case FunctionToken:
		funcDecl := p.parseFuncDecl()
		stmt = &funcDecl
	case AsyncToken: // async function
		p.next()
		if p.tt != FunctionToken {
			p.fail("async function statement", FunctionToken)
			return
		}
		p.asyncLevel++
		funcDecl := p.parseFuncDecl()
		funcDecl.Async = true
		p.asyncLevel--
		stmt = &funcDecl
	case ClassToken:
		classDecl := p.parseClassDecl()
		stmt = &classDecl
	case ThrowToken:
		p.next()
		var value Expr
		if !p.prevLineTerminator {
			value = p.parseExpr()
		}
		stmt = &ThrowStmt{value}
	case TryToken:
		p.next()
		body := p.parseBlockStmt("try statement")
		var binding IBinding
		var catch, finally BlockStmt
		if p.tt == CatchToken {
			p.next()
			if p.tt == OpenParenToken {
				p.next()
				binding = p.parseBinding()
				if p.tt != CloseParenToken {
					p.fail("try statement", CloseParenToken)
					return
				}
				p.next()
			}
			catch = p.parseBlockStmt("try statement")
		}
		if p.tt == FinallyToken {
			p.next()
			finally = p.parseBlockStmt("try statement")
		}
		stmt = &TryStmt{body, binding, catch, finally}
	case DebuggerToken:
		p.next()
		stmt = &DebuggerStmt{}
	case SemicolonToken, ErrorToken:
		stmt = &EmptyStmt{}
	default:
		stmt = &ExprStmt{p.parseExpr()}
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return
}

func (p *Parser) parseVarDecl() (varDecl VarDecl) {
	// assume we're at var, let or const
	varDecl.TokenType = p.tt
	p.next()
	for {
		varDecl.List = append(varDecl.List, p.parseBindingElement())
		if p.tt == CommaToken {
			p.next()
		} else {
			break
		}
	}
	return
}

func (p *Parser) parseBlockStmt(in string) (blockStmt BlockStmt) {
	if p.tt != OpenBraceToken {
		p.fail(in, OpenBraceToken)
		return
	}
	p.next()
	for p.tt != ErrorToken {
		if p.tt == CloseBraceToken {
			break
		}
		blockStmt.List = append(blockStmt.List, p.parseStmt())
	}
	p.next()
	return
}

func (p *Parser) parseImportStmt() (importStmt ImportStmt) {
	// assume we're at import
	p.next()
	if p.tt == StringToken {
		importStmt.Module = p.data
		p.next()
	} else {
		if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			importStmt.Default = p.data
			p.next()
			if p.tt == CommaToken {
				p.next()
			}
		}
		if p.tt == MulToken {
			p.next()
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("as")) {
				p.fail("import statement", AsToken)
				return
			}
			p.next()
			if p.tt != IdentifierToken && p.tt != YieldToken && p.tt != AwaitToken {
				p.fail("import statement", IdentifierToken)
				return
			}
			importStmt.List = []Alias{Alias{[]byte("*"), p.data}}
			p.next()
		} else if p.tt == OpenBraceToken {
			p.next()
			for IsIdentifier(p.tt) {
				var name, binding []byte = nil, p.data
				p.next()
				if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
					p.next()
					if p.tt != IdentifierToken && p.tt != YieldToken && p.tt != AwaitToken {
						p.fail("import statement", IdentifierToken)
						return
					}
					name = binding
					binding = p.data
					p.next()
				}
				importStmt.List = append(importStmt.List, Alias{name, binding})
				if p.tt == CommaToken {
					p.next()
					if p.tt == CloseBraceToken {
						importStmt.List = append(importStmt.List, Alias{})
						break
					}
				}
			}
			if p.tt != CloseBraceToken {
				p.fail("import statement", CloseBraceToken)
				return
			}
			p.next()
		}
		if importStmt.Default == nil && len(importStmt.List) == 0 {
			p.fail("import statement", StringToken, IdentifierToken, MulToken, OpenBraceToken)
			return
		}

		if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("from")) {
			p.fail("import statement", FromToken)
			return
		}
		p.next()
		if p.tt != StringToken {
			p.fail("import statement", StringToken)
			return
		}
		importStmt.Module = p.data
		p.next()
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return
}

func (p *Parser) parseExportStmt() (exportStmt ExportStmt) {
	// assume we're at export
	p.next()
	if p.tt == MulToken || p.tt == OpenBraceToken {
		if p.tt == MulToken {
			p.next()
			if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
				p.next()
				if !IsIdentifier(p.tt) {
					p.fail("export statement", IdentifierToken)
					return
				}
				exportStmt.List = []Alias{Alias{[]byte("*"), p.data}}
				p.next()
			} else {
				exportStmt.List = []Alias{Alias{nil, []byte("*")}}
			}
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("from")) {
				p.fail("export statement", FromToken)
				return
			}
		} else {
			p.next()
			for IsIdentifier(p.tt) {
				var name, binding []byte = nil, p.data
				p.next()
				if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("as")) {
					p.next()
					if !IsIdentifier(p.tt) {
						p.fail("export statement", IdentifierToken)
						return
					}
					name = binding
					binding = p.data
					p.next()
				}
				exportStmt.List = append(exportStmt.List, Alias{name, binding})
				if p.tt == CommaToken {
					p.next()
					if p.tt == CloseBraceToken {
						exportStmt.List = append(exportStmt.List, Alias{})
						break
					}
				}
			}
			if p.tt != CloseBraceToken {
				p.fail("export statement", CloseBraceToken)
				return
			}
			p.next()
		}
		if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("from")) {
			p.next()
			if p.tt != StringToken {
				p.fail("export statement", StringToken)
				return
			}
			exportStmt.Module = p.data
			p.next()
		}
	} else if p.tt == VarToken || p.tt == ConstToken || p.tt == LetToken {
		varDecl := p.parseVarDecl()
		exportStmt.Decl = &varDecl
	} else if p.tt == FunctionToken {
		funcDecl := p.parseFuncDecl()
		exportStmt.Decl = &funcDecl
	} else if p.tt == AsyncToken { // async function
		p.next()
		if p.tt != FunctionToken {
			p.fail("export statement", FunctionToken)
			return
		}
		p.asyncLevel++
		funcDecl := p.parseFuncDecl()
		funcDecl.Async = true
		p.asyncLevel--
		exportStmt.Decl = &funcDecl
	} else if p.tt == ClassToken {
		classDecl := p.parseClassDecl()
		exportStmt.Decl = &classDecl
	} else if p.tt == DefaultToken {
		exportStmt.Default = true
		p.next()
		if p.tt == FunctionToken {
			funcDecl := p.parseFuncDecl()
			exportStmt.Decl = &funcDecl
		} else if p.tt == AsyncToken { // async function
			p.next()
			if p.tt != FunctionToken {
				p.fail("export statement", FunctionToken)
				return
			}
			p.asyncLevel++
			funcDecl := p.parseFuncDecl()
			funcDecl.Async = true
			p.asyncLevel--
			exportStmt.Decl = &funcDecl
		} else if p.tt == ClassToken {
			classDecl := p.parseClassDecl()
			exportStmt.Decl = &classDecl
		} else {
			assignExpr := p.parseAssignmentExpr()
			exportStmt.Decl = &Expr{[]AssignExpr{assignExpr}}
		}
	} else {
		p.fail("export statement", MulToken, OpenBraceToken, VarToken, LetToken, ConstToken, FunctionToken, AsyncToken, ClassToken, DefaultToken)
		return
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return
}

func (p *Parser) parseFuncDecl() (funcDecl FuncDecl) {
	// assume we're at function
	p.next()
	if p.tt == MulToken {
		funcDecl.Generator = true
		p.next()
	}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		funcDecl.Name = p.data
		p.next()
	}
	funcDecl.Params = p.parseFuncParams("function declaration")
	funcDecl.Body = p.parseBlockStmt("function declaration")
	return
}

func (p *Parser) parseFuncParams(in string) (params Params) {
	if !p.consume(in, OpenParenToken) {
		return
	}

	for p.tt != CloseParenToken {
		// binding rest element
		if p.tt == EllipsisToken {
			p.next()
			rest := p.parseBindingElement()
			params.Rest = &rest
			break
		}

		params.List = append(params.List, p.parseBindingElement())

		if p.tt == CommaToken {
			p.next()
		} else if p.tt == CloseParenToken {
			break
		} else {
			p.fail(in, CommaToken, CloseParenToken)
			return
		}
	}
	if !p.consume(in, CloseParenToken) {
		return
	}
	return
}

func (p *Parser) parseClassDecl() (classDecl ClassDecl) {
	// assume we're at class
	p.next()
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		classDecl.Name = p.data
		p.next()
	}
	if p.tt == ExtendsToken {
		p.next()
		leftHandSideExpr := Expr{[]AssignExpr{AssignExpr{p.parseLeftHandSideExpr(nil)}}}
		classDecl.Extends = &leftHandSideExpr
	}

	if !p.consume("class statement", OpenBraceToken) {
		return
	}
	for p.tt != ErrorToken {
		if p.tt == SemicolonToken {
			p.next()
			continue
		} else if p.tt == CloseBraceToken {
			break
		}
		classDecl.Methods = append(classDecl.Methods, p.parseMethod())
	}
	if !p.consume("class statement", CloseBraceToken) {
		return
	}
	return
}

func (p *Parser) parseMethod() (method Method) {
	if p.tt == StaticToken {
		method.Static = true
		p.next()
	}
	if p.tt == AsyncToken || p.tt == MulToken {
		if p.tt == AsyncToken {
			method.Async = true
			p.next()
		}
		if p.tt == MulToken {
			method.Generator = true
			p.next()
		}
	} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("get")) {
		method.Get = true
		p.next()
	} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("set")) {
		method.Set = true
		p.next()
	}

	if IsIdentifier(p.tt) {
		method.Name = PropertyName{Token{IdentifierToken, p.data}, nil}
		p.next()
	} else if p.tt == StringToken || IsNumeric(p.tt) {
		method.Name = PropertyName{Token{p.tt, p.data}, nil}
		p.next()
	} else if p.tt == OpenBracketToken {
		p.next()
		assignExpr := p.parseAssignmentExpr()
		method.Name = PropertyName{Token{}, &assignExpr}
		if p.tt != CloseBracketToken {
			p.fail("method definition", CloseBracketToken)
			return
		}
		p.next()
	} else {
		p.fail("method definition", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
		return
	}
	if method.Async {
		p.asyncLevel++
	}
	method.Params = p.parseFuncParams("method definition")
	method.Body = p.parseBlockStmt("method definition")
	if method.Async {
		p.asyncLevel--
	}
	return
}

func (p *Parser) parseBindingElement() (bindingElement BindingElement) {
	// binding element
	bindingElement.Binding = p.parseBinding()
	if p.tt == EqToken {
		p.next()
		assignExpr := p.parseAssignmentExpr()
		bindingElement.Default = &assignExpr
	}
	return
}

func (p *Parser) parseBinding() (binding IBinding) {
	// binding identifier or binding pattern
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		binding = &BindingName{p.data}
		p.next()
	} else if p.tt == OpenBracketToken {
		p.next()
		array := BindingArray{}
		if p.tt == CommaToken {
			array.List = append(array.List, BindingElement{})
		}
		for p.tt != CloseBracketToken {
			// elision
			for p.tt == CommaToken {
				p.next()
				if p.tt == CommaToken || p.tt == CloseBracketToken {
					array.List = append(array.List, BindingElement{})
				}
			}
			// binding rest element
			if p.tt == EllipsisToken {
				p.next()
				array.Rest = p.parseBinding()
				if p.tt != CloseBracketToken {
					p.fail("array binding pattern", CloseBracketToken)
					return
				}
				break
			}
			if p.tt == CloseBracketToken {
				break
			}

			array.List = append(array.List, p.parseBindingElement())

			if p.tt != CommaToken && p.tt != CloseBracketToken {
				p.fail("array binding pattern", CommaToken, CloseBracketToken)
				return
			}
		}
		p.next()
		binding = &array
	} else if p.tt == OpenBraceToken {
		p.next()
		object := BindingObject{}
		for p.tt != CloseBraceToken {
			// binding rest property
			if p.tt == EllipsisToken {
				p.next()
				if p.tt != IdentifierToken && p.tt != YieldToken && p.tt != AwaitToken {
					p.fail("object binding pattern", IdentifierToken)
					return
				}
				object.Rest = &BindingName{p.data}
				p.next()
				if p.tt != CloseBraceToken {
					p.fail("object binding pattern", CloseBraceToken)
					return
				}
				break
			}

			item := BindingObjectItem{}
			if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
				ident := Token{p.tt, p.data}
				p.next()
				if p.tt == ColonToken {
					// property name + : + binding element
					p.next()
					item.Key = &PropertyName{Name: ident}
					item.Value = p.parseBindingElement()
				} else {
					// single name binding
					item.Value.Binding = &BindingName{ident.Data}
					if p.tt == EqToken {
						p.next()
						assignExpr := p.parseAssignmentExpr()
						item.Value.Default = &assignExpr
					}
				}
			} else if IsIdentifier(p.tt) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
				// property name + : + binding element
				if p.tt == OpenBracketToken {
					p.next()
					assignExpr := p.parseAssignmentExpr()
					item.Key = &PropertyName{ComputedName: &assignExpr}
					if p.tt != CloseBracketToken {
						p.fail("object binding pattern", CloseBracketToken)
						return
					}
					p.next()
				} else if IsIdentifier(p.tt) {
					item.Key = &PropertyName{Name: Token{IdentifierToken, p.data}}
					p.next()
				} else {
					item.Key = &PropertyName{Name: Token{p.tt, p.data}}
					p.next()
				}
				if p.tt != ColonToken {
					p.fail("object binding pattern", ColonToken)
					return
				}
				p.next()
				item.Value = p.parseBindingElement()
			} else {
				p.fail("object binding pattern", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
				return
			}
			object.List = append(object.List, item)

			if p.tt == CommaToken {
				p.next()
			} else if p.tt != CloseBraceToken {
				p.fail("object binding pattern", CommaToken, CloseBraceToken)
				return
			}
		}
		p.next()
		binding = &object
	} else {
		p.fail("binding")
		return
	}
	return
}

func (p *Parser) parseObjectLiteral(nodes Nodes) Nodes {
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
			property := Nodes{}
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

			last := Node{}
			if 0 < len(property) {
				last, _ = property[len(property)-1].(Node)
			}
			if (p.tt == EqToken || p.tt == CommaToken || p.tt == CloseBraceToken) && len(property) == 1 && (last.TokenType == IdentifierToken || last.TokenType == YieldToken || last.TokenType == AwaitToken && p.asyncLevel == 0) {
				nodes = append(nodes, TokenNode(IdentifierToken, last.Data))
				if p.tt == EqToken {
					nodes = append(nodes, p.parseToken())
					nodes = append(nodes, p.parseAssignmentExpr())
				}
			} else if 0 < len(property) && IsIdentifier(last.TokenType) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
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
					funcParams := p.parseFuncParams("method definition")
					property = append(property, funcParams)
					property = append(property, p.parseBlockStmt("method definition"))
					if async {
						p.asyncLevel--
					}
					nodes = append(nodes, GrammarNode(MethodGrammar, property))
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

func (p *Parser) parseTemplateLiteral(nodes Nodes) Nodes {
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

func (p *Parser) parsePrimaryExpr(nodes Nodes) Nodes {
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
		nodes = append(nodes, p.parseClassDecl())
	case FunctionToken:
		nodes = append(nodes, p.parseFuncDecl())
	case AsyncToken:
		// async function
		async := p.parseToken()
		if !p.prevLineTerminator {
			if p.tt == FunctionToken {
				p.asyncLevel++
				funcDecl := p.parseFuncDecl()
				funcDecl.Async = true
				nodes = append(nodes, funcDecl)
				p.asyncLevel--
			} else {
				p.fail("async function expression", FunctionToken)
				return nil
			}
		} else {
			nodes = append(nodes, async)
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

func (p *Parser) parseLeftHandSideExprEnd(nodes Nodes) Nodes {
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

func (p *Parser) parseLeftHandSideExpr(nodes Nodes) Nodes {
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
			return nil
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

func (p *Parser) parseAssignmentExpr() AssignExpr {
	nodes := Nodes{}
	if p.tt == YieldToken {
		yield := p.parseToken()
		if p.tt == ArrowToken {
			nodes = append(nodes, GrammarNode(ParamsGrammar, Nodes{GrammarNode(BindingGrammar, Nodes{yield})}))
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
		return AssignExpr{nodes}
	} else if p.tt == AsyncToken {
		async := p.parseToken()
		if p.prevLineTerminator {
			p.fail("async function expression")
			return AssignExpr{nil}
		}
		if p.tt == FunctionToken {
			// primary expression
			p.asyncLevel++
			funcDecl := p.parseFuncDecl()
			funcDecl.Async = true
			nodes = append(nodes, funcDecl)
			p.asyncLevel--
		} else if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			nodes = append(nodes, async)
			nodes = append(nodes, GrammarNode(ParamsGrammar, Nodes{GrammarNode(BindingGrammar, Nodes{p.parseTokenAs(IdentifierToken)})}))
			if p.tt != ArrowToken {
				p.fail("async arrow function expression", ArrowToken)
				return AssignExpr{nil}
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
			return AssignExpr{nil}
		}
		return AssignExpr{nodes}
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
		await := p.parseToken()
		if p.tt == ArrowToken {
			nodes = append(nodes, Node{GrammarType: TokenGrammar, TokenType: IdentifierToken, Data: await.Data})
			goto ASSIGNSWITCH
		}
		nodes = append(nodes, await)
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
			return AssignExpr{nil}
		}
		nodes = append(nodes, p.parseToken())
		nodes = append(nodes, p.parseAssignmentExpr())
	case ArrowToken:
		// we allow the start of an arrow function expressions to be anything in a left-hand-side expression, but that should be fine
		if p.prevLineTerminator {
			p.fail("expression")
			return AssignExpr{nil}
		}
		// previous token should be identifier, yield, await, or arrow parameter list (end with CloseParenToken)
		last, _ := nodes[len(nodes)-1].(Node)
		if len(nodes) == 1 && last.TokenType == IdentifierToken || last.TokenType == YieldToken || last.TokenType == AwaitToken {
			ident := TokenNode(IdentifierToken, last.Data)
			nodes = append(nodes[:len(nodes)-1], GrammarNode(ParamsGrammar, Nodes{GrammarNode(BindingGrammar, Nodes{ident})}))
		} else if last.TokenType == CloseParenToken {
			i := len(nodes) - 2
			for {
				n, _ := nodes[i].(Node)
				if n.TokenType == OpenParenToken {
					break
				}
				if n.GrammarType == ExprGrammar {
					n.GrammarType = BindingGrammar
					nodes[i] = n
				} else if n.TokenType == CommaToken {
					nodes = append(nodes[:i], nodes[i+1:]...)
				}
				i--
			}
			nodes = append(nodes[:i:i], GrammarNode(ParamsGrammar, nodes[i+1:len(nodes)-1]))
			fmt.Println(nodes)
		} else {
			p.fail("arrow function expression")
			return AssignExpr{nil}
		}
		nodes = append(nodes, p.parseToken())
		if p.tt == OpenBraceToken {
			nodes = append(nodes, p.parseBlockStmt("arrow function expression"))
		} else {
			nodes = append(nodes, p.parseAssignmentExpr())
		}
	}
	return AssignExpr{nodes}
}

func (p *Parser) parseExpr() (expr Expr) {
	assignExpr := p.parseAssignmentExpr()
	expr.List = append(expr.List, assignExpr)
	for p.tt == CommaToken {
		p.next()
		assignExpr = p.parseAssignmentExpr()
		expr.List = append(expr.List, assignExpr)
	}
	return
}

func (p *Parser) parseToken() Node {
	node := TokenNode(p.tt, p.data)
	p.next()
	return node
}

func (p *Parser) parseTokenAs(tt TokenType) Node {
	if len(p.data) == 0 {
		fmt.Println(p.tt, tt)
	}
	node := TokenNode(tt, p.data)
	p.next()
	return node
}
