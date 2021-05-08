package js

// IVisitor represents the AST Visitor
// Each INode encountered by `Walk` is passed to `Enter`, children nodes will be ignored if the returned IVisitor is nil
type IVisitor interface {
	Enter(n INode) IVisitor
}

// Walk traverses an AST in depth-first order
func Walk(v IVisitor, n INode) {
	n = pointer(n)

	if isnil(n) {
		return
	}

	if v = v.Enter(n); v == nil {
		return
	}

	switch n := n.(type) {
	case *AST:
		Walk(v, n.BlockStmt)
	case *Var:
		return
	case *BlockStmt:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *EmptyStmt:
		return
	case *ExprStmt:
		Walk(v, n.Value)
	case *IfStmt:
		Walk(v, n.Body)
		Walk(v, n.Else)
		Walk(v, n.Cond)
	case *DoWhileStmt:
		Walk(v, n.Body)
		Walk(v, n.Cond)
	case *WhileStmt:
		Walk(v, n.Body)
		Walk(v, n.Cond)
	case *ForStmt:
		Walk(v, n.Body)
		Walk(v, n.Init)
		Walk(v, n.Cond)
		Walk(v, n.Post)
	case *ForInStmt:
		Walk(v, n.Body)
		Walk(v, n.Init)
		Walk(v, n.Value)
	case *ForOfStmt:
		Walk(v, n.Body)
		Walk(v, n.Init)
		Walk(v, n.Value)
	case *CaseClause:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Cond)
	case *SwitchStmt:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Init)
	case *BranchStmt:
		return
	case *ReturnStmt:
		Walk(v, n.Value)
	case *WithStmt:
		Walk(v, n.Body)
		Walk(v, n.Cond)
	case *LabelledStmt:

		Walk(v, n.Value)
	case *ThrowStmt:
		Walk(v, n.Value)
	case *TryStmt:
		Walk(v, n.Body)
		Walk(v, n.Catch)
		Walk(v, n.Finally)
		Walk(v, n.Binding)
	case *DebuggerStmt:
		return
	case *Alias:
		return
	case *ImportStmt:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *ExportStmt:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Decl)
	case *DirectivePrologueStmt:
		return
	case *PropertyName:
		Walk(v, n.Literal)
		Walk(v, n.Computed)
	case *BindingArray:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Rest)
	case *BindingObjectItem:

		Walk(v, n.Key)
		Walk(v, n.Value)
	case *BindingObject:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Rest)
	case *BindingElement:
		Walk(v, n.Binding)
		Walk(v, n.Default)
	case *VarDecl:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *Params:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Rest)
	case *FuncDecl:
		Walk(v, n.Body)
		Walk(v, n.Params)
		Walk(v, n.Name)
	case *MethodDecl:
		Walk(v, n.Body)
		Walk(v, n.Params)
		Walk(v, n.Name)
	case *FieldDefinition:
		Walk(v, n.Name)
		Walk(v, n.Init)
	case *ClassDecl:
		Walk(v, n.Name)
		Walk(v, n.Extends)

		if n.Definitions != nil {
			for _, item := range n.Definitions {
				Walk(v, item)
			}
		}

		if n.Methods != nil {
			for _, item := range n.Methods {
				Walk(v, item)
			}
		}
	case *LiteralExpr:
		return
	case *Element:
		Walk(v, n.Value)
	case *ArrayExpr:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *Property:
		Walk(v, n.Name)
		Walk(v, n.Value)
		Walk(v, n.Init)
	case *ObjectExpr:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *TemplatePart:
		Walk(v, n.Expr)
	case *TemplateExpr:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}

		Walk(v, n.Tag)
	case *GroupExpr:
		Walk(v, n.X)
	case *IndexExpr:
		Walk(v, n.X)
		Walk(v, n.Y)
	case *DotExpr:
		Walk(v, n.X)
		Walk(v, n.Y)
	case *NewTargetExpr:
		return
	case *ImportMetaExpr:
		return
	case *Arg:
		Walk(v, n.Value)
	case *Args:
		if n.List != nil {
			for _, item := range n.List {
				Walk(v, item)
			}
		}
	case *NewExpr:
		Walk(v, n.Args)
		Walk(v, n.X)
	case *CallExpr:
		Walk(v, n.Args)
		Walk(v, n.X)
	case *OptChainExpr:
		Walk(v, n.X)
		Walk(v, n.Y)
	case *UnaryExpr:
		Walk(v, n.X)
	case *BinaryExpr:
		Walk(v, n.X)
		Walk(v, n.Y)
	case *CondExpr:
		Walk(v, n.Cond)
		Walk(v, n.X)
		Walk(v, n.Y)
	case *YieldExpr:
		Walk(v, n.X)
	case *ArrowFunc:
		Walk(v, n.Body)
		Walk(v, n.Params)
	default:
		return
	}
}

func pointer(n INode) INode {
	switch n := n.(type) {
	case Var:
		return &n
	case BlockStmt:
		return &n
	case EmptyStmt:
		return &n
	case ExprStmt:
		return &n
	case IfStmt:
		return &n
	case DoWhileStmt:
		return &n
	case WhileStmt:
		return &n
	case ForStmt:
		return &n
	case ForInStmt:
		return &n
	case ForOfStmt:
		return &n
	case CaseClause:
		return &n
	case SwitchStmt:
		return &n
	case BranchStmt:
		return &n
	case ReturnStmt:
		return &n
	case WithStmt:
		return &n
	case LabelledStmt:
		return &n
	case ThrowStmt:
		return &n
	case TryStmt:
		return &n
	case DebuggerStmt:
		return &n
	case Alias:
		return &n
	case ImportStmt:
		return &n
	case ExportStmt:
		return &n
	case DirectivePrologueStmt:
		return &n
	case PropertyName:
		return &n
	case BindingArray:
		return &n
	case BindingObjectItem:
		return &n
	case BindingObject:
		return &n
	case BindingElement:
		return &n
	case VarDecl:
		return &n
	case Params:
		return &n
	case FuncDecl:
		return &n
	case MethodDecl:
		return &n
	case FieldDefinition:
		return &n
	case ClassDecl:
		return &n
	case LiteralExpr:
		return &n
	case Element:
		return &n
	case ArrayExpr:
		return &n
	case Property:
		return &n
	case ObjectExpr:
		return &n
	case TemplatePart:
		return &n
	case TemplateExpr:
		return &n
	case GroupExpr:
		return &n
	case IndexExpr:
		return &n
	case DotExpr:
		return &n
	case NewTargetExpr:
		return &n
	case ImportMetaExpr:
		return &n
	case Arg:
		return &n
	case Args:
		return &n
	case NewExpr:
		return &n
	case CallExpr:
		return &n
	case OptChainExpr:
		return &n
	case UnaryExpr:
		return &n
	case BinaryExpr:
		return &n
	case CondExpr:
		return &n
	case YieldExpr:
		return &n
	case ArrowFunc:
		return &n
	}

	return n
}

func isnil(n INode) bool {
	switch n := n.(type) {
	case *Var:
		return n == nil
	case *BlockStmt:
		return n == nil
	case *EmptyStmt:
		return n == nil
	case *ExprStmt:
		return n == nil
	case *IfStmt:
		return n == nil
	case *DoWhileStmt:
		return n == nil
	case *WhileStmt:
		return n == nil
	case *ForStmt:
		return n == nil
	case *ForInStmt:
		return n == nil
	case *ForOfStmt:
		return n == nil
	case *CaseClause:
		return n == nil
	case *SwitchStmt:
		return n == nil
	case *BranchStmt:
		return n == nil
	case *ReturnStmt:
		return n == nil
	case *WithStmt:
		return n == nil
	case *LabelledStmt:
		return n == nil
	case *ThrowStmt:
		return n == nil
	case *TryStmt:
		return n == nil
	case *DebuggerStmt:
		return n == nil
	case *Alias:
		return n == nil
	case *ImportStmt:
		return n == nil
	case *ExportStmt:
		return n == nil
	case *DirectivePrologueStmt:
		return n == nil
	case *PropertyName:
		return n == nil
	case *BindingArray:
		return n == nil
	case *BindingObjectItem:
		return n == nil
	case *BindingObject:
		return n == nil
	case *BindingElement:
		return n == nil
	case *VarDecl:
		return n == nil
	case *Params:
		return n == nil
	case *FuncDecl:
		return n == nil
	case *MethodDecl:
		return n == nil
	case *FieldDefinition:
		return n == nil
	case *ClassDecl:
		return n == nil
	case *LiteralExpr:
		return n == nil
	case *Element:
		return n == nil
	case *ArrayExpr:
		return n == nil
	case *Property:
		return n == nil
	case *ObjectExpr:
		return n == nil
	case *TemplatePart:
		return n == nil
	case *TemplateExpr:
		return n == nil
	case *GroupExpr:
		return n == nil
	case *IndexExpr:
		return n == nil
	case *DotExpr:
		return n == nil
	case *NewTargetExpr:
		return n == nil
	case *ImportMetaExpr:
		return n == nil
	case *Arg:
		return n == nil
	case *Args:
		return n == nil
	case *NewExpr:
		return n == nil
	case *CallExpr:
		return n == nil
	case *OptChainExpr:
		return n == nil
	case *UnaryExpr:
		return n == nil
	case *BinaryExpr:
		return n == nil
	case *CondExpr:
		return n == nil
	case *YieldExpr:
		return n == nil
	case *ArrowFunc:
		return n == nil
	}

	return false
}
