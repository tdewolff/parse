package js

import (
	"bytes"
	"io"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/buffer"
)

type BindingScope int

const (
	FunctionBindingScope BindingScope = iota
	BlockBindingScope
)

// Parser is the state for the parser.
type Parser struct {
	l   *Lexer
	src Src
	err error

	tt                 TokenType
	data               []byte
	prevLineTerminator bool
	inFor              bool

	functionState
	unbound map[string]int
}

// Parse returns a JS AST tree of.
func Parse(r *parse.Input) (AST, error) {
	p := &Parser{
		l:       NewLexer(r),
		tt:      WhitespaceToken, // trick so that next() works
		unbound: map[string]int{},
	}
	p.src = Src(p.l.r.Bytes())

	p.next()
	ast := p.parseModule()
	ast.Src = p.src
	ast.UnboundVars = make([]string, 0, len(p.unbound))
	for name, n := range p.unbound {
		if 0 < n {
			ast.UnboundVars = append(ast.UnboundVars, name)
		}
	}

	if p.err == nil {
		p.err = p.l.Err()
	}
	if p.err == io.EOF {
		p.err = nil
	}
	return ast, p.err
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
		msg := "unexpected"
		if 0 < len(expected) {
			msg = "expected"
			for i, tt := range expected[:len(expected)-1] {
				if 0 < i {
					msg += ","
				}
				msg += " '" + tt.String() + "'"
			}
			if 2 < len(expected) {
				msg += ", or"
			} else if 1 < len(expected) {
				msg += " or"
			}
			msg += " '" + expected[len(expected)-1].String() + "' instead of"
		}

		if p.tt == ErrorToken {
			if p.l.Err() == io.EOF {
				msg += " EOF"
			} else if lexerErr, ok := p.l.Err().(*parse.Error); ok {
				msg = lexerErr.Message
			} else {
				msg = p.l.Err().Error()
			}
		} else {
			msg += " '" + string(p.data) + "'"
		}

		offset := p.l.r.Offset() - len(p.data)
		p.err = parse.NewError(buffer.NewReader(p.l.r.Bytes()), offset, "%s in %s", msg, in)
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

type functionState struct {
	functionScope *Scope
	blockScope    *Scope
	async         bool
	generator     bool
}

func (p *Parser) enterFunctionScope(scope *Scope, async, generator bool) (state functionState) {
	*scope = Scope{p.blockScope, map[string]bool{}}
	state = p.functionState
	p.functionScope = scope
	p.blockScope = scope
	p.async = async
	p.generator = generator
	return
}

func (p *Parser) restoreFunctionScope(state functionState) {
	p.functionState = state
}

func (p *Parser) enterBlockScope(scope *Scope) (blockScope *Scope) {
	*scope = Scope{p.blockScope, map[string]bool{}}
	blockScope = p.blockScope
	p.blockScope = scope
	return
}

func (p *Parser) restoreBlockScope(blockScope *Scope) {
	p.blockScope = blockScope
}

func (p *Parser) varUse(name []byte) {
	if !p.blockScope.use(name) {
		p.unbound[string(name)]++
	}
	if !p.functionScope.Bound[string(name)] && (p.functionScope == p.blockScope || !p.blockScope.Bound[string(name)]) {
		p.functionScope.Unbound[string(name)]++
	}
}

func (p *Parser) varDefine(name []byte, bindingScope BindingScope) {
	if bindingScope == FunctionBindingScope {
		p.functionScope.Bound[string(name)] = true
	} else {
		p.blockScope.Bound[string(name)] = true
	}
}

func (p *Parser) parseModule() (ast AST) {
	p.enterFunctionScope(&ast.Scope, false, false)
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
		blockStmt := p.parseBlockStmt("block statement", true)
		stmt = &blockStmt
	case LetToken, ConstToken, VarToken:
		varDecl := p.parseVarDecl()
		stmt = &varDecl
	case IfToken:
		p.next()
		if !p.consume("if statement", OpenParenToken) {
			return
		}
		cond := p.parseExpression(OpExpr)
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
	case ContinueToken, BreakToken:
		tt := p.tt
		p.next()
		var name Ref
		if !p.prevLineTerminator && p.isIdentifierReference(p.tt) {
			name = p.src.Ref(p.tt, p.data)
			p.next()
		}
		stmt = &BranchStmt{tt, name}
	case ReturnToken:
		p.next()
		var value IExpr
		if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != CloseBraceToken && p.tt != ErrorToken {
			value = p.parseExpression(OpExpr)
		}
		stmt = &ReturnStmt{value}
	case WithToken:
		p.next()
		if !p.consume("with statement", OpenParenToken) {
			return
		}
		cond := p.parseExpression(OpExpr)
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
		stmt = &DoWhileStmt{p.parseExpression(OpExpr), body}
		if !p.consume("do statement", CloseParenToken) {
			return
		}
	case WhileToken:
		p.next()
		if !p.consume("while statement", OpenParenToken) {
			return
		}
		cond := p.parseExpression(OpExpr)
		if !p.consume("while statement", CloseParenToken) {
			return
		}
		stmt = &WhileStmt{cond, p.parseStmt()}
	case ForToken:
		p.next()
		await := p.async && p.tt == AwaitToken
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
			init = p.parseExpression(OpExpr)
		}
		p.inFor = false

		if p.tt == SemicolonToken {
			var cond, post IExpr
			if await {
				p.fail("for statement", OfToken)
				return
			}
			p.next()
			if p.tt != SemicolonToken {
				cond = p.parseExpression(OpExpr)
			}
			if !p.consume("for statement", SemicolonToken) {
				return
			}
			if p.tt != CloseParenToken {
				post = p.parseExpression(OpExpr)
			}
			if !p.consume("for statement", CloseParenToken) {
				return
			}
			stmt = &ForStmt{init, cond, post, p.parseStmt()}
		} else if p.tt == InToken {
			if await {
				p.fail("for statement", OfToken)
				return
			}
			p.next()
			value := p.parseExpression(OpExpr)
			if !p.consume("for statement", CloseParenToken) {
				return
			}
			stmt = &ForInStmt{init, value, p.parseStmt()}
		} else if p.tt == IdentifierToken && bytes.Equal(p.data, []byte("of")) {
			p.next()
			value := p.parseExpression(OpAssign)
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
		expr := p.parseExpression(OpExpr)
		stmt = &ExprStmt{expr}
		lit, ok := expr.(*LiteralExpr)
		if p.tt == ColonToken && ok && p.isIdentifierReference(lit.TokenType) {
			p.next() // colon
			lit.TokenType = IdentifierToken
			stmt = &LabelledStmt{Ref(*lit), p.parseStmt()}
		} else if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != CloseBraceToken && p.tt != ErrorToken {
			p.fail("expression")
		}
	case SwitchToken:
		p.next()
		if !p.consume("switch statement", OpenParenToken) {
			return
		}
		init := p.parseExpression(OpExpr)
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
			var list IExpr
			if p.tt == CaseToken {
				p.next()
				list = p.parseExpression(OpExpr)
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
			if p.tt == OpenParenToken || p.tt == IdentifierToken || !p.generator && p.tt == YieldToken {
				arrowFuncDecl := p.parseAsyncArrowFunc()
				stmt = &ExprStmt{&arrowFuncDecl}
				break
			}
			p.fail("function statement", FunctionToken)
			return
		}
		funcDecl := p.parseAsyncFuncDecl()
		stmt = &funcDecl
	case ClassToken:
		classDecl := p.parseClassDecl()
		stmt = &classDecl
	case ThrowToken:
		p.next()
		var value IExpr
		if !p.prevLineTerminator {
			value = p.parseExpression(OpExpr)
		}
		stmt = &ThrowStmt{value}
	case TryToken:
		p.next()
		body := p.parseBlockStmt("try statement", true)
		var binding IBinding
		var catch, finally BlockStmt
		if p.tt == CatchToken {
			p.next()

			var scope Scope
			parentBlockScope := p.enterBlockScope(&scope)
			if p.tt == OpenParenToken {
				p.next()
				binding = p.parseBinding(BlockBindingScope) // local to block scope of catch
				if p.tt != CloseParenToken {
					p.fail("try statement", CloseParenToken)
					return
				}
				p.next()
			}
			catch = p.parseBlockStmt("try statement", false)
			catch.Scope = scope
			p.restoreBlockScope(parentBlockScope)
		}
		if p.tt == FinallyToken {
			p.next()
			finally = p.parseBlockStmt("try statement", true)
		}
		stmt = &TryStmt{body, binding, catch, finally}
	case DebuggerToken:
		p.next()
		stmt = &DebuggerStmt{}
	case SemicolonToken, ErrorToken:
		stmt = &EmptyStmt{}
	default:
		stmt = &ExprStmt{p.parseExpression(OpExpr)}
		if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != CloseBraceToken && p.tt != ErrorToken {
			p.fail("expression")
		}
	}
	if p.tt == SemicolonToken {
		p.next()
	}
	return
}

