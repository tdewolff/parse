package css

import (
	"bytes"
	"reflect"
)

////////////////////////////////////////////////////////////////

// NodeType determines the type of node, eg. a block or a declaration.
type NodeType uint32

const (
	ErrorNode NodeType = iota // extra node when errors occur
	StylesheetNode
	RulesetNode
	SelectorGroupNode
	SelectorNode
	DeclarationListNode
	DeclarationNode
	FunctionNode
	AtRuleNode
	TokenNode // extra node for simple tokens
)

func (t NodeType) Type() NodeType {
	return t
}

////////////////////////////////////////////////////////////////

type Node interface {
	Type() NodeType
	String() string
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

type NodeError struct {
	NodeType
	Err error
}

func NewError(err error) *NodeError {
	return &NodeError{
		NodeType: ErrorNode,
		Err: err,
	}
}

func (n NodeError) String() string {
	return n.Err.Error()
}

////////////////////////////////////////////////////////////////

type NodeToken struct {
	NodeType
	TokenType
	Data string
}

func NewToken(tt TokenType, data string) *NodeToken {
	return &NodeToken{
		NodeType: TokenNode,
		TokenType: tt,
		Data: data,
	}
}

func (n NodeToken) String() string {
	return n.Data
}

////////////////////////////////////////////////////////////////

type NodeStylesheet struct {
	NodeType
	Nodes []Node
}

func NewStylesheet() *NodeStylesheet {
	return &NodeStylesheet{
		NodeType: StylesheetNode,
	}
}

func (n NodeStylesheet) String() string {
	return NodesString(n.Nodes, "")
}

////////////////////////////////////////////////////////////////

type NodeRuleset struct {
	NodeType
	SelGroups []*NodeSelectorGroup
	DeclList *NodeDeclarationList
}

func NewRuleset() *NodeRuleset {
	return &NodeRuleset{
		NodeType: RulesetNode,
	}
}

func (n NodeRuleset) String() string {
	if n.DeclList == nil {
		return NodesString(n.SelGroups, ",") + "{}"
	}
	return NodesString(n.SelGroups, ",") + "{" + n.DeclList.String() + "}"
}

////////////////////////////////////////////////////////////////

type NodeSelectorGroup struct {
	NodeType
	Selectors []*NodeSelector
}

func NewSelectorGroup() *NodeSelectorGroup {
	return &NodeSelectorGroup{
		NodeType: SelectorGroupNode,
	}
}

func (n NodeSelectorGroup) String() string {
	return NodesString(n.Selectors, " ")
}

////////////////////////////////////////////////////////////////

type NodeSelector struct {
	NodeType
	Nodes []*NodeToken
}

func NewSelector() *NodeSelector {
	return &NodeSelector{
		NodeType: SelectorNode,
	}
}

func (n NodeSelector) String() string {
	return NodesString(n.Nodes, "")
}

////////////////////////////////////////////////////////////////

type NodeDeclarationList struct {
	NodeType
	Decls []*NodeDeclaration
}

func NewDeclarationList() *NodeDeclarationList {
	return &NodeDeclarationList{
		NodeType: DeclarationListNode,
	}
}

func (n NodeDeclarationList) String() string {
	return NodesString(n.Decls, "")
}

////////////////////////////////////////////////////////////////

type NodeDeclaration struct {
	NodeType
	Prop *NodeToken
	Vals []Node
}

func NewDeclaration(prop *NodeToken) *NodeDeclaration {
	return &NodeDeclaration{
		NodeType: DeclarationNode,
		Prop: prop,
	}
}

func (n NodeDeclaration) String() string {
	if n.Prop == nil {
		return ""
	}
	return n.Prop.String() + ":" + NodesString(n.Vals, " ") + ";"
}

////////////////////////////////////////////////////////////////

type NodeFunction struct {
	NodeType
	Func *NodeToken
	Args []*NodeToken
}

func NewFunction(f *NodeToken) *NodeFunction {
	return &NodeFunction{
		NodeType: FunctionNode,
		Func: f,
	}
}

func (n NodeFunction) String() string {
	if n.Func == nil {
		return ""
	}
	return n.Func.String() + NodesString(n.Args, ",") + ")"
}

////////////////////////////////////////////////////////////////

type NodeAtRule struct {
	NodeType
	At *NodeToken
	Nodes []*NodeToken
	Block []Node
}

func NewAtRule(at *NodeToken) *NodeAtRule {
	return &NodeAtRule{
		NodeType: AtRuleNode,
		At: at,
	}
}

func (n NodeAtRule) String() string {
	if len(n.Block) > 0 {
		return n.At.String() + " " + NodesString(n.Nodes, " ") + "{" + NodesString(n.Block, "") + "}"
	}
	if len(n.Nodes) > 0 {
		return n.At.String() + " " + NodesString(n.Nodes, " ") + ";"
	}
	return n.At.String() + ";"
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

func NodesString(inodes interface{}, delim string) string {
	if reflect.TypeOf(inodes).Kind() != reflect.Slice {
		panic("can only print a _slice_ of Node")
	}
	nodes := reflect.ValueOf(inodes)
	if nodes.Len() == 0 {
		return ""
	}

	b := &bytes.Buffer{}
	for i := 0; i < nodes.Len(); i++ {
		if n, ok := nodes.Index(i).Interface().(Node); ok {
			b.WriteString(n.String()+delim)
		} else {
			panic("can only print a slice of _Node_")
		}
	}
	s := b.String()
	return s[:len(s)-len(delim)]
}