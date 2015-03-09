package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
	"os"
)

////////////////////////////////////////////////////////////////

// Node is an interface that all nodes implement.
type Node interface {
	Serialize(io.Writer) error
}

////////////////////////////////////////////////////////////////

// TokenNode is a leaf node of a single token.
type TokenNode struct {
	TokenType
	Data []byte
}

// NewToken returns a new TokenNode.
func NewToken(tt TokenType, data []byte) *TokenNode {
	return &TokenNode{
		tt,
		data,
	}
}

// Serialize write to Writer the string representation of the node.
func (n TokenNode) Serialize(w io.Writer) error {
	_, err := w.Write(n.Data)
	return err
}

////////////////////////////////////////////////////////////////

// StylesheetNode is the apex node of the whole stylesheet.
// Nodes contains TokenNode, AtRuleNode, DeclarationNode and RulesetNode nodes.
type StylesheetNode struct {
	Nodes []Node
}

// NewStylesheet returns a new StylesheetNode.
func NewStylesheet() *StylesheetNode {
	return &StylesheetNode{
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n StylesheetNode) Serialize(w io.Writer) error {
	for _, m := range n.Nodes {
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// AtRuleNode contains several nodes and/or a block node.
type AtRuleNode struct {
	At    *TokenNode
	Nodes []Node
	Rules []Node
}

// NewAtRule returns a new AtRuleNode.
func NewAtRule(at *TokenNode) *AtRuleNode {
	return &AtRuleNode{
		at,
		nil,
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n AtRuleNode) Serialize(w io.Writer) error {
	if err := n.At.Serialize(w); err != nil {
		return err
	}
	for i, m := range n.Nodes {
		if i != 0 {
			var t *TokenNode
			if k, ok := n.Nodes[i-1].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			} else if k, ok := n.Nodes[i].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			}
			if t == nil || t.Data[0] != ',' {
				if _, err := w.Write([]byte(" ")); err != nil {
					return err
				}
			}
		} else {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if len(n.Rules) > 0 {
		if _, err := w.Write([]byte("{")); err != nil {
			return err
		}
		for _, m := range n.Rules {
			if err := m.Serialize(w); err != nil {
				return err
			}
		}
		if _, err := w.Write([]byte("}")); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte(";")); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// RulesetNode consists of selector groups (separated by commas) and a declaration list.
type RulesetNode struct {
	Selectors []*SelectorNode
	Decls     []*DeclarationNode
}

// NewRuleset returns a new RulesetNode.
func NewRuleset() *RulesetNode {
	return &RulesetNode{
		nil,
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n RulesetNode) Serialize(w io.Writer) error {
	for i, m := range n.Selectors {
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

// SelectorNode contains the tokens of a single selector, either TokenNode or AttributeSelectorNode.
type SelectorNode struct {
	Elems []*TokenNode
}

// NewSelector returns a new SelectorNode.
func NewSelector() *SelectorNode {
	return &SelectorNode{
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n SelectorNode) Serialize(w io.Writer) error {
	for _, m := range n.Elems {
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// DeclarationNode represents a property declaration.
// Vals contains FunctionNode and TokenNode nodes.
type DeclarationNode struct {
	Prop      *TokenNode
	Vals      []Node
	Important bool
}

// NewDeclaration returns a new DeclarationNode.
func NewDeclaration(prop *TokenNode) *DeclarationNode {
	prop.Data = bytes.ToLower(prop.Data)
	return &DeclarationNode{
		prop,
		nil,
		false,
	}
}

// Serialize write to Writer the string representation of the node.
func (n DeclarationNode) Serialize(w io.Writer) error {
	if err := n.Prop.Serialize(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte(":")); err != nil {
		return err
	}
	for i, m := range n.Vals {
		if i != 0 {
			var t *TokenNode
			if k, ok := n.Vals[i-1].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			} else if k, ok := n.Vals[i].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			}
			if t == nil || (t.Data[0] != ',' && t.Data[0] != '/' && t.Data[0] != ':' && t.Data[0] != '.') {
				if _, err := w.Write([]byte(" ")); err != nil {
					return err
				}
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	if n.Important {
		if _, err := w.Write([]byte(" !important")); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte(";")); err != nil {
		return err
	}
	return nil
}

////////////////////////////////////////////////////////////////

// FunctionNode represents a function and its arguments (separated by commas).
type FunctionNode struct {
	Func *TokenNode
	Args []*ArgumentNode
}

// NewFunction returns a new FunctionNode.
func NewFunction(f *TokenNode) *FunctionNode {
	f.Data = f.Data[:len(f.Data)-1]
	return &FunctionNode{
		f,
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n FunctionNode) Serialize(w io.Writer) error {
	if err := n.Func.Serialize(w); err != nil {
		return err
	}
	if _, err := w.Write([]byte("(")); err != nil {
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

// ArgumentNode represents the key and parts of an argument separated by spaces.
type ArgumentNode struct {
	Vals []Node
}

// NewArgument returns a new ArgumentNode.
func NewArgument() *ArgumentNode {
	return &ArgumentNode{
		[]Node{},
	}
}

// Serialize write to Writer the string representation of the node.
func (n ArgumentNode) Serialize(w io.Writer) error {
	for i, m := range n.Vals {
		if i != 0 {
			var t *TokenNode
			if k, ok := n.Vals[i-1].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			} else if k, ok := n.Vals[i].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			}
			if t == nil || (t.Data[0] != '=' && t.Data[0] != '*' && t.Data[0] != '/') {
				if _, err := w.Write([]byte(" ")); err != nil {
					return err
				}
			}
		}
		if err := m.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////

// BlockNode contains the contents of a block (brace, bracket or parenthesis blocks).
// Nodes contains AtRuleNode, DeclarationNode, RulesetNode and BlockNode nodes.
type BlockNode struct {
	Open  *TokenNode
	Nodes []Node
	Close *TokenNode
}

// NewBlock returns a new BlockNode.
func NewBlock(open *TokenNode) *BlockNode {
	return &BlockNode{
		open,
		nil,
		nil,
	}
}

// Serialize write to Writer the string representation of the node.
func (n BlockNode) Serialize(w io.Writer) error {
	if len(n.Nodes) > 0 {
		if err := n.Open.Serialize(w); err != nil {
			return err
		}
		if n.Close == nil {
			w = os.Stderr
		}
		for i, m := range n.Nodes {
			if i != 0 {
				var t *TokenNode
				if k, ok := n.Nodes[i-1].(*TokenNode); ok && len(k.Data) == 1 {
					t = k
				} else if k, ok := n.Nodes[i].(*TokenNode); ok && len(k.Data) == 1 {
					t = k
				}
				if t == nil || t.Data[0] != ':' {
					if _, err := w.Write([]byte(" ")); err != nil {
						return err
					}
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