func (p *Parser) parseBlockStmt(in string, enterScope bool) (blockStmt BlockStmt) {
	if p.tt != OpenBraceToken {
		p.fail(in, OpenBraceToken)
		return
	}
	p.next()
	var parentBlockScope *Scope
	if enterScope {
		parentBlockScope = p.enterBlockScope(&blockStmt.Scope)
	}
	for p.tt != ErrorToken {
		if p.tt == CloseBraceToken {
			break
		}
		blockStmt.List = append(blockStmt.List, p.parseStmt())
	}
	if enterScope {
		p.restoreBlockScope(parentBlockScope)
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
		funcDecl := p.parseAsyncFuncDecl()
		exportStmt.Decl = &funcDecl
	} else if p.tt == ClassToken {
		classDecl := p.parseClassDecl()
		exportStmt.Decl = &classDecl
	} else if p.tt == DefaultToken {
		exportStmt.Default = true
		p.next()
		if p.tt == FunctionToken {
			funcDecl := p.parseFuncExpr()
			exportStmt.Decl = &funcDecl
		} else if p.tt == AsyncToken { // async function or async arrow function
			p.next()
			if p.tt != FunctionToken {
				if p.tt == OpenParenToken || p.tt == IdentifierToken || !p.generator && p.tt == YieldToken {
					arrowFuncDecl := p.parseAsyncArrowFunc()
					exportStmt.Decl = &arrowFuncDecl
					return
				}
				p.fail("export statement", FunctionToken)
				return
			}
			funcDecl := p.parseAsyncFuncExpr()
			exportStmt.Decl = &funcDecl
		} else if p.tt == ClassToken {
			classDecl := p.parseClassDecl()
			exportStmt.Decl = &classDecl
		} else {
			exportStmt.Decl = p.parseExpression(OpAssign)
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

func (p *Parser) parseVarDecl() (varDecl VarDecl) {
	// assume we're at var, let or const
	varDecl.TokenType = p.tt
	bindingScope := BlockBindingScope
	if p.tt == VarToken {
		bindingScope = FunctionBindingScope
	}
	p.next()
	for {
		varDecl.List = append(varDecl.List, p.parseBindingElement(bindingScope))
		if p.tt == CommaToken {
			p.next()
		} else {
			break
		}
	}
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
			params.Rest = p.parseBinding(FunctionBindingScope)
			break
		}

		params.List = append(params.List, p.parseBindingElement(FunctionBindingScope))

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

func (p *Parser) parseFuncDecl() (funcDecl FuncDecl) {
	return p.parseAnyFuncDecl(false, false)
}

func (p *Parser) parseAsyncFuncDecl() (funcDecl FuncDecl) {
	return p.parseAnyFuncDecl(true, false)
}

func (p *Parser) parseFuncExpr() (funcDecl FuncDecl) {
	return p.parseAnyFuncDecl(false, true)
}

func (p *Parser) parseAsyncFuncExpr() (funcDecl FuncDecl) {
	return p.parseAnyFuncDecl(true, true)
}

func (p *Parser) parseAnyFuncDecl(async, inExpr bool) (funcDecl FuncDecl) {
	// assume we're at function
	p.next()
	funcDecl.Async = async
	funcDecl.Generator = p.tt == MulToken
	if funcDecl.Generator {
		p.next()
	}
	if inExpr && (p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken) || !inExpr && p.isIdentifierReference(p.tt) {
		if !inExpr {
			p.varDefine(p.data, FunctionBindingScope)
		}
		funcDecl.Name = p.data
		p.next()
	} else if p.tt != OpenParenToken {
		p.fail("function declaration", IdentifierToken, OpenParenToken)
		return
	}
	parent := p.enterFunctionScope(&funcDecl.Scope, funcDecl.Async, funcDecl.Generator)
	if inExpr && funcDecl.Name != nil {
		p.varDefine(funcDecl.Name, FunctionBindingScope)
	}
	funcDecl.Params = p.parseFuncParams("function declaration")
	funcDecl.Body = p.parseBlockStmt("function declaration", false)
	p.restoreFunctionScope(parent)
	return
}

func (p *Parser) parseClassDecl() (classDecl ClassDecl) {
	// assume we're at class
	p.next()
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		classDecl.Name = p.data
		p.varDefine(p.data, BlockBindingScope)
		p.next()
	}
	if p.tt == ExtendsToken {
		p.next()
		classDecl.Extends = p.parseExpression(OpLHS)
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

func (p *Parser) parseMethod() (method MethodDecl) {
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
		method.Name = PropertyName{LiteralExpr(p.src.Ref(p.tt, p.data)), nil}
		p.next()
	} else if p.tt == StringToken || IsNumeric(p.tt) {
		method.Name = PropertyName{LiteralExpr(p.src.Ref(p.tt, p.data)), nil}
		p.next()
	} else if p.tt == OpenBracketToken {
		p.next()
		method.Name = PropertyName{LiteralExpr{}, p.parseExpression(OpAssign)}
		if p.tt != CloseBracketToken {
			p.fail("method definition", CloseBracketToken)
			return
		}
		p.next()
	} else {
		p.fail("method definition", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
		return
	}
	parentAsync, parentGenerator := p.async, p.generator
	p.async, p.generator = method.Async, method.Generator
	method.Params = p.parseFuncParams("method definition")
	method.Body = p.parseBlockStmt("method definition", false)
	p.async, p.generator = parentAsync, parentGenerator
	return
}

func (p *Parser) parseBindingElement(bindingScope BindingScope) (bindingElement BindingElement) {
	// binding element
	bindingElement.Binding = p.parseBinding(bindingScope)
	if p.tt == EqToken {
		p.next()
		bindingElement.Default = p.parseExpression(OpAssign)
	}
	return
}

func (p *Parser) parseBinding(bindingScope BindingScope) (binding IBinding) {
	// binding identifier or binding pattern
	if p.tt == IdentifierToken || !p.generator && p.tt == YieldToken || !p.async && p.tt == AwaitToken {
		bindingName := BindingName(p.src.Ref(p.tt, p.data))
		binding = &bindingName
		p.varDefine(p.data, bindingScope)
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
				if p.tt == CommaToken {
					array.List = append(array.List, BindingElement{})
				}
			}
			// binding rest element
			if p.tt == EllipsisToken {
				p.next()
				array.Rest = p.parseBinding(bindingScope)
				if p.tt != CloseBracketToken {
					p.fail("array binding pattern", CloseBracketToken)
					return
				}
				break
			} else if p.tt == CloseBracketToken {
				break
			}

			array.List = append(array.List, p.parseBindingElement(bindingScope))

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
				if p.tt != IdentifierToken && (p.generator || p.tt != YieldToken) && (p.async || p.tt != AwaitToken) {
					p.fail("object binding pattern", IdentifierToken)
					return
				}
				object.Rest = BindingName(p.src.Ref(p.tt, p.data))
				p.varDefine(p.data, bindingScope)
				p.next()
				if p.tt != CloseBraceToken {
					p.fail("object binding pattern", CloseBraceToken)
					return
				}
				break
			}

			item := BindingObjectItem{}
			if p.tt == IdentifierToken || !p.generator && p.tt == YieldToken || !p.async && p.tt == AwaitToken {
				lit := LiteralExpr(p.src.Ref(p.tt, p.data))
				p.next()
				if p.tt == ColonToken {
					// property name + : + binding element
					p.next()
					item.Key = &PropertyName{Literal: lit}
					item.Value = p.parseBindingElement(bindingScope)
				} else {
					// single name binding
					item.Value.Binding = (*BindingName)(&lit)
					p.varDefine(lit.Data(p.src), bindingScope)
					if p.tt == EqToken {
						p.next()
						item.Value.Default = p.parseExpression(OpAssign)
					}
				}
			} else if IsIdentifier(p.tt) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
				// property name + : + binding element
				if p.tt == OpenBracketToken {
					p.next()
					item.Key = &PropertyName{Computed: p.parseExpression(OpAssign)}
					if p.tt != CloseBracketToken {
						p.fail("object binding pattern", CloseBracketToken)
						return
					}
					p.next()
				} else if IsIdentifier(p.tt) {
					item.Key = &PropertyName{Literal: LiteralExpr(p.src.Ref(IdentifierToken, p.data))}
					p.next()
				} else {
					item.Key = &PropertyName{Literal: LiteralExpr(p.src.Ref(p.tt, p.data))}
					p.next()
				}
				if p.tt != ColonToken {
					p.fail("object binding pattern", ColonToken)
					return
				}
				p.next()
				item.Value = p.parseBindingElement(bindingScope)
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

func (p *Parser) parseObjectLiteral() (object ObjectExpr) {
	// assume we're on {
	p.next()
	for p.tt != CloseBraceToken && p.tt != ErrorToken {
		property := Property{}
		if p.tt == EllipsisToken {
			p.next()
			property.Spread = true
			property.Value = p.parseExpression(OpAssign)
		} else {
			// try to parse as MethodDefinition, otherwise fall back to PropertyName:AssignExpr or IdentifierReference
			method := MethodDecl{}
			data := p.data
			if p.tt == MulToken {
				p.next()
				method.Generator = true
			} else if p.tt == AsyncToken {
				p.next()
				if !p.prevLineTerminator {
					method.Async = true
					if p.tt == MulToken {
						p.next()
						method.Generator = true
					}
				} else {
					method.Name.Literal = LiteralExpr(p.src.Ref(IdentifierToken, data))
				}
			} else if p.tt == IdentifierToken && len(p.data) == 3 {
				if bytes.Equal(p.data, []byte("get")) {
					p.next()
					method.Get = true
				} else if bytes.Equal(p.data, []byte("set")) {
					p.next()
					method.Set = true
				}
			}

			// PropertyName
			if method.Name.Literal.TokenType == ErrorToken { // did not parse: async [LT]
				if IsIdentifier(p.tt) {
					method.Name.Literal = LiteralExpr(p.src.Ref(IdentifierToken, p.data))
					p.next()
				} else if p.tt == StringToken || IsNumeric(p.tt) {
					method.Name.Literal = LiteralExpr(p.src.Ref(p.tt, p.data))
					p.next()
				} else if p.tt == OpenBracketToken {
					p.next()
					method.Name.Computed = p.parseExpression(OpAssign)
					if p.tt != CloseBracketToken {
						p.fail("object literal", CloseBracketToken)
						return
					}
					p.next()
				} else if !method.Generator && (method.Async || method.Get || method.Set) {
					// interpret async, get, or set as PropertyName instead of method keyword
					method.Async = false
					method.Get = false
					method.Set = false
					method.Name.Literal = LiteralExpr(p.src.Ref(IdentifierToken, data))
				} else {
					p.fail("object literal", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
					return
				}
			}

			if p.tt == OpenParenToken {
				// MethodDefinition
				parentAsync, parentGenerator := p.async, p.generator
				p.async, p.generator = method.Async, method.Generator
				method.Params = p.parseFuncParams("method definition")
				method.Body = p.parseBlockStmt("method definition", false)
				p.async, p.generator = parentAsync, parentGenerator
				property.Value = &method
			} else if p.tt == ColonToken {
				// PropertyName : AssignmentExpression
				p.next()
				property.Key = &method.Name
				property.Value = p.parseExpression(OpAssign)
			} else if !p.isIdentifierReference(method.Name.Literal.TokenType) {
				p.fail("object literal", ColonToken, OpenParenToken)
				return
			} else {
				// IdentifierReference (= AssignmentExpression)?
				property.Value = &method.Name.Literal
				p.varUse(method.Name.Literal.Data(p.src))
				if p.tt == EqToken {
					p.next()
					property.Init = p.parseExpression(OpAssign)
				}
			}
		}
		object.List = append(object.List, property)
		if p.tt != CloseBraceToken && !p.consume("object literal", CommaToken) {
			return
		}
	}
	if p.tt == CloseBraceToken {
		p.next()
	}
	return
}

func (p *Parser) parseTemplateLiteral() (template TemplateExpr) {
	// assume we're on 'Template' or 'TemplateStart'
	for p.tt == TemplateStartToken || p.tt == TemplateMiddleToken {
		tpl := p.data
		p.next()
		template.List = append(template.List, TemplatePart{tpl, p.parseExpression(OpExpr)})
		if p.tt == TemplateEndToken {
			break
		} else {
			p.fail("template literal", TemplateToken)
			return
		}
	}
	template.Tail = p.data
	p.next()
	return
}

func (p *Parser) parseArgs() (args Arguments) {
	// assume we're on (
	p.next()
	args.List = make([]IExpr, 0, 4)
	for {
		if p.tt == EllipsisToken {
			p.next()
			args.Rest = p.parseExpression(OpAssign)
			if p.tt == CommaToken {
				p.next()
			}
			break
		}

		if p.tt == CloseParenToken || p.tt == ErrorToken {
			break
		}
		args.List = append(args.List, p.parseExpression(OpAssign))
		if p.tt == CommaToken {
			p.next()
		}
	}
	p.consume("arguments", CloseParenToken)
	return
}

func (p *Parser) parseAsyncArrowFuncParams() Params {
	if p.tt == IdentifierToken || !p.generator && p.tt == YieldToken {
		tt := p.tt
		data := p.data
		p.next()
		bindingName := BindingName(p.src.Ref(tt, data))
		return Params{List: []BindingElement{{Binding: &bindingName}}}
	} else if p.tt == AwaitToken {
		p.fail("arrow function")
	}
	return p.parseFuncParams("arrow function")
}

func (p *Parser) parseArrowFunc(params Params) (arrowFunc ArrowFunc) {
	// function scoping handled in parenthesized expression
	return p.parseAnyArrowFunc(false, params)
}

func (p *Parser) parseAsyncArrowFunc() (arrowFunc ArrowFunc) {
	parent := p.enterFunctionScope(&arrowFunc.Scope, true, false)
	params := p.parseAsyncArrowFuncParams()
	arrowFunc = p.parseAnyArrowFunc(true, params)
	p.restoreFunctionScope(parent)
	return
}

func (p *Parser) parseAnyArrowFunc(async bool, params Params) (arrowFunc ArrowFunc) {
	if p.tt != ArrowToken {
		p.fail("arrow function", ArrowToken)
		return
	} else if p.prevLineTerminator {
		p.fail("expression")
		return
	}

	arrowFunc.Async = async
	arrowFunc.Params = params
	p.next()
	if p.tt == OpenBraceToken {
		arrowFunc.Body = p.parseBlockStmt("arrow function", false)
	} else {
		arrowFunc.Body = BlockStmt{[]IStmt{&ReturnStmt{p.parseExpression(OpAssign)}}, Scope{}}
	}
	return
}

// parseExpression parses an expression that has a precendence of prec or higher.
func (p *Parser) parseExpression(prec OpPrec) IExpr {
	// reparse input if we have / or /= as the beginning of a new expression, this should be a regular expression!
	if p.tt == DivToken || p.tt == DivEqToken {
		p.tt, p.data = p.l.RegExp()
		if p.tt == ErrorToken {
			p.fail("regular expression")
			return nil
		}
	}

	var left IExpr
	precLeft := OpPrimary
	switch tt := p.tt; tt {
	case IdentifierToken:
		lit := LiteralExpr(p.src.Ref(p.tt, p.data))
		left = &lit
		p.varUse(p.data)
		p.next()
	case StringToken, ThisToken, NullToken, TrueToken, FalseToken, RegExpToken:
		lit := LiteralExpr(p.src.Ref(p.tt, p.data))
		left = &lit
		p.next()
	case OpenBracketToken:
		// array literal and [expression]
		array := ArrayExpr{}
		p.next()
		prevComma := true
		for p.tt != CloseBracketToken && p.tt != ErrorToken {
			if p.tt == CommaToken {
				if prevComma {
					array.List = append(array.List, Element{})
				}
				prevComma = true
				p.next()
			} else {
				spread := p.tt == EllipsisToken
				if spread {
					p.next()
				}
				array.List = append(array.List, Element{p.parseExpression(OpAssign), spread})
				prevComma = false
			}
		}
		p.next()
		left = &array
	case OpenBraceToken:
		object := p.parseObjectLiteral()
		left = &object
	case OpenParenToken:
		// parenthesized expression or arrow parameter list
		p.next()
		var list []IExpr
		var rest IExpr
		for p.tt != CloseParenToken && p.tt != ErrorToken {
			if p.tt == EllipsisToken {
				p.next()
				rest = p.parseExpression(OpLHS) // all variables are saved as unbound
			} else if p.tt == CommaToken {
				p.next()
			} else {
				list = append(list, p.parseExpression(OpAssign))
			}
		}
		p.next()
		if p.tt == ArrowToken {
			// arrow function
			if OpAssign < prec {
				return left
			} else if precLeft < OpPrimary {
				p.fail("expression")
				return nil
			}
			var fail bool
			var scope Scope
			parent := p.enterFunctionScope(&scope, false, false)
			params := Params{List: make([]BindingElement, len(list)), Rest: nil}
			for i, item := range list {
				params.List[i], fail = p.exprToBindingElement(item, parent.functionScope)
				if fail {
					p.fail("expression")
					return nil
				}
			}
			if rest != nil {
				params.Rest, fail = p.exprToBinding(rest, parent.functionScope)
				if fail {
					p.fail("expression")
					return nil
				}
			}
			arrowFunc := p.parseArrowFunc(params)
			arrowFunc.Scope = scope
			p.restoreFunctionScope(parent)
			left = &arrowFunc
			precLeft = OpAssign
		} else if len(list) == 0 || rest != nil {
			p.fail("arrow function", ArrowToken)
			return nil
		} else {
			// parenthesized expression
			left = list[0]
			for _, item := range list[1:] {
				left = &BinaryExpr{CommaToken, left, item}
			}
			left = &GroupExpr{left}
		}
	case NotToken, BitNotToken, TypeofToken, VoidToken, DeleteToken:
		if OpUnary < prec {
			p.fail("expression")
			return nil
		}
		p.next()
		left = &UnaryExpr{tt, p.parseExpression(OpUnary)}
		precLeft = OpUnary
	case AddToken:
		if OpUnary < prec {
			p.fail("expression")
			return nil
		}
		p.next()
		left = &UnaryExpr{PosToken, p.parseExpression(OpUnary)}
		precLeft = OpUnary
	case SubToken:
		if OpUnary < prec {
			p.fail("expression")
			return nil
		}
		p.next()
		left = &UnaryExpr{NegToken, p.parseExpression(OpUnary)}
		precLeft = OpUnary
	case IncrToken:
		if OpUpdate < prec {
			p.fail("expression")
			return nil
		}
		p.next()
		left = &UnaryExpr{PreIncrToken, p.parseExpression(OpUnary)}
		precLeft = OpUnary
	case DecrToken:
		if OpUpdate < prec {
			p.fail("expression")
			return nil
		}
		p.next()
		left = &UnaryExpr{PreDecrToken, p.parseExpression(OpUnary)}
		precLeft = OpUnary
	case AwaitToken:
		// either accepted as IdentifierReference or as AwaitExpression, if followed by => it could be a BindingIdentifier for an arrow function
		await := LiteralExpr(p.src.Ref(IdentifierToken, p.data))
		if p.async && (p.tt != ArrowToken || p.prevLineTerminator) && prec <= OpUnary {
			p.next()
			left = &UnaryExpr{tt, p.parseExpression(OpUnary)}
			precLeft = OpUnary
		} else if p.async && (p.tt != ArrowToken || p.prevLineTerminator) {
			p.fail("expression")
			return nil
		} else {
			p.varUse(p.data)
			p.next()
			left = &await
		}
	case NewToken:
		p.next()
		if p.tt == DotToken {
			p.next()
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("target")) {
				p.fail("new.target expression", TargetToken)
				return nil
			}
			left = &NewTargetExpr{}
			precLeft = OpMember
			p.next()
		} else {
			newExpr := &NewExpr{p.parseExpression(OpMember), nil}
			if p.tt == OpenParenToken {
				args := p.parseArgs()
				if len(args.List) == 0 && args.Rest == nil {
					precLeft = OpLHS
				} else {
					newExpr.Args = &args
					precLeft = OpMember
				}
			} else {
				precLeft = OpLHS
			}
			left = newExpr
		}
	case ImportToken:
		if OpMember < prec {
			p.fail("expression")
			return nil
		}
		lit := LiteralExpr(p.src.Ref(p.tt, p.data))
		left = &lit
		p.next()
		if p.tt == DotToken {
			p.next()
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("meta")) {
				p.fail("import.meta expression", MetaToken)
				return nil
			}
			left = &ImportMetaExpr{}
			precLeft = OpMember
			p.next()
		} else if p.tt != OpenParenToken {
			p.fail("import expression", OpenParenToken)
			return nil
		} else if prec == OpMember {
			p.fail("expression")
			return nil
		} else {
			precLeft = OpLHS
		}
	case SuperToken:
		if OpMember < prec {
			p.fail("expression")
			return nil
		}
		lit := LiteralExpr(p.src.Ref(p.tt, p.data))
		left = &lit
		p.next()
		if prec == OpMember && p.tt != DotToken && p.tt != OpenBracketToken {
			p.fail("super expression", OpenBracketToken, DotToken)
			return nil
		} else if p.tt != DotToken && p.tt != OpenBracketToken && p.tt != OpenParenToken {
			p.fail("super expression", OpenBracketToken, OpenParenToken, DotToken)
			return nil
		}
		precLeft = OpLHS
	case YieldToken:
		// either accepted as IdentifierReference or as YieldExpression
		yield := LiteralExpr(p.src.Ref(IdentifierToken, p.data))
		if p.generator && prec <= OpAssign {
			// YieldExpression
			p.next()
			yieldExpr := YieldExpr{}
			if !p.prevLineTerminator {
				yieldExpr.Generator = p.tt == MulToken
				if yieldExpr.Generator {
					p.next()
				}
				yieldExpr.X = p.parseExpression(OpAssign)
			}
			left = &yieldExpr
			precLeft = OpAssign
		} else if p.generator {
			p.fail("expression")
			return nil
		} else {
			p.varUse(p.data)
			p.next()
			left = &yield
		}
	case AsyncToken:
		p.next()
		if p.tt == FunctionToken {
			// primary expression
			if p.prevLineTerminator {
				p.fail("function declaration")
				return nil
			}
			funcDecl := p.parseAsyncFuncExpr()
			left = &funcDecl
		} else if OpAssign < prec {
			if p.prevLineTerminator {
				p.fail("function declaration")
				return nil
			} else {
				p.fail("function declaration", FunctionToken)
				return nil
			}
			return nil
		} else if p.tt == OpenParenToken || p.tt == IdentifierToken || !p.generator && p.tt == YieldToken {
			// async arrow function expression
			if p.prevLineTerminator {
				p.fail("arrow function")
				return nil
			}
			arrowFunc := p.parseAsyncArrowFunc()
			left = &arrowFunc
			precLeft = OpAssign
		} else {
			if p.prevLineTerminator {
				p.fail("function declaration")
			} else {
				p.fail("function declaration", FunctionToken, IdentifierToken)
			}
			return nil
		}
	case ClassToken:
		classDecl := p.parseClassDecl()
		left = &classDecl
	case FunctionToken:
		funcDecl := p.parseFuncExpr()
		left = &funcDecl
	case TemplateToken, TemplateStartToken:
		template := p.parseTemplateLiteral()
		left = &template
	default:
		if IsNumeric(p.tt) {
			lit := LiteralExpr(p.src.Ref(p.tt, p.data))
			left = &lit
			p.next()
		} else {
			p.fail("expression")
			return nil
		}
	}

	for {
		switch tt := p.tt; tt {
		case EqToken, MulEqToken, DivEqToken, ModEqToken, ExpEqToken, AddEqToken, SubEqToken, LtLtEqToken, GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitXorEqToken, BitOrEqToken:
			if OpAssign < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAssign)}
			precLeft = OpAssign
		case LtToken, LtEqToken, GtToken, GtEqToken, InToken, InstanceofToken:
			if OpCompare < prec || p.inFor && tt == InToken {
				return left
			} else if precLeft < OpCompare {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpShift)}
			precLeft = OpCompare
		case EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken:
			if OpEquals < prec {
				return left
			} else if precLeft < OpEquals {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpCompare)}
			precLeft = OpEquals
		case AndToken:
			if OpAnd < prec {
				return left
			} else if precLeft < OpAnd {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitOr)}
			precLeft = OpAnd
		case OrToken:
			if OpOr < prec {
				return left
			} else if precLeft < OpOr {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAnd)}
			precLeft = OpOr
		case NullishToken:
			if OpCoalesce < prec {
				return left
			} else if precLeft < OpBitOr && precLeft != OpCoalesce {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitOr)}
			precLeft = OpCoalesce
		case DotToken:
			if OpMember < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			p.next()
			if !IsIdentifier(p.tt) {
				p.fail("dot expression", IdentifierToken)
				return nil
			}
			left = &DotExpr{left, LiteralExpr(p.src.Ref(IdentifierToken, p.data))}
			precLeft = OpMember
			p.next()
		case OpenBracketToken:
			if OpMember < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &IndexExpr{left, p.parseExpression(OpExpr)}
			if !p.consume("index expression", CloseBracketToken) {
				return nil
			}
			precLeft = OpMember
		case OpenParenToken:
			if OpLHS < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			left = &CallExpr{left, p.parseArgs()}
			precLeft = OpLHS
		case TemplateToken, TemplateStartToken:
			if OpMember < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			template := p.parseTemplateLiteral()
			template.Tag = left
			left = &template
			precLeft = OpMember
		case OptChainToken:
			// left must be LHS
			if OpLHS < prec {
				return left
			}
			p.next()
			if p.tt == OpenParenToken {
				left = &OptChainExpr{left, &CallExpr{nil, p.parseArgs()}}
			} else if p.tt == OpenBracketToken {
				p.next()
				left = &OptChainExpr{left, &IndexExpr{nil, p.parseExpression(OpExpr)}}
				if !p.consume("optional chaining expression", CloseBracketToken) {
					return nil
				}
			} else if p.tt == TemplateToken || p.tt == TemplateStartToken {
				template := p.parseTemplateLiteral()
				left = &OptChainExpr{left, &template}
			} else if IsIdentifier(p.tt) {
				lit := LiteralExpr(p.src.Ref(p.tt, p.data))
				left = &OptChainExpr{left, &lit}
				p.next()
			} else {
				p.fail("optional chaining expression", IdentifierToken, OpenParenToken, OpenBracketToken, TemplateToken)
				return nil
			}
			precLeft = OpLHS
		case IncrToken:
			if p.prevLineTerminator || OpUpdate < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &UnaryExpr{PostIncrToken, left}
			precLeft = OpUpdate
		case DecrToken:
			if p.prevLineTerminator || OpUpdate < prec {
				return left
			} else if precLeft < OpLHS {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &UnaryExpr{PostDecrToken, left}
			precLeft = OpUpdate
		case ExpToken:
			if OpExp < prec {
				return left
			} else if precLeft < OpUpdate {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpExp)}
			precLeft = OpExp
		case MulToken, DivToken, ModToken:
			if OpMul < prec {
				return left
			} else if precLeft < OpMul {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpExp)}
			precLeft = OpMul
		case AddToken, SubToken:
			if OpAdd < prec {
				return left
			} else if precLeft < OpAdd {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpMul)}
			precLeft = OpAdd
		case LtLtToken, GtGtToken, GtGtGtToken:
			if OpShift < prec {
				return left
			} else if precLeft < OpShift {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAdd)}
			precLeft = OpShift
		case BitAndToken:
			if OpBitAnd < prec {
				return left
			} else if precLeft < OpBitAnd {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpEquals)}
			precLeft = OpBitAnd
		case BitXorToken:
			if OpBitXor < prec {
				return left
			} else if precLeft < OpBitXor {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitAnd)}
			precLeft = OpBitXor
		case BitOrToken:
			if OpBitOr < prec {
				return left
			} else if precLeft < OpBitOr {
				p.fail("expression")
				return nil
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitXor)}
			precLeft = OpBitOr
		case QuestionToken:
			if OpAssign < prec {
				return left
			} else if precLeft < OpCoalesce {
				p.fail("expression")
				return nil
			}
			p.next()
			ifExpr := p.parseExpression(OpAssign)
			if !p.consume("conditional expression", ColonToken) {
				return nil
			}
			elseExpr := p.parseExpression(OpAssign)
			left = &CondExpr{left, ifExpr, elseExpr}
			precLeft = OpAssign
		case CommaToken:
			if OpExpr < prec {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAssign)}
			precLeft = OpExpr
		case ArrowToken:
			// handle identifier => ..., where identifier could also be yield or await
			if OpAssign < prec {
				return left
			} else if precLeft < OpPrimary {
				p.fail("expression")
				return nil
			}
			lit, ok := left.(*LiteralExpr)
			if !ok || !p.isIdentifierReference(lit.TokenType) {
				p.fail("expression")
				return nil
			}
			var scope Scope
			p.functionScope.Unbound[string(lit.Data(p.src))]--
			parent := p.enterFunctionScope(&scope, false, false)
			p.varDefine(lit.Data(p.src), FunctionBindingScope)

			params := Params{[]BindingElement{{Binding: (*BindingName)(lit)}}, nil}
			arrowFunc := p.parseArrowFunc(params)
			arrowFunc.Scope = scope
			p.restoreFunctionScope(parent)

			left = &arrowFunc
			precLeft = OpAssign
		default:
			return left
		}
	}
}

