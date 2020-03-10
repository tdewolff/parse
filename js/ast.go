package js

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
	Value IExpr
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
	Cond IExpr
	Post IExpr
	Body IStmt
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
	Body  IStmt
}

func (n ForInStmt) String() string {
	return "Stmt(for " + n.Init.String() + " in " + n.Value.String() + " " + n.Body.String() + ")"
}

type ForOfStmt struct {
	Await bool
	Init  IExpr
	Value IExpr
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
	Cond IExpr
	Body []IStmt
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
		for _, item := range clause.Body {
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
	if val[0] == '(' {
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
	Name         Token
	ComputedName IExpr // can be nil
}

func (n PropertyName) String() string {
	if n.ComputedName != nil {
		name := n.ComputedName.String()
		if name[0] == '(' {
			return "[" + name[1:len(name)-1] + "]"
		}
		return "[" + name + "]"
	}
	return n.Name.String()
}

type Property struct {
	Init   bool
	Spread bool
	Key    *PropertyName // can be nil
	Value  IExpr         // can be nil, method or assignment expression
}

func (n Property) String() string {
	s := ""
	if n.Key != nil {
		s += n.Key.String()
		if n.Value != nil {
			if n.Init {
				s += " = "
			} else {
				s += ": "
			}
			s += n.Value.String()
		}
	} else {
		if n.Spread {
			s += "..."
		}
		s += n.Value.String()
	}
	return s
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
	Binding IBinding // can be nil
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
		s += "*"
	}
	if n.Name != nil {
		s += " " + string(n.Name)
	}
	return s + " " + n.Params.String() + " " + n.Body.String() + ")"
}

type ClassDecl struct {
	Name    []byte // can be nil
	Extends IExpr  // can be nil TODO LHS EXPR
	Methods []MethodDecl
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

type ArrowFunctionDecl struct {
	Async  bool
	Params Params
	Body   BlockStmt
}

func (n ArrowFunctionDecl) String() string {
	s := "("
	if n.Async {
		s += "async "
	}
	return s + n.Params.String() + " => " + n.Body.String() + ")"
}

func (n VarDecl) stmtNode()   {}
func (n FuncDecl) stmtNode()  {}
func (n ClassDecl) stmtNode() {}

func (n VarDecl) exprNode()           {}
func (n FuncDecl) exprNode()          {}
func (n ClassDecl) exprNode()         {}
func (n MethodDecl) exprNode()        {}
func (n ArrowFunctionDecl) exprNode() {}

////////////////////////////////////////////////////////////////

type GroupExpr struct {
	List []IExpr
	Rest IBinding
}

func (n GroupExpr) String() string {
	s := "("
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	if n.Rest != nil {
		s += ", ...Binding(" + n.Rest.String() + ")"
	}
	return s + ")"
}

type ArrayExpr struct {
	List []IExpr
	Rest IExpr // can be nil
}

func (n ArrayExpr) String() string {
	s := "["
	for i, item := range n.List {
		if i != 0 {
			s += ", "
		}
		s += item.String()
	}
	if n.Rest != nil {
		s += ", ..." + n.Rest.String()
	}
	return s + "]"
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

type NewExpr struct {
	X IExpr
}

func (n NewExpr) String() string {
	return "(new " + n.X.String() + ")"
}

type NewTargetExpr struct {
}

func (n NewTargetExpr) String() string {
	return "(new.target)"
}

type YieldExpr struct {
	Generator bool
	Value     IExpr // can be nil
}

func (n YieldExpr) String() string {
	if n.Value == nil {
		return "(yield)"
	}
	s := "(yield"
	if n.Generator {
		s += "*"
	}
	return s + " " + n.Value.String() + ")"
}

type ConditionalExpr struct {
	X, Y, Z IExpr
}

func (n ConditionalExpr) String() string {
	return "(" + n.X.String() + " ? " + n.Y.String() + " : " + n.Z.String() + ")"
}

type CallExpr struct {
	X    IExpr
	Args Arguments
}

func (n CallExpr) String() string {
	return n.X.String() + n.Args.String()
}

type IndexExpr struct {
	X     IExpr
	Index IExpr
}

func (n IndexExpr) String() string {
	return n.X.String() + "[" + n.Index.String() + "]"
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
	} else if IsIdentifier(n.Op) {
		return "(" + n.Op.String() + " " + n.X.String() + ")"
	}
	return "(" + n.Op.String() + n.X.String() + ")"
}

type BinaryExpr struct {
	Op   TokenType
	X, Y IExpr
}

func (n BinaryExpr) String() string {
	if IsIdentifier(n.Op) {
		return "(" + n.X.String() + " " + n.Op.String() + " " + n.Y.String() + ")"
	}
	return "(" + n.X.String() + n.Op.String() + n.Y.String() + ")"
}

type LiteralExpr Token

func (n LiteralExpr) String() string {
	return string(n.Data)
}

func (n GroupExpr) exprNode()       {}
func (n ArrayExpr) exprNode()       {}
func (n ObjectExpr) exprNode()      {}
func (n TemplateExpr) exprNode()    {}
func (n NewExpr) exprNode()         {}
func (n NewTargetExpr) exprNode()   {}
func (n YieldExpr) exprNode()       {}
func (n ConditionalExpr) exprNode() {}
func (n CallExpr) exprNode()        {}
func (n IndexExpr) exprNode()       {}
func (n OptChainExpr) exprNode()    {}
func (n UnaryExpr) exprNode()       {}
func (n BinaryExpr) exprNode()      {}
func (n LiteralExpr) exprNode()     {}
