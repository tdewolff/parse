package js

import (
	"fmt"
	"unsafe"
)

func init() {
	fmt.Println("sizeof Ref", unsafe.Sizeof(Ref{}))
	fmt.Println("sizeof TokenType", unsafe.Sizeof(TokenType(0)))
	fmt.Println("sizeof LiteralExpr", unsafe.Sizeof(LiteralExpr{}))
	fmt.Println("sizeof BinaryExpr", unsafe.Sizeof(BinaryExpr{}))
	fmt.Println("sizeof DotExpr", unsafe.Sizeof(DotExpr{}))
	fmt.Println("sizeof CallExpr", unsafe.Sizeof(CallExpr{}))
	fmt.Println("sizeof FuncDecl", unsafe.Sizeof(FuncDecl{}))
	fmt.Println("sizeof MethodDecl", unsafe.Sizeof(MethodDecl{}))
}

type Ref struct {
	offset uint32 // maximum file size 4GB
	length uint16
	TokenType
}

func (ref Ref) Data(src Src) []byte {
	if ref.length == 0 {
		return nil
	}
	end := ref.offset + uint32(ref.length)
	return src[ref.offset:end:end]
}

func (ref Ref) String(src Src) string {
	return string(ref.Data(src))
}

type Src []byte

func (src Src) Ref(tt TokenType, data []byte) Ref {
	offset := uint32(uintptr(unsafe.Pointer(&data[0])) - uintptr(unsafe.Pointer(&src[0])))
	return Ref{offset, uint16(len(data)), tt}
}

type Var struct {
	Uses     int
	Declared bool
}

type Scope struct {
	Parent *Scope
	Vars   map[string]Var
}

func (s Scope) declare(b []byte) {
	v := s.Vars[string(b)]
	v.Uses++
	v.Declared = true
	s.Vars[string(b)] = v
}

func (s Scope) use(b []byte) bool {
	if _, ok := s.Vars[string(b)]; ok {
		v := s.Vars[string(b)]
		v.Uses++
		s.Vars[string(b)] = v
		return true
	} else if s.Parent != nil {
		return s.Parent.use(b)
	}
	return false
}

////////////////////////////////////////////////////////////////

type AST struct {
	List []IStmt

	Src
	Scope
	Undeclared map[string]struct{}
}

func (n AST) String() string {
	s := ""
	for i, item := range n.List {
		if i != 0 {
			s += " "
		}
		s += item.String(n.Src)
	}
	return s
}

////////////////////////////////////////////////////////////////

type IStmt interface {
	String(Src) string
	stmtNode()
}

type IBinding interface {
	String(Src) string
	bindingNode()
}

type IExpr interface {
	String(Src) string
	exprNode()
}

////////////////////////////////////////////////////////////////

type BlockStmt struct {
	List []IStmt
	Scope
}

func (n BlockStmt) String(src Src) string {
	s := "Stmt({"
	for _, item := range n.List {
		s += " " + item.String(src)
	}
	return s + " })"
}

type BranchStmt struct {
	Type TokenType
	Ref  // can be nil
}

func (n BranchStmt) String(src Src) string {
	s := "Stmt(" + n.Type.String()
	if n.Ref.length != 0 {
		s += " " + n.Ref.String(src)
	}
	return s + ")"
}

type LabelledStmt struct {
	Ref
	Value IStmt
}

func (n LabelledStmt) String(src Src) string {
	return "Stmt(" + n.Ref.String(src) + " : " + n.Value.String(src) + ")"
}

type ReturnStmt struct {
	Value IExpr // can be nil
}

func (n ReturnStmt) String(src Src) string {
	s := "Stmt(return"
	if n.Value != nil {
		s += " " + n.Value.String(src)
	}
	return s + ")"
}

type IfStmt struct {
	Cond IExpr
	Body IStmt
	Else IStmt // can be nil
}

func (n IfStmt) String(src Src) string {
	s := "Stmt(if " + n.Cond.String(src) + " " + n.Body.String(src)
	if n.Else != nil {
		s += " else " + n.Else.String(src)
	}
	return s + ")"
}

type WithStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WithStmt) String(src Src) string {
	return "Stmt(with " + n.Cond.String(src) + " " + n.Body.String(src) + ")"
}

type DoWhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n DoWhileStmt) String(src Src) string {
	return "Stmt(do " + n.Body.String(src) + " while " + n.Cond.String(src) + ")"
}

type WhileStmt struct {
	Cond IExpr
	Body IStmt
}

func (n WhileStmt) String(src Src) string {
	return "Stmt(while " + n.Cond.String(src) + " " + n.Body.String(src) + ")"
}

type ForStmt struct {
	Init IExpr // can be nil
	Cond IExpr // can be nil
	Post IExpr // can be nil
	Body IStmt
}

func (n ForStmt) String(src Src) string {
	s := "Stmt(for"
	if n.Init != nil {
		s += " " + n.Init.String(src)
	}
	s += " ;"
	if n.Cond != nil {
		s += " " + n.Cond.String(src)
	}
	s += " ;"
	if n.Post != nil {
		s += " " + n.Post.String(src)
	}
	return s + " " + n.Body.String(src) + ")"
}

type ForInStmt struct {
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForInStmt) String(src Src) string {
	return "Stmt(for " + n.Init.String(src) + " in " + n.Value.String(src) + " " + n.Body.String(src) + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
	Body  IStmt
}

