package js

import (
	"bytes"
	"fmt"
	"strconv"
	"unsafe"
)

type DeclType uint16

const (
	NoDecl           DeclType = iota // unbound variables
	ArgumentNoDecl                   // unbound variable used in argument so that f(a=b){var b} refers to different b's
	VariableDecl                     // var and function
	LexicalDecl                      // let, const, class
	ArgumentDecl                     // function arguments and catch statement argument
	FuncExprNameDecl                 // function expression name
)

func (decl DeclType) String() string {
	switch decl {
	case NoDecl:
		return "NoDecl"
	case ArgumentNoDecl:
		return "ArgumentNoDecl"
	case VariableDecl:
		return "Variable"
	case LexicalDecl:
		return "Lexical"
	case ArgumentDecl:
		return "Argument"
	case FuncExprNameDecl:
		return "FuncExprName"
	}
	return "Invalid(" + strconv.Itoa(int(decl)) + ")"
}

// TODO: use
//length uint16
//offset uint32
//
//end := v.offset + uint32(v.length)
//return c.src[v.offset:end:end]
//
//offset := uintptr(unsafe.Pointer(&data[0])) - uintptr(unsafe.Pointer(&c.src[0]))
//if math.MaxUint32 < uint64(offset) || math.MaxUint16 < len(data) || math.MaxUint32 == len(c.list) {
//	// TODO: or make uints bigger
//	panic("variable name too long")
//}
//uint16(len(data)), uint32(offset)

// VarRef is an index into VarCtx.vars and is used by the AST to refer to a variable
// The chain of pointers: VarRef --(idx)--> VarArray --(ptr)--> Var --([]byte)--> data
type VarRef uint32 // *VarRef is faster than VarRef

func (ref *VarRef) Get(ctx *VarCtx) *Var {
	return ctx.vars[*ref]
}

func (ref *VarRef) String(ctx *VarCtx) string {
	return string(ctx.vars[*ref].Name)
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

func init() {
	fmt.Println("Var:", unsafe.Sizeof(Var{}))
}

// Var is a variable, where Decl is the type of declaration and can be var|function for function scoped variables, let|const|class for block scoped variables
type Var struct {
	Ref  VarRef
	Uses uint32
	Decl DeclType

	IsRenamed bool
	Name      []byte
	OrigName  []byte
}

func (v *Var) String() string {
	name := string(v.Name)
	if !bytes.Equal(v.Name, v.OrigName) {
		name = string(v.OrigName) + "=>" + name
	}
	return fmt.Sprintf("Var{%v %v %v %v %s}", v.Ref, v.Uses, v.Decl, v.IsRenamed, name)
}

// VarCtx holds the context needed for variable identifiers. It holds a list of all variables to which VarRef is indexing.
type VarCtx struct {
	src  []byte
	vars VarArray
}

func NewVarCtx(src []byte) *VarCtx {
	return &VarCtx{
		src:  src,
		vars: VarArray{nil},
	}
}

func (ctx *VarCtx) Add(decl DeclType, data []byte) *Var {
	v := &Var{VarRef(len(ctx.vars)), 0, decl, false, data, data}
	ctx.vars = append(ctx.vars, v)
	return v
}

func (ctx *VarCtx) String() string {
	s := "["
	for i, v := range ctx.vars {
		if i != 0 {
			s += ", "
		}
		s += v.String()
	}
	return s + "]"
}

// Scope is a function or block scope with a list of variables declared and used
// TODO: handle with statement and eval function calls in scope
type Scope struct {
	Parent, Func    *Scope
	Declared        VarArray // TODO: merge with others?
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
		if (v.Decl == LexicalDecl || decl == LexicalDecl) && v.Decl != FuncExprNameDecl {
			// redeclaration of let, const, class on an already declared name is an error, except if the declared name is a function expression name
			return nil, false
		}
		if v.Decl == FuncExprNameDecl {
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
				//v.Uses += uv.Uses
				//uv.Uses = 0          // remove from undeclared
				//ctx.vars[uv.Ref] = v // point undeclared variable reference to the new declared variable
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
	//s.Undeclared = s.Undeclared[:0]
}

func (s *Scope) UndeclareScope(ctx *VarCtx) {
	// move all declared variables to the parent scope as undeclared variables. Look if the variable already exists in the parent scope, if so replace the Var pointer in original use as it will still be referenced.
	for _, vorig := range s.Declared {
		name := vorig.Name
		if v, _ := s.Parent.findDeclared(name); v != nil {
			v.Uses++
			ctx.vars[vorig.Ref] = v
			break
		} else if v = s.Parent.findUndeclared(name); v != nil {
			// check if variable is already used before in the current or lower scopes
			v.Uses++
			ctx.vars[vorig.Ref] = v
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

func (n BlockStmt) String(c *VarCtx) string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String(c)
	}
	return s + " })"
}

type BranchStmt struct {
	Type  TokenType
	Label []byte // can be nil
}

func (n BranchStmt) String(c *VarCtx) string {
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

func (n LabelledStmt) String(c *VarCtx) string {
	return "Stmt(" + string(n.Label) + " : " + n.Value.String(c) + ")"
}

type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String(c *VarCtx) string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String(c)
	}
	return s + ")"
}

type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String(c *VarCtx) string {
	s := "Stmt(if " + n.Cond.String(c) + " " + n.Body.String(c)
	if n.Else != nil {
		s += " else " + n.Else.String(c)
	}
	return s + ")"
}

type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String(c *VarCtx) string {
	return "Stmt(with " + n.Cond.String(c) + " " + n.Body.String(c) + ")"
}

type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String(c *VarCtx) string {
	return "Stmt(do " + n.Body.String(c) + " while " + n.Cond.String(c) + ")"
}

type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String(c *VarCtx) string {
	return "Stmt(while " + n.Cond.String(c) + " " + n.Body.String(c) + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body IStmt
}

func (n ForStmt) String(c *VarCtx) string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String(c)
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String(c)
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String(c)
	}
	return s + " " + n.Body.String(c) + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForInStmt) String(c *VarCtx) string {
	return "Stmt(for " + n.Init.String(c) + " in " + n.Value.String(c) + " " + n.Body.String(c) + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForOfStmt) String(c *VarCtx) string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String(c) + " of " + n.Value.String(c) + " " + n.Body.String(c) + ")"
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

