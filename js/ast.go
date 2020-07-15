package js

import (
	"bytes"
	"fmt"
	"strconv"
)

type DeclType uint16

const (
	NoDecl       DeclType = iota // undeclared variables
	VariableDecl                 // var and function
	LexicalDecl                  // let, const, class
	ArgumentDecl                 // function, method, and catch statement arguments
	ExprDecl                     // function expression name or class expression name
)

func (decl DeclType) String() string {
	switch decl {
	case NoDecl:
		return "NoDecl"
	case VariableDecl:
		return "VariableDecl"
	case LexicalDecl:
		return "LexicalDecl"
	case ArgumentDecl:
		return "ArgumentDecl"
	case ExprDecl:
		return "ExprDecl"
	}
	return "Invalid(" + strconv.Itoa(int(decl)) + ")"
}

// VarCtx holds the context needed for variable identifiers. It holds a list of all variables to which VarRef is indexing.
type VarCtx struct {
	Vars VarArray
}

func NewVarCtx() *VarCtx {
	return &VarCtx{
		Vars: VarArray{},
	}
}

func (ctx *VarCtx) Add(decl DeclType, data []byte) *Var {
	v := &Var{VarRef(len(ctx.Vars)), 0, decl, data}
	ctx.Vars = append(ctx.Vars, v)
	return v
}

func (ctx *VarCtx) String() string {
	s := "["
	for i, v := range ctx.Vars {
		if i != 0 {
			s += ", "
		}
		s += v.String()
	}
	return s + "]"
}

// VarRef is an index into VarCtx.Vars and is used by the AST to refer to a variable
// The chain of pointers: *VarRef --(ptr)--> VarRef(in Var) --(Vars[idx])--> *Var --(ptr)--> Var
type VarRef uint32 // *VarRef is faster than VarRef and refers to the VarRef stored in Var

func (ref VarRef) Get(ctx *VarCtx) *Var {
	return ctx.Vars[ref]
}

func (ref VarRef) String(ctx *VarCtx) string {
	return string(ctx.Vars[ref].Name)
}

// VarArray is sortable by uses
type VarArray []*Var

func (vs VarArray) Len() int {
	return len(vs)
}

func (vs VarArray) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs VarArray) Less(i, j int) bool {
	return vs[i].Uses < vs[j].Uses
}

func (vs VarArray) String() string {
	s := "["
	for i, item := range vs {
		if i != 0 {
			s += ", "
		}
		s += fmt.Sprintf("%v=>", item) + item.String()
	}
	return s + "]"
}

// Var is a variable, where Decl is the type of declaration and can be var|function for function scoped variables, let|const|class for block scoped variables
type Var struct {
	Ref  VarRef
	Uses uint16
	Decl DeclType
	Name []byte
}

func (v *Var) String() string {
	return fmt.Sprintf("Var{%v %v %v %s}", v.Ref, v.Uses, v.Decl, string(v.Name))
}

// Scope is a function or block scope with a list of variables declared and used
// TODO: handle with statement and eval function calls in scope?
type Scope struct {
	Parent, Func    *Scope
	Declared        VarArray
	Undeclared      VarArray
	argumentsOffset int
}

func (s Scope) String() string {
	return "Scope{Declared: " + s.Declared.String() + ", Undeclared: " + s.Undeclared.String() + "}"
}