func (n ForOfStmt) String(src Src) string {
	s := "Stmt(for"
	if n.Await {
		s += " await"
	}
	return s + " " + n.Init.String(src) + " of " + n.Value.String(src) + " " + n.Body.String(src) + ")"
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

func (n SwitchStmt) String(src Src) string {
	s := "Stmt(switch " + n.Init.String(src)
	for _, clause := range n.List {
		s += " Clause(" + clause.TokenType.String()
		if clause.Cond != nil {
			s += " " + clause.Cond.String(src)
		}
		for _, item := range clause.List {
			s += " " + item.String(src)
		}
		s += ")"
	}
	return s + ")"
}

type ThrowStmt struct {
	Value IExpr
}

func (n ThrowStmt) String(src Src) string {
	return "Stmt(throw " + n.Value.String(src) + ")"
}

type TryStmt struct {
	Body    BlockStmt
	Binding IBinding // can be nil
	Catch   BlockStmt
	Finally BlockStmt
}

func (n TryStmt) String(src Src) string {
	s := "Stmt(try " + n.Body.String(src)
	if len(n.Catch.List) != 0 || n.Binding != nil {
		s += " catch"
		if n.Binding != nil {
			s += " Binding(" + n.Binding.String(src) + ")"
		}
		s += " " + n.Catch.String(src)
	}
	if len(n.Finally.List) != 0 {
		s += " finally " + n.Finally.String(src)
	}
	return s + ")"
}

type DebuggerStmt struct {
}

func (n DebuggerStmt) String(src Src) string {
	return "Stmt(debugger)"
}

type EmptyStmt struct {
}

func (n EmptyStmt) String(src Src) string {
	return "Stmt(;)"
}

type Alias struct {
	Name    []byte // can be nil
	Binding []byte // can be nil
}

func (alias Alias) String(src Src) string {
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

func (n ImportStmt) String(src Src) string {
	s := "Stmt(import"
	if n.Default != nil {
		s += " " + string(n.Default)
		if len(n.List) != 0 {
			s += " ,"
		}
	}
	if len(n.List) == 1 {
		s += " " + n.List[0].String(src)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(src)
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

func (n ExportStmt) String(src Src) string {
	s := "Stmt(export"
	if n.Decl != nil {
		if n.Default {
			s += " default"
		}
		return s + " " + n.Decl.String(src) + ")"
	} else if len(n.List) == 1 {
		s += " " + n.List[0].String(src)
	} else if 1 < len(n.List) {
		s += " {"
		for i, item := range n.List {
			if i != 0 {
				s += " ,"
			}
			if item.Binding != nil {
				s += " " + item.String(src)
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

func (n ExprStmt) String(src Src) string {
	val := n.Value.String(src)
	if val[0] == '(' {
		return "Stmt" + n.Value.String(src)
	}
	return "Stmt(" + n.Value.String(src) + ")"
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

// TODO: merge Token and IExpr, others don't have to use *PropertyName in that case
type PropertyName struct {
	Literal  LiteralExpr
	Computed IExpr // can be nil
}

func (n PropertyName) String(src Src) string {
	if n.Computed != nil {
		name := n.Computed.String(src)
		if name[0] == '(' {
			return "[" + name[1:len(name)-1] + "]"
		}
		return "[" + name + "]"
	}
	return n.Literal.String(src)
}

type Property struct {
	// either Key, Init, or Spread are set. When Key or Spread are set then Value is AssignmentExpression
	// if Init is set then Value is IdentifierReference, otherwise it can also be MethodDefinition
	Key    *PropertyName
	Init   IExpr // can be nil
	Spread bool
	Value  IExpr
}

func (n Property) String(src Src) string {
	s := ""
	if n.Key != nil {
		s += n.Key.String(src) + ": "
	} else if n.Spread {
		s += "..."
	}
	s += n.Value.String(src)
	if n.Init != nil {
		s += " = " + n.Init.String(src)
	}
	return s
}

type BindingName Ref

func (n BindingName) Data(src Src) []byte {
	return Ref(n).Data(src)
}

func (n BindingName) String(src Src) string {
	return Ref(n).String(src)
}

type BindingArray struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n BindingArray) String(src Src) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		s += " " + item.String(src)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(src) + ")"
	}
	return s + " ]"
}

type BindingObjectItem struct {
	Key   *PropertyName // can be nil
	Value BindingElement
}

type BindingObject struct {
	List []BindingObjectItem
	Rest BindingName // can be nil
}

func (n BindingObject) String(src Src) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ","
		}
		if item.Key != nil {
			s += " " + item.Key.String(src) + ":"
		}
		s += " " + item.Value.String(src)
	}
	if n.Rest.Data(src) != nil {
		if len(n.List) != 0 {
			s += ","
		}
		s += " ...Binding(" + n.Rest.String(src) + ")"
	}
	return s + " }"
}

type BindingElement struct {
	Binding IBinding // can be nil
	Default IExpr    // can be nil
}

func (n BindingElement) String(src Src) string {
	if n.Binding == nil {
		return "Binding()"
	}
	s := "Binding(" + n.Binding.String(src)
	if n.Default != nil {
		s += " = " + n.Default.String(src)
	}
	return s + ")"
}

func (n BindingName) bindingNode()   {}
func (n BindingArray) bindingNode()  {}
func (n BindingObject) bindingNode() {}

////////////////////////////////////////////////////////////////

type Params struct {
	List []BindingElement
	Rest IBinding // can be nil
}

func (n Params) String(src Src) string {
	s := "Params("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(src)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "...Binding(" + n.Rest.String(src) + ")"
	}
	return s + ")"
}

type Arguments struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n Arguments) String(src Src) string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(src)
	}
	if n.Rest != nil {
		if len(n.List) != 0 {
			s += ", "
		}
		s += "..." + n.Rest.String(src)
	}
	return s + ")"
}

type VarDecl struct {
	TokenType
	List []BindingElement
}

func (n VarDecl) String(src Src) string {
	s := "Decl(" + n.TokenType.String()
	for _, item := range n.List {
		s += " " + item.String(src)
	}
	return s + ")"
}

type FuncDecl struct {
	Async     bool
	Generator bool
	Name      []byte // can be nil
	Params    Params
	Body      BlockStmt
	Scope
}

func (n FuncDecl) String(src Src) string {
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
		s += " " + string(n.Name)
	}
	return s + " " + n.Params.String(src) + " " + n.Body.String(src) + ")"
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

func (n MethodDecl) String(src Src) string {
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
	s += " " + n.Name.String(src) + " " + n.Params.String(src) + " " + n.Body.String(src)
	return "Method(" + s[1:] + ")"
}

type ClassDecl struct {
	Name    []byte // can be nil
	Extends IExpr  // can be nil
	Methods []MethodDecl
}

