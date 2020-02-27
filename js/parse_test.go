package js

import (
	"bytes"
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
		{"{}", "Stmt({ })"},
		{"/a/i", "Stmt(Expr(/a/i))"},
		{"var a = b;", "Stmt(var Binding(a) = Expr(b))"},
		{"const a = b;", "Stmt(const Binding(a) = Expr(b))"},
		{"let a = b;", "Stmt(let Binding(a) = Expr(b))"},
		{"let [a,b] = [1, 2];", "Stmt(let Binding([ Binding(a) , Binding(b) ]) = Expr([ Expr(1) , Expr(2) ]))"},
		{"let [a,[b,c]] = [1, [2, 3]];", "Stmt(let Binding([ Binding(a) , Binding([ Binding(b) , Binding(c) ]) ]) = Expr([ Expr(1) , Expr([ Expr(2) , Expr(3) ]) ]))"},
		{"let [,,c] = [1, 2, 3];", "Stmt(let Binding([ , , Binding(c) ]) = Expr([ Expr(1) , Expr(2) , Expr(3) ]))"},
		{"let [a, ...b] = [1, 2, 3];", "Stmt(let Binding([ Binding(a) , ... Binding(b) ]) = Expr([ Expr(1) , Expr(2) , Expr(3) ]))"},
		{"let {a, b} = {a: 3, b: 4};", "Stmt(let Binding({ a , b }) = Expr({ a : Expr(3) , b : Expr(4) }))"},
		{"let {a: [b, {c}]} = {a: [5, {c: 3}]};", "Stmt(let Binding({ a : Binding([ Binding(b) , Binding({ c }) ]) }) = Expr({ a : Expr([ Expr(5) , Expr({ c : Expr(3) }) ]) }))"},
		{"let [a = 2] = [];", "Stmt(let Binding([ Binding(a) = Expr(2) ]) = Expr([ ]))"},
		{"let {a: b = 2} = {};", "Stmt(let Binding({ a : Binding(b) = Expr(2) }) = Expr({ }))"},
		{"var a = 5 * 4 / 3 ** 2 + ( 5 - 3 );", "Stmt(var Binding(a) = Expr(5 * 4 / 3 ** 2 + ( Expr(5 - 3) )))"},
		{";", "Stmt()"},
		{"{; var a = 3;}", "Stmt({ Stmt() Stmt(var Binding(a) = Expr(3)) })"},
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
		{"for (var a = 0; a < 4; a++) b = a", "Stmt(for Stmt(var Binding(a) = Expr(0)) Expr(a < 4) Expr(a ++) Stmt(Expr(b = a)))"},
		{"for (5; a < 4; a++) {}", "Stmt(for Expr(5) Expr(a < 4) Expr(a ++) Stmt({ }))"},
		{"for (var a in b) {}", "Stmt(for Stmt(var Binding(a)) in Expr(b) Stmt({ }))"},
		{"for (var a of b) {}", "Stmt(for Stmt(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"for await (var a of b) {}", "Stmt(for await Stmt(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"throw 5", "Stmt(throw Expr(5))"},
		{"try {} catch {}", "Stmt(try Stmt({ }) catch Stmt({ }))"},
		{"try {} finally {}", "Stmt(try Stmt({ }) finally Stmt({ }))"},
		{"try {} catch {} finally {}", "Stmt(try Stmt({ }) catch Stmt({ }) finally Stmt({ }))"},
		{"debugger", "Stmt(debugger)"},
		{"label: var a", "Stmt(label Stmt(var Binding(a)))"},
		{"switch (5) {}", "Stmt(switch Expr(5))"},
		{"switch (5) { case 3: {} default: {}}", "Stmt(switch Expr(5) Clause(case Expr(3) Stmt({ })) Clause(default Stmt({ })))"},
		{"function (b) {}", "Stmt(function Param(Binding(b)) Stmt({ }))"},
		{"function a(b) {}", "Stmt(function a Param(Binding(b)) Stmt({ }))"},
		{"class { }", "Stmt(class)"},
		{"class A { }", "Stmt(class A)"},
		{"class A extends B { }", "Stmt(class A extends Expr(B))"},
		{"class { a(b) {} }", "Stmt(class Method(a Param(Binding(b)) Stmt({ })))"},
		{"class { get a() {} }", "Stmt(class Method(get a Stmt({ })))"},
		{"class { set a(b) {} }", "Stmt(class Method(set a Param(Binding(b)) Stmt({ })))"},
		{"class { * a(b) {} }", "Stmt(class Method(* a Param(Binding(b)) Stmt({ })))"},
		{"class { async a(b) {} }", "Stmt(class Method(async a Param(Binding(b)) Stmt({ })))"},
		{"class { async * a(b) {} }", "Stmt(class Method(async * a Param(Binding(b)) Stmt({ })))"},
		{"class { static a(b) {} }", "Stmt(class Method(static a Param(Binding(b)) Stmt({ })))"},
		{"class { ; }", "Stmt(class)"},
	}
	for _, tt := range parseTests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(bytes.NewBufferString(tt.js))
			if err != io.EOF {
				test.Error(t, err)
			}
			test.String(t, ast.String(), tt.expected)
		})
	}
}
