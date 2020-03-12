package js

import (
	"bytes"
	"io"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/buffer"
)

// Parser is the state for the parser.
type Parser struct {
	l   *Lexer
	err error

	tt                 TokenType
	data               []byte
	prevLineTerminator bool
	asyncLevel         int
	generatorLevel     int
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
	case ContinueToken, BreakToken:
		tt := p.tt
		p.next()
		var name *Token
		if !p.prevLineTerminator && p.isIdentRef(p.tt) {
			name = &Token{IdentifierToken, p.data}
			p.next()
		}
		stmt = &BranchStmt{tt, name}
	case ReturnToken:
		p.next()
		var value IExpr
		if !p.prevLineTerminator && p.tt != SemicolonToken && p.tt != LineTerminatorToken && p.tt != ErrorToken {
			value = p.parseExpr()
		}
		stmt = &ReturnStmt{value}
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
			var cond, post IExpr
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
			if await {
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
		if p.tt == ColonToken {
			if literal, ok := expr.(*LiteralExpr); ok && (literal.TokenType != AwaitToken || p.asyncLevel == 0) {
				p.next() // colon
				stmt = &LabelledStmt{Token{IdentifierToken, literal.Data}, p.parseStmt()}
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
			var list IExpr
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
		funcDecl := p.parseFuncDecl(false, false)
		stmt = &funcDecl
	case AsyncToken: // async function
		p.next()
		if p.tt != FunctionToken {
			p.fail("function statement", FunctionToken)
			return
		}
		funcDecl := p.parseFuncDecl(true, false)
		stmt = &funcDecl
	case ClassToken:
		classDecl := p.parseClassDecl()
		stmt = &classDecl
	case ThrowToken:
		p.next()
		var value IExpr
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
		funcDecl := p.parseFuncDecl(false, false)
		exportStmt.Decl = &funcDecl
	} else if p.tt == AsyncToken { // async function
		p.next()
		if p.tt != FunctionToken {
			p.fail("export statement", FunctionToken)
			return
		}
		funcDecl := p.parseFuncDecl(true, false)
		exportStmt.Decl = &funcDecl
	} else if p.tt == ClassToken {
		classDecl := p.parseClassDecl()
		exportStmt.Decl = &classDecl
	} else if p.tt == DefaultToken {
		exportStmt.Default = true
		p.next()
		if p.tt == FunctionToken {
			funcDecl := p.parseFuncDecl(false, true)
			exportStmt.Decl = &funcDecl
		} else if p.tt == AsyncToken { // async function
			p.next()
			if p.tt != FunctionToken {
				p.fail("export statement", FunctionToken)
				return
			}
			funcDecl := p.parseFuncDecl(true, true)
			exportStmt.Decl = &funcDecl
		} else if p.tt == ClassToken {
			classDecl := p.parseClassDecl()
			exportStmt.Decl = &classDecl
		} else {
			exportStmt.Decl = p.parseAssignmentExpr()
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

func (p *Parser) parseFuncDecl(async, nameOptional bool) (funcDecl FuncDecl) {
	// assume we're at function
	p.next()
	funcDecl.Async = async
	funcDecl.Generator = p.tt == MulToken
	if funcDecl.Generator {
		p.next()
	}
	if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
		funcDecl.Name = p.data
		p.next()
	} else if !nameOptional {
		p.fail("function declaration", IdentifierToken)
		return
	} else if p.tt != OpenParenToken {
		p.fail("function declaration", IdentifierToken, OpenParenToken)
		return
	}
	if funcDecl.Async {
		p.asyncLevel++
	}
	if funcDecl.Generator {
		p.generatorLevel++
	}
	funcDecl.Params = p.parseFuncParams("function declaration")
	funcDecl.Body = p.parseBlockStmt("function declaration")
	if funcDecl.Generator {
		p.generatorLevel--
	}
	if funcDecl.Async {
		p.asyncLevel--
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
		classDecl.Extends = p.parseLeftHandSideExpr()
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
		method.Name = PropertyName{Token{IdentifierToken, p.data}, nil}
		p.next()
	} else if p.tt == StringToken || IsNumeric(p.tt) {
		method.Name = PropertyName{Token{p.tt, p.data}, nil}
		p.next()
	} else if p.tt == OpenBracketToken {
		p.next()
		method.Name = PropertyName{Token{}, p.parseAssignmentExpr()}
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
		bindingElement.Default = p.parseAssignmentExpr()
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
					item.Key = &PropertyName{Literal: ident}
					item.Value = p.parseBindingElement()
				} else {
					// single name binding
					item.Value.Binding = &BindingName{ident.Data}
					if p.tt == EqToken {
						p.next()
						item.Value.Default = p.parseAssignmentExpr()
					}
				}
			} else if IsIdentifier(p.tt) || p.tt == StringToken || IsNumeric(p.tt) || p.tt == OpenBracketToken {
				// property name + : + binding element
				if p.tt == OpenBracketToken {
					p.next()
					item.Key = &PropertyName{Computed: p.parseAssignmentExpr()}
					if p.tt != CloseBracketToken {
						p.fail("object binding pattern", CloseBracketToken)
						return
					}
					p.next()
				} else if IsIdentifier(p.tt) {
					item.Key = &PropertyName{Literal: Token{IdentifierToken, p.data}}
					p.next()
				} else {
					item.Key = &PropertyName{Literal: Token{p.tt, p.data}}
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

func (p *Parser) isIdentRef(tt TokenType) bool {
	return tt == IdentifierToken || tt == YieldToken && p.generatorLevel == 0 || tt == AwaitToken && p.asyncLevel == 0
}

func (p *Parser) parseObjectLiteral() (object ObjectExpr) {
	// assume we're on {
	p.next()
	for p.tt != CloseBraceToken && p.tt != ErrorToken {
		property := Property{}
		if p.tt == EllipsisToken {
			p.next()
			property.Spread = true
			property.Value = p.parseAssignmentExpr()
		} else {
			// try to parse as MethodDefinition, otherwise fall back to PropertyName:AssignExpr or IdentifierReference
			method := MethodDecl{}
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
					method.Name.Literal = Token{IdentifierToken, []byte("async")}
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
					method.Name.Literal = Token{IdentifierToken, p.data}
					p.next()
				} else if p.tt == StringToken || IsNumeric(p.tt) {
					method.Name.Literal = Token{p.tt, p.data}
					p.next()
				} else if p.tt == OpenBracketToken {
					p.next()
					method.Name.Computed = p.parseAssignmentExpr()
					if p.tt != CloseBracketToken {
						p.fail("object literal", CloseBracketToken)
						return
					}
					p.next()
				} else if !method.Generator && (method.Async || method.Get || method.Set) {
					// interpret async, get, or set as PropertyName instead of method keyword
					if method.Async {
						method.Name.Literal = Token{IdentifierToken, []byte("async")}
						method.Async = false
					} else if method.Get {
						method.Name.Literal = Token{IdentifierToken, []byte("get")}
						method.Get = false
					} else if method.Set {
						method.Name.Literal = Token{IdentifierToken, []byte("set")}
						method.Set = false
					}
				} else {
					p.fail("object literal", IdentifierToken, StringToken, NumericToken, OpenBracketToken)
					return
				}
			}

			if p.tt == OpenParenToken {
				// MethodDefinition
				if method.Async {
					p.asyncLevel++
				}
				method.Params = p.parseFuncParams("method definition")
				method.Body = p.parseBlockStmt("method definition")
				if method.Async {
					p.asyncLevel--
				}
				property.Value = &method
			} else if p.tt == ColonToken {
				// PropertyName : AssignmentExpression
				p.next()
				property.Key = &method.Name
				property.Value = p.parseAssignmentExpr()
			} else if !p.isIdentRef(method.Name.Literal.TokenType) {
				p.fail("object literal", ColonToken, OpenParenToken)
				return
			} else {
				// IdentifierReference (= AssignmentExpression)?
				property.Value = (*LiteralExpr)(&method.Name.Literal)
				if p.tt == EqToken {
					p.next()
					property.Init = p.parseAssignmentExpr()
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
		template.List = append(template.List, TemplatePart{tpl, p.parseExpr()})
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
	args.List = []IExpr{}
	for {
		if p.tt == EllipsisToken {
			p.next()
			args.Rest = p.parseAssignmentExpr()
			if p.tt == CommaToken {
				p.next()
			}
			break
		}

		if p.tt == CloseParenToken || p.tt == ErrorToken {
			break
		}
		args.List = append(args.List, p.parseAssignmentExpr())
		if p.tt == CommaToken {
			p.next()
		}
	}
	p.consume("arguments", CloseParenToken)
	return
}

func (p *Parser) parseExpr() (expr IExpr) {
	return p.parseExpression(OpEnd)
}

func (p *Parser) parseAssignmentExpr() (expr IExpr) {
	return p.parseExpression(OpComma)
}

func (p *Parser) parseLeftHandSideExpr() (expr IExpr) {
	return p.parseExpression(OpNew)
}

func (p *Parser) parseExpression(prec OpPrec) (expr IExpr) {
	// reparse input if we have / or /= as the beginning of a new expression, this should be a regular expression!
	if p.tt == DivToken || p.tt == DivEqToken {
		p.tt, p.data = p.l.RegExp()
	}

	var left IExpr
	switch tt := p.tt; tt {
	case IdentifierToken, StringToken, ThisToken, NullToken, TrueToken, FalseToken, RegExpToken:
		left = &LiteralExpr{p.tt, p.data}
		p.next()
	case OpenBracketToken:
		// array literal and [expression]
		array := ArrayExpr{}
		p.next()
		commas := 1
		for p.tt != CloseBracketToken && p.tt != ErrorToken {
			if p.tt == EllipsisToken {
				p.next()
				array.Rest = p.parseAssignmentExpr()
				break
			} else if p.tt == CommaToken {
				commas++
				p.next()
			} else {
				for 1 < commas {
					array.List = append(array.List, nil)
					commas--
				}
				commas = 0
				array.List = append(array.List, p.parseAssignmentExpr())
			}
		}
		p.next()
		left = &array
	case OpenBraceToken:
		object := p.parseObjectLiteral()
		left = &object
	case OpenParenToken:
		// parenthesized expression and arrow parameter list
		group := GroupExpr{}
		p.next()
		for p.tt != CloseParenToken && p.tt != ErrorToken {
			if p.tt == EllipsisToken {
				p.next()
				group.Rest = p.parseBinding()
			} else if p.tt == CommaToken {
				p.next()
			} else {
				group.List = append(group.List, p.parseAssignmentExpr())
			}
		}
		p.next()
		left = &group
	case NotToken, BitNotToken, TypeofToken, VoidToken, DeleteToken:
		p.next()
		left = &UnaryExpr{tt, p.parseExpression(OpPrefix - 1)}
	case AddToken:
		p.next()
		left = &UnaryExpr{PosToken, p.parseExpression(OpPrefix - 1)}
	case SubToken:
		p.next()
		left = &UnaryExpr{NegToken, p.parseExpression(OpPrefix - 1)}
	case IncrToken:
		p.next()
		left = &UnaryExpr{PreIncrToken, p.parseExpression(OpPrefix - 1)}
	case DecrToken:
		p.next()
		left = &UnaryExpr{PreDecrToken, p.parseExpression(OpPrefix - 1)}
	case NewToken:
		p.next()
		if p.tt == DotToken {
			p.next()
			if p.tt != IdentifierToken || !bytes.Equal(p.data, []byte("target")) {
				p.fail("new expression", TargetToken)
				return
			}
			left = &NewTargetExpr{}
			p.next()
		} else {
			left = &NewExpr{p.parseExpression(OpNew)}
		}
	case ImportToken:
		left = &LiteralExpr{p.tt, p.data}
		p.next()
		if p.tt != OpenParenToken {
			p.fail("import expression", OpenParenToken)
		}
	case SuperToken:
		left = &LiteralExpr{p.tt, p.data}
		p.next()
		if p.tt != DotToken && p.tt != OpenBracketToken && p.tt != OpenParenToken {
			p.fail("super expression", OpenBracketToken, OpenParenToken, DotToken)
		}
	case AwaitToken:
		p.next()
		if 0 < p.asyncLevel && (p.tt != ArrowToken || p.prevLineTerminator) {
			left = &UnaryExpr{tt, p.parseExpression(OpPrefix - 1)}
		} else {
			left = &LiteralExpr{IdentifierToken, []byte("await")}
		}
	case YieldToken:
		p.next()
		if 0 < p.generatorLevel {
			// YieldExpression
			yieldExpr := YieldExpr{}
			if !p.prevLineTerminator {
				yieldExpr.Generator = p.tt == MulToken
				if yieldExpr.Generator {
					p.next()
				}
				yieldExpr.X = p.parseExpression(OpYield - 1)
			}
			left = &yieldExpr
		} else {
			left = &LiteralExpr{IdentifierToken, []byte("yield")}
		}
	case AsyncToken:
		p.next()
		if p.prevLineTerminator {
			p.fail("function declaration")
			return nil
		}
		if p.tt == FunctionToken {
			// primary expression
			funcDecl := p.parseFuncDecl(true, true)
			left = &funcDecl
		} else if OpAssign < prec {
			p.fail("function declaration", FunctionToken)
			return nil
		} else if p.tt == IdentifierToken || p.tt == YieldToken || p.tt == AwaitToken {
			name := p.data
			p.next()
			if p.tt != ArrowToken {
				p.fail("arrow function declaration", ArrowToken)
				return nil
			}

			arrowFuncDecl := ArrowFuncDecl{}
			arrowFuncDecl.Async = true
			arrowFuncDecl.Params = Params{List: []BindingElement{{Binding: &BindingName{name}}}}
			p.next()
			p.asyncLevel++
			if p.tt == OpenBraceToken {
				arrowFuncDecl.Body = p.parseBlockStmt("arrow function declaration")
			} else {
				arrowFuncDecl.Body = BlockStmt{[]IStmt{ExprStmt{p.parseAssignmentExpr()}}}
			}
			p.asyncLevel--
			left = &arrowFuncDecl
		} else {
			p.fail("function declaration", FunctionToken, IdentifierToken)
			return nil
		}
	case ClassToken:
		classDecl := p.parseClassDecl()
		left = &classDecl
	case FunctionToken:
		funcDecl := p.parseFuncDecl(false, true)
		left = &funcDecl
	case TemplateToken, TemplateStartToken:
		template := p.parseTemplateLiteral()
		left = &template
	default:
		if IsNumeric(p.tt) {
			left = &LiteralExpr{p.tt, p.data}
			p.next()
		} else {
			p.fail("expression")
			return
		}
	}

	for {
		switch tt := p.tt; tt {
		case EqToken, MulEqToken, DivEqToken, ModEqToken, ExpEqToken, AddEqToken, SubEqToken, LtLtEqToken, GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitXorEqToken, BitOrEqToken:
			if prec >= OpAssign {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAssign - 1)}
		case LtToken, LtEqToken, GtToken, GtEqToken, InToken, InstanceofToken:
			if prec >= OpCompare || p.inFor && tt == InToken {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpCompare)}
		case EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken:
			if prec >= OpEquals {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpEquals)}
		case AndToken:
			if prec >= OpOr {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAnd)}
		case OrToken:
			if prec >= OpOr {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpOr)}
		case DotToken:
			if prec >= OpCall {
				return left
			}
			p.next()
			if !IsIdentifier(p.tt) {
				p.fail("dot expression", IdentifierToken)
				return nil
			}
			left = &DotExpr{left, LiteralExpr{IdentifierToken, p.data}}
			p.next()
		case OpenBracketToken:
			if prec >= OpCall {
				return left
			}
			p.next()
			left = &IndexExpr{left, p.parseExpr()}
			if !p.consume("index expression", CloseBracketToken) {
				return nil
			}
		case OpenParenToken:
			if prec >= OpCall {
				return left
			}
			left = &CallExpr{left, p.parseArgs()}
		case OptChainToken:
			if prec >= OpCall {
				return left
			}
			p.next()
			if p.tt == OpenParenToken {
				left = &OptChainExpr{left, &CallExpr{nil, p.parseArgs()}}
			} else if p.tt == OpenBracketToken {
				p.next()
				left = &OptChainExpr{left, &IndexExpr{nil, p.parseExpr()}}
				if !p.consume("optional chaining expression", CloseBracketToken) {
					return nil
				}
			} else if p.tt == TemplateToken || p.tt == TemplateStartToken {
				template := p.parseTemplateLiteral()
				left = &OptChainExpr{left, &template}
			} else if IsIdentifier(p.tt) {
				left = &OptChainExpr{left, &LiteralExpr{p.tt, p.data}}
				p.next()
			} else {
				p.fail("optional chaining expression", IdentifierToken, OpenParenToken, OpenBracketToken, TemplateToken)
				return nil
			}
		case IncrToken:
			if p.prevLineTerminator || prec >= OpPostfix {
				return left
			}
			p.next()
			left = &UnaryExpr{PostIncrToken, left}
		case DecrToken:
			if p.prevLineTerminator || prec >= OpPostfix {
				return left
			}
			p.next()
			left = &UnaryExpr{PostDecrToken, left}
		case ExpToken:
			if prec >= OpExp {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpExp - 1)}
		case MulToken, DivToken, ModToken:
			if prec >= OpMul {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpMul)}
		case AddToken, SubToken:
			if prec >= OpAdd {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpAdd)}
		case LtLtToken, GtGtToken, GtGtGtToken:
			if prec >= OpShift {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpShift)}
		case BitAndToken:
			if prec >= OpBitAnd {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitAnd)}
		case BitXorToken:
			if prec >= OpBitXor {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitXor)}
		case BitOrToken:
			if prec >= OpBitOr {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpBitOr)}
		case NullishToken:
			if prec >= OpNullish {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpNullish)}
		case QuestionToken:
			if prec >= OpCond {
				return left
			}
			p.next()
			ifExpr := p.parseExpression(OpCond - 1)
			if !p.consume("conditional expression", ColonToken) {
				return nil
			}
			elseExpr := p.parseExpression(OpCond - 1)
			left = &ConditionalExpr{left, ifExpr, elseExpr}
		case CommaToken:
			if prec >= OpComma {
				return left
			}
			p.next()
			left = &BinaryExpr{tt, left, p.parseExpression(OpComma)}
		case TemplateToken, TemplateStartToken:
			if prec >= OpCall {
				return left
			}
			template := p.parseTemplateLiteral()
			template.Tag = left
			left = &template
		case ArrowToken:
			if p.prevLineTerminator {
				p.fail("expression")
				return nil
			}

			var fail bool
			arrowFuncDecl := ArrowFuncDecl{}
			arrowFuncDecl.Params, fail = p.exprToParams(left)
			if fail {
				p.fail("expression")
				return nil
			}
			p.next()
			if p.tt == OpenBraceToken {
				arrowFuncDecl.Body = p.parseBlockStmt("arrow function expression")
			} else {
				arrowFuncDecl.Body = BlockStmt{[]IStmt{ExprStmt{p.parseAssignmentExpr()}}}
			}
			left = &arrowFuncDecl
		default:
			return left
		}
	}
}

func (p *Parser) exprToBinding(expr IExpr) (binding IBinding, fail bool) {
	if literal, ok := expr.(*LiteralExpr); ok && (literal.TokenType == IdentifierToken || literal.TokenType == YieldToken || literal.TokenType == AwaitToken) {
		binding = &BindingName{literal.Data}
	} else if array, ok := expr.(*ArrayExpr); ok {
		bindingArray := BindingArray{}
		for _, item := range array.List {
			var bindingElement BindingElement
			bindingElement, fail = p.exprToBindingElement(item)
			if fail {
				return
			}
			bindingArray.List = append(bindingArray.List, bindingElement)
		}
		if array.Rest != nil {
			bindingArray.Rest, fail = p.exprToBinding(array.Rest)
		}
		binding = &bindingArray
	} else if object, ok := expr.(*ObjectExpr); ok {
		bindingObject := BindingObject{}
		for _, item := range object.List {
			if item.Init != nil || item.Spread {
				fail = true
				return
			}
			var bindingElement BindingElement
			bindingElement, fail = p.exprToBindingElement(item.Value)
			if fail {
				return
			}
			bindingObject.List = append(bindingObject.List, BindingObjectItem{Key: item.Key, Value: bindingElement})
		}
		binding = &bindingObject
	} else if expr != nil {
		fail = true
	}
	return
}

func (p *Parser) exprToBindingElement(expr IExpr) (bindingElement BindingElement, fail bool) {
	if assign, ok := expr.(*BinaryExpr); ok && assign.Op == EqToken {
		bindingElement.Default = assign.Y
		expr = assign.X
	}
	bindingElement.Binding, fail = p.exprToBinding(expr)
	return
}

func (p *Parser) exprToParams(expr IExpr) (params Params, fail bool) {
	if literal, ok := expr.(*LiteralExpr); ok && (literal.TokenType == IdentifierToken || literal.TokenType == YieldToken || literal.TokenType == AwaitToken) {
		params.List = append(params.List, BindingElement{Binding: &BindingName{literal.Data}})
	} else if group, ok := expr.(*GroupExpr); ok {
		for _, item := range group.List {
			var bindingElement BindingElement
			bindingElement, fail = p.exprToBindingElement(item)
			if fail {
				return
			}
			params.List = append(params.List, bindingElement)
		}
		if group.Rest != nil {
			params.Rest = &BindingElement{Binding: group.Rest}
		}
	} else {
		fail = true
	}
	return
}
