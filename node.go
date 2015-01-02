package css

import (
	"bytes"
	"reflect"
)

////////////////////////////////////////////////////////////////

// NodeType determines the type of node, eg. a block or a declaration.
type NodeType uint32

// NodeType values, it is safe to cast a node to the referred node type
const (
	ErrorNode NodeType = iota // extra node when errors occur
	StylesheetNode
	RulesetNode
	SelectorGroupNode
	SelectorNode
	DeclarationListNode
	DeclarationNode
	ArgumentNode
	FunctionNode
	BlockNode
	AtRuleNode
	TokenNode // extra node for simple tokens
)

// Type returns the node type, it implements the function in interface Node for all nodes
func (t NodeType) Type() NodeType {
	return t
}

////////////////////////////////////////////////////////////////

// Node is an interface that all nodes implement
type Node interface {
	Type() NodeType
	String() string
}

////////////////////////////////////////////////////////////////

// NodeToken is a leaf node of a single token
type NodeToken struct {
	NodeType
	TokenType
	Data []byte
}

// NewToken returns a new NodeToken
func NewToken(tt TokenType, data []byte) *NodeToken {
	return &NodeToken{
		NodeType:  TokenNode,
		TokenType: tt,
		Data:      data,
	}
}

// String returns the string representation of the node
func (n NodeToken) String() string {
	return string(n.Data)
}

////////////////////////////////////////////////////////////////

// NodeStylesheet is the apex node of the whole stylesheet
// Nodes contains NodeToken, NodeAtRule, NodeDeclaration and NodeRuleset nodes
type NodeStylesheet struct {
	NodeType
	Nodes []Node
}

// NewStylesheet returns a new NodeStylesheet
func NewStylesheet() *NodeStylesheet {
	return &NodeStylesheet{
		StylesheetNode,
		make([]Node, 0, 10),
	}
}

// String returns the string representation of the node
func (n NodeStylesheet) String() string {
	return NodesString(n.Nodes, "")
}

////////////////////////////////////////////////////////////////

// NodeRuleset consists of selector groups (separated by commas) and a declaration list
type NodeRuleset struct {
	NodeType
	SelGroups []*NodeSelectorGroup
	Decls     []*NodeDeclaration
}

// NewRuleset returns a new NodeRuleset
func NewRuleset() *NodeRuleset {
	return &NodeRuleset{
		RulesetNode,
		make([]*NodeSelectorGroup, 0, 1),
		make([]*NodeDeclaration, 0, 5),
	}
}

// String returns the string representation of the node
func (n NodeRuleset) String() string {
	return NodesString(n.SelGroups, ",") + "{" + NodesString(n.Decls, "") + "}"
}

////////////////////////////////////////////////////////////////

// NodeSelectorGroup contains selectors separated by whitespace
type NodeSelectorGroup struct {
	NodeType
	Selectors []*NodeSelector
}

// NewSelectorGroup returns a new NodeSelectorGroup
func NewSelectorGroup() *NodeSelectorGroup {
	return &NodeSelectorGroup{
		SelectorGroupNode,
		make([]*NodeSelector, 0, 3),
	}
}

// String returns the string representation of the node
func (n NodeSelectorGroup) String() string {
	return NodesString(n.Selectors, " ")
}

////////////////////////////////////////////////////////////////

// NodeSelector contains thee tokens of a single selector
type NodeSelector struct {
	NodeType
	Nodes []*NodeToken
}

// NewSelector returns a new NodeSelector
func NewSelector() *NodeSelector {
	return &NodeSelector{
		SelectorNode,
		make([]*NodeToken, 0, 2),
	}
}

// String returns the string representation of the node
func (n NodeSelector) String() string {
	return NodesString(n.Nodes, "")
}

////////////////////////////////////////////////////////////////

// NodeDeclaration represents a property declaration
// Vals contains NodeFunction and NodeToken nodes
type NodeDeclaration struct {
	NodeType
	Prop *NodeToken
	Vals []Node
}

// NewDeclaration returns a new NodeDeclaration
func NewDeclaration(prop *NodeToken) *NodeDeclaration {
	return &NodeDeclaration{
		DeclarationNode,
		prop,
		make([]Node, 0, 1),
	}
}

// String returns the string representation of the node
func (n NodeDeclaration) String() string {
	return n.Prop.String() + ":" + NodesString(n.Vals, " ") + ";"
}

////////////////////////////////////////////////////////////////

// NodeFunction represents a function and its arguments
type NodeArgument struct {
	NodeType
	Key *NodeToken
	Val *NodeToken
}

// NewArgument returns a new NodeArgument
func NewArgument(key, val *NodeToken) *NodeArgument {
	return &NodeArgument{
		ArgumentNode,
		key,
		val,
	}
}

// String returns the string representation of the node
func (n NodeArgument) String() string {
	if n.Key == nil {
		return n.Val.String()
	}
	return n.Key.String() + "=" + n.Val.String()
}

////////////////////////////////////////////////////////////////

// NodeFunction represents a function and its arguments
type NodeFunction struct {
	NodeType
	Func *NodeToken
	Args []*NodeArgument
}

// NewFunction returns a new NodeFunction
func NewFunction(f *NodeToken) *NodeFunction {
	return &NodeFunction{
		FunctionNode,
		f,
		make([]*NodeArgument, 0, 3),
	}
}

// String returns the string representation of the node
func (n NodeFunction) String() string {
	return n.Func.String() + NodesString(n.Args, ",") + ")"
}

////////////////////////////////////////////////////////////////

// NodeBlock contains the contents of a block (brace, bracket or parenthesis blocks)
// Nodes contains NodeAtRule, NodeDeclaration, NodeRuleset and NodeBlock nodes
type NodeBlock struct {
	NodeType
	Open  *NodeToken
	Nodes []Node
	Close *NodeToken
}

// NewBlock returns a new NodeBlock
func NewBlock(open *NodeToken) *NodeBlock {
	return &NodeBlock{
		BlockNode,
		open,
		make([]Node, 0, 5),
		nil,
	}
}

// String returns the string representation of the node
func (n NodeBlock) String() string {
	if len(n.Nodes) > 0 {
		return n.Open.String() + NodesString(n.Nodes, " ") + n.Close.String()
	}
	return ""
}

////////////////////////////////////////////////////////////////

// NodeAtRule contains several nodes and/or a block node
type NodeAtRule struct {
	NodeType
	At    *NodeToken
	Nodes []*NodeToken
	Block *NodeBlock
}

// NewAtRule returns a new NodeAtRule
func NewAtRule(at *NodeToken) *NodeAtRule {
	return &NodeAtRule{
		AtRuleNode,
		at,
		make([]*NodeToken, 0, 3),
		nil,
	}
}

// String returns the string representation of the node
func (n NodeAtRule) String() string {
	s := n.At.String()
	if len(n.Nodes) > 0 {
		s += " " + NodesString(n.Nodes, " ")
	}
	if n.Block != nil {
		return s + " " + n.Block.String()
	}
	return s + ";"
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

// NodesString returns the joined node String()s by delim
func NodesString(inodes interface{}, delim string) string {
	if reflect.TypeOf(inodes).Kind() != reflect.Slice {
		return ""
	}
	b := &bytes.Buffer{}
	nodes := reflect.ValueOf(inodes)
	for i := 0; i < nodes.Len(); i++ {
		if n, ok := nodes.Index(i).Interface().(Node); ok {
			if _, err := b.WriteString(n.String() + delim); err != nil {
				break
			}
		} else {
			break
		}
	}
	s := b.String()
	return s[:len(s)-len(delim)]
}
