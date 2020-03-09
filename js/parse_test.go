package js

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/tdewolff/test"
)

////////////////////////////////////////////////////////////////

func TestParse(t *testing.T) {
	var tests = []struct {
		js       string
		expected string
	}{
		// grammar
		{"", ""},
		{"/* comment */", ""},
		{"{}", "Stmt({ })"},
		{"var a = b;", "Decl(var Binding(a = Expr(b)))"},
		{"const a = b;", "Decl(const Binding(a = Expr(b)))"},
		{"let a = b;", "Decl(let Binding(a = Expr(b)))"},
		{"let [a,b] = [1, 2];", "Decl(let Binding([ Binding(a) Binding(b) ] = Expr([ Expr(1) , Expr(2) ])))"},
		{"let [a,[b,c]] = [1, [2, 3]];", "Decl(let Binding([ Binding(a) Binding([ Binding(b) Binding(c) ]) ] = Expr([ Expr(1) , Expr([ Expr(2) , Expr(3) ]) ])))"},
		{"let [,,c] = [1, 2, 3];", "Decl(let Binding([ Binding() Binding() Binding(c) ] = Expr([ Expr(1) , Expr(2) , Expr(3) ])))"},
		{"let [a, ...b] = [1, 2, 3];", "Decl(let Binding([ Binding(a) ... Binding(b) ] = Expr([ Expr(1) , Expr(2) , Expr(3) ])))"},
		{"let {a, b} = {a: 3, b: 4};", "Decl(let Binding({ Binding(a) Binding(b) } = Expr({ a : Expr(3) , b : Expr(4) })))"},
		{"let {a: [b, {c}]} = {a: [5, {c: 3}]};", "Decl(let Binding({ a : Binding([ Binding(b) Binding({ Binding(c) }) ]) } = Expr({ a : Expr([ Expr(5) , Expr({ c : Expr(3) }) ]) })))"},
		{"let [a = 2] = [];", "Decl(let Binding([ Binding(a = Expr(2)) ] = Expr([ ])))"},
		{"let {a: b = 2} = {};", "Decl(let Binding({ a : Binding(b = Expr(2)) } = Expr({ })))"},
		{"var a = 5 * 4 / 3 ** 2 + ( 5 - 3 );", "Decl(var Binding(a = Expr(5 * 4 / 3 ** 2 + ( Expr(5 - 3) ))))"},
		{"var a, b = c;", "Decl(var Binding(a) Binding(b = Expr(c)))"},
		{"var a,\nb = c;", "Decl(var Binding(a) Binding(b = Expr(c)))"},
		{";", "Stmt(;)"},
		{"{; var a = 3;}", "Stmt({ Stmt(;) Decl(var Binding(a = Expr(3))) })"},
		{"return", "Stmt(return)"},
		{"return 5*3", "Stmt(return Expr(5 * 3))"},
		{"break", "Stmt(break)"},
		{"break LABEL", "Stmt(break LABEL)"},
		{"continue", "Stmt(continue)"},
		{"continue LABEL", "Stmt(continue LABEL)"},
		{"if (a == 5) return true", "Stmt(if Expr(a == 5) Stmt(return Expr(true)))"},
		{"if (a == 5) return true else return false", "Stmt(if Expr(a == 5) Stmt(return Expr(true)) else Stmt(return Expr(false)))"},
		{"if (a) b; else if (c) d;", "Stmt(if Expr(a) Stmt(Expr(b)) else Stmt(if Expr(c) Stmt(Expr(d))))"},
		{"with (a = 5) return true", "Stmt(with Expr(a = Expr(5)) Stmt(return Expr(true)))"},
		{"do a++ while (a < 4)", "Stmt(do Stmt(Expr(a ++)) while Expr(a < 4))"},
		{"do {a++} while (a < 4)", "Stmt(do Stmt({ Stmt(Expr(a ++)) }) while Expr(a < 4))"},
		{"while (a < 4) a++", "Stmt(while Expr(a < 4) Stmt(Expr(a ++)))"},
		{"for (var a = 0; a < 4; a++) b = a", "Stmt(for Decl(var Binding(a = Expr(0))) ; Expr(a < 4) ; Expr(a ++) Stmt(Expr(b = Expr(a))))"},
		{"for (5; a < 4; a++) {}", "Stmt(for Expr(5) ; Expr(a < 4) ; Expr(a ++) Stmt({ }))"},
		{"for (;;) {}", "Stmt(for ; ; Stmt({ }))"},
		{"for (a,b=5;;) {}", "Stmt(for Expr(Expr(a) , Expr(b = Expr(5))) ; ; Stmt({ }))"},
		{"for (let a;;) {}", "Stmt(for Decl(let Binding(a)) ; ; Stmt({ }))"},
		{"for (var a in b) {}", "Stmt(for Decl(var Binding(a)) in Expr(b) Stmt({ }))"},
		{"for (var a of b) {}", "Stmt(for Decl(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"for (var a=5 of b) {}", "Stmt(for Decl(var Binding(a = Expr(5))) of Expr(b) Stmt({ }))"},
		{"for await (var a of b) {}", "Stmt(for await Decl(var Binding(a)) of Expr(b) Stmt({ }))"},
		{"throw 5", "Stmt(throw Expr(5))"},
		{"try {} catch {b}", "Stmt(try Stmt({ }) catch Stmt({ Stmt(Expr(b)) }))"},
		{"try {} finally {c}", "Stmt(try Stmt({ }) finally Stmt({ Stmt(Expr(c)) }))"},
		{"try {} catch {b} finally {c}", "Stmt(try Stmt({ }) catch Stmt({ Stmt(Expr(b)) }) finally Stmt({ Stmt(Expr(c)) }))"},
		{"try {} catch (e) {b}", "Stmt(try Stmt({ }) catch Binding(e) Stmt({ Stmt(Expr(b)) }))"},
		{"debugger", "Stmt(debugger)"},
		{"label: var a", "Stmt(label : Decl(var Binding(a)))"},
		{"switch (5) {}", "Stmt(switch Expr(5))"},
		{"switch (5) { case 3: {} default: {}}", "Stmt(switch Expr(5) Clause(case Expr(3) Stmt({ })) Clause(default Stmt({ })))"},
		{"function (b) {}", "Decl(function Params(Binding(b)) Stmt({ }))"},
		{"function a(b) {}", "Decl(function a Params(Binding(b)) Stmt({ }))"},
		{"async function (b) {}", "Decl(async function Params(Binding(b)) Stmt({ }))"},
		{"function* (b) {}", "Decl(function * Params(Binding(b)) Stmt({ }))"},
		{"function (a,) {}", "Decl(function Params(Binding(a)) Stmt({ }))"},
		{"function (a, b) {}", "Decl(function Params(Binding(a) , Binding(b)) Stmt({ }))"},
		{"function (...a) {}", "Decl(function Params(... Binding(a)) Stmt({ }))"},
		{"function (a, ...b) {}", "Decl(function Params(Binding(a) , ... Binding(b)) Stmt({ }))"},
		{"class { }", "Decl(class)"},
		{"class { ; }", "Decl(class)"},
		{"class A { }", "Decl(class A)"},
		{"class A extends B { }", "Decl(class A extends Expr(B))"},
		{"class { a(b) {} }", "Decl(class Method(a Params(Binding(b)) Stmt({ })))"},
		{"class { get a() {} }", "Decl(class Method(get a Params() Stmt({ })))"},
		{"class { set a(b) {} }", "Decl(class Method(set a Params(Binding(b)) Stmt({ })))"},
		{"class { * a(b) {} }", "Decl(class Method(* a Params(Binding(b)) Stmt({ })))"},
		{"class { async a(b) {} }", "Decl(class Method(async a Params(Binding(b)) Stmt({ })))"},
		{"class { async * a(b) {} }", "Decl(class Method(async * a Params(Binding(b)) Stmt({ })))"},
		{"class { static a(b) {} }", "Decl(class Method(static a Params(Binding(b)) Stmt({ })))"},
		{"class { [5](b) {} }", "Decl(class Method([ Expr(5) ] Params(Binding(b)) Stmt({ })))"},
		{"`tmpl`", "Stmt(Expr(`tmpl`))"},
		{"`tmpl${x}`", "Stmt(Expr(`tmpl${ Expr(x) }`))"},
		{"`tmpl` x `tmpl`", "Stmt(Expr(`tmpl`)) Stmt(Expr(x `tmpl`))"},
		{"import \"pkg\";", "Stmt(import \"pkg\")"},
		{"import yield from \"pkg\"", "Stmt(import yield from \"pkg\")"},
		{"import * as yield from \"pkg\"", "Stmt(import * as yield from \"pkg\")"},
		{"import {yield, for as yield,} from \"pkg\"", "Stmt(import { yield , for as yield , } from \"pkg\")"},
		{"import yield, * as yield from \"pkg\"", "Stmt(import yield , * as yield from \"pkg\")"},
		{"import yield, {yield} from \"pkg\"", "Stmt(import yield , yield from \"pkg\")"},
		{"import {yield,} from \"pkg\"", "Stmt(import { yield , } from \"pkg\")"},
		{"export * from \"pkg\";", "Stmt(export * from \"pkg\")"},
		{"export * as for from \"pkg\"", "Stmt(export * as for from \"pkg\")"},
		{"export {if, for as switch} from \"pkg\"", "Stmt(export { if , for as switch } from \"pkg\")"},
		{"export {if, for as switch}", "Stmt(export { if , for as switch })"},
		{"export var a", "Stmt(export Decl(var Binding(a)))"},
		{"export function(b){}", "Stmt(export Decl(function Params(Binding(b)) Stmt({ })))"},
		{"export async function(b){}", "Stmt(export Decl(async function Params(Binding(b)) Stmt({ })))"},
		{"export class{}", "Stmt(export Decl(class))"},
		{"export default function(b){}", "Stmt(export default Decl(function Params(Binding(b)) Stmt({ })))"},
		{"export default async function(b){}", "Stmt(export default Decl(async function Params(Binding(b)) Stmt({ })))"},
		{"export default class{}", "Stmt(export default Decl(class))"},
		{"export default a", "Stmt(export default Expr(a))"},

		// edge-cases
		{"let\nawait 0", "Decl(let Binding(await)) Stmt(Expr(0))"},
		{"yield a = 5", "Stmt(Expr(yield Expr(a = Expr(5))))"},
		{"yield * a = 5", "Stmt(Expr(yield * Expr(a = Expr(5))))"},
		{"yield\na = 5", "Stmt(Expr(yield)) Stmt(Expr(a = Expr(5)))"},
		{"yield yield a", "Stmt(Expr(yield Expr(yield Expr(a))))"},
		{"yield * yield * a", "Stmt(Expr(yield * Expr(yield * Expr(a))))"},
		{"if (a) 1 else if (b) 2 else 3", "Stmt(if Expr(a) Stmt(Expr(1)) else Stmt(if Expr(b) Stmt(Expr(2)) else Stmt(Expr(3))))"},
		{"x = await => a++", "Stmt(Expr(x = Expr(Params(Binding(await)) => Expr(a ++))))"},
		{"async function(){x = await => a++}", "Decl(async function Params() Stmt({ Stmt(Expr(x = Expr(Params(Binding(await)) => Expr(a ++)))) }))"},
		{"x = {await}", "Stmt(Expr(x = Expr({ await })))"},
		{"x = {async a(b){}}", "Stmt(Expr(x = Expr({ Method(async a Params(Binding(b)) Stmt({ })) })))"},
		{"async function(){ x = {await: 5} }", "Decl(async function Params() Stmt({ Stmt(Expr(x = Expr({ await : Expr(5) }))) }))"},
		{"async function(){ x = await a }", "Decl(async function Params() Stmt({ Stmt(Expr(x = Expr(await a))) }))"},
		{"for (var a in b) {}", "Stmt(for Decl(var Binding(a)) in Expr(b) Stmt({ }))"},
		{"for (a in b) {}", "Stmt(for Expr(a) in Expr(b) Stmt({ }))"},
		{"for (a = b;;) {}", "Stmt(for Expr(a = Expr(b)) ; ; Stmt({ }))"},
		{"!!a", "Stmt(Expr(! ! a))"},

		// bindings
		{"let []", "Decl(let Binding([ ]))"},
		{"let [name = 5]", "Decl(let Binding([ Binding(name = Expr(5)) ]))"},
		{"let [name = 5,,]", "Decl(let Binding([ Binding(name = Expr(5)) Binding() Binding() ]))"},
		{"let [name = 5,, ...yield]", "Decl(let Binding([ Binding(name = Expr(5)) Binding() ... Binding(yield) ]))"},
		{"let [...yield]", "Decl(let Binding([ ... Binding(yield) ]))"},
		{"let [,,...yield]", "Decl(let Binding([ Binding() Binding() ... Binding(yield) ]))"},
		{"let [name = 5,, ...[yield]]", "Decl(let Binding([ Binding(name = Expr(5)) Binding() ... Binding([ Binding(yield) ]) ]))"},
		{"let [name = 5,, ...{yield}]", "Decl(let Binding([ Binding(name = Expr(5)) Binding() ... Binding({ Binding(yield) }) ]))"},
		{"let {}", "Decl(let Binding({ }))"},
		{"let {name = 5}", "Decl(let Binding({ Binding(name = Expr(5)) }))"},
		{"let {await = 5}", "Decl(let Binding({ Binding(await = Expr(5)) }))"},
		{"let {if: name}", "Decl(let Binding({ if : Binding(name) }))"},
		{"let {\"string\": name}", "Decl(let Binding({ \"string\" : Binding(name) }))"},
		{"let {[a = 5]: name}", "Decl(let Binding({ [ Expr(a = Expr(5)) ] : Binding(name) }))"},
		{"let {if: name = 5}", "Decl(let Binding({ if : Binding(name = Expr(5)) }))"},
		{"let {if: yield = 5}", "Decl(let Binding({ if : Binding(yield = Expr(5)) }))"},
		{"let {if: [name] = 5}", "Decl(let Binding({ if : Binding([ Binding(name) ] = Expr(5)) }))"},
		{"let {if: {name} = 5}", "Decl(let Binding({ if : Binding({ Binding(name) } = Expr(5)) }))"},
		{"let {...yield}", "Decl(let Binding({ ... Binding(yield) }))"},
		{"let {if: name, ...yield}", "Decl(let Binding({ if : Binding(name) ... Binding(yield) }))"},

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
		{"x = {a, if: b, do(){}, ...d}", "Stmt(Expr(x = Expr({ a , if : Expr(b) , Method(do Params() Stmt({ })) , ... Expr(d) })))"},
		{"x = (a, b, ...c)", "Stmt(Expr(x = Expr(( Expr(a) , Expr(b) , ... Binding(c) ))))"},
		{"x = function() {}", "Stmt(Expr(x = Expr(Decl(function Params() Stmt({ })))))"},
		{"x = async function() {}", "Stmt(Expr(x = Expr(Decl(async function Params() Stmt({ })))))"},
		{"x = class {}", "Stmt(Expr(x = Expr(Decl(class))))"},
		{"x = class {a(){}}", "Stmt(Expr(x = Expr(Decl(class Method(a Params() Stmt({ }))))))"},
		{"x = a => a++", "Stmt(Expr(x = Expr(Params(Binding(a)) => Expr(a ++))))"},
		{"x = yield => a++", "Stmt(Expr(x = Expr(Params(Binding(yield)) => Expr(a ++))))"},
		{"x = yield => {a++}", "Stmt(Expr(x = Expr(Params(Binding(yield)) => Stmt({ Stmt(Expr(a ++)) }))))"},
		{"x = (a) => a++", "Stmt(Expr(x = Expr(Params(Binding(a)) => Expr(a ++))))"},
		{"x = (a) => {a++}", "Stmt(Expr(x = Expr(Params(Binding(a)) => Stmt({ Stmt(Expr(a ++)) }))))"},
		{"x = (a,b) => {a++}", "Stmt(Expr(x = Expr(Params(Binding(a) Binding(b)) => Stmt({ Stmt(Expr(a ++)) }))))"},
		{"x = async a => a++", "Stmt(Expr(x = Expr(async Params(Binding(a)) => Expr(a ++))))"},
		{"x = async a => {a++}", "Stmt(Expr(x = Expr(async Params(Binding(a)) => Stmt({ Stmt(Expr(a ++)) }))))"},
		{"x = a??b", "Stmt(Expr(x = Expr(a ?? b)))"},
		{"x = import(a)", "Stmt(Expr(x = Expr(import Expr(a))))"},
		{"x = a?.b?.c.d", "Stmt(Expr(x = Expr(a ?. b ?. c . d)))"},
		{"x = a?.[b]?.c", "Stmt(Expr(x = Expr(a ?. [ Expr(b) ] ?. c)))"},
		{"x = super(a)(b)(c)", "Stmt(Expr(x = Expr(super ( Expr(a) ) ( Expr(b) ) ( Expr(c) ))))"},
		{"x = a(a,b,...c,)", "Stmt(Expr(x = Expr(a ( Expr(a) , Expr(b) , ... Expr(c) ))))"},
		{"x = new new.target", "Stmt(Expr(x = Expr(new new . target)))"},
		{"x = ++a", "Stmt(Expr(x = Expr(++ a)))"},
		{"x = +a", "Stmt(Expr(x = Expr(+ a)))"},
		{"x = !a", "Stmt(Expr(x = Expr(! a)))"},
		{"x = delete a", "Stmt(Expr(x = Expr(delete a)))"},
		{"x = a in b", "Stmt(Expr(x = Expr(a in b)))"},
		{"class a extends async function(){}{}", "Decl(class a extends Expr(Decl(async function Params() Stmt({ }))))"},
		{"x = a.replace(b, c)", "Stmt(Expr(x = Expr(a . replace ( Expr(b) , Expr(c) ))))"},

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
		{"f(); function foo() {} /42/i", "Stmt(Expr(f ( ))) Decl(function foo Params() Stmt({ })) Stmt(Expr(/42/i))"},
		{"x = function() {} /42/i", "Stmt(Expr(x = Expr(Decl(function Params() Stmt({ })) / 42 / i)))"},
		{"x = function foo() {} /42/i", "Stmt(Expr(x = Expr(Decl(function foo Params() Stmt({ })) / 42 / i)))"},
		{"x = /foo/", "Stmt(Expr(x = Expr(/foo/)))"},
		{"x = (/foo/)", "Stmt(Expr(x = Expr(( Expr(/foo/) ))))"},
		{"x = {a: /foo/}", "Stmt(Expr(x = Expr({ a : Expr(/foo/) })))"},
		{"x = (a) / foo", "Stmt(Expr(x = Expr(( Expr(a) ) / foo)))"},
		{"do { /foo/ } while (a)", "Stmt(do Stmt({ Stmt(Expr(/foo/)) }) while Expr(a))"},
		{"if (true) /foo/", "Stmt(if Expr(true) Stmt(Expr(/foo/)))"},
		{"/abc/ ? /def/ : /geh/", "Stmt(Expr(/abc/ ? Expr(/def/) : Expr(/geh/)))"},
		{"yield /abc/", "Stmt(Expr(yield Expr(/abc/)))"},
		{"yield * /abc/", "Stmt(Expr(yield * Expr(/abc/)))"},

		// ASI
		{"return a", "Stmt(return Expr(a))"},
		{"return; a", "Stmt(return) Stmt(Expr(a))"},
		{"return\na", "Stmt(return) Stmt(Expr(a))"},
		{"return /*comment*/ a", "Stmt(return Expr(a))"},
		{"return /*com\nment*/ a", "Stmt(return) Stmt(Expr(a))"},
		{"return //comment\n a", "Stmt(return) Stmt(Expr(a))"},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			fmt.Println()
			fmt.Println(tt.js)
			ast, err := Parse(bytes.NewBufferString(tt.js))
			fmt.Println(ast)
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
		{"for(a", "expected 'in', 'of', or ';' instead of EOF in for statement"},
		{"for(a;a", "expected ';' instead of EOF in for statement"},
		{"for(a;a;a", "expected ')' instead of EOF in for statement"},
		{"switch", "expected '(' instead of EOF in switch statement"},
		{"switch(a", "expected ')' instead of EOF in switch statement"},
		{"switch(a)", "expected '{' instead of EOF in switch statement"},
		{"switch(a){bad:5}", "expected 'case' or 'default' instead of 'bad' in switch statement"},
		{"switch(a){case", "unexpected EOF in expression"},
		{"switch(a){case a", "expected ':' instead of EOF in switch statement"},
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
		{"class a extends ||", "unexpected '||' in expression"},
		{"class a extends =", "unexpected '=' in expression"},
		{"class a extends ?", "unexpected '?' in expression"},
		{"class a extends =>", "unexpected '=>' in expression"},
		{"class a extends async", "expected 'function' instead of EOF in async function expression"},
		{"x=a?b", "expected ':' instead of EOF in conditional expression"},
		{"x=async a", "expected '=>' instead of EOF in async arrow function expression"},
		{"x=async", "expected 'function' or 'Identifier' instead of EOF in async function expression"},
		{"x=async\n", "unexpected EOF in async function expression"},
		{"x=?.?.b", "unexpected '?.' in expression"},
		{"x=a?.?.b", "expected 'Identifier', '(', '[', or 'Template' instead of '?.' in left hand side expression"},
		{"x=a?..b", "expected 'Identifier', '(', '[', or 'Template' instead of '.' in left hand side expression"},
		{"`tmp${", "unexpected EOF in expression"},
		{"`tmp${x", "expected 'Template' instead of EOF in template literal"},
		{"x=5=>", "unexpected '=>' in arrow function expression"},
		{"x=new.bad", "expected 'target' instead of 'bad' in left hand side expression"},
		{"x=super", "expected '(', '[', '.', or 'Template' instead of EOF in left hand side expression"},
		{"x=super `tmpl`", "unexpected '`tmpl`' in left hand side expression"},
		{"x=super(a", "expected ')' instead of EOF in left hand side expression"},
		{"x=super[a", "expected ']' instead of EOF in left hand side expression"},
		{"x=super.", "expected 'Identifier' instead of EOF in left hand side expression"},
		{"x=import", "expected '(' instead of EOF in left hand side expression"},
		{"x=import(5", "expected ')' instead of EOF in left hand side expression"},
		{"import", "expected 'String', 'Identifier', '*', or '{' instead of EOF in import statement"},
		{"import *", "expected 'as' instead of EOF in import statement"},
		{"import * as", "expected 'Identifier' instead of EOF in import statement"},
		{"import {", "expected '}' instead of EOF in import statement"},
		{"import {yield", "expected '}' instead of EOF in import statement"},
		{"import {yield as", "expected 'Identifier' instead of EOF in import statement"},
		{"import {yield,", "expected '}' instead of EOF in import statement"},
		{"import yield", "expected 'from' instead of EOF in import statement"},
		{"import yield from", "expected 'String' instead of EOF in import statement"},
		{"export", "expected '*', '{', 'var', 'let', 'const', 'function', 'async', 'class', or 'default' instead of EOF in export statement"},
		{"export *", "expected 'from' instead of EOF in export statement"},
		{"export * as", "expected 'Identifier' instead of EOF in export statement"},
		{"export * as if", "expected 'from' instead of EOF in export statement"},
		{"export {", "expected '}' instead of EOF in export statement"},
		{"export {yield", "expected '}' instead of EOF in export statement"},
		{"export {yield,", "expected '}' instead of EOF in export statement"},
		{"export {yield as", "expected 'Identifier' instead of EOF in export statement"},
		{"export {} from", "expected 'String' instead of EOF in export statement"},
		{"export {} from", "expected 'String' instead of EOF in export statement"},
		{"export async", "expected 'function' instead of EOF in export statement"},
		{"export default async", "expected 'function' instead of EOF in export statement"},

		// specific cases
		{"{a, if: b, do(){}, ...d}", "unexpected 'if' in expression"}, // block stmt
		{"let {if = 5}", "expected ':' instead of '=' in object binding pattern"},
		{"let {...[]}", "expected 'Identifier' instead of '[' in object binding pattern"},
		{"let {...{}}", "expected 'Identifier' instead of '{' in object binding pattern"},
		{"for", "expected '(' instead of EOF in for statement"},
		{"for b", "expected '(' instead of 'b' in for statement"},
		{"for (a b)", "expected 'in', 'of', or ';' instead of 'b' in for statement"},
		{"for (var a in b;) {}", "expected ')' instead of ';' in for statement"},
		{"async function (a) { class a extends await", "unexpected 'await' in expression"},
		{"x = await\n=> a++", "unexpected '=>' in expression"},

		// regular expressions
		{"x = x / foo /", "unexpected EOF in expression"},
		{"bar (true) /foo/", "unexpected EOF in expression"},
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
