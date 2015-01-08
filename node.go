package css

import (
	"bytes"
	"io"
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
	AttributeSelectorNode
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
	Serialize(io.Writer)
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
		TokenNode,
		tt,
		data,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeToken) Equals(other *NodeToken) bool {
	return n.TokenType == other.TokenType && bytes.Equal(n.Data, other.Data)
}

// Serialize write to Writer the string representation of the node
func (n NodeToken) Serialize(w io.Writer) {
	w.Write(n.Data)
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
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeStylesheet) Equals(other *NodeStylesheet) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		if n.Nodes[i].Type() != otherNode.Type() {
			return false
		}
		switch n.Nodes[i].Type() {
		case TokenNode:
			if !n.Nodes[i].(*NodeToken).Equals(otherNode.(*NodeToken)) {
				return false
			}
		case AtRuleNode:
			if !n.Nodes[i].(*NodeAtRule).Equals(otherNode.(*NodeAtRule)) {
				return false
			}
		case DeclarationNode:
			if !n.Nodes[i].(*NodeDeclaration).Equals(otherNode.(*NodeDeclaration)) {
				return false
			}
		case RulesetNode:
			if !n.Nodes[i].(*NodeRuleset).Equals(otherNode.(*NodeRuleset)) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeStylesheet) Serialize(w io.Writer) {
	for _, m := range n.Nodes {
		m.Serialize(w)
	}
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
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeRuleset) Equals(other *NodeRuleset) bool {
	if len(n.SelGroups) != len(other.SelGroups) || len(n.Decls) != len(other.Decls) {
		return false
	}
	for i, otherNode := range other.SelGroups {
		if !n.SelGroups[i].Equals(otherNode) {
			return false
		}
	}
	for i, otherNode := range other.Decls {
		if !n.Decls[i].Equals(otherNode) {
			return false
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeRuleset) Serialize(w io.Writer) {
	for i, m := range n.SelGroups {
		if i != 0 {
			w.Write([]byte(","))
		}
		m.Serialize(w)
	}
	w.Write([]byte("{"))
	for _, m := range n.Decls {
		m.Serialize(w)
	}
	w.Write([]byte("}"))
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
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeSelectorGroup) Equals(other *NodeSelectorGroup) bool {
	if len(n.Selectors) != len(other.Selectors) {
		return false
	}
	for i, otherNode := range other.Selectors {
		if !n.Selectors[i].Equals(otherNode) {
			return false
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeSelectorGroup) Serialize(w io.Writer) {
	for i, m := range n.Selectors {
		if i != 0 {
			w.Write([]byte(" "))
		}
		m.Serialize(w)
	}
}

////////////////////////////////////////////////////////////////

// NodeSelector contains the tokens of a single selector, either TokenNode or AttributeSelectorNode
type NodeSelector struct {
	NodeType
	Nodes []Node
}

// NewSelector returns a new NodeSelector
func NewSelector() *NodeSelector {
	return &NodeSelector{
		SelectorNode,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeSelector) Equals(other *NodeSelector) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		if n.Nodes[i].Type() != otherNode.Type() {
			return false
		}
		switch n.Nodes[i].Type() {
		case TokenNode:
			if !n.Nodes[i].(*NodeToken).Equals(otherNode.(*NodeToken)) {
				return false
			}
		case AttributeSelectorNode:
			if !n.Nodes[i].(*NodeAttributeSelector).Equals(otherNode.(*NodeAttributeSelector)) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeSelector) Serialize(w io.Writer) {
	for _, m := range n.Nodes {
		m.Serialize(w)
	}
}

////////////////////////////////////////////////////////////////

// NodeAttributeSelector contains the key and possible the operators with values as TokenNodes of an attribute selector
type NodeAttributeSelector struct {
	NodeType
	Key *NodeToken
	Op *NodeToken
	Vals []*NodeToken
}

// NewAttributeSelector returns a new NodeSelector
func NewAttributeSelector(key *NodeToken) *NodeAttributeSelector {
	return &NodeAttributeSelector{
		AttributeSelectorNode,
		key,
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeAttributeSelector) Equals(other *NodeAttributeSelector) bool {
	if !n.Key.Equals(other.Key) || len(n.Vals) != len(other.Vals) {
		return false
	}
	if n.Op == nil && other.Op != nil || !n.Op.Equals(other.Op) {
		return false
	}
	for i, otherNode := range other.Vals {
		if !n.Vals[i].Equals(otherNode) {
			return false
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeAttributeSelector) Serialize(w io.Writer) {
	w.Write([]byte("["))
	n.Key.Serialize(w)
	if n.Op != nil {
		n.Op.Serialize(w)
		for _, m := range n.Vals {
			m.Serialize(w)
		}
	}
	w.Write([]byte("]"))
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
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeDeclaration) Equals(other *NodeDeclaration) bool {
	if !n.Prop.Equals(other.Prop) || len(n.Vals) != len(other.Vals) {
		return false
	}
	for i, otherNode := range other.Vals {
		if n.Vals[i].Type() != otherNode.Type() {
			return false
		}
		switch n.Vals[i].Type() {
		case TokenNode:
			if !n.Vals[i].(*NodeToken).Equals(otherNode.(*NodeToken)) {
				return false
			}
		case FunctionNode:
			if !n.Vals[i].(*NodeFunction).Equals(otherNode.(*NodeFunction)) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeDeclaration) Serialize(w io.Writer) {
	n.Prop.Serialize(w)
	w.Write([]byte(":"))
	for i, m := range n.Vals {
		if i != 0 {
			w.Write([]byte(" "))
		}
		m.Serialize(w)
	}
	w.Write([]byte(";"))
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

// Equals returns true when the nodes match (deep)
func (n NodeArgument) Equals(other *NodeArgument) bool {
	if !n.Val.Equals(other.Val) {
		return false
	}
	if n.Key == nil && other.Key != nil || !n.Key.Equals(other.Key) {
		return false
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeArgument) Serialize(w io.Writer) {
	if n.Key != nil {
		n.Key.Serialize(w)
		w.Write([]byte("="))
	}
	n.Val.Serialize(w)
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
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeFunction) Equals(other *NodeFunction) bool {
	if !n.Func.Equals(other.Func) {
		return false
	}
	for i, otherNode := range other.Args {
		if !n.Args[i].Equals(otherNode) {
			return false
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeFunction) Serialize(w io.Writer) {
	n.Func.Serialize(w)
	for i, m := range n.Args {
		if i != 0 {
			w.Write([]byte(","))
		}
		m.Serialize(w)
	}
	w.Write([]byte(")"))
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
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeBlock) Equals(other *NodeBlock) bool {
	if !n.Open.Equals(other.Open) || !n.Close.Equals(other.Close) {
		return false
	}
	for i, otherNode := range other.Nodes {
		if n.Nodes[i].Type() != otherNode.Type() {
			return false
		}
		switch n.Nodes[i].Type() {
		case AtRuleNode:
			if !n.Nodes[i].(*NodeAtRule).Equals(otherNode.(*NodeAtRule)) {
				return false
			}
		case DeclarationNode:
			if !n.Nodes[i].(*NodeDeclaration).Equals(otherNode.(*NodeDeclaration)) {
				return false
			}
		case RulesetNode:
			if !n.Nodes[i].(*NodeRuleset).Equals(otherNode.(*NodeRuleset)) {
				return false
			}
		case BlockNode:
			if !n.Nodes[i].(*NodeBlock).Equals(otherNode.(*NodeBlock)) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeBlock) Serialize(w io.Writer) {
	if len(n.Nodes) > 0 {
		n.Open.Serialize(w)
		for i, m := range n.Nodes {
			if i != 0 {
				w.Write([]byte(" "))
			}
			m.Serialize(w)
		}
		n.Close.Serialize(w)
	}
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
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeAtRule) Equals(other *NodeAtRule) bool {
	if !n.At.Equals(other.At) || len(n.Nodes) != len(other.Nodes) {
		return false
	}
	if n.Block == nil && other.Block != nil || !n.Block.Equals(other.Block) {
		return false
	}
	for i, otherNode := range other.Nodes {
		if !n.Nodes[i].Equals(otherNode) {
			return false
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n NodeAtRule) Serialize(w io.Writer) {
	n.At.Serialize(w)
	for _, m := range n.Nodes {
		w.Write([]byte(" "))
		m.Serialize(w)
	}
	if n.Block != nil {
		w.Write([]byte(" "))
		n.Block.Serialize(w)
	} else {
		w.Write([]byte(";"))
	}
}
