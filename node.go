package css

import (
	"bytes"
	"io"
)

////////////////////////////////////////////////////////////////

// Node is an interface that all nodes implement
type Node interface {
	Serialize(io.Writer)
}

////////////////////////////////////////////////////////////////

// NodeToken is a leaf node of a single token
type NodeToken struct {
	TokenType
	Data []byte
}

// NewToken returns a new NodeToken
func NewToken(tt TokenType, data []byte) *NodeToken {
	return &NodeToken{
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
	Nodes []Node
}

// NewStylesheet returns a new NodeStylesheet
func NewStylesheet() *NodeStylesheet {
	return &NodeStylesheet{
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeStylesheet) Equals(other *NodeStylesheet) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		switch m := n.Nodes[i].(type) {
		case *NodeToken:
			if o, ok := otherNode.(*NodeToken); !ok || !m.Equals(o) {
				return false
			}
		case *NodeAtRule:
			if o, ok := otherNode.(*NodeAtRule); !ok || !m.Equals(o) {
				return false
			}
		case *NodeDeclaration:
			if o, ok := otherNode.(*NodeDeclaration); !ok || !m.Equals(o) {
				return false
			}
		case *NodeRuleset:
			if o, ok := otherNode.(*NodeRuleset); !ok || !m.Equals(o) {
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
	SelGroups []*NodeSelectorGroup
	Decls     []*NodeDeclaration
}

// NewRuleset returns a new NodeRuleset
func NewRuleset() *NodeRuleset {
	return &NodeRuleset{
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
	Selectors []*NodeSelector
}

// NewSelectorGroup returns a new NodeSelectorGroup
func NewSelectorGroup() *NodeSelectorGroup {
	return &NodeSelectorGroup{
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
	Nodes []Node
}

// NewSelector returns a new NodeSelector
func NewSelector() *NodeSelector {
	return &NodeSelector{
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n NodeSelector) Equals(other *NodeSelector) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		switch m := n.Nodes[i].(type) {
		case *NodeToken:
			if o, ok := otherNode.(*NodeToken); !ok || !m.Equals(o) {
				return false
			}
		case *NodeAttributeSelector:
			if o, ok := otherNode.(*NodeAttributeSelector); !ok || !m.Equals(o) {
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
	Key *NodeToken
	Op *NodeToken
	Vals []*NodeToken
}

// NewAttributeSelector returns a new NodeSelector
func NewAttributeSelector(key *NodeToken) *NodeAttributeSelector {
	return &NodeAttributeSelector{
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
	Prop *NodeToken
	Vals []Node
}

// NewDeclaration returns a new NodeDeclaration
func NewDeclaration(prop *NodeToken) *NodeDeclaration {
	return &NodeDeclaration{
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
		switch m := n.Vals[i].(type) {
		case *NodeToken:
			if o, ok := otherNode.(*NodeToken); !ok || !m.Equals(o) {
				return false
			}
		case *NodeFunction:
			if o, ok := otherNode.(*NodeFunction); !ok || !m.Equals(o) {
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
	Key *NodeToken
	Val *NodeToken
}

// NewArgument returns a new NodeArgument
func NewArgument(key, val *NodeToken) *NodeArgument {
	return &NodeArgument{
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
	Func *NodeToken
	Args []*NodeArgument
}

// NewFunction returns a new NodeFunction
func NewFunction(f *NodeToken) *NodeFunction {
	return &NodeFunction{
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
	Open  *NodeToken
	Nodes []Node
	Close *NodeToken
}

// NewBlock returns a new NodeBlock
func NewBlock(open *NodeToken) *NodeBlock {
	return &NodeBlock{
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
		switch m := n.Nodes[i].(type) {
		case *NodeAtRule:
			if o, ok := otherNode.(*NodeAtRule); !ok || !m.Equals(o) {
				return false
			}
		case *NodeDeclaration:
			if o, ok := otherNode.(*NodeDeclaration); !ok || !m.Equals(o) {
				return false
			}
		case *NodeRuleset:
			if o, ok := otherNode.(*NodeRuleset); !ok || !m.Equals(o) {
				return false
			}
		case *NodeBlock:
			if o, ok := otherNode.(*NodeBlock); !ok || !m.Equals(o) {
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
	At    *NodeToken
	Nodes []*NodeToken
	Block *NodeBlock
}

// NewAtRule returns a new NodeAtRule
func NewAtRule(at *NodeToken) *NodeAtRule {
	return &NodeAtRule{
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
