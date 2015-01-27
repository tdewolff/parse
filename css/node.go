package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
)

////////////////////////////////////////////////////////////////

// Node is an interface that all nodes implement
type Node interface {
	Serialize(io.Writer) error
}

////////////////////////////////////////////////////////////////

// TokenNode is a leaf node of a single token
type TokenNode struct {
	TokenType
	Data []byte
}

// NewToken returns a new TokenNode
func NewToken(tt TokenType, data []byte) *TokenNode {
	return &TokenNode{
		tt,
		data,
	}
}

// Equals returns true when the nodes match (deep)
func (n TokenNode) Equals(other *TokenNode) bool {
	return n.TokenType == other.TokenType && bytes.Equal(n.Data, other.Data)
}

// Serialize write to Writer the string representation of the node
func (n TokenNode) Serialize(w io.Writer) error {
	_, err := w.Write(n.Data)
	return err
}

////////////////////////////////////////////////////////////////

// StylesheetNode is the apex node of the whole stylesheet
// Nodes contains TokenNode, AtRuleNode, DeclarationNode and RulesetNode nodes
type StylesheetNode struct {
	Nodes []Node
}

// NewStylesheet returns a new StylesheetNode
func NewStylesheet() *StylesheetNode {
	return &StylesheetNode{
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n StylesheetNode) Equals(other *StylesheetNode) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		switch m := n.Nodes[i].(type) {
		case *TokenNode:
			if o, ok := otherNode.(*TokenNode); !ok || !m.Equals(o) {
				return false
			}
		case *AtRuleNode:
			if o, ok := otherNode.(*AtRuleNode); !ok || !m.Equals(o) {
				return false
			}
		case *DeclarationNode:
			if o, ok := otherNode.(*DeclarationNode); !ok || !m.Equals(o) {
				return false
			}
		case *RulesetNode:
			if o, ok := otherNode.(*RulesetNode); !ok || !m.Equals(o) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n StylesheetNode) Serialize(w io.Writer) error {
	for _, m := range n.Nodes {
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// RulesetNode consists of selector groups (separated by commas) and a declaration list
type RulesetNode struct {
	SelGroups []*SelectorGroupNode
	Decls     []*DeclarationNode
}

// NewRuleset returns a new RulesetNode
func NewRuleset() *RulesetNode {
	return &RulesetNode{
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n RulesetNode) Equals(other *RulesetNode) bool {
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
func (n RulesetNode) Serialize(w io.Writer) error {
	for i, m := range n.SelGroups {
		if i != 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("{")); err != nil {
		return err
	}
	for _, m := range n.Decls {
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte("}")); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// SelectorGroupNode contains selectors separated by whitespace
type SelectorGroupNode struct {
	Selectors []*SelectorNode
}

// NewSelectorGroup returns a new SelectorGroupNode
func NewSelectorGroup() *SelectorGroupNode {
	return &SelectorGroupNode{
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n SelectorGroupNode) Equals(other *SelectorGroupNode) bool {
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
func (n SelectorGroupNode) Serialize(w io.Writer) error {
	for i, m := range n.Selectors {
		if i != 0 {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// SelectorNode contains the tokens of a single selector, either TokenNode or AttributeSelectorNode
type SelectorNode struct {
	Nodes []Node
}

// NewSelector returns a new SelectorNode
func NewSelector() *SelectorNode {
	return &SelectorNode{
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n SelectorNode) Equals(other *SelectorNode) bool {
	if len(n.Nodes) != len(other.Nodes) {
		return false
	}
	for i, otherNode := range other.Nodes {
		switch m := n.Nodes[i].(type) {
		case *TokenNode:
			if o, ok := otherNode.(*TokenNode); !ok || !m.Equals(o) {
				return false
			}
		case *AttributeSelectorNode:
			if o, ok := otherNode.(*AttributeSelectorNode); !ok || !m.Equals(o) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n SelectorNode) Serialize(w io.Writer) error {
	for _, m := range n.Nodes {
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// AttributeSelectorNode contains the key and possible the operators with values as TokenNodes of an attribute selector
type AttributeSelectorNode struct {
	Key  *TokenNode
	Op   *TokenNode
	Vals []*TokenNode
}

// NewAttributeSelector returns a new SelectorNode
func NewAttributeSelector(key *TokenNode) *AttributeSelectorNode {
	return &AttributeSelectorNode{
		key,
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n AttributeSelectorNode) Equals(other *AttributeSelectorNode) bool {
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
func (n AttributeSelectorNode) Serialize(w io.Writer) error {
	if _, err := w.Write([]byte("[")); err != nil {
		return err
	}
	if err := n.Key.Serialize(w); err != nil {
		return err
	}
	if n.Op != nil {
		if err := n.Op.Serialize(w); err != nil {
			return err
		}
		for _, m := range n.Vals {
			if err := m.Serialize(w); err != nil {
				return err
			}
		}
	}
	if _, err := w.Write([]byte("]")); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// DeclarationNode represents a property declaration
// Vals contains FunctionNode and TokenNode nodes
type DeclarationNode struct {
	Prop *TokenNode
	Vals []Node
}

// NewDeclaration returns a new DeclarationNode
func NewDeclaration(prop *TokenNode) *DeclarationNode {
	return &DeclarationNode{
		prop,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n DeclarationNode) Equals(other *DeclarationNode) bool {
	if !n.Prop.Equals(other.Prop) || len(n.Vals) != len(other.Vals) {
		return false
	}
	for i, otherNode := range other.Vals {
		switch m := n.Vals[i].(type) {
		case *TokenNode:
			if o, ok := otherNode.(*TokenNode); !ok || !m.Equals(o) {
				return false
			}
		case *FunctionNode:
			if o, ok := otherNode.(*FunctionNode); !ok || !m.Equals(o) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n DeclarationNode) Serialize(w io.Writer) error {
	if err := n.Prop.Serialize(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte(":")); err != nil {
		return err
	}
	for i, m := range n.Vals {
		if i != 0 {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte(";")); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// FunctionNode represents a function and its arguments
type ArgumentNode struct {
	Key *TokenNode
	Val *TokenNode
}

// NewArgument returns a new ArgumentNode
func NewArgument(key, val *TokenNode) *ArgumentNode {
	return &ArgumentNode{
		key,
		val,
	}
}

// Equals returns true when the nodes match (deep)
func (n ArgumentNode) Equals(other *ArgumentNode) bool {
	if !n.Val.Equals(other.Val) {
		return false
	}
	if n.Key == nil && other.Key != nil || !n.Key.Equals(other.Key) {
		return false
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n ArgumentNode) Serialize(w io.Writer) error {
	if n.Key != nil {
		if err := n.Key.Serialize(w); err != nil {
			return err
		}
		if _, err := w.Write([]byte("=")); err != nil {
			return err
		}
	}
	if err := n.Val.Serialize(w); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// FunctionNode represents a function and its arguments
type FunctionNode struct {
	Func *TokenNode
	Args []*ArgumentNode
}

// NewFunction returns a new FunctionNode
func NewFunction(f *TokenNode) *FunctionNode {
	return &FunctionNode{
		f,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n FunctionNode) Equals(other *FunctionNode) bool {
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
func (n FunctionNode) Serialize(w io.Writer) error {
	if err := n.Func.Serialize(w); err != nil {
		return err
	}
	for i, m := range n.Args {
		if i != 0 {
			if _, err := w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte(")")); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// BlockNode contains the contents of a block (brace, bracket or parenthesis blocks)
// Nodes contains AtRuleNode, DeclarationNode, RulesetNode and BlockNode nodes
type BlockNode struct {
	Open  *TokenNode
	Nodes []Node
	Close *TokenNode
}

// NewBlock returns a new BlockNode
func NewBlock(open *TokenNode) *BlockNode {
	return &BlockNode{
		open,
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n BlockNode) Equals(other *BlockNode) bool {
	if !n.Open.Equals(other.Open) || !n.Close.Equals(other.Close) {
		return false
	}
	for i, otherNode := range other.Nodes {
		switch m := n.Nodes[i].(type) {
		case *AtRuleNode:
			if o, ok := otherNode.(*AtRuleNode); !ok || !m.Equals(o) {
				return false
			}
		case *DeclarationNode:
			if o, ok := otherNode.(*DeclarationNode); !ok || !m.Equals(o) {
				return false
			}
		case *RulesetNode:
			if o, ok := otherNode.(*RulesetNode); !ok || !m.Equals(o) {
				return false
			}
		case *BlockNode:
			if o, ok := otherNode.(*BlockNode); !ok || !m.Equals(o) {
				return false
			}
		}
	}
	return true
}

// Serialize write to Writer the string representation of the node
func (n BlockNode) Serialize(w io.Writer) error {
	if len(n.Nodes) > 0 {
		if err := n.Open.Serialize(w); err != nil {
			return err
		}
		for i, m := range n.Nodes {
			if i != 0 {
				if _, err := w.Write([]byte(" ")); err != nil {
					return err
				}
			}
			if err := m.Serialize(w); err != nil {
				return err
			}
		}
		if err := n.Close.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// AtRuleNode contains several nodes and/or a block node
type AtRuleNode struct {
	At    *TokenNode
	Nodes []*TokenNode
	Block *BlockNode
}

// NewAtRule returns a new AtRuleNode
func NewAtRule(at *TokenNode) *AtRuleNode {
	return &AtRuleNode{
		at,
		nil,
		nil,
	}
}

// Equals returns true when the nodes match (deep)
func (n AtRuleNode) Equals(other *AtRuleNode) bool {
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
func (n AtRuleNode) Serialize(w io.Writer) error {
	if err := n.At.Serialize(w); err != nil {
		return err
	}
	for _, m := range n.Nodes {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if n.Block != nil {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if err := n.Block.Serialize(w); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte(";")); err != nil {
			return err
		}
	}
	return nil
}