// Declare a new variable
func (s *Scope) Declare(ctx *VarCtx, decl DeclType, name []byte) (*VarRef, bool) {
	// refer to new variable for previously undeclared symbols in the current and lower scopes
	// this happens in `{a=5} var a` where both a's refer to the same variable
	curScope := s
	if decl == VariableDecl {
		// find function scope for var and function declarations
		s = s.Func
	}

	if v := s.findScopeVar(name); v != nil {
		// variable already declared, might be an error or a duplicate declaration
		if (v.Decl == LexicalDecl || decl == LexicalDecl) && v.Decl != ExprDecl {
			// redeclaration of let, const, class on an already declared name is an error, except if the declared name is a function expression name
			return nil, false
		}
		if v.Decl == ExprDecl {
			v.Decl = decl
		}
		v.Uses++
		if s != curScope {
			curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
		}
		return &v.Ref, true
	}

	var v *Var
	if decl != ArgumentDecl { // in case of function f(a=b,b), where the first b is different from the second
		for i, uv := range s.Undeclared[s.argumentsOffset:] {
			if 0 < uv.Uses && bytes.Equal(uv.Name, name) {
				v = uv
				s.Undeclared = append(s.Undeclared[:s.argumentsOffset+i], s.Undeclared[s.argumentsOffset+i+1:]...)
				break
			}
		}
	}
	if v == nil {
		// add variable to the context list and to the scope
		v = ctx.Add(decl, name)
	} else {
		v.Decl = decl
	}
	v.Uses++
	s.Declared = append(s.Declared, v)
	if s != curScope {
		curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
	}
	return &v.Ref, true
}

// Use a variable
func (s *Scope) Use(ctx *VarCtx, name []byte) *VarRef {
	// check if variable is declared in the current or upper scopes
	v, declaredScope := s.findDeclared(name)
	if v == nil {
		// check if variable is already used before in the current or lower scopes
		v = s.findUndeclared(name)
		if v == nil {
			// add variable to the context list and to the scope's undeclared
			v = ctx.Add(NoDecl, name)
			s.Undeclared = append(s.Undeclared, v)
		}
	} else if declaredScope != s {
		s.Undeclared = append(s.Undeclared, v)
	}
	v.Uses++
	return &v.Ref
}

// find declared variable in the current scope
func (s *Scope) findScopeVar(name []byte) *Var {
	for _, v := range s.Declared {
		if bytes.Equal(name, v.Name) {
			return v
		}
	}
	return nil
}

// find declared variable in the current and upper scopes
func (s *Scope) findDeclared(name []byte) (*Var, *Scope) {
	if v := s.findScopeVar(name); v != nil {
		return v, s
	} else if s.Parent != nil {
		return s.Parent.findDeclared(name)
	}
	return nil, nil
}

// find undeclared variable in the current and lower scopes
func (s *Scope) findUndeclared(name []byte) *Var {
	for _, v := range s.Undeclared {
		if v.Uses != 0 && bytes.Equal(v.Name, name) {
			return v
		}
	}
	return nil
}

func (s *Scope) MarkUndeclaredAsArguments() {
	// set the offset for variables used for arguments, to ensure different b's for: function f(a=b){var b}
	s.argumentsOffset = len(s.Undeclared)
}

func (s *Scope) HoistUndeclared() {
	// copy all undeclared variables to the parent scope
	// TODO: don't add duplicate entries? or remove at the end?
	for _, v := range s.Undeclared {
		if 0 < v.Uses && v.Decl == NoDecl {
			s.Parent.Undeclared = append(s.Parent.Undeclared, v)
		}
	}
}

func (s *Scope) UndeclareScope(ctx *VarCtx) {
	// move all declared variables to the parent scope as undeclared variables. Look if the variable already exists in the parent scope, if so replace the Var pointer in original use
	for _, vorig := range s.Declared {
		name := vorig.Name
		if v, _ := s.Parent.findDeclared(name); v != nil {
			v.Uses++
			ctx.Vars[vorig.Ref] = v
			break
		} else if v = s.Parent.findUndeclared(name); v != nil {
			// check if variable is already used before in the current or lower scopes
			v.Uses++
			ctx.Vars[vorig.Ref] = v
			break
		} else {
			// add variable to the context list and to the scope's undeclared
			vorig.Decl = NoDecl
			s.Parent.Undeclared = append(s.Undeclared, vorig)
		}
	}
}

////////////////////////////////////////////////////////////////

type AST struct {
	List []IStmt
	Ctx  *VarCtx
	Scope
}

