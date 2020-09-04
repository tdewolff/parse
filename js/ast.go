package js

import (
	"bytes"
	"fmt"
	"strconv"
)

type AST struct {
	Comments  [][]byte // first comments in file
	BlockStmt          // module
}

func (ast *AST) String() string {
	s := ""
	for i, item := range ast.BlockStmt.List {
		if i != 0 {
			s += " "
		}
		s += item.String()
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

// Var is a variable, where Decl is the type of declaration and can be var|function for function scoped variables, let|const|class for block scoped variables
type Var struct {
	Decl DeclType
	Data []byte
	Link *Var // is set when merging variable uses, as in:  {a} {var a}  where the first lins to the second
	Uses uint16
}

func (v *Var) Name() []byte {
	for v.Link != nil {
		v = v.Link
	}
	return v.Data
}

func (v *Var) String() string {
	return string(v.Name())
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
	for i, v := range vs {
		if i != 0 {
			s += ", "
		}
		links := 0
		for v.Link != nil {
			v = v.Link
			links++
		}
		s += fmt.Sprintf("Var{%v %s %v %v}", v.Decl, string(v.Data), links, v.Uses)
	}
	return s + "]"
}

// Scope is a function or block scope with a list of variables declared and used
type Scope struct {
	Parent, Func   *Scope   // Parent is nil for global scope, Parent equals Func for function scope
	Declared       VarArray // Link in Var are always nil
	Undeclared     VarArray
	NumVarDecls    uint16 // number of variable declaration statements in a function scope
	NumForInit     uint16 // offset into Declared to mark variables used in for initializer
	NumArguments   uint16 // offset into Undeclared to mark variables used in arguments
	IsGlobalOrFunc bool
	HasWith        bool
}

func (s Scope) String() string {
	return "Scope{Declared: " + s.Declared.String() + ", Undeclared: " + s.Undeclared.String() + "}"
}

// Declare a new variable
func (s *Scope) Declare(decl DeclType, name []byte) (*Var, bool) {
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
			return nil, false
		}
		if v.Decl == ExprDecl {
			v.Decl = decl
		}
		v.Uses++
		if s != curScope {
			curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
		}
		return v, true
	}

	var v *Var
	// reuse variable if previously used, as in:  a;var a
	if decl != ArgumentDecl { // in case of function f(a=b,b), where the first b is different from the second
		for i, uv := range s.Undeclared[s.NumArguments:] {
			// no need to evaluate v.Link as v.Data stays the same and Link is nil in the active scope
			if 0 < uv.Uses && bytes.Equal(name, uv.Data) {
				v = uv
				s.Undeclared = append(s.Undeclared[:int(s.NumArguments)+i], s.Undeclared[int(s.NumArguments)+i+1:]...)
				break
			}
		}
	}
	if v == nil {
		// add variable to the context list and to the scope
		v = &Var{decl, name, nil, 0}
	} else {
		v.Decl = decl
	}
	v.Uses++
	s.Declared = append(s.Declared, v)
	if s != curScope {
		curScope.Undeclared = append(curScope.Undeclared, v) // add variable declaration as used variable to the current scope
	}
	return v, true
}

// Use a variable
func (s *Scope) Use(name []byte) *Var {
	// check if variable is declared in the current scope
	v := s.findDeclared(name)
	if v == nil {
		// check if variable is already used before in the current or lower scopes
		v = s.findUndeclared(name)
		if v == nil {
			// add variable to the context list and to the scope's undeclared
			v = &Var{NoDecl, name, nil, 0}
			s.Undeclared = append(s.Undeclared, v)
		}
	}
	v.Uses++
	return v
}

// find declared variable in the current scope
func (s *Scope) findDeclared(name []byte) *Var {
	for _, v := range s.Declared {
		// no need to evaluate v.Link as v.Data stays the same, and Link is always nil in Declared
		if bytes.Equal(name, v.Data) {
			return v
		}
	}
	return nil
}

// find undeclared variable in the current and lower scopes
func (s *Scope) findUndeclared(name []byte) *Var {
	for _, v := range s.Undeclared {
		// no need to evaluate v.Link as v.Data stays the same and Link is nil in the active scope
		if 0 < v.Uses && bytes.Equal(name, v.Data) {
			return v
		}
	}
	return nil
}

func (s *Scope) MarkForInit() {
	// set the offset for variables declared in for initializer to distinguish from declarations in body
	s.NumForInit = uint16(len(s.Declared))
}

func (s *Scope) MarkArguments() {
	// set the offset for variables used for arguments, to ensure different b's for: function f(a=b){var b}
	s.NumArguments = uint16(len(s.Undeclared))
}

func (s *Scope) HoistUndeclared() {
	// copy all undeclared variables to the parent scope
	for i, vorig := range s.Undeclared {
		// no need to evaluate vorig.Link as vorig.Data stays the same
		if 0 < vorig.Uses && vorig.Decl == NoDecl {
			if v := s.Parent.findDeclared(vorig.Data); v != nil {
				// check if variable is declared in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				s.Undeclared[i] = v // point reference to existing var (to avoid many Link chains)
			} else if v := s.Parent.findUndeclared(vorig.Data); v != nil {
				// check if variable is already used before in parent scope
				v.Uses += vorig.Uses
				vorig.Link = v
				s.Undeclared[i] = v // point reference to existing var (to avoid many Link chains)
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
	for _, vorig := range s.Declared {
		// no need to evaluate vorig.Link as vorig.Data stays the same, and Link is always nil in Declared
		// vorig.Uses will be atleast 1
		if v := s.Parent.findDeclared(vorig.Data); v != nil {
			// check if variable has been declared in this scope
			v.Uses += vorig.Uses
			vorig.Link = v
		} else if v := s.Parent.findUndeclared(vorig.Data); v != nil {
			// check if variable is already used before in the current or lower scopes
			v.Uses += vorig.Uses
			vorig.Link = v
		} else {
			// add variable to the context list and to the scope's undeclared
			vorig.Decl = NoDecl
			s.Parent.Undeclared = append(s.Parent.Undeclared, vorig)
		}
	}
	s.Declared = s.Declared[:0]
	s.Undeclared = s.Undeclared[:0]
}

////////////////////////////////////////////////////////////////

type IStmt interface {
	String() string
	stmtNode()
}

type IBinding interface {
	String() string
	bindingNode()
}

type IExpr interface {
	String() string
	exprNode()
}

////////////////////////////////////////////////////////////////

type BlockStmt struct {
	List []IStmt
	Scope
}

func (n BlockStmt) String() string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + " })"
}

type BranchStmt struct {
	Type  TokenType
	Label []byte // can be nil
}

func (n BranchStmt) String() string {
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

func (n LabelledStmt) String() string {
	return "Stmt(" + string(n.Label) + " : " + n.Value.String() + ")"
}

type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String() string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String()
	}
	return s + ")"
}

type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String() string {
	s := "Stmt(if " + n.Cond.String() + " " + n.Body.String()
	if n.Else != nil {
		s += " else " + n.Else.String()
	}
	return s + ")"
}

type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String() string {
	return "Stmt(with " + n.Cond.String() + " " + n.Body.String() + ")"
}

type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String() string {
	return "Stmt(do " + n.Body.String() + " while " + n.Cond.String() + ")"
}

type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String() string {
	return "Stmt(while " + n.Cond.String() + " " + n.Body.String() + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body BlockStmt
}

func (n ForStmt) String() string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String()
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String()
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String()
	}
	return s + " " + n.Body.String() + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  BlockStmt
}

func (n ForInStmt) String() string {
	return "Stmt(for " + n.Init.String() + " in " + n.Value.String() + " " + n.Body.String() + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  BlockStmt
}

func (n ForOfStmt) String() string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String() + " of " + n.Value.String() + " " + n.Body.String() + ")"
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

func (n SwitchStmt) String() string {
	s := "Stmt(switch " + n.Init.String()
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if clause.Cond != nil {
			s += " " + clause.Cond.String()
		}
		for _, item := range clause.List {
			s += " " + item.String()
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String() string {
	return "Stmt(throw " + n.Value.String() + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding   // can be nil
	Catch   *BlockStmt // can be nil
	Finally *BlockStmt // can be nil
}

func (n TryStmt) String() string {
	s := "Stmt(try " + n.Body.String()
	if n.Catch != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String() + ")"
		}
		s += " " + n.Catch.String()
	}
	if n.Finally != nil {
		s += " finally " + n.Finally.String()
	}
	return s + ")"
}

type DebuggerStmt struct {
}

func (n DebuggerStmt) String() string {
	return "Stmt(debugger)"
}

type EmptyStmt struct {
}

func (n EmptyStmt) String() string {
	return "Stmt(;)"
}