func (n SwitchStmt) String(c *VarCtx) string {
	s := "Stmt(switch " + n.Init.String(c)
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if clause.Cond != nil {
			s += " " + clause.Cond.String(c)
		}
		for _, item := range clause.List {
			s += " " + item.String(c)
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String(c *VarCtx) string {
	return "Stmt(throw " + n.Value.String(c) + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding // can be nil
	Catch   BlockStmt
	Finally BlockStmt
}

func (n TryStmt) String(c *VarCtx) string {
	s := "Stmt(try " + n.Body.String(c)
	if len(n.Catch.List) != 0 || n.Binding != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String(c) + ")"
		}
		s += " " + n.Catch.String(c)
	}
	if len(n.Finally.List) != 0 {
		s += " finally " + n.Finally.String(c)
	}
	return s + ")"
}

type DebuggerStmt struct {
}

func (n DebuggerStmt) String(c *VarCtx) string {
	return "Stmt(debugger)"
}

type EmptyStmt struct {
}

func (n EmptyStmt) String(c *VarCtx) string {
	return "Stmt(;)"
}

type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String(c *VarCtx) string {
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

func (n ImportStmt) String(c *VarCtx) string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 {
		s += " " + n.List[0].String(c)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(c)
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

func (n ExportStmt) String(c *VarCtx) string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String(c) + ")"
	} else if len(n.List) == 1 {
		s += " " + n.List[0].String(c)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(c)
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

func (n ExprStmt) String(c *VarCtx) string {
	val := n.Value.String(c)
	if val[0] == '(' && val[len(val)-1] == ')' {
		return "Stmt" + n.Value.String(c)
	}
	return "Stmt(" + n.Value.String(c) + ")"
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

func (n PropertyName) String(c *VarCtx) string {
	if n.Computed != nil {
		val := n.Computed.String(c)
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

func (n BindingArray) String(c *VarCtx) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String(c)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(c) + ")"
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

func (n BindingObject) String(c *VarCtx) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		if item.Key.IsSet() {
			if ref, ok := item.Value.Binding.(*VarRef); !ok || !item.Key.IsIdent(ref.Get(c).Name) {
				s += " " + item.Key.String(c) + ":"
			}
		}
		s += " " + item.Value.String(c)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(c) + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding // can be nil
	Default IExpr    // can be nil
}

func (n BindingElement) String(c *VarCtx) string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String(c)
	if n.Default != nil {
		s += " = " + n.Default.String(c)
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

func (n Params) String(c *VarCtx) string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(c)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String(c) + ")"
	}
	return s + ")"
}

type Arguments struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n Arguments) String(c *VarCtx) string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(c)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.String(c)
	}
	return s + ")"
}