func (ast AST) String() string {
	s := ""
	for i, item := range ast.List {
		if i != 0 {
			s += " "
		}
		s += item.String(ast.Ctx)
	}
	return s
}

type IStmt interface {
	String(*VarCtx) string
	stmtNode()
}

type IBinding interface {
	String(*VarCtx) string
	bindingNode()
}

type IExpr interface {
	String(*VarCtx) string
	exprNode()
}

////////////////////////////////////////////////////////////////

type BlockStmt struct {
	List []IStmt
	Scope
}

func (n BlockStmt) String(ctx *VarCtx) string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String(ctx)
	}
	return s + " })"
}

type BranchStmt struct {
	Type  TokenType
	Label []byte // can be nil
}

func (n BranchStmt) String(ctx *VarCtx) string {
	s := "Stmt(" + n.Type.String()
	if n.Label != nil {
		s += " " + string(n.Label)
	}
	return s + ")"
}

type LabelledStmt struct {
	Label []byte
	Value IStmt
}

func (n LabelledStmt) String(ctx *VarCtx) string {
	return "Stmt(" + string(n.Label) + " : " + n.Value.String(ctx) + ")"
}

type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String(ctx *VarCtx) string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String(ctx)
	}
	return s + ")"
}

type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String(ctx *VarCtx) string {
	s := "Stmt(if " + n.Cond.String(ctx) + " " + n.Body.String(ctx)
	if n.Else != nil {
		s += " else " + n.Else.String(ctx)
	}
	return s + ")"
}

type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String(ctx *VarCtx) string {
	return "Stmt(with " + n.Cond.String(ctx) + " " + n.Body.String(ctx) + ")"
}

type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String(ctx *VarCtx) string {
	return "Stmt(do " + n.Body.String(ctx) + " while " + n.Cond.String(ctx) + ")"
}

type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String(ctx *VarCtx) string {
	return "Stmt(while " + n.Cond.String(ctx) + " " + n.Body.String(ctx) + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body IStmt
}

func (n ForStmt) String(ctx *VarCtx) string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String(ctx)
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String(ctx)
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String(ctx)
	}
	return s + " " + n.Body.String(ctx) + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForInStmt) String(ctx *VarCtx) string {
	return "Stmt(for " + n.Init.String(ctx) + " in " + n.Value.String(ctx) + " " + n.Body.String(ctx) + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForOfStmt) String(ctx *VarCtx) string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String(ctx) + " of " + n.Value.String(ctx) + " " + n.Body.String(ctx) + ")"
}

type CaseClause struct {
	TokenType
	Cond IExpr // can be nil
	List []IStmt
}

type SwitchStmt struct {
	Init IExpr
	List []CaseClause
}

func (n SwitchStmt) String(ctx *VarCtx) string {
	s := "Stmt(switch " + n.Init.String(ctx)
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if clause.Cond != nil {
			s += " " + clause.Cond.String(ctx)
		}
		for _, item := range clause.List {
			s += " " + item.String(ctx)
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String(ctx *VarCtx) string {
	return "Stmt(throw " + n.Value.String(ctx) + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding // can be nil
	Catch   BlockStmt
	Finally BlockStmt
}

func (n TryStmt) String(ctx *VarCtx) string {
	s := "Stmt(try " + n.Body.String(ctx)
	if len(n.Catch.List) != 0 || n.Binding != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String(ctx) + ")"
		}
		s += " " + n.Catch.String(ctx)
	}
	if len(n.Finally.List) != 0 {
		s += " finally " + n.Finally.String(ctx)
	}
	return s + ")"
}

type DebuggerStmt struct {
}

func (n DebuggerStmt) String(ctx *VarCtx) string {
	return "Stmt(debugger)"
}

type EmptyStmt struct {
}

func (n EmptyStmt) String(ctx *VarCtx) string {
	return "Stmt(;)"
}

type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String(ctx *VarCtx) string {
	s := ""
	if alias.Name != nil {
		s += string(alias.Name) + " as "
	}
	return s + string(alias.Binding)
}

