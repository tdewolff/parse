package js

import "fmt"

type AST struct {
	List []IStmt
}

func (n AST) String() string {
	s := ""
	for i, item := range n.List {
		if i != 0 {
			s += " "
		}
		s += item.String()
	}
	return s
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

type Token struct {
	TokenType
	Data []byte
}

func (n Token) String() string {
	return string(n.Data)
}

////////////////////////////////////////////////////////////////

type BlockStmt struct {
	List []IStmt
}

func (n BlockStmt) String() string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String()
	}
	return s + " })"
}

type BranchStmt struct {
	Type TokenType
	Name *Token // can be nil
}

func (n BranchStmt) String() string {
	s := "Stmt(" + n.Type.String()
	if n.Name != nil {
		s += " " + n.Name.String()
	}
	return s + ")"
}

type LabelledStmt struct {
	Token
	Value IStmt
}

func (n LabelledStmt) String() string {
	return "Stmt(" + n.Token.String() + " : " + n.Value.String() + ")"
}

type ReturnStmt struct {
	Value Expr
}

func (n ReturnStmt) String() string {
	s := "Stmt(return"
	if len(n.Value.List) != 0 {
		s += " " + n.Value.String()
	}
	return s + ")"
}

type IfStmt struct {
	Cond Expr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String() string {
	fmt.Println(n.Cond, n.Body, n.Else)
	s := "Stmt(if " + n.Cond.String() + " " + n.Body.String()
	if n.Else != nil {
		s += " else " + n.Else.String()
	}
	return s + ")"
}

type WithStmt struct {
	Cond Expr
	Body IStmt
}

func (n WithStmt) String() string {
	return "Stmt(with " + n.Cond.String() + " " + n.Body.String() + ")"
}

type DoWhileStmt struct {
	Cond Expr
	Body IStmt
}

func (n DoWhileStmt) String() string {
	return "Stmt(do " + n.Body.String() + " while " + n.Cond.String() + ")"
}

type WhileStmt struct {
	Cond Expr
	Body IStmt
}

func (n WhileStmt) String() string {
	return "Stmt(while " + n.Cond.String() + " " + n.Body.String() + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond Expr
	Post Expr
	Body IStmt
}

func (n ForStmt) String() string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String()
	}
	s += " ;"
	if len(n.Cond.List) != 0 {
		s += " " + n.Cond.String()
	}
	s += " ;"
	if len(n.Post.List) != 0 {
		s += " " + n.Post.String()
	}
	return s + " " + n.Body.String() + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value Expr
	Body  IStmt
}

func (n ForInStmt) String() string {
	return "Stmt(for " + n.Init.String() + " in " + n.Value.String() + " " + n.Body.String() + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value AssignExpr
	Body  IStmt
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
	Cond Expr
	Body []IStmt
}

type SwitchStmt struct {
	Init Expr
	List []CaseClause
}

func (n SwitchStmt) String() string {
	s := "Stmt(switch " + n.Init.String()
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if len(clause.Cond.List) != 0 {
			s += " " + clause.Cond.String()
		}
		for _, item := range clause.Body {
			s += " " + item.String()
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value Expr
}

func (n ThrowStmt) String() string {
	return "Stmt(throw " + n.Value.String() + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding // can be nil
	Catch   BlockStmt
	Finally BlockStmt
}

func (n TryStmt) String() string {
	s := "Stmt(try " + n.Body.String()
	if len(n.Catch.List) != 0 || n.Binding != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String() + ")"
		}
		s += " " + n.Catch.String()
	}
	if len(n.Finally.List) != 0 {
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
	if alias.Binding == nil {
		return ""
	}
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
			s += " " + item.String()
		}
		s += " }"
	}
	if n.Module != nil {
		s += " from " + string(n.Module)
	}
	return s + ")"
}

type ExprStmt struct {
	Value Expr
}

func (n ExprStmt) String() string {
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
	Name         Token
	ComputedName *AssignExpr // can be nil
}

func (n PropertyName) String() string {
	if n.ComputedName != nil {
		return "[ " + n.ComputedName.String() + " ]"
	}
	return n.Name.String()
}

type BindingName struct {
	Name []byte // can be nil
}

func (n BindingName) String() string {
	return string(n.Name)
}

type BindingArray struct {
	List []BindingElement
	Rest IBinding
}

func (n BindingArray) String() string {
	s := "["
	for _, item := range n.List {
		s += " " + item.String()
	}
	if n.Rest != nil {
		s += " ... Binding(" + n.Rest.String() + ")"
	}
	return s + " ]"
}

type BindingObjectItem struct {
	Key   *PropertyName // can be nil
	Value BindingElement
}

type BindingObject struct {
	List []BindingObjectItem
	Rest *BindingName // can be nil
}

func (n BindingObject) String() string {
	s := "{"
	for _, item := range n.List {
		if item.Key != nil {
			s += " " + item.Key.String() + " :"
		}
		s += " " + item.Value.String()
	}
	if n.Rest != nil {
		s += " ... Binding(" + n.Rest.String() + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding    // can be nil
	Default *AssignExpr // can be nil
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

func (n BindingName) bindingNode()   {}
func (n BindingArray) bindingNode()  {}
func (n BindingObject) bindingNode() {}

////////////////////////////////////////////////////////////////

type Params struct {
	List []BindingElement
	Rest *BindingElement // can be nil
}

func (n Params) String() string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += " , "
		}
		s += item.String()
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += " , "
		}
		s += "... " + n.Rest.String()
	}
	return s + ")"
}

type Method struct {
	Static    bool
	Async     bool
	Generator bool
	Get       bool
	Set       bool
	Name      PropertyName
	Params    Params
	Body      BlockStmt
}

func (n Method) String() string {
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
	Name      []byte // can be nil
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
		s += " *"
	}
	if n.Name != nil {
		s += " " + string(n.Name)
	}
	return s + " " + n.Params.String() + " " + n.Body.String() + ")"
}

type ClassDecl struct {
	Name    []byte // can be nil
	Extends IExpr  // can be nil TODO LHS EXPR
	Methods []Method
}

func (n ClassDecl) String() string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + string(n.Name)
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

func (n VarDecl) exprNode()   {}
func (n FuncDecl) exprNode()  {}
func (n ClassDecl) exprNode() {}

////////////////////////////////////////////////////////////////

type AssignExpr struct {
	Nodes []interface{ String() string }
}

func (n AssignExpr) String() string {
	s := "Expr("
	for i, item := range n.Nodes {
		if i != 0 {
			s += " "
		}
		s += item.String()
	}
	return s + ")"
}

type Expr struct {
	List []AssignExpr
}

func (n Expr) String() string {
	if len(n.List) == 1 {
		return n.List[0].String()
	}
	s := "Expr("
	for i, item := range n.List {
		if i != 0 {
			s += " , "
		}
		s += item.String()
	}
	return s + ")"
}

func (n AssignExpr) exprNode() {}
func (n Expr) exprNode()       {}
