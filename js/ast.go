package js

import (
	"bytes"
	"fmt"
	"strconv"
)

type AST struct {
	Comment   []byte // first comment in file
	BlockStmt        // module

	// Vars holds a list of all variables to which VarRef is indexing
	Vars VarArray
}

func newAST() *AST {
	return &AST{
		Vars: VarArray{nil}, // VarRef(0) => nil
	}
}

func (ast *AST) AddVar(decl DeclType, name []byte) *Var {
	v := &Var{VarRef(len(ast.Vars)), nil, 0, decl, name}
	ast.Vars = append(ast.Vars, v)
	return v
}

func (ast *AST) String() string {
	s := ""
	for i, item := range ast.BlockStmt.List {
		if i != 0 {
			s += " "
		}
		s += item.String(ast)
	}
	return s
}

////////////////////////////////////////////////////////////////

type DeclType uint16

const (
	NoDecl       DeclType = iota // undeclared variables
	VariableDecl                 // var
	FunctionDecl                 // function
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
	case FunctionDecl:
		return "FunctionDecl"
	case LexicalDecl:
		return "LexicalDecl"
	case ArgumentDecl:
		return "ArgumentDecl"
	case ExprDecl:
		return "ExprDecl"
	}
	return "Invalid(" + strconv.Itoa(int(decl)) + ")"
}

// VarRef is an index into AST.Vars and is used by the AST to refer to a variable
// The chain of pointers: VarRef --(idx in Vars)--> *Var --(ptr)--> Var
type VarRef uint32 // fits as value in interface

func (ref VarRef) Var(ast *AST) *Var {
	v := ast.Vars[ref]
	for v.Link != nil {
		v = v.Link
	}
	return v
}

func (ref VarRef) Name(ast *AST) []byte {
	return ref.Var(ast).Name
}

func (ref VarRef) String(ast *AST) string {
	return string(ref.Var(ast).Name)
}

// Var is a variable, where Decl is the type of declaration and can be var|function for function scoped variables, let|const|class for block scoped variables
type Var struct {
	Ref  VarRef
	Link *Var
	Uses uint16
	Decl DeclType
	Name []byte
}

func (v *Var) String() string {
	for v.Link != nil {
		v = v.Link
	}
	return fmt.Sprintf("Var{%v %v %v %s}", v.Ref, v.Uses, v.Decl, string(v.Name))
}

// VarsByUses is sortable by uses in descending order
type VarsByUses VarArray

func (vs VarsByUses) Len() int {
	return len(vs)
}

func (vs VarsByUses) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (vs VarsByUses) Less(i, j int) bool {
	return vs[i].Uses > vs[j].Uses
}

////////////////////////////////////////////////////////////////

type VarArray []*Var

func (vs VarArray) String() string {
	s := "["
	for i, item := range vs {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	return s + "]"
}

// Scope is a function or block scope with a list of variables declared and used
// TODO: handle with statement and eval function calls in scope?
// TODO: don't add used statement to scope when defining a function or var (without define) in a block
type Scope struct {
	Parent, Func    *Scope
	Declared        VarArray
	Undeclared      VarArray
	NVarDecls       int // number of variable declaration statements in a function scope
	argumentsOffset int // offset into Undeclared to mark variables used in arguments initializers
}

func (s Scope) String() string {
	return "Scope{Declared: " + s.Declared.String() + ", Undeclared: " + s.Undeclared.String() + "}"
}

// Declare a new variable
func (s *Scope) Declare(ast *AST, decl DeclType, name []byte) (VarRef, bool) {
	// refer to new variable for previously undeclared symbols in the current and lower scopes
	// this happens in `{ a = 5; } var a` where both a's refer to the same variable
	curScope := s
	if decl == VariableDecl || decl == FunctionDecl {
		// find function scope for var and function declarations
		s = s.Func
	}

	if v := s.findDeclared(name); v != nil {
		// variable already declared, might be an error or a duplicate declaration
		if (v.Decl == LexicalDecl || decl == LexicalDecl) && v.Decl != ExprDecl {
			// redeclaration of let, const, class on an already declared name is an error, except if the declared name is a function expression name
			return 0, false
		}
		if v.Decl == ExprDecl {
			v.Decl = decl
		}
		v.Uses++
		if s != curScope {
			curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
		}
		return v.Ref, true
	}

	var v *Var
	if decl != ArgumentDecl { // in case of function f(a=b,b), where the first b is different from the second
		for i, uv := range s.Undeclared[s.argumentsOffset:] {
			if 0 < uv.Uses && bytes.Equal(name, uv.Name) {
				v = uv
				s.Undeclared = append(s.Undeclared[:s.argumentsOffset+i], s.Undeclared[s.argumentsOffset+i+1:]...)
				break
			}
		}
	}
	if v == nil {
		// add variable to the context list and to the scope
		v = ast.AddVar(decl, name)
	} else {
		v.Decl = decl
	}
	v.Uses++
	s.Declared = append(s.Declared, v)
	if s != curScope {
		curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
	}
	return v.Ref, true
}

// Use a variable
func (s *Scope) Use(ast *AST, name []byte) VarRef {
	// check if variable is declared in the current scope
	v := s.findDeclared(name)
	if v == nil {
		// check if variable is already used before in the current or lower scopes
		v = s.findUndeclared(name)
		if v == nil {
			// add variable to the context list and to the scope's undeclared
			v = ast.AddVar(NoDecl, name)
			s.Undeclared = append(s.Undeclared, v)
		}
	}
	v.Uses++
	return v.Ref
}

// find declared variable in the current scope
func (s *Scope) findDeclared(name []byte) *Var {
	for _, v := range s.Declared {
		if bytes.Equal(name, v.Name) {
			return v
		}
	}
	return nil
}

// find declared variable in the current and upper scopes
func (s *Scope) findVarDeclaration(name []byte) (*Var, *Scope) {
	if v := s.findDeclared(name); v != nil {
		return v, s
	} else if s.Parent != nil {
		return s.Parent.findVarDeclaration(name)
	}
	return nil, nil
}

// find undeclared variable in the current and lower scopes
func (s *Scope) findUndeclared(name []byte) *Var {
	for _, v := range s.Undeclared {
		if 0 < v.Uses && bytes.Equal(name, v.Name) {
			return v
		}
	}
	return nil
}

func (s *Scope) MarkArguments() {
	// set the offset for variables used for arguments, to ensure different b's for: function f(a=b){var b}
	s.argumentsOffset = len(s.Undeclared)
}

func (s *Scope) HoistUndeclared() {
	// copy all undeclared variables to the parent scope
	for _, vorig := range s.Undeclared {
		if 0 < vorig.Uses && vorig.Decl == NoDecl {
			if v := s.Parent.findDeclared(vorig.Name); v != nil {
				// check if variable is declared in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				//ast.Vars[vorig.Ref] = v // point reference to existing var
			} else if v := s.Parent.findUndeclared(vorig.Name); v != nil {
				// check if variable is already used before in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				//ast.Vars[vorig.Ref] = v // point reference to existing var
			} else {
				// add variable to the context list and to the scope's undeclared
				s.Parent.Undeclared = append(s.Parent.Undeclared, vorig)
			}
		}
	}
}

func (s *Scope) UndeclareScope() {
	// called when possibly arrow func ends up being a parenthesized expression, scope not futher used
	// move all declared variables to the parent scope as undeclared variables. Look if the variable already exists in the parent scope, if so replace the Var pointer in original use
	// TODO; remove new vars, pass difference to this function?
	for _, vorig := range s.Declared {
		if v, _ := s.Parent.findVarDeclaration(vorig.Name); v != nil {
			// check if variable has been declared in this scope
			v.Uses += vorig.Uses
			vorig.Link = v
			//ast.Vars[vorig.Ref] = v // point reference to existing var
		} else if v := s.Parent.findUndeclared(vorig.Name); v != nil {
			// check if variable is already used before in the current or lower scopes
			v.Uses += vorig.Uses
			vorig.Link = v
			//ast.Vars[vorig.Ref] = v // point reference to existing var
		} else {
			// add variable to the context list and to the scope's undeclared
			vorig.Decl = NoDecl
			s.Parent.Undeclared = append(s.Parent.Undeclared, vorig)
		}
	}
}

////////////////////////////////////////////////////////////////

type IStmt interface {
	String(*AST) string
	stmtNode()
}

type IBinding interface {
	String(*AST) string
	bindingNode()
}

type IExpr interface {
	String(*AST) string
	exprNode()
}

////////////////////////////////////////////////////////////////

type BlockStmt struct {
	List []IStmt
	Scope
}

func (n BlockStmt) String(ast *AST) string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String(ast)
	}
	return s + " })"
}

type BranchStmt struct {
	Type  TokenType
	Label []byte // can be nil
}