type ImportStmt struct {
	List    []Alias
	Default []byte // can be nil
	Module  []byte
}

func (n ImportStmt) String(ctx *VarCtx) string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 {
		s += " " + n.List[0].String(ctx)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(ctx)
			}
		}
		s += " }"
	}
	if n.Default != nil || len(n.List) != 0 {
		s += " from"
	}
	return s + " " + string(n.Module) + ")"
}

type ExportStmt struct {
	List    []Alias
	Module  []byte // can be nil
	Default bool
	Decl    IExpr
}

func (n ExportStmt) String(ctx *VarCtx) string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String(ctx) + ")"
	} else if len(n.List) == 1 {
		s += " " + n.List[0].String(ctx)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(ctx)
			}
		}
		s += " }"
	}
	if n.Module != nil {
		s += " from " + string(n.Module)
	}
	return s + ")"
}

type ExprStmt struct {
	Value IExpr
}

func (n ExprStmt) String(ctx *VarCtx) string {
	val := n.Value.String(ctx)
	if val[0] == '(' && val[len(val)-1] == ')' {
		return "Stmt" + n.Value.String(ctx)
	}
	return "Stmt(" + n.Value.String(ctx) + ")"
}

func (n BlockStmt) stmtNode()    {}
func (n BranchStmt) stmtNode()   {}
func (n LabelledStmt) stmtNode() {}
func (n ReturnStmt) stmtNode()   {}
func (n IfStmt) stmtNode()       {}
func (n WithStmt) stmtNode()     {}
func (n DoWhileStmt) stmtNode()  {}
func (n WhileStmt) stmtNode()    {}
func (n ForStmt) stmtNode()      {}
func (n ForInStmt) stmtNode()    {}
func (n ForOfStmt) stmtNode()    {}
func (n SwitchStmt) stmtNode()   {}
func (n ThrowStmt) stmtNode()    {}
func (n TryStmt) stmtNode()      {}
func (n DebuggerStmt) stmtNode() {}
func (n EmptyStmt) stmtNode()    {}
func (n ImportStmt) stmtNode()   {}
func (n ExportStmt) stmtNode()   {}
func (n ExprStmt) stmtNode()     {}

////////////////////////////////////////////////////////////////

type PropertyName struct {
	Literal  LiteralExpr
	Computed IExpr // can be nil
}

func (n PropertyName) IsSet() bool {
	return n.IsComputed() || n.Literal.TokenType != ErrorToken
}

func (n PropertyName) IsComputed() bool {
	return n.Computed != nil
}

func (n PropertyName) IsIdent(data []byte) bool {
	return !n.IsComputed() && n.Literal.TokenType == IdentifierToken && bytes.Equal(data, n.Literal.Data)
}

func (n PropertyName) String(ctx *VarCtx) string {
	if n.Computed != nil {
		val := n.Computed.String(ctx)
		if val[0] == '(' {
			return "[" + val[1:len(val)-1] + "]"
		}
		return "[" + val + "]"
	}
	return string(n.Literal.Data)
}

type BindingArray struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n BindingArray) String(ctx *VarCtx) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String(ctx)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(ctx) + ")"
	}
	return s + " ]"
}

type BindingObjectItem struct {
	Key   PropertyName // can be unset
	Value BindingElement
}

type BindingObject struct {
	List []BindingObjectItem
	Rest *VarRef // can be nil
}

func (n BindingObject) String(ctx *VarCtx) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		if item.Key.IsSet() {
			if ref, ok := item.Value.Binding.(*VarRef); !ok || !item.Key.IsIdent(ref.Get(ctx).Name) {
				s += " " + item.Key.String(ctx) + ":"
			}
		}
		s += " " + item.Value.String(ctx)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(ctx) + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding // can be nil
	Default IExpr    // can be nil
}