func (n ClassDecl) String(src Src) string {
	s := "Decl(class"
	if n.Name != nil {
		s += " " + string(n.Name)
	}
	if n.Extends != nil {
		s += " extends " + n.Extends.String(src)
	}
	for _, item := range n.Methods {
		s += " " + item.String(src)
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

func (n GroupExpr) String(src Src) string {
	return "(" + n.X.String(src) + ")"
}

type Element struct {
	Value  IExpr // can be nil
	Spread bool
}

type ArrayExpr struct {
	List []Element
}

func (n ArrayExpr) String(src Src) string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		if item.Value != nil {
			if item.Spread {
				s += "..."
			}
			s += item.Value.String(src)
		}
	}
	if 0 < len(n.List) && n.List[len(n.List)-1].Value == nil {
		s += ","
	}
	return s + "]"
}

type ObjectExpr struct {
	List []Property
}

func (n ObjectExpr) String(src Src) string {
	s := "{"
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String(src)
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

func (n TemplateExpr) String(src Src) string {
	s := ""
	if n.Tag != nil {
		s += n.Tag.String(src)
	}
	for _, item := range n.List {
		s += string(item.Value) + item.Expr.String(src)
	}
	return s + string(n.Tail)
}

type NewExpr struct {
	X    IExpr
	Args *Arguments // can be nil
}

func (n NewExpr) String(src Src) string {
	if n.Args != nil {
		return "(new " + n.X.String(src) + n.Args.String(src) + ")"
	}
	return "(new " + n.X.String(src) + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String(src Src) string {
	return "(new.target)"
}

type ImportMetaExpr struct {
}

func (n ImportMetaExpr) String(src Src) string {
	return "(import.meta)"
}

type YieldExpr struct {
	Generator bool
	X         IExpr // can be nil
}

func (n YieldExpr) String(src Src) string {
	if n.X == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.X.String(src) + ")"
}

type CondExpr struct {
	Cond, X, Y IExpr
}

func (n CondExpr) String(src Src) string {
	return "(" + n.Cond.String(src) + " ? " + n.X.String(src) + " : " + n.Y.String(src) + ")"
}

type DotExpr struct {
	X IExpr
	Y LiteralExpr
}

func (n DotExpr) String(src Src) string {
	return "(" + n.X.String(src) + "." + n.Y.String(src) + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String(src Src) string {
	return "(" + n.X.String(src) + n.Args.String(src) + ")"
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
}

func (n IndexExpr) String(src Src) string {
	return "(" + n.X.String(src) + "[" + n.Index.String(src) + "])"
}

type OptChainExpr struct {
	X IExpr
	Y IExpr // can be CallExpr, IndexExpr, LiteralExpr, or TemplateExpr
}

func (n OptChainExpr) String(src Src) string {
	s := "(" + n.X.String(src) + "?."
	switch y := n.Y.(type) {
	case *CallExpr:
		return s + y.Args.String(src) + ")"
	case *IndexExpr:
		return s + "[" + y.Index.String(src) + "])"
	default:
		return s + y.String(src) + ")"
	}
}

type UnaryExpr struct {
	Op TokenType
	X  IExpr
}

func (n UnaryExpr) String(src Src) string {
	if n.Op == PostIncrToken || n.Op == PostDecrToken {
		return "(" + n.X.String(src) + n.Op.String() + ")"
	} else if IsIdentifier(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String(src) + ")"
	}
	return "(" + n.Op.String() + n.X.String(src) + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String(src Src) string {
	if IsIdentifier(n.Op) {
		return "(" + n.X.String(src) + " " + n.Op.String() + " " + n.Y.String(src) + ")"
	}
	return "(" + n.X.String(src) + n.Op.String() + n.Y.String(src) + ")"
}

type LiteralExpr Ref

func (n LiteralExpr) Data(src Src) []byte {
	return Ref(n).Data(src)
}

func (n LiteralExpr) String(src Src) string {
	return Ref(n).String(src)
}

type ArrowFunc struct {
	Async  bool
	Params Params
	Body   BlockStmt
	Scope
}

func (n ArrowFunc) String(src Src) string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String(src) + " => " + n.Body.String(src) + ")"
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