type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String() string {
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

func (n ImportStmt) String() string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 {
		s += " " + n.List[0].String()
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String()
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

func (n ExportStmt) String() string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String() + ")"
	} else if len(n.List) == 1 {
		s += " " + n.List[0].String()
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String()
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

func (n ExprStmt) String() string {
	val := n.Value.String()
	if val[0] == '(' && val[len(val)-1] == ')' {
		return "Stmt" + n.Value.String()
	}
	return "Stmt(" + n.Value.String() + ")"
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

func (n PropertyName) String() string {
	if n.Computed != nil {
		val := n.Computed.String()
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

func (n BindingArray) String() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String() + ")"
	}
	return s + " ]"
}

type BindingObjectItem struct {
	Key   *PropertyName // can be nil
	Value BindingElement
}

type BindingObject struct {
	List []BindingObjectItem
	Rest *Var // can be nil
}

func (n BindingObject) String() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		if item.Key != nil {
			if v, ok := item.Value.Binding.(*Var); !ok || !item.Key.IsIdent(v.Data) {
				s += " " + item.Key.String() + ":"
			}
		}
		s += " " + item.Value.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + string(n.Rest.Data) + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding // can be nil (in case of ellision)
	Default IExpr    // can be nil
}

func (n BindingElement) String() string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String()
	if n.Default != nil {
		s += " = " + n.Default.String()
	}
	return s + ")"
}

func (n *Var) bindingNode()          {}
func (n BindingArray) bindingNode()  {}
func (n BindingObject) bindingNode() {}

////////////////////////////////////////////////////////////////

type Params struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n Params) String() string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String() + ")"
	}
	return s + ")"
}

type Arguments struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n Arguments) String() string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.String()
	}
	return s + ")"
}

type VarDecl struct {
	TokenType
	List []BindingElement
}

func (n VarDecl) String() string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + ")"
}

type FuncDecl struct {
	Async     bool
	Generator bool
	Name      *Var // can be nil
	Params    Params
	Body      BlockStmt
}

func (n FuncDecl) String() string {
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
		s += " " + string(n.Name.Data)
	}
	return s + " " + n.Params.String() + " " + n.Body.String() + ")"
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

func (n MethodDecl) String() string {
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
	s += " " + n.Name.String() + " " + n.Params.String() + " " + n.Body.String()
	return "Method(" + s[1:] + ")"
}

type ClassDecl struct {
	Name    *Var  // can be nil
	Extends IExpr // can be nil
	Methods []MethodDecl
}

func (n ClassDecl) String() string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + string(n.Name.Data)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String()
	}
	for _, item := range n.Methods {
		s += " " + item.String()
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

func (n GroupExpr) String() string {
	return "(" + n.X.String() + ")"
}

type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String()
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

func (n Property) String() string {
	s := ""
	if n.Name != nil {
		if v, ok := n.Value.(*Var); !ok || !n.Name.IsIdent(v.Data) {
			s += n.Name.String() + ": "
		}
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String()
	if n.Init != nil {
		s += " = " + n.Init.String()
	}
	return s
}

type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String() string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
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

func (n TemplateExpr) String() string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String()
	}
	for _, item := range n.List {
		s += string(item.Value) + item.Expr.String()
	}
	return s + string(n.Tail)
}

type NewExpr struct {
	X    IExpr
	Args *Arguments // can be nil
}

func (n NewExpr) String() string {
	if n.Args != nil {
		return "(new " + n.X.String() + n.Args.String() + ")"
	}
	return "(new " + n.X.String() + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String() string {
	return "(new.target)"
}

type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String() string {
	return "(import.meta)"
}

type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String() string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String() + ")"
}

type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String() string {
	return "(" + n.Cond.String() + " ? " + n.X.String() + " : " + n.Y.String() + ")"
}

type DotExpr struct {
	X    IExpr
	Y    LiteralExpr
	Prec OpPrec
}

func (n DotExpr) String() string {
	return "(" + n.X.String() + "." + n.Y.String() + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String() string {
	return "(" + n.X.String() + n.Args.String() + ")"
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
	Prec  OpPrec
}

func (n IndexExpr) String() string {
	return "(" + n.X.String() + "[" + n.Index.String() + "])"
}

type OptChainExpr struct {
	X IExpr
	Y IExpr // can be CallExpr, IndexExpr, LiteralExpr, or TemplateExpr
}

func (n OptChainExpr) String() string {
	s := "(" + n.X.String() + "?."
	switch y := n.Y.(type) {
	case *CallExpr:
		return s + y.Args.String() + ")"
	case *IndexExpr:
		return s + "[" + y.Index.String() + "])"
	default:
		return s + y.String() + ")"
	}
}

type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String() string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String() + n.Op.String() + ")"
	} else if IsIdentifierName(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String() + ")"
	}
	return "(" + n.Op.String() + n.X.String() + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String() string {
	if IsIdentifierName(n.Op) {
		return "(" + n.X.String() + " " + n.Op.String() + " " + n.Y.String() + ")"
	}
	return "(" + n.X.String() + n.Op.String() + n.Y.String() + ")"
}

type LiteralExpr struct {
	TokenType
	Data []byte
}

func (n LiteralExpr) String() string {
	return string(n.Data)
}

type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
}

func (n ArrowFunc) String() string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String() + " => " + n.Body.String() + ")"
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
func (n *Var) exprNode()           {}
