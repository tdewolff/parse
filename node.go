package css

import (
	"bytes"
)

////////////////////////////////////////////////////////////////

// NodeType determines the type of node, eg. a block or a declaration.
type NodeType uint32

const (
	ErrorNode NodeType = iota // extra node when errors occur
	StylesheetNode
	RulesetNode
	SelectorNode
	DeclarationBlockNode
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
	err error
}

func newError(err error) *NodeError {
	return &NodeError{
		NodeType: ErrorNode,
		err: err,
	}
}

func (n NodeError) String() string {
	return n.err.Error()
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

func (n NodeToken) String() string {
	return n.data
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

func (n NodeStylesheet) String() string {
	return listString(n.Nodes)
}

////////////////////////////////////////////////////////////////

type NodeRuleset struct {
	NodeType
	Selectors []Node
	Decl Node
}

func newRuleset() *NodeRuleset {
	return &NodeRuleset{
		NodeType: RulesetNode,
	}
}

func (n NodeRuleset) String() string {
	if n.Decl == nil {
		return listString(n.Selectors)
	}
	return listString(n.Selectors) + "=" + n.Decl.String()
}

////////////////////////////////////////////////////////////////

type NodeSelector struct {
	NodeType
	Selector []Node
}

func newSelector() *NodeSelector {
	return &NodeSelector{
		NodeType: SelectorNode,
	}
}

func (n NodeSelector) String() string {
	return listString(n.Selector)
}

////////////////////////////////////////////////////////////////

type NodeDeclarationBlock struct {
	NodeType
	Decls []Node
}

func newDeclarationBlock() *NodeDeclarationBlock {
	return &NodeDeclarationBlock{
		NodeType: DeclarationBlockNode,
	}
}

func (n NodeDeclarationBlock) String() string {
	return listString(n.Decls)
}

////////////////////////////////////////////////////////////////

type NodeDeclaration struct {
	NodeType
	Prop Node
	Val []Node
}

func newDeclaration(prop Node) *NodeDeclaration {
	return &NodeDeclaration{
		NodeType: DeclarationNode,
		Prop: prop,
	}
}

func (n NodeDeclaration) String() string {
	if n.Prop == nil {
		return ""
	}
	if len(n.Val) > 0 {
		return n.Prop.String() + ":" + listString(n.Val)
	}
	return n.Prop.String()
}

////////////////////////////////////////////////////////////////

type NodeFunction struct {
	NodeType
	Func Node
	Arg []Node
}

func newFunction(f Node) *NodeFunction {
	return &NodeFunction{
		NodeType: FunctionNode,
		Func: f,
	}
}

func (n NodeFunction) String() string {
	if n.Func == nil {
		return ""
	}
	if len(n.Arg) > 0 {
		return n.Func.String() + ":" + listString(n.Arg)
	}
	return n.Func.String()
}

////////////////////////////////////////////////////////////////

type NodeAtRule struct {
	NodeType
	At Node
	Nodes []Node
	Block []Node
}

func newAtRule(at Node) *NodeAtRule {
	return &NodeAtRule{
		NodeType: AtRuleNode,
		At: at,
	}
}

func (n NodeAtRule) String() string {
	if len(n.Block) > 0 {
		return n.At.String() + ":" + listString(n.Nodes) + ":" + listString(n.Block)
	}
	if len(n.Nodes) > 0 {
		return n.At.String() + ":" + listString(n.Nodes)
	}
	return n.At.String()
}

////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////

func listString(nodes []Node) string {
	if len(nodes) == 0 {
		return ""
	} else if len(nodes) == 1 {
		return nodes[0].String()
	}

	b := &bytes.Buffer{}
	b.WriteByte('[')
	for _, n := range nodes {
		b.WriteString(n.String()+" ")
	}
	s := b.String()
	return s[:len(s)-1] + "]"
}