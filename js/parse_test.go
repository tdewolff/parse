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
	var tests = []struct {
		js       string
		expected string
	}{
		// grammar
		{"", ""},
		{"{}", "Stmt({ })"},
		{"var a = b;", "Stmt(var Binding(a = Expr(b)))"},
		{"const a = b;", "Stmt(const Binding(a = Expr(b)))"},
		{"let a = b;", "Stmt(let Binding(a = Expr(b)))"},
		{"let [a,b] = [1, 2];", "Stmt(let Binding([ Binding(a) Binding(b) ] = Expr([ Expr(1) , Expr(2) ])))"},
		{"let [a,[b,c]] = [1, [2, 3]];", "Stmt(let Binding([ Binding(a) Binding([ Binding(b) Binding(c) ]) ] = Expr([ Expr(1) , Expr([ Expr(2) , Expr(3) ]) ])))"},
		{"let [,,c] = [1, 2, 3];", "Stmt(let Binding([ Binding(c) ] = Expr([ Expr(1) , Expr(2) , Expr(3) ])))"},
		{"let [a, ...b] = [1, 2, 3];", "Stmt(let Binding([ Binding(a) ... Binding(b) ] = Expr([ Expr(1) , Expr(2) , Expr(3) ])))"},
		{"let {a, b} = {a: 3, b: 4};", "Stmt(let Binding({ Binding(a) Binding(b) } = Expr({ a : Expr(3) , b : Expr(4) })))"},
		{"let {a: [b, {c}]} = {a: [5, {c: 3}]};", "Stmt(let Binding({ a : Binding([ Binding(b) Binding({ Binding(c) }) ]) } = Expr({ a : Expr([ Expr(5) , Expr({ c : Expr(3) }) ]) })))"},
		{"let [a = 2] = [];", "Stmt(let Binding([ Binding(a = Expr(2)) ] = Expr([ ])))"},
		{"let {a: b = 2} = {};", "Stmt(let Binding({ a : Binding(b = Expr(2)) } = Expr({ })))"},
		{"var a = 5 * 4 / 3 ** 2 + ( 5 - 3 );", "Stmt(var Binding(a = Expr(5 * 4 / 3 ** 2 + ( Expr(5 - 3) ))))"},
		{"var a, b = c;", "Stmt(var Binding(a) Binding(b = Expr(c)))"},
		{";", "Stmt()"},
		{"{; var a = 3;}", "Stmt({ Stmt() Stmt(var Binding(a = Expr(3))) })"},
		{"return", "Stmt(return)"},
		{"return 5*3", "Stmt(return Expr(5 * 3))"},
		{"break", "Stmt(break)"},
		{"break LABEL", "Stmt(break LABEL)"},
		{"continue", "Stmt(continue)"},
		{"continue LABEL", "Stmt(continue LABEL)"},
		{"if (a == 5) return true", "Stmt(if Expr(a == 5) Stmt(return Expr(true)))"},
		{"if (a == 5) return true else return false", "Stmt(if Expr(a == 5) Stmt(return Expr(true)) else Stmt(return Expr(false)))"},
		{"with (a = 5) return true", "Stmt(with Expr(a = Expr(5)) Stmt(return Expr(true)))"},
		{"do a++ while (a < 4)", "Stmt(do Stmt(Expr(a ++)) while Expr(a < 4))"},
		{"do {a++} while (a < 4)", "Stmt(do Stmt({ Stmt(Expr(a ++)) }) while Expr(a < 4))"},
		{"while (a < 4) a++", "Stmt(while Expr(a < 4) Stmt(Expr(a ++)))"},
		{"for (var a = 0; a < 4; a++) b = a", "Stmt(for Stmt(var Binding(a = Expr(0))) Expr(a < 4) Expr(a ++) Stmt(Expr(b = Expr(a))))"},
		{"for (5; a < 4; a++) {}", "Stmt(for Expr(5) Expr(a < 4) Expr(a ++) Stmt({ }))"},
		{"for (var a in b) {}", "Stmt(for Stmt(var Binding(a)) in Expr(b) Stmt({ }))"},
		{"for (var a of b) {}", "Stmt(for Stmt(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"for await (var a of b) {}", "Stmt(for await Stmt(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"throw 5", "Stmt(throw Expr(5))"},
		{"try {} catch {}", "Stmt(try Stmt({ }) catch Stmt({ }))"},
		{"try {} finally {}", "Stmt(try Stmt({ }) finally Stmt({ }))"},
		{"try {} catch {} finally {}", "Stmt(try Stmt({ }) catch Stmt({ }) finally Stmt({ }))"},
		{"try {} catch (e) {}", "Stmt(try Stmt({ }) catch Binding(e) Stmt({ }))"},
		{"debugger", "Stmt(debugger)"},
		{"label: var a", "Stmt(label Stmt(var Binding(a)))"},
		{"switch (5) {}", "Stmt(switch Expr(5))"},
		{"switch (5) { case 3: {} default: {}}", "Stmt(switch Expr(5) Clause(case Expr(3) Stmt({ })) Clause(default Stmt({ })))"},
		{"function (b) {}", "Stmt(function Binding(b) Stmt({ }))"},
		{"function a(b) {}", "Stmt(function a Binding(b) Stmt({ }))"},
		{"async function (b) {}", "Stmt(async function Binding(b) Stmt({ }))"},
		{"function* (b) {}", "Stmt(function * Binding(b) Stmt({ }))"},
		{"function (a,) {}", "Stmt(function Binding(a) Stmt({ }))"},
		{"function (a, b) {}", "Stmt(function Binding(a) Binding(b) Stmt({ }))"},
		{"function (...a) {}", "Stmt(function ... Binding(a) Stmt({ }))"},
		{"function (a, ...b) {}", "Stmt(function Binding(a) ... Binding(b) Stmt({ }))"},
		{"class { }", "Stmt(class)"},
		{"class { ; }", "Stmt(class)"},
		{"class A { }", "Stmt(class A)"},
		{"class A extends B { }", "Stmt(class A extends Expr(B))"},
		{"class { a(b) {} }", "Stmt(class Method(a Binding(b) Stmt({ })))"},
		{"class { get a() {} }", "Stmt(class Method(get a Stmt({ })))"},
		{"class { set a(b) {} }", "Stmt(class Method(set a Binding(b) Stmt({ })))"},
		{"class { * a(b) {} }", "Stmt(class Method(* a Binding(b) Stmt({ })))"},
		{"class { async a(b) {} }", "Stmt(class Method(async a Binding(b) Stmt({ })))"},
		{"class { async * a(b) {} }", "Stmt(class Method(async * a Binding(b) Stmt({ })))"},
		{"class { static a(b) {} }", "Stmt(class Method(static a Binding(b) Stmt({ })))"},
		{"class { [5](b) {} }", "Stmt(class Method([ Expr(5) ] Binding(b) Stmt({ })))"},

		// edge-cases
		{"let\nawait 0", "Stmt(let Binding(await)) Stmt(Expr(0))"},
		{"yield a = 5", "Stmt(Expr(yield Expr(a = Expr(5))))"},
		{"yield * a = 5", "Stmt(Expr(yield * Expr(a = Expr(5))))"},
		{"yield\na = 5", "Stmt(Expr(yield)) Stmt(Expr(a = Expr(5)))"},
		{"yield yield a", "Stmt(Expr(yield Expr(yield Expr(a))))"},
		{"yield * yield * a", "Stmt(Expr(yield * Expr(yield * Expr(a))))"},
		{"if (a) 1 else if (b) 2 else 3", "Stmt(if Expr(a) Stmt(Expr(1)) else Stmt(if Expr(b) Stmt(Expr(2)) else Stmt(Expr(3))))"},

		// bindings
		{"let []", "Stmt(let Binding([ ]))"},
		{"let [name = 5]", "Stmt(let Binding([ Binding(name = Expr(5)) ]))"},
		{"let [name = 5,,]", "Stmt(let Binding([ Binding(name = Expr(5)) ]))"},
		{"let [name = 5,, ...yield]", "Stmt(let Binding([ Binding(name = Expr(5)) ... Binding(yield) ]))"},
		{"let [...yield]", "Stmt(let Binding([ ... Binding(yield) ]))"},
		{"let [,,...yield]", "Stmt(let Binding([ ... Binding(yield) ]))"},
		{"let [name = 5,, ...[yield]]", "Stmt(let Binding([ Binding(name = Expr(5)) ... Binding([ Binding(yield) ]) ]))"},
		{"let [name = 5,, ...{yield}]", "Stmt(let Binding([ Binding(name = Expr(5)) ... Binding({ Binding(yield) }) ]))"},
		{"let {}", "Stmt(let Binding({ }))"},
		{"let {name = 5}", "Stmt(let Binding({ Binding(name = Expr(5)) }))"},
		{"let {await = 5}", "Stmt(let Binding({ Binding(await = Expr(5)) }))"},
		{"let {if: name}", "Stmt(let Binding({ if : Binding(name) }))"},
		{"let {\"string\": name}", "Stmt(let Binding({ \"string\" : Binding(name) }))"},
		{"let {[a = 5]: name}", "Stmt(let Binding({ [ Expr(a = Expr(5)) ] : Binding(name) }))"},
		{"let {if: name = 5}", "Stmt(let Binding({ if : Binding(name = Expr(5)) }))"},
		{"let {if: yield = 5}", "Stmt(let Binding({ if : Binding(yield = Expr(5)) }))"},
		{"let {if: [name] = 5}", "Stmt(let Binding({ if : Binding([ Binding(name) ] = Expr(5)) }))"},
		{"let {if: {name} = 5}", "Stmt(let Binding({ if : Binding({ Binding(name) } = Expr(5)) }))"},
		{"let {...yield}", "Stmt(let Binding({ ... Binding(yield) }))"},
		{"let {if: name, ...yield}", "Stmt(let Binding({ if : Binding(name) ... Binding(yield) }))"},

		// expressions
		{"x = {a}", "Stmt(Expr(x = Expr({ a })))"},
		{"x = {a=5}", "Stmt(Expr(x = Expr({ a = Expr(5) })))"},
		{"x = {yield=5}", "Stmt(Expr(x = Expr({ yield = Expr(5) })))"},
		{"x = {a:5}", "Stmt(Expr(x = Expr({ a : Expr(5) })))"},
		{"x = {yield:5}", "Stmt(Expr(x = Expr({ yield : Expr(5) })))"},
		{"x = {if:5}", "Stmt(Expr(x = Expr({ if : Expr(5) })))"},
		{"x = {\"string\":5}", "Stmt(Expr(x = Expr({ \"string\" : Expr(5) })))"},
		{"x = {3:5}", "Stmt(Expr(x = Expr({ 3 : Expr(5) })))"},
		{"x = {[3]:5}", "Stmt(Expr(x = Expr({ [ Expr(3) ] : Expr(5) })))"},
		{"x = {a, if: b, do(){}, ...d}", "Stmt(Expr(x = Expr({ a , if : Expr(b) , Method(do Stmt({ })) , ... Expr(d) })))"},
		{"x = (a, b, ...c)", "Stmt(Expr(x = Expr(( Expr(a) , Expr(b) , ... Binding(c) ))))"},
		{"x = function() {}", "Stmt(Expr(x = Expr(function Stmt({ }))))"},
		{"x = async function() {}", "Stmt(Expr(x = Expr(async function Stmt({ }))))"},
		{"x = class {}", "Stmt(Expr(x = Expr(class)))"},
		{"x = class {a(){}}", "Stmt(Expr(x = Expr(class Method(a Stmt({ })))))"},
		{"x = a => a++", "Stmt(Expr(x = Expr(Binding(a) => Expr(a ++))))"},
		{"x = yield => a++", "Stmt(Expr(x = Expr(Binding(yield) => Expr(a ++))))"},
		{"x = await => a++", "Stmt(Expr(x = Expr(Binding(await) => Expr(a ++))))"},
		{"x = (a) => a++", "Stmt(Expr(x = Expr(( Expr(a) ) => Expr(a ++))))"},
		{"x = (a) => {a++}", "Stmt(Expr(x = Expr(( Expr(a) ) => Stmt({ Stmt(Expr(a ++)) }))))"},
		{"x = async a => a++", "Stmt(Expr(x = Expr(async a => Expr(a ++))))"},
		{"x = async a => {a++}", "Stmt(Expr(x = Expr(async a => Stmt({ Stmt(Expr(a ++)) }))))"},

		// regular expressions
		{"/abc/", "Stmt(Expr(/abc/))"},
		{"return /abc/;", "Stmt(return Expr(/abc/))"},
		{"a/b/g", "Stmt(Expr(a / b / g))"},
		{"{}/1/g", "Stmt({ }) Stmt(Expr(/1/g))"},
		{"i(0)/1/g", "Stmt(Expr(i ( Expr(0) ) / 1 / g))"},
		{"if(0)/1/g", "Stmt(if Expr(0) Stmt(Expr(/1/g)))"},
		{"a.if(0)/1/g", "Stmt(Expr(a . if ( Expr(0) ) / 1 / g))"},
		{"this/1/g", "Stmt(Expr(this / 1 / g))"},
		{"switch(a){case /1/g:}", "Stmt(switch Expr(a) Clause(case Expr(/1/g)))"},
		{"(a+b)/1/g", "Stmt(Expr(( Expr(a + b) ) / 1 / g))"},
		{"f(); function foo() {} /42/i", "Stmt(Expr(f ( ))) Stmt(function foo Stmt({ })) Stmt(Expr(/42/i))"},
		{"x = function() {} /42/i", "Stmt(Expr(x = Expr(function Stmt({ }) / 42 / i)))"},
		{"x = function foo() {} /42/i", "Stmt(Expr(x = Expr(function foo Stmt({ }) / 42 / i)))"},
		{"x = /foo/", "Stmt(Expr(x = Expr(/foo/)))"},
		{"x = x / foo /", "Stmt(Expr(x = Expr(x / foo /)))"},
		{"x = (/foo/)", "Stmt(Expr(x = Expr(( Expr(/foo/) ))))"},
		{"x = {a: /foo/}", "Stmt(Expr(x = Expr({ a : Expr(/foo/) })))"},
		{"do { /foo/ } while (a)", "Stmt(do Stmt({ Stmt(Expr(/foo/)) }) while Expr(a))"},
		{"if (true) /foo/", "Stmt(if Expr(true) Stmt(Expr(/foo/)))"},
		{"x = (a) / foo", "Stmt(Expr(x = Expr(( Expr(a) ) / foo)))"},
		{"bar (true) /foo/", "Stmt(Expr(bar ( Expr(true) ) / foo /))"},
		{"/abc/ ? /def/ : /geh/", "Stmt(Expr(/abc/ ? Expr(/def/) : Expr(/geh/)))"},
		{"yield /abc/", "Stmt(Expr(yield Expr(/abc/)))"},
		{"yield * /abc/", "Stmt(Expr(yield * Expr(/abc/)))"},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(bytes.NewBufferString(tt.js))
			if err != io.EOF {
				test.Error(t, err)
			}
			test.String(t, ast.String(), tt.expected)
		})
	}

	// coverage
	for i := 0; ; i++ {
		if GrammarType(i).String() == fmt.Sprintf("Invalid(%d)", i) {
			break
		}
	}
}

func TestParseError(t *testing.T) {
	var tests = []struct {
		js  string
		err string
	}{
		{"{a, if: b, do(){}, ...d}", "unexpected ':' in statement"},
		{"let {if = 5}", "expected ':' instead of '=' in object binding pattern"},
		{"let {...[]}", "expected 'Identifier' instead of '[' in object binding pattern"},
		{"let {...{}}", "expected 'Identifier' instead of '{' in object binding pattern"},
		{"for", "expected '(' instead of EOF in for statement"},
		{"for b", "expected '(' instead of 'b' in for statement"},
		{"for (a b)", "expected 'in', 'of', or ';' instead of 'b' in for statement"},

		{"if", "expected '(' instead of EOF in if statement"},
		{"if(a", "expected ')' instead of EOF in if statement"},
		{"with", "expected '(' instead of EOF in with statement"},
		{"with(a", "expected ')' instead of EOF in with statement"},
		{"do a++", "expected 'while' instead of EOF in do statement"},
		{"do a++ while", "expected '(' instead of EOF in do statement"},
		{"do a++ while(a", "expected ')' instead of EOF in do statement"},
		{"while", "expected '(' instead of EOF in while statement"},
		{"while(a", "expected ')' instead of EOF in while statement"},
		{"for", "expected '(' instead of EOF in for statement"},
		{"for(a;a", "expected ';' instead of EOF in for statement"},
		{"for(a;a;a", "expected ')' instead of EOF in for statement"},
		{"switch", "expected '(' instead of EOF in switch statement"},
		{"switch(a", "expected ')' instead of EOF in switch statement"},
		{"switch(a)", "expected '{' instead of EOF in switch statement"},
		{"switch(a){bad:5}", "expected 'case' or 'default' instead of 'bad' in switch statement"},
		{"switch(a){case", "expected ':' instead of EOF in switch statement"},
		{"async", "expected 'function' instead of EOF in async function statement"},
		{"try{}catch(a", "expected ')' instead of EOF in try statement"},
		{"function", "expected '(' instead of EOF in function declaration"},
		{"function(a", "expected ',' or ')' instead of EOF in function declaration"},
		{"function(...a", "expected ')' instead of EOF in function declaration"},
		{"function(...a", "expected ')' instead of EOF in function declaration"},
		{"function()", "expected '{' instead of EOF in function declaration"},
		{"class A", "expected '{' instead of EOF in class statement"},
		{"class A{", "expected '}' instead of EOF in class statement"},
		{"class A extends a b {}", "expected '{' instead of 'b' in class statement"},
		{"class A{+", "expected 'Identifier', 'String', 'Numeric', or '[' instead of '+' in method definition"},
		{"class A{[a", "expected ']' instead of EOF in method definition"},
		{"var [...a", "expected ']' instead of EOF in array binding pattern"},
		{"var [a", "expected ',' or ']' instead of EOF in array binding pattern"},
		{"var {[a", "expected ']' instead of EOF in object binding pattern"},
		{"var {+", "expected 'Identifier', 'String', 'Numeric', or '[' instead of '+' in object binding pattern"},
		{"var {a", "expected ',' or '}' instead of EOF in object binding pattern"},
		{"var 0", "unexpected '0' in binding"},
		{"x={[a", "expected ']' instead of EOF in object literal"},
		{"x={[a]", "expected ':' or '(' instead of EOF in object literal"},
		{"x={+", "expected '=', ',', '}', '...', 'Identifier', 'String', 'Numeric', or '[' instead of '+' in object literal"},
		{"class a extends ||", "expected '{' instead of '||' in class statement"},
		{"class a extends =", "expected '{' instead of '=' in class statement"},
		{"class a extends ?", "expected '{' instead of '?' in class statement"},
		{"class a extends =>", "expected '{' instead of '=>' in class statement"},
		{"class a extends async", "expected 'function' instead of EOF in async function expression"},
		{"x=a?b", "expected ':' instead of EOF in expression"},
		{"x=async a", "expected '=>' instead of EOF in async arrow function expression"},
		{"x=async", "expected 'function' or 'Identifier' instead of EOF in async function expression"},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			_, err := Parse(bytes.NewBufferString(tt.js))
			test.That(t, err != io.EOF && err != nil)

			e := err.Error()
			if len(tt.err) < len(err.Error()) {
				e = e[:len(tt.err)]
			}
			test.String(t, e, tt.err)
		})
	}
}
