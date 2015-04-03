package css // import "github.com/tdewolff/parse/css"

import (
	"io"
)

var (
	commaBytes            = []byte(",")
	spaceBytes            = []byte(" ")
	colonBytes            = []byte(":")
	semicolonBytes        = []byte(";")
	leftParenthesisBytes  = []byte("(")
	rightParenthesisBytes = []byte(")")
	leftBracketBytes      = []byte("{")
	rightBracketBytes     = []byte("}")
)

// Node is an interface that all nodes implement.
type Node interface {
	WriteTo(io.Writer) (int64, error)
}

////////////////////////////////////////////////////////////////

// TokenNode is a leaf node of a single token.
type TokenNode struct {
	TokenType
	Data []byte
}

// WriteTo writes the string representation of the node to the writer.
func (token TokenNode) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(token.Data)
	return int64(n), err
}

////////////////////////////////////////////////////////////////

// StylesheetNode is the apex node of the whole stylesheet.
// Nodes contains TokenNode, AtRuleNode, DeclarationNode and RulesetNode nodes.
type StylesheetNode struct {
	Nodes []Node
}

// WriteTo writes the string representation of the node to the writer.
func (stylesheet StylesheetNode) WriteTo(w io.Writer) (size int64, err error) {
	var m int64
	for _, node := range stylesheet.Nodes {
		m, err = node.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	return
}

////////////////////////////////////////////////////////////////

// AtRuleNode contains several nodes and/or a block node.
type AtRuleNode struct {
	Name  *TokenNode
	Nodes []Node
	Rules []Node
}

// WriteTo writes the string representation of the node to the writer.
func (atrule AtRuleNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	m, err = atrule.Name.WriteTo(w)
	if err != nil {
		return
	}
	size += m
	for i, node := range atrule.Nodes {
		if i != 0 {
			var t *TokenNode
			if k, ok := atrule.Nodes[i-1].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			} else if k, ok := atrule.Nodes[i].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			}
			if t == nil || t.Data[0] != ',' {
				n, err = w.Write(spaceBytes)
				if err != nil {
					return
				}
				size += int64(n)
			}
		} else {
			n, err = w.Write(spaceBytes)
			if err != nil {
				return
			}
			size += int64(n)
		}
		m, err = node.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	if len(atrule.Rules) > 0 {
		n, err = w.Write(leftBracketBytes)
		if err != nil {
			return
		}
		size += int64(n)
		for _, rule := range atrule.Rules {
			m, err = rule.WriteTo(w)
			if err != nil {
				return
			}
			size += m
		}
		n, err = w.Write(rightBracketBytes)
		if err != nil {
			return
		}
		size += int64(n)
	} else {
		n, err = w.Write(semicolonBytes)
		if err != nil {
			return
		}
		size += int64(n)
	}
	return
}

////////////////////////////////////////////////////////////////

// RulesetNode consists of selector groups (separated by commas) and a declaration list.
type RulesetNode struct {
	Selectors []SelectorNode
	Decls     []*DeclarationNode
}

// WriteTo writes the string representation of the node to the writer.
func (ruleset RulesetNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	for i, sel := range ruleset.Selectors {
		if i != 0 {
			n, err = w.Write(commaBytes)
			if err != nil {
				return
			}
			size += int64(n)
		}
		m, err = sel.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	n, err = w.Write(leftBracketBytes)
	if err != nil {
		return
	}
	size += int64(n)
	for _, decl := range ruleset.Decls {
		m, err = decl.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	n, err = w.Write(rightBracketBytes)
	if err != nil {
		return
	}
	size += int64(n)
	return
}

////////////////////////////////////////////////////////////////

// SelectorNode contains the tokens of a single selector.
type SelectorNode struct {
	Elems []*TokenNode
}

// WriteTo writes the string representation of the node to the writer.
func (sel SelectorNode) WriteTo(w io.Writer) (size int64, err error) {
	var m int64
	for _, elem := range sel.Elems {
		m, err = elem.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	return
}

////////////////////////////////////////////////////////////////

// DeclarationNode represents a property declaration.
// Vals contains FunctionNode and TokenNode nodes.
type DeclarationNode struct {
	Prop *TokenNode
	Vals []Node
}

// WriteTo writes the string representation of the node to the writer.
func (decl DeclarationNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	m, err = decl.Prop.WriteTo(w)
	if err != nil {
		return
	}
	size += m
	n, err = w.Write(colonBytes)
	if err != nil {
		return
	}
	size += int64(n)
	for i, val := range decl.Vals {
		if i != 0 {
			var t *TokenNode
			if k, ok := decl.Vals[i-1].(*TokenNode); ok && len(k.Data) == 1 && k.TokenType != IdentToken {
				t = k
			} else if k, ok := decl.Vals[i].(*TokenNode); ok && len(k.Data) == 1 && k.TokenType != IdentToken {
				t = k
			}
			if t == nil || (t.Data[0] != ',' && t.Data[0] != '/' && t.Data[0] != ':' && t.Data[0] != '.' && t.Data[0] != '!') {
				n, err = w.Write(spaceBytes)
				if err != nil {
					return
				}
				size += int64(n)
			}
		}
		m, err = val.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	n, err = w.Write(semicolonBytes)
	if err != nil {
		return
	}
	size += int64(n)
	return
}

////////////////////////////////////////////////////////////////

// FunctionNode represents a function and its arguments (separated by commas).
type FunctionNode struct {
	Name *TokenNode
	Args []ArgumentNode
}

// WriteTo writes the string representation of the node to the writer.
func (fun FunctionNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	m, err = fun.Name.WriteTo(w)
	if err != nil {
		return
	}
	size += m
	n, err = w.Write(leftParenthesisBytes)
	if err != nil {
		return
	}
	size += int64(n)
	for i, arg := range fun.Args {
		if i != 0 {
			n, err = w.Write(commaBytes)
			if err != nil {
				return
			}
			size += int64(n)
		}
		m, err = arg.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	n, err = w.Write(rightParenthesisBytes)
	if err != nil {
		return
	}
	size += int64(n)
	return
}

////////////////////////////////////////////////////////////////

// ArgumentNode represents the key and parts of an argument separated by spaces.
type ArgumentNode struct {
	Vals []Node
}

// WriteTo writes the string representation of the node to the writer.
func (arg ArgumentNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	for i, val := range arg.Vals {
		if i != 0 {
			var t *TokenNode
			if k, ok := arg.Vals[i-1].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			} else if k, ok := arg.Vals[i].(*TokenNode); ok && len(k.Data) == 1 {
				t = k
			}
			if t == nil || (t.Data[0] != '=' && t.Data[0] != '*' && t.Data[0] != '/') {
				n, err = w.Write(spaceBytes)
				if err != nil {
					return
				}
				size += int64(n)
			}
		}
		m, err = val.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	return
}

////////////////////////////////////////////////////////////////

// BlockNode contains the contents of a block (brace, bracket or parenthesis blocks).
// Nodes contains AtRuleNode, DeclarationNode, RulesetNode and BlockNode nodes.
type BlockNode struct {
	Open  *TokenNode
	Nodes []Node
	Close *TokenNode
}

// WriteTo writes the string representation of the node to the writer.
func (block BlockNode) WriteTo(w io.Writer) (size int64, err error) {
	var n int
	var m int64
	if block.Open != nil && block.Close != nil && len(block.Nodes) > 0 {
		m, err = block.Open.WriteTo(w)
		if err != nil {
			return
		}
		size += m
		for i, node := range block.Nodes {
			if i != 0 {
				var t *TokenNode
				if k, ok := block.Nodes[i-1].(*TokenNode); ok && len(k.Data) == 1 {
					t = k
				} else if k, ok := block.Nodes[i].(*TokenNode); ok && len(k.Data) == 1 {
					t = k
				}
				if t == nil || t.Data[0] != ':' {
					n, err = w.Write(spaceBytes)
					if err != nil {
						return
					}
					size += int64(n)
				}
			}
			m, err = node.WriteTo(w)
			if err != nil {
				return
			}
			size += m
		}
		m, err = block.Close.WriteTo(w)
		if err != nil {
			return
		}
		size += m
	}
	return
}