type VarDecl struct {
	TokenType
	List []BindingElement
}

func (n VarDecl) String(c *VarCtx) string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String(c)
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

func (n FuncDecl) String(c *VarCtx) string {
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
		s += " " + n.Name.String(c)
	}
	return s + " " + n.Params.String(c) + " " + n.Body.String(c) + ")"
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

func (n MethodDecl) String(c *VarCtx) string {
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
	s += " " + n.Name.String(c) + " " + n.Params.String(c) + " " + n.Body.String(c)
	return "Method(" + s[1:] + ")"
}

type ClassDecl struct {
	Name    *VarRef // can be nil
	Extends IExpr   // can be nil
	Methods []MethodDecl
}

func (n ClassDecl) String(c *VarCtx) string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + n.Name.String(c)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String(c)
	}
	for _, item := range n.Methods {
		s += " " + item.String(c)
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

func (n GroupExpr) String(c *VarCtx) string {
	return "(" + n.X.String(c) + ")"
}

type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String(c *VarCtx) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String(c)
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

func (n Property) String(c *VarCtx) string {
	s := ""
	if n.Name.IsSet() {
		if ref, ok := n.Value.(*VarRef); !ok || !n.Name.IsIdent(ref.Get(c).Name) {
			s += n.Name.String(c) + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String(c)
	if n.Init != nil {
		s += " = " + n.Init.String(c)
	}
	return s
}

type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String(c *VarCtx) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(c)
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

func (n TemplateExpr) String(c *VarCtx) string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String(c)
	}
	for _, item := range n.List {
		s += string(item.Value) + item.Expr.String(c)
	}
	return s + string(n.Tail)
}

type NewExpr struct {
	X    IExpr
	Args *Arguments // can be nil
}

func (n NewExpr) String(c *VarCtx) string {
	if n.Args != nil {
		return "(new " + n.X.String(c) + n.Args.String(c) + ")"
	}
	return "(new " + n.X.String(c) + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String(c *VarCtx) string {
	return "(new.target)"
}

type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String(c *VarCtx) string {
	return "(import.meta)"
}

type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String(c *VarCtx) string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String(c) + ")"
}

type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String(c *VarCtx) string {
	return "(" + n.Cond.String(c) + " ? " + n.X.String(c) + " : " + n.Y.String(c) + ")"
}

type DotExpr struct {
	X IExpr
	Y LiteralExpr
}

func (n DotExpr) String(c *VarCtx) string {
	return "(" + n.X.String(c) + "." + n.Y.String(c) + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String(c *VarCtx) string {
	return "(" + n.X.String(c) + n.Args.String(c) + ")"
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
}

func (n IndexExpr) String(c *VarCtx) string {
	return "(" + n.X.String(c) + "[" + n.Index.String(c) + "])"
}

type OptChainExpr struct {
	X IExpr
	Y IExpr // can be CallExpr, IndexExpr, LiteralExpr, or TemplateExpr
}

func (n OptChainExpr) String(c *VarCtx) string {
	s := "(" + n.X.String(c) + "?."
	switch y := n.Y.(type) {
	case *CallExpr:
		return s + y.Args.String(c) + ")"
	case *IndexExpr:
		return s + "[" + y.Index.String(c) + "])"
	default:
		return s + y.String(c) + ")"
	}
}

type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String(c *VarCtx) string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String(c) + n.Op.String() + ")"
	} else if IsIdentifierName(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String(c) + ")"
	}
	return "(" + n.Op.String() + n.X.String(c) + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String(c *VarCtx) string {
	if IsIdentifierName(n.Op) {
		return "(" + n.X.String(c) + " " + n.Op.String() + " " + n.Y.String(c) + ")"
	}
	return "(" + n.X.String(c) + n.Op.String() + n.Y.String(c) + ")"
}

type LiteralExpr struct {
	TokenType
	Data []byte
}

func (n LiteralExpr) String(c *VarCtx) string {
	return string(n.Data)
}

type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
	Scope
}

func (n ArrowFunc) String(c *VarCtx) string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String(c) + " => " + n.Body.String(c) + ")"
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
