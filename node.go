package css

import (
	"strconv"
)

////////////////////////////////////////////////////////////////

// NodeType determines the type of node, eg. a block or a declaration.
type NodeType uint32

const (
	ErrorNode NodeType = iota // extra node when errors occur
	StylesheetNode
	DeclarationListNode
	DeclarationNode
	TokenNode // extra node for simple tokens
)

// String returns the string representation of a NodeType.
func (t NodeType) String() string {
	switch t {
	case ErrorNode:
		return "Error"
	case StylesheetNode:
		return "Stylesheet"
	case DeclarationListNode:
		return "DeclarationList"
	case DeclarationNode:
		return "Declaration"
	case TokenNode:
		return "Token"
	}
	return "Invalid(" + strconv.Itoa(int(t)) + ")"
}

func (t NodeType) Type() NodeType {
	return t
}

////////////////////////////////////////////////////////////////

type Node interface {
	Type() NodeType
}

////////////////////////////////////////////////////////////////

type NodeError struct {
	NodeType
	err error
}

func newError(err error) *NodeError {
	return &NodeError{
		NodeType: ErrorNode,
		err: err,
	}
}

////////////////////////////////////////////////////////////////

type NodeToken struct {
	NodeType
	tt TokenType
	data string
}

func newToken(tt TokenType, data string) *NodeToken {
	return &NodeToken{
		NodeType: TokenNode,
		tt: tt,
		data: data,
	}
}

func (n NodeToken) Token() (TokenType, string) {
	return n.tt, n.data
}

////////////////////////////////////////////////////////////////

type NodeStylesheet struct {
	NodeType
	Nodes []Node
}

func newStylesheet() *NodeStylesheet {
	return &NodeStylesheet{
		NodeType: StylesheetNode,
	}
}

////////////////////////////////////////////////////////////////

type NodeDeclarationList struct {
	NodeType
}

func newDeclarationList() *NodeDeclarationList {
	return &NodeDeclarationList{
		NodeType: DeclarationListNode,
	}
}