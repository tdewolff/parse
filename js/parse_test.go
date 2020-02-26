package js

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/tdewolff/test"
)

////////////////////////////////////////////////////////////////

func (n Node) String() string {
	if n.gt == TokenGrammar {
		return string(n.data)
	}
	s := ""
	for _, child := range n.nodes {
		s += " " + child.String()
	}
	if 0 < len(s) {
		s = s[1:]
	}
	if n.gt == ModuleGrammar {
		return s
	}
	return n.gt.String() + "(" + s + ")"
}

func TestParse(t *testing.T) {
	var parseTests = []struct {
		js       string
		expected string
	}{
		{"var a = b;", "Stmt(Decl(var Binding(a = Expr(b))))"},
		{"const a = b;", "Stmt(Decl(const Binding(a = Expr(b))))"},
		{"let a = b;", "Stmt(Decl(let Binding(a = Expr(b))))"},
		{"let [a,b] = [1, 2];", ""},
		{"let [a,[b,c]] = [1, [2, 3]];", ""},
		{"let [,,c] = [1, 2, 3];", ""},
		{"let [a, ...b] = [1, 2, 3];", ""},
		{"let {a, b} = {a: 3, b: 4};", ""},
		{"let {a: [b, {c}]} = {a: [5, {c: 3}]};", ""},
		{"let [a = 2] = [];", ""},
		{"let {a: b = 2} = {};", ""},
		{"var a = 5 * 4 / 3 ** 2 + ( 5 - 3 );", "Stmt(Decl(var Binding(a = Expr(5 * 4 / 3 ** 2 + ( 5 - 3 )))))"},
		{";", "Stmt()"},
		{"{; var a = 3;}", "Stmt({ Stmt() Stmt(Decl(var Binding(a = Expr(3)))) })"},
		{"return", "Stmt(return)"},
		{"return 5*3", "Stmt(return Expr(5 * 3))"},
		{"break", "Stmt(break)"},
		{"break LABEL", "Stmt(break LABEL)"},
		{"continue", "Stmt(continue)"},
		{"continue LABEL", "Stmt(continue LABEL)"},
		{"if (a == 5) return true", "Stmt(if Expr(a == 5) Stmt(return Expr(true)))"},
		{"with (a = 5) return true", "Stmt(with Expr(a = 5) Stmt(return Expr(true)))"},
		{"do a++ while (a < 4)", "Stmt(do Stmt(Expr(a ++)) while Expr(a < 4))"},
		{"while (a < 4) a++", "Stmt(while Expr(a < 4) Stmt(Expr(a ++)))"},
		{"for (var a = 0; a < 4; a++) b = a", "Stmt(for Decl(var Binding(a = Expr(0))) Expr(a < 4) Expr(a ++) Stmt(Expr(b = a)))"},
	}
	for _, tt := range parseTests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(bytes.NewBufferString(tt.js))
			fmt.Println(ast)
			if err != io.EOF {
				test.Error(t, err)
			}
			test.String(t, ast.String(), tt.expected)
		})
	}
}