func (p *Parser) exprToBinding(expr IExpr, parentFunctionScope *Scope) (binding IBinding, fail bool) {
	if lit, ok := expr.(*LiteralExpr); ok && p.isIdentifierReference(lit.TokenType) {
		parentFunctionScope.Unbound[string(lit.Data(p.src))]--
		p.varDefine(lit.Data(p.src), FunctionBindingScope)
		binding = (*BindingName)(lit)
	} else if array, ok := expr.(*ArrayExpr); ok {
		bindingArray := BindingArray{}
		for i, item := range array.List {
			if item.Spread && i != len(array.List)-1 {
				fail = true
				return
			} else if item.Spread {
				bindingArray.Rest, fail = p.exprToBinding(item.Value, parentFunctionScope)
				if fail {
					return
				}
				break
			}
			var bindingElement BindingElement
			bindingElement, fail = p.exprToBindingElement(item.Value, parentFunctionScope)
			if fail {
				return
			}
			bindingArray.List = append(bindingArray.List, bindingElement)
		}
		binding = &bindingArray
	} else if object, ok := expr.(*ObjectExpr); ok {
		bindingObject := BindingObject{}
		for i, item := range object.List {
			if item.Spread && i != len(object.List)-1 {
				fail = true
				return
			} else if item.Spread {
				if lit, ok := item.Value.(*LiteralExpr); ok && p.isIdentifierReference(lit.TokenType) {
					parentFunctionScope.Unbound[string(lit.Data(p.src))]--
					p.varDefine(lit.Data(p.src), FunctionBindingScope)
					bindingObject.Rest = BindingName(*lit)
					break
				} else {
					fail = true
					return
				}
			}
			var bindingElement BindingElement
			bindingElement.Binding, fail = p.exprToBinding(item.Value, parentFunctionScope)
			if fail {
				return
			}
			if item.Init != nil {
				p.exprRebindVars(item.Init, parentFunctionScope)
				bindingElement.Default = item.Init
			}
			if item.Key != nil && item.Key.Computed != nil {
				p.exprRebindVars(item.Key.Computed, parentFunctionScope)
			}
			bindingObject.List = append(bindingObject.List, BindingObjectItem{Key: item.Key, Value: bindingElement})
		}
		binding = &bindingObject
	} else if expr != nil {
		fail = true
	}
	return
}

func (p *Parser) exprToBindingElement(expr IExpr, parentFunctionScope *Scope) (bindingElement BindingElement, fail bool) {
	if assign, ok := expr.(*BinaryExpr); ok && assign.Op == EqToken {
		p.exprRebindVars(assign.Y, parentFunctionScope)
		bindingElement.Default = assign.Y
		expr = assign.X
	}
	bindingElement.Binding, fail = p.exprToBinding(expr, parentFunctionScope)
	return
}

func (p *Parser) exprRebindVars(iexpr IExpr, parentFunctionScope *Scope) {
	switch expr := iexpr.(type) {
	case *LiteralExpr:
		parentFunctionScope.Unbound[string(expr.Data(p.src))]--
		p.functionScope.Unbound[string(expr.Data(p.src))]++
	}
}

func (p *Parser) isIdentifierReference(tt TokenType) bool {
	return tt == IdentifierToken || tt == YieldToken && !p.generator || tt == AwaitToken && !p.async
}