func (n BindingElement) String(ctx *VarCtx) string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String(ctx)
	if n.Default != nil {
		s += " = " + n.Default.String(ctx)
	}
	return s + ")"
}

func (n VarRef) bindingNode()        {}
func (n BindingArray) bindingNode()  {}
func (n BindingObject) bindingNode() {}

////////////////////////////////////////////////////////////////

type Params struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n Params) String(ctx *VarCtx) string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ctx)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String(ctx) + ")"
	}
	return s + ")"
}

type Arguments struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n Arguments) String(ctx *VarCtx) string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ctx)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.String(ctx)
	}
	return s + ")"
}

type VarDecl struct {
	TokenType
	List []BindingElement
}

func (n VarDecl) String(ctx *VarCtx) string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String(ctx)
	}
	return s + ")"
}

type FuncDecl struct {
	Async     bool
	Generator bool
	Name      *VarRef // can be nil
	Params    Params
	Body      BlockStmt
	Scope
}

func (n FuncDecl) String(ctx *VarCtx) string {
	s := "Decl("
	if n.Async {
		s += "async function"
	} else {
		s += "function"
	}
	if n.Generator {
		s += "*"
	}
	if n.Name != nil {
		s += " " + n.Name.String(ctx)
	}
	return s + " " + n.Params.String(ctx) + " " + n.Body.String(ctx) + ")"
}

type MethodDecl struct {
	Static    bool
	Async     bool
	Generator bool
	Get       bool
	Set       bool
	Name      PropertyName
	Params    Params
	Body      BlockStmt
	Scope
}

func (n MethodDecl) String(ctx *VarCtx) string {
	s := ""
	if n.Static {
		s += " static"
	}
	if n.Async {
		s += " async"
	}
	if n.Generator {
		s += " *"
	}
	if n.Get {
		s += " get"
	}
	if n.Set {
		s += " set"
	}
	s += " " + n.Name.String(ctx) + " " + n.Params.String(ctx) + " " + n.Body.String(ctx)
	return "Method(" + s[1:] + ")"
}

type ClassDecl struct {
	Name    *VarRef // can be nil
	Extends IExpr   // can be nil
	Methods []MethodDecl
}

func (n ClassDecl) String(ctx *VarCtx) string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + n.Name.String(ctx)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String(ctx)
	}
	for _, item := range n.Methods {
		s += " " + item.String(ctx)
	}
	return s + ")"
}

func (n VarDecl) stmtNode()   {}
func (n FuncDecl) stmtNode()  {}
func (n ClassDecl) stmtNode() {}

func (n VarDecl) exprNode()    {} // not a real IExpr, used for ForInit and ExportDecl
func (n FuncDecl) exprNode()   {}
func (n ClassDecl) exprNode()  {}
func (n MethodDecl) exprNode() {} // not a real IExpr, used for ObjectExpression PropertyName

////////////////////////////////////////////////////////////////

type GroupExpr struct {
	X IExpr
}

func (n GroupExpr) String(ctx *VarCtx) string {
	return "(" + n.X.String(ctx) + ")"
}

type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String(ctx *VarCtx) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String(ctx)
		}
	}
	if 0 < len(n.List) && n.List[len(n.List)-1].Value == nil {
		s += ","
	}
	return s + "]"
}

type Property struct {
	// either Name or Spread are set. When Spread is set then Value is AssignmentExpression
	// if Init is set then Value is IdentifierReference, otherwise it can also be MethodDefinition
	Name   PropertyName // can be unset
	Spread bool
	Value  IExpr
	Init   IExpr // can be nil
}

func (n Property) String(ctx *VarCtx) string {
	s := ""
	if n.Name.IsSet() {
		if ref, ok := n.Value.(*VarRef); !ok || !n.Name.IsIdent(ref.Get(ctx).Name) {
			s += n.Name.String(ctx) + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String(ctx)
	if n.Init != nil {
		s += " = " + n.Init.String(ctx)
	}
	return s
}

type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String(ctx *VarCtx) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ctx)
	}
	return s + "}"
}

type TemplatePart struct {
	Value []byte
	Expr  IExpr
}

type TemplateExpr struct {
	Tag  IExpr // can be nil
	List []TemplatePart
	Tail []byte
}

func (n TemplateExpr) String(ctx *VarCtx) string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String(ctx)
	}
	for _, item := range n.List {
		s += string(item.Value) + item.Expr.String(ctx)
	}
	return s + string(n.Tail)
}

type NewExpr struct {
	X    IExpr
	Args *Arguments // can be nil
}

func (n NewExpr) String(ctx *VarCtx) string {
	if n.Args != nil {
		return "(new " + n.X.String(ctx) + n.Args.String(ctx) + ")"
	}
	return "(new " + n.X.String(ctx) + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String(ctx *VarCtx) string {
	return "(new.target)"
}

type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String(ctx *VarCtx) string {
	return "(import.meta)"
}

type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String(ctx *VarCtx) string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String(ctx) + ")"
}

type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String(ctx *VarCtx) string {
	return "(" + n.Cond.String(ctx) + " ? " + n.X.String(ctx) + " : " + n.Y.String(ctx) + ")"
}

type DotExpr struct {
	X IExpr
	Y LiteralExpr
}

func (n DotExpr) String(ctx *VarCtx) string {
	return "(" + n.X.String(ctx) + "." + n.Y.String(ctx) + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String(ctx *VarCtx) string {
	return "(" + n.X.String(ctx) + n.Args.String(ctx) + ")"
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
}

func (n IndexExpr) String(ctx *VarCtx) string {
	return "(" + n.X.String(ctx) + "[" + n.Index.String(ctx) + "])"
}

type OptChainExpr struct {
	X IExpr
	Y IExpr // can be CallExpr, IndexExpr, LiteralExpr, or TemplateExpr
}

func (n OptChainExpr) String(ctx *VarCtx) string {
	s := "(" + n.X.String(ctx) + "?."
	switch y := n.Y.(type) {
	case *CallExpr:
		return s + y.Args.String(ctx) + ")"
	case *IndexExpr:
		return s + "[" + y.Index.String(ctx) + "])"
	default:
		return s + y.String(ctx) + ")"
	}
}

type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String(ctx *VarCtx) string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String(ctx) + n.Op.String() + ")"
	} else if IsIdentifierName(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String(ctx) + ")"
	}
	return "(" + n.Op.String() + n.X.String(ctx) + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String(ctx *VarCtx) string {
	if IsIdentifierName(n.Op) {
		return "(" + n.X.String(ctx) + " " + n.Op.String() + " " + n.Y.String(ctx) + ")"
	}
	return "(" + n.X.String(ctx) + n.Op.String() + n.Y.String(ctx) + ")"
}

type LiteralExpr struct {
	TokenType
	Data []byte
}

func (n LiteralExpr) String(ctx *VarCtx) string {
	return string(n.Data)
}

type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
	Scope
}

func (n ArrowFunc) String(ctx *VarCtx) string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String(ctx) + " => " + n.Body.String(ctx) + ")"
}

func (n GroupExpr) exprNode()      {}
func (n ArrayExpr) exprNode()      {}
func (n ObjectExpr) exprNode()     {}
func (n TemplateExpr) exprNode()   {}
func (n NewExpr) exprNode()        {}
func (n NewTargetExpr) exprNode()  {}
func (n ImportMetaExpr) exprNode() {}
func (n YieldExpr) exprNode()      {}
func (n CondExpr) exprNode()       {}
func (n DotExpr) exprNode()        {}
func (n CallExpr) exprNode()       {}
func (n IndexExpr) exprNode()      {}
func (n OptChainExpr) exprNode()   {}
func (n UnaryExpr) exprNode()      {}
func (n BinaryExpr) exprNode()     {}
func (n LiteralExpr) exprNode()    {}
func (n ArrowFunc) exprNode()      {}
func (n VarRef) exprNode()         {}