func (n BranchStmt) String(ast *AST) string {
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

func (n LabelledStmt) String(ast *AST) string {
	return "Stmt(" + string(n.Label) + " : " + n.Value.String(ast) + ")"
}

type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String(ast *AST) string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String(ast)
	}
	return s + ")"
}

type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String(ast *AST) string {
	s := "Stmt(if " + n.Cond.String(ast) + " " + n.Body.String(ast)
	if n.Else != nil {
		s += " else " + n.Else.String(ast)
	}
	return s + ")"
}

type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String(ast *AST) string {
	return "Stmt(with " + n.Cond.String(ast) + " " + n.Body.String(ast) + ")"
}

type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String(ast *AST) string {
	return "Stmt(do " + n.Body.String(ast) + " while " + n.Cond.String(ast) + ")"
}

type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String(ast *AST) string {
	return "Stmt(while " + n.Cond.String(ast) + " " + n.Body.String(ast) + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body IStmt
}

func (n ForStmt) String(ast *AST) string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String(ast)
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String(ast)
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String(ast)
	}
	return s + " " + n.Body.String(ast) + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForInStmt) String(ast *AST) string {
	return "Stmt(for " + n.Init.String(ast) + " in " + n.Value.String(ast) + " " + n.Body.String(ast) + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForOfStmt) String(ast *AST) string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String(ast) + " of " + n.Value.String(ast) + " " + n.Body.String(ast) + ")"
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

func (n SwitchStmt) String(ast *AST) string {
	s := "Stmt(switch " + n.Init.String(ast)
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if clause.Cond != nil {
			s += " " + clause.Cond.String(ast)
		}
		for _, item := range clause.List {
			s += " " + item.String(ast)
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String(ast *AST) string {
	return "Stmt(throw " + n.Value.String(ast) + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding   // can be nil
	Catch   *BlockStmt // can be nil
	Finally *BlockStmt // can be nil
}

func (n TryStmt) String(ast *AST) string {
	s := "Stmt(try " + n.Body.String(ast)
	if n.Catch != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String(ast) + ")"
		}
		s += " " + n.Catch.String(ast)
	}
	if n.Finally != nil {
		s += " finally " + n.Finally.String(ast)
	}
	return s + ")"
}

type DebuggerStmt struct {
}

func (n DebuggerStmt) String(ast *AST) string {
	return "Stmt(debugger)"
}

type EmptyStmt struct {
}

func (n EmptyStmt) String(ast *AST) string {
	return "Stmt(;)"
}

type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String(ast *AST) string {
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

func (n ImportStmt) String(ast *AST) string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 {
		s += " " + n.List[0].String(ast)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(ast)
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

func (n ExportStmt) String(ast *AST) string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String(ast) + ")"
	} else if len(n.List) == 1 {
		s += " " + n.List[0].String(ast)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(ast)
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

func (n ExprStmt) String(ast *AST) string {
	val := n.Value.String(ast)
	if val[0] == '(' && val[len(val)-1] == ')' {
		return "Stmt" + n.Value.String(ast)
	}
	return "Stmt(" + n.Value.String(ast) + ")"
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

func (n PropertyName) String(ast *AST) string {
	if n.Computed != nil {
		val := n.Computed.String(ast)
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

func (n BindingArray) String(ast *AST) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String(ast)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(ast) + ")"
	}
	return s + " ]"
}

type BindingObjectItem struct {
	Key   *PropertyName // can be nil
	Value BindingElement
}

type BindingObject struct {
	List []BindingObjectItem
	Rest VarRef // can be nil
}

func (n BindingObject) String(ast *AST) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		if item.Key != nil {
			if ref, ok := item.Value.Binding.(VarRef); !ok || !item.Key.IsIdent(ref.Name(ast)) {
				s += " " + item.Key.String(ast) + ":"
			}
		}
		s += " " + item.Value.String(ast)
	}
	if n.Rest != 0 {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(ast) + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding // can be nil (in case of ellision)
	Default IExpr    // can be nil
}

func (n BindingElement) String(ast *AST) string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String(ast)
	if n.Default != nil {
		s += " = " + n.Default.String(ast)
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

func (n Params) String(ast *AST) string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ast)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String(ast) + ")"
	}
	return s + ")"
}

type Arguments struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n Arguments) String(ast *AST) string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ast)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.String(ast)
	}
	return s + ")"
}

type VarDecl struct {
	TokenType
	List []BindingElement
}

func (n VarDecl) String(ast *AST) string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String(ast)
	}
	return s + ")"
}

