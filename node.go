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
	FunctionNode
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
	Data string
}

// NewToken returns a new NodeToken
func NewToken(tt TokenType, data string) *NodeToken {
	return &NodeToken{
		NodeType:  TokenNode,
		TokenType: tt,
		Data:      data,
	}
}

// String returns the string representation of the node
func (n NodeToken) String() string {
	return n.Data
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
		NodeType: StylesheetNode,
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
		NodeType: RulesetNode,
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
		NodeType: SelectorGroupNode,
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
		NodeType: SelectorNode,
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
		NodeType: DeclarationNode,
		Prop:     prop,
	}
}

// String returns the string representation of the node
func (n NodeDeclaration) String() string {
	return n.Prop.String() + ":" + NodesString(n.Vals, " ") + ";"
}

////////////////////////////////////////////////////////////////

// NodeFunction represents a function and its arguments
type NodeFunction struct {
	NodeType
	Func *NodeToken
	Args []*NodeToken
}

// NewFunction returns a new NodeFunction
func NewFunction(f *NodeToken) *NodeFunction {
	return &NodeFunction{
		NodeType: FunctionNode,
		Func:     f,
	}
}

// String returns the string representation of the node
func (n NodeFunction) String() string {
	return n.Func.String() + NodesString(n.Args, ",") + ")"
}

////////////////////////////////////////////////////////////////

// NodeAtRule contains several nodes and/or a brace-block with nodes
// Block contains NodeDeclaration and NodeRuleset nodes
type NodeAtRule struct {
	NodeType
	At    *NodeToken
	Nodes []*NodeToken
	Block []Node
}

// NewAtRule returns a new NodeAtRule
func NewAtRule(at *NodeToken) *NodeAtRule {
	return &NodeAtRule{
		NodeType: AtRuleNode,
		At:       at,
	}
}

// String returns the string representation of the node
func (n NodeAtRule) String() string {
	s := n.At.String()
	if len(n.Nodes) > 0 {
		s += " " + NodesString(n.Nodes, " ")
	}
	if len(n.Block) > 0 {
		return s + "{" + NodesString(n.Block, "") + "}"
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