type FuncDecl struct {
	Async     bool
	Generator bool
	Name      VarRef // can be nil
	Params    Params
	Body      BlockStmt
}

func (n FuncDecl) String(ast *AST) string {
	s := "Decl("
	if n.Async {
		s += "async function"
	} else {
		s += "function"
	}
	if n.Generator {
		s += "*"
	}
	if n.Name != 0 {
		s += " " + n.Name.String(ast)
	}
	return s + " " + n.Params.String(ast) + " " + n.Body.String(ast) + ")"
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
}

func (n MethodDecl) String(ast *AST) string {
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
	s += " " + n.Name.String(ast) + " " + n.Params.String(ast) + " " + n.Body.String(ast)
	return "Method(" + s[1:] + ")"
}

type ClassDecl struct {
	Name    VarRef // can be nil
	Extends IExpr  // can be nil
	Methods []MethodDecl
}

func (n ClassDecl) String(ast *AST) string {
	s := "Decl(class"
	if n.Name != 0 {
		s += " " + n.Name.String(ast)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String(ast)
	}
	for _, item := range n.Methods {
		s += " " + item.String(ast)
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

func (n GroupExpr) String(ast *AST) string {
	return "(" + n.X.String(ast) + ")"
}

type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String(ast *AST) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String(ast)
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
	Name   *PropertyName // can be nil
	Spread bool
	Value  IExpr
	Init   IExpr // can be nil
}

func (n Property) String(ast *AST) string {
	s := ""
	if n.Name != nil {
		if ref, ok := n.Value.(VarRef); !ok || !n.Name.IsIdent(ref.Name(ast)) {
			s += n.Name.String(ast) + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String(ast)
	if n.Init != nil {
		s += " = " + n.Init.String(ast)
	}
	return s
}

type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String(ast *AST) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(ast)
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
	Prec OpPrec
}

func (n TemplateExpr) String(ast *AST) string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String(ast)
	}
	for _, item := range n.List {
		s += string(item.Value) + item.Expr.String(ast)
	}
	return s + string(n.Tail)
}

type NewExpr struct {
	X    IExpr
	Args *Arguments // can be nil
}

func (n NewExpr) String(ast *AST) string {
	if n.Args != nil {
		return "(new " + n.X.String(ast) + n.Args.String(ast) + ")"
	}
	return "(new " + n.X.String(ast) + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String(ast *AST) string {
	return "(new.target)"
}

type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String(ast *AST) string {
	return "(import.meta)"
}

type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String(ast *AST) string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String(ast) + ")"
}

type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String(ast *AST) string {
	return "(" + n.Cond.String(ast) + " ? " + n.X.String(ast) + " : " + n.Y.String(ast) + ")"
}

type DotExpr struct {
	X    IExpr
	Y    LiteralExpr
	Prec OpPrec
}

func (n DotExpr) String(ast *AST) string {
	return "(" + n.X.String(ast) + "." + n.Y.String(ast) + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String(ast *AST) string {
	return "(" + n.X.String(ast) + n.Args.String(ast) + ")"
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
	Prec  OpPrec
}

func (n IndexExpr) String(ast *AST) string {
	return "(" + n.X.String(ast) + "[" + n.Index.String(ast) + "])"
}

type OptChainExpr struct {
	X IExpr
	Y IExpr // can be CallExpr, IndexExpr, LiteralExpr, or TemplateExpr
}

func (n OptChainExpr) String(ast *AST) string {
	s := "(" + n.X.String(ast) + "?."
	switch y := n.Y.(type) {
	case *CallExpr:
		return s + y.Args.String(ast) + ")"
	case *IndexExpr:
		return s + "[" + y.Index.String(ast) + "])"
	default:
		return s + y.String(ast) + ")"
	}
}

type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String(ast *AST) string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String(ast) + n.Op.String() + ")"
	} else if IsIdentifierName(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String(ast) + ")"
	}
	return "(" + n.Op.String() + n.X.String(ast) + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String(ast *AST) string {
	if IsIdentifierName(n.Op) {
		return "(" + n.X.String(ast) + " " + n.Op.String() + " " + n.Y.String(ast) + ")"
	}
	return "(" + n.X.String(ast) + n.Op.String() + n.Y.String(ast) + ")"
}

type LiteralExpr struct {
	TokenType
	Data []byte
}

func (n LiteralExpr) String(ast *AST) string {
	return string(n.Data)
}

type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
}

func (n ArrowFunc) String(ast *AST) string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String(ast) + " => " + n.Body.String(ast) + ")"
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
