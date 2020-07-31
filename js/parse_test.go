package js

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/tdewolff/parse/v2"
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
		{"var a = b;", "Decl(var Binding(a = b))"},
		{"const a = b;", "Decl(const Binding(a = b))"},
		{"let a = b;", "Decl(let Binding(a = b))"},
		{"let [a,b] = [1, 2];", "Decl(let Binding([ Binding(a), Binding(b) ] = [1, 2]))"},
		{"let [a,[b,c]] = [1, [2, 3]];", "Decl(let Binding([ Binding(a), Binding([ Binding(b), Binding(c) ]) ] = [1, [2, 3]]))"},
		{"let [,,c] = [1, 2, 3];", "Decl(let Binding([ Binding(), Binding(), Binding(c) ] = [1, 2, 3]))"},
		{"let [a, ...b] = [1, 2, 3];", "Decl(let Binding([ Binding(a), ...Binding(b) ] = [1, 2, 3]))"},
		{"let {a, b} = {a: 3, b: 4};", "Decl(let Binding({ Binding(a), Binding(b) } = {a: 3, b: 4}))"},
		{"let {a: [b, {c}]} = {a: [5, {c: 3}]};", "Decl(let Binding({ a: Binding([ Binding(b), Binding({ Binding(c) }) ]) } = {a: [5, {c: 3}]}))"},
		{"let [a = 2] = [];", "Decl(let Binding([ Binding(a = 2) ] = []))"},
		{"let {a: b = 2} = {};", "Decl(let Binding({ a: Binding(b = 2) } = {}))"},
		{"var a = 5 * 4 / 3 ** 2 + ( 5 - 3 );", "Decl(var Binding(a = (((5*4)/(3**2))+((5-3)))))"},
		{"var a, b = c;", "Decl(var Binding(a) Binding(b = c))"},
		{"var a,\nb = c;", "Decl(var Binding(a) Binding(b = c))"},
		{";", "Stmt(;)"},
		{"{; var a = 3;}", "Stmt({ Stmt(;) Decl(var Binding(a = 3)) })"},
		{"{a=5}", "Stmt({ Stmt(a=5) })"},
		{"return", "Stmt(return)"},
		{"return 5*3", "Stmt(return (5*3))"},
		{"break", "Stmt(break)"},
		{"break LABEL", "Stmt(break LABEL)"},
		{"continue", "Stmt(continue)"},
		{"continue LABEL", "Stmt(continue LABEL)"},
		{"if (a == 5) return true", "Stmt(if (a==5) Stmt(return true))"},
		{"if (a == 5) return true else return false", "Stmt(if (a==5) Stmt(return true) else Stmt(return false))"},
		{"if (a) b; else if (c) d;", "Stmt(if a Stmt(b) else Stmt(if c Stmt(d)))"},
		{"if (a) 1; else if (b) 2; else 3", "Stmt(if a Stmt(1) else Stmt(if b Stmt(2) else Stmt(3)))"},
		{"with (a = 5) return true", "Stmt(with (a=5) Stmt(return true))"},
		{"do a++; while (a < 4)", "Stmt(do Stmt(a++) while (a<4))"},
		{"do {a++} while (a < 4)", "Stmt(do Stmt({ Stmt(a++) }) while (a<4))"},
		{"while (a < 4) a++", "Stmt(while (a<4) Stmt(a++))"},
		{"for (var a = 0; a < 4; a++) b = a", "Stmt(for Decl(var Binding(a = 0)) ; (a<4) ; (a++) Stmt(b=a))"},
		{"for (5; a < 4; a++) {}", "Stmt(for 5 ; (a<4) ; (a++) Stmt({ }))"},
		{"for (;;) {}", "Stmt(for ; ; Stmt({ }))"},
		{"for (a,b=5;;) {}", "Stmt(for (a,(b=5)) ; ; Stmt({ }))"},
		{"for (let a;;) {}", "Stmt(for Decl(let Binding(a)) ; ; Stmt({ }))"},
		{"for (var a in b) {}", "Stmt(for Decl(var Binding(a)) in b Stmt({ }))"},
		{"for (var a of b) {}", "Stmt(for Decl(var Binding(a)) of b Stmt({ }))"},
		{"for (var a=5 of b) {}", "Stmt(for Decl(var Binding(a = 5)) of b Stmt({ }))"},
		{"for (var a in b) {}", "Stmt(for Decl(var Binding(a)) in b Stmt({ }))"},
		{"for (a in b) {}", "Stmt(for a in b Stmt({ }))"},
		{"for (a = b;;) {}", "Stmt(for (a=b) ; ; Stmt({ }))"},
		{"throw 5", "Stmt(throw 5)"},
		{"try {} catch {b}", "Stmt(try Stmt({ }) catch Stmt({ Stmt(b) }))"},
		{"try {} finally {c}", "Stmt(try Stmt({ }) finally Stmt({ Stmt(c) }))"},
		{"try {} catch {b} finally {c}", "Stmt(try Stmt({ }) catch Stmt({ Stmt(b) }) finally Stmt({ Stmt(c) }))"},
		{"try {} catch (e) {b}", "Stmt(try Stmt({ }) catch Binding(e) Stmt({ Stmt(b) }))"},
		{"debugger", "Stmt(debugger)"},
		{"label: var a", "Stmt(label : Decl(var Binding(a)))"},
		{"yield: var a", "Stmt(yield : Decl(var Binding(a)))"},
		{"await: var a", "Stmt(await : Decl(var Binding(a)))"},
		{"switch (5) {}", "Stmt(switch 5)"},
		{"switch (5) { case 3: {} default: {}}", "Stmt(switch 5 Clause(case 3 Stmt({ })) Clause(default Stmt({ })))"},
		{"function a(b) {}", "Decl(function a Params(Binding(b)) Stmt({ }))"},
		{"async function a(b) {}", "Decl(async function a Params(Binding(b)) Stmt({ }))"},
		{"function* a(b) {}", "Decl(function* a Params(Binding(b)) Stmt({ }))"},
		{"function a(b,) {}", "Decl(function a Params(Binding(b)) Stmt({ }))"},
		{"function a(b, c) {}", "Decl(function a Params(Binding(b), Binding(c)) Stmt({ }))"},
		{"function a(...b) {}", "Decl(function a Params(...Binding(b)) Stmt({ }))"},
		{"function a(b, ...c) {}", "Decl(function a Params(Binding(b), ...Binding(c)) Stmt({ }))"},
		{"function a(b) {return}", "Decl(function a Params(Binding(b)) Stmt({ Stmt(return) }))"},
		{"class { }", "Decl(class)"},
		{"class { ; }", "Decl(class)"},
		{"class A { }", "Decl(class A)"},
		{"class A extends B { }", "Decl(class A extends B)"},
		{"class { a(b) {} }", "Decl(class Method(a Params(Binding(b)) Stmt({ })))"},
		{"class { 5(b) {} }", "Decl(class Method(5 Params(Binding(b)) Stmt({ })))"},
		{"class { 'a'(b) {} }", "Decl(class Method(a Params(Binding(b)) Stmt({ })))"},
		{"class { '5'(b) {} }", "Decl(class Method(5 Params(Binding(b)) Stmt({ })))"},
		{"class { '%'(b) {} }", "Decl(class Method('%' Params(Binding(b)) Stmt({ })))"},
		{"class { get() {} }", "Decl(class Method(get Params() Stmt({ })))"},
		{"class { get a() {} }", "Decl(class Method(get a Params() Stmt({ })))"},
		{"class { set a(b) {} }", "Decl(class Method(set a Params(Binding(b)) Stmt({ })))"},
		{"class { * a(b) {} }", "Decl(class Method(* a Params(Binding(b)) Stmt({ })))"},
		{"class { async a(b) {} }", "Decl(class Method(async a Params(Binding(b)) Stmt({ })))"},
		{"class { async * a(b) {} }", "Decl(class Method(async * a Params(Binding(b)) Stmt({ })))"},
		{"class { static() {} }", "Decl(class Method(static Params() Stmt({ })))"},
		{"class { static a(b) {} }", "Decl(class Method(static a Params(Binding(b)) Stmt({ })))"},
		{"class { [5](b) {} }", "Decl(class Method([5] Params(Binding(b)) Stmt({ })))"},
		{"`tmpl`", "Stmt(`tmpl`)"},
		{"`tmpl${x}`", "Stmt(`tmpl${x}`)"},
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
		{"export {if, for as switch,}", "Stmt(export { if , for as switch , })"},
		{"export var a", "Stmt(export Decl(var Binding(a)))"},
		{"export function a(b){}", "Stmt(export Decl(function a Params(Binding(b)) Stmt({ })))"},
		{"export async function a(b){}", "Stmt(export Decl(async function a Params(Binding(b)) Stmt({ })))"},
		{"export class{}", "Stmt(export Decl(class))"},
		{"export default function(b){}", "Stmt(export default Decl(function Params(Binding(b)) Stmt({ })))"},
		{"export default async function(b){}", "Stmt(export default Decl(async function Params(Binding(b)) Stmt({ })))"},
		{"export default class{}", "Stmt(export default Decl(class))"},
		{"export default a", "Stmt(export default a)"},
		{"export default async", "Stmt(export default async)"},

		// yield, await, async
		{"yield\na = 5", "Stmt(yield) Stmt(a=5)"},
		{"yield * yield * a", "Stmt((yield*yield)*a)"},
		{"async", "Stmt(async)"},
		{"async = a", "Stmt(async=a)"},
		{"async\n= a", "Stmt(async=a)"},
		{"async a => b", "Stmt(async Params(Binding(a)) => Stmt({ Stmt(return b) }))"},
		{"async (a) => b", "Stmt(async Params(Binding(a)) => Stmt({ Stmt(return b) }))"},
		{"async\nawait => b", "Stmt(async) Stmt(Params(Binding(await)) => Stmt({ Stmt(return b) }))"},
		{"a + async\nb", "Stmt(a+async) Stmt(b)"},
		{"a + async\nfunction(){}", "Stmt(a+async) Decl(function Params() Stmt({ }))"},
		{"class a extends async {}", "Decl(class a extends async)"},
		{"function*a(){ yield a = 5 }", "Decl(function* a Params() Stmt({ Stmt(yield (a=5)) }))"},
		{"function*a(){ yield * a = 5 }", "Decl(function* a Params() Stmt({ Stmt(yield* (a=5)) }))"},
		{"function*a(){ yield\na = 5 }", "Decl(function* a Params() Stmt({ Stmt(yield) Stmt(a=5) }))"},
		{"function*a(){ yield yield a }", "Decl(function* a Params() Stmt({ Stmt(yield (yield a)) }))"},
		{"function*a(){ yield * yield * a }", "Decl(function* a Params() Stmt({ Stmt(yield* (yield* a)) }))"},
		{"function*a(b = yield c){}", "Decl(function* a Params(Binding(b = (yield c))) Stmt({ }))"},
		{"function*a(){ x = function yield(){} }", "Decl(function* a Params() Stmt({ Stmt(x=Decl(function yield Params() Stmt({ }))) }))"},
		{"function*a(){ x = function b(){ x = yield } }", "Decl(function* a Params() Stmt({ Stmt(x=Decl(function b Params() Stmt({ Stmt(x=yield) }))) }))"},
		{"function*a(){ (yield a) }", "Decl(function* a Params() Stmt({ Stmt((yield a)) }))"},
		{"let\nawait", "Decl(let Binding(await))"},
		{"x = {await}", "Stmt(x={await})"},
		{"async function a(){ x = {await: 5} }", "Decl(async function a Params() Stmt({ Stmt(x={await: 5}) }))"},
		{"async function a(){ x = await a }", "Decl(async function a Params() Stmt({ Stmt(x=(await a)) }))"},
		{"async function a(){ x = await a+y }", "Decl(async function a Params() Stmt({ Stmt(x=((await a)+y)) }))"},
		{"async function a(b = await c){}", "Decl(async function a Params(Binding(b = (await c))) Stmt({ }))"},
		{"async function a(){ x = function await(){} }", "Decl(async function a Params() Stmt({ Stmt(x=Decl(function await Params() Stmt({ }))) }))"},
		{"async function a(){ x = function b(){ x = await } }", "Decl(async function a Params() Stmt({ Stmt(x=Decl(function b Params() Stmt({ Stmt(x=await) }))) }))"},
		{"async function a(){ for await (var a of b) {} }", "Decl(async function a Params() Stmt({ Stmt(for await Decl(var Binding(a)) of b Stmt({ })) }))"},
		{"async function a(){ (await a) }", "Decl(async function a Params() Stmt({ Stmt((await a)) }))"},
		{"x = {async a(b){}}", "Stmt(x={Method(async a Params(Binding(b)) Stmt({ }))})"},

		// bindings
		{"let []", "Decl(let Binding([ ]))"},
		{"let [,]", "Decl(let Binding([ Binding() ]))"},
		{"let [,a]", "Decl(let Binding([ Binding(), Binding(a) ]))"},
		{"let [name = 5]", "Decl(let Binding([ Binding(name = 5) ]))"},
		{"let [name = 5,]", "Decl(let Binding([ Binding(name = 5) ]))"},
		{"let [name = 5,,]", "Decl(let Binding([ Binding(name = 5), Binding() ]))"},
		{"let [name = 5,, ...yield]", "Decl(let Binding([ Binding(name = 5), Binding(), ...Binding(yield) ]))"},
		{"let [...yield]", "Decl(let Binding([ ...Binding(yield) ]))"},
		{"let [,,...yield]", "Decl(let Binding([ Binding(), Binding(), ...Binding(yield) ]))"},
		{"let [name = 5,, ...[yield]]", "Decl(let Binding([ Binding(name = 5), Binding(), ...Binding([ Binding(yield) ]) ]))"},
		{"let [name = 5,, ...{yield}]", "Decl(let Binding([ Binding(name = 5), Binding(), ...Binding({ Binding(yield) }) ]))"},
		{"let {}", "Decl(let Binding({ }))"},
		{"let {name = 5}", "Decl(let Binding({ Binding(name = 5) }))"},
		{"let {await = 5}", "Decl(let Binding({ Binding(await = 5) }))"},
		{"let {if: name}", "Decl(let Binding({ if: Binding(name) }))"},
		{"let {\"string\": name}", "Decl(let Binding({ string: Binding(name) }))"},
		{"let {[a = 5]: name}", "Decl(let Binding({ [a=5]: Binding(name) }))"},
		{"let {if: name = 5}", "Decl(let Binding({ if: Binding(name = 5) }))"},
		{"let {if: yield = 5}", "Decl(let Binding({ if: Binding(yield = 5) }))"},
		{"let {if: [name] = 5}", "Decl(let Binding({ if: Binding([ Binding(name) ] = 5) }))"},
		{"let {if: {name} = 5}", "Decl(let Binding({ if: Binding({ Binding(name) } = 5) }))"},
		{"let {...yield}", "Decl(let Binding({ ...Binding(yield) }))"},
		{"let {if: name, ...yield}", "Decl(let Binding({ if: Binding(name), ...Binding(yield) }))"},

		// expressions
		{"x = [a, ...b]", "Stmt(x=[a, ...b])"},
		{"x = [...b]", "Stmt(x=[...b])"},
		{"x = [...a, ...b]", "Stmt(x=[...a, ...b])"},
		{"x = [,]", "Stmt(x=[,])"},
		{"x = [,,]", "Stmt(x=[, ,])"},
		{"x = [a,]", "Stmt(x=[a])"},
		{"x = [a,,]", "Stmt(x=[a, ,])"},
		{"x = [,a]", "Stmt(x=[, a])"},
		{"x = {a}", "Stmt(x={a})"},
		{"x = {...a}", "Stmt(x={...a})"},
		{"x = {a, ...b}", "Stmt(x={a, ...b})"},
		{"x = {...a, ...b}", "Stmt(x={...a, ...b})"},
		{"x = {a=5}", "Stmt(x={a = 5})"},
		{"x = {yield=5}", "Stmt(x={yield = 5})"},
		{"x = {a:5}", "Stmt(x={a: 5})"},
		{"x = {yield:5}", "Stmt(x={yield: 5})"},
		{"x = {async:5}", "Stmt(x={async: 5})"},
		{"x = {if:5}", "Stmt(x={if: 5})"},
		{"x = {\"string\":5}", "Stmt(x={string: 5})"},
		{"x = {3:5}", "Stmt(x={3: 5})"},
		{"x = {[3]:5}", "Stmt(x={[3]: 5})"},
		{"x = {a, if: b, do(){}, ...d}", "Stmt(x={a, if: b, Method(do Params() Stmt({ })), ...d})"},
		{"x = {*a(){}}", "Stmt(x={Method(* a Params() Stmt({ }))})"},
		{"x = {async*a(){}}", "Stmt(x={Method(async * a Params() Stmt({ }))})"},
		{"x = {get a(){}}", "Stmt(x={Method(get a Params() Stmt({ }))})"},
		{"x = {set a(){}}", "Stmt(x={Method(set a Params() Stmt({ }))})"},
		{"x = {get(){}}", "Stmt(x={Method(get Params() Stmt({ }))})"},
		{"x = {set(){}}", "Stmt(x={Method(set Params() Stmt({ }))})"},
		{"x = (a, b)", "Stmt(x=((a,b)))"},
		{"x = function() {}", "Stmt(x=Decl(function Params() Stmt({ })))"},
		{"x = async function() {}", "Stmt(x=Decl(async function Params() Stmt({ })))"},
		{"x = class {}", "Stmt(x=Decl(class))"},
		{"x = class {a(){}}", "Stmt(x=Decl(class Method(a Params() Stmt({ }))))"},
		{"x = a => a++", "Stmt(x=(Params(Binding(a)) => Stmt({ Stmt(return (a++)) })))"},
		{"x = a => {a++}", "Stmt(x=(Params(Binding(a)) => Stmt({ Stmt(a++) })))"},
		{"x = a => {return}", "Stmt(x=(Params(Binding(a)) => Stmt({ Stmt(return) })))"},
		{"x = a => {return a}", "Stmt(x=(Params(Binding(a)) => Stmt({ Stmt(return a) })))"},
		{"x = yield => a++", "Stmt(x=(Params(Binding(yield)) => Stmt({ Stmt(return (a++)) })))"},
		{"x = yield => {a++}", "Stmt(x=(Params(Binding(yield)) => Stmt({ Stmt(a++) })))"},
		{"x = async a => a++", "Stmt(x=(async Params(Binding(a)) => Stmt({ Stmt(return (a++)) })))"},
		{"x = async a => {a++}", "Stmt(x=(async Params(Binding(a)) => Stmt({ Stmt(a++) })))"},
		{"x = async a => await b", "Stmt(x=(async Params(Binding(a)) => Stmt({ Stmt(return (await b)) })))"},
		{"x = await => a++", "Stmt(x=(Params(Binding(await)) => Stmt({ Stmt(return (a++)) })))"},
		{"x = a??b", "Stmt(x=(a??b))"},
		{"x = a[b]", "Stmt(x=(a[b]))"},
		{"x = a?.b?.c.d", "Stmt(x=(((a?.b)?.c).d))"},
		{"x = a?.[b]?.`tpl`", "Stmt(x=((a?.[b])?.`tpl`))"},
		{"x = a?.(b)", "Stmt(x=(a?.(b)))"},
		{"x = super(a)", "Stmt(x=(super(a)))"},
		{"x = a(a,b,...c,)", "Stmt(x=(a(a, b, ...c)))"},
		{"x = new a", "Stmt(x=(new a))"},
		{"x = new a()", "Stmt(x=(new a))"},
		{"x = new a(b)", "Stmt(x=(new a(b)))"},
		{"x = new new.target", "Stmt(x=(new (new.target)))"},
		{"x = new import.meta", "Stmt(x=(new (import.meta)))"},
		{"x = import(a)", "Stmt(x=(import(a)))"},
		{"x = +a", "Stmt(x=(+a))"},
		{"x = ++a", "Stmt(x=(++a))"},
		{"x = -a", "Stmt(x=(-a))"},
		{"x = --a", "Stmt(x=(--a))"},
		{"x = a--", "Stmt(x=(a--))"},
		{"x = a<<b", "Stmt(x=(a<<b))"},
		{"x = a|b", "Stmt(x=(a|b))"},
		{"x = a&b", "Stmt(x=(a&b))"},
		{"x = a^b", "Stmt(x=(a^b))"},
		{"x = a||b", "Stmt(x=(a||b))"},
		{"x = a&&b", "Stmt(x=(a&&b))"},
		{"x = !a", "Stmt(x=(!a))"},
		{"x = delete a", "Stmt(x=(delete a))"},
		{"x = a in b", "Stmt(x=(a in b))"},
		{"x = a.replace(b, c)", "Stmt(x=((a.replace)(b, c)))"},
		{"class a extends async function(){}{}", "Decl(class a extends Decl(async function Params() Stmt({ })))"},
		{"x = a?b:c=d", "Stmt(x=(a ? b : (c=d)))"},
		{"implements = 0", "Stmt(implements=0)"},
		{"interface = 0", "Stmt(interface=0)"},
		{"let = 0", "Stmt(let=0)"},
		{"(let [a] = 0)", "Stmt(((let[a])=0))"},
		{"package = 0", "Stmt(package=0)"},
		{"private = 0", "Stmt(private=0)"},
		{"protected = 0", "Stmt(protected=0)"},
		{"public = 0", "Stmt(public=0)"},
		{"static = 0", "Stmt(static=0)"},

		// expression to arrow function parameters
		{"x = (a,b,c) => {a++}", "Stmt(x=(Params(Binding(a), Binding(b), Binding(c)) => Stmt({ Stmt(a++) })))"},
		{"x = (a,b,...c) => {a++}", "Stmt(x=(Params(Binding(a), Binding(b), ...Binding(c)) => Stmt({ Stmt(a++) })))"},
		{"x = ([a, ...b]) => {a++}", "Stmt(x=(Params(Binding([ Binding(a), ...Binding(b) ])) => Stmt({ Stmt(a++) })))"},
		{"x = ([,a,]) => {a++}", "Stmt(x=(Params(Binding([ Binding(), Binding(a) ])) => Stmt({ Stmt(a++) })))"},
		{"x = ({a}) => {a++}", "Stmt(x=(Params(Binding({ Binding(a) })) => Stmt({ Stmt(a++) })))"},
		{"x = ({a:b, c:d}) => {a++}", "Stmt(x=(Params(Binding({ a: Binding(b), c: Binding(d) })) => Stmt({ Stmt(a++) })))"},
		{"x = ({a:[b]}) => {a++}", "Stmt(x=(Params(Binding({ a: Binding([ Binding(b) ]) })) => Stmt({ Stmt(a++) })))"},
		{"x = ({a=5}) => {a++}", "Stmt(x=(Params(Binding({ Binding(a = 5) })) => Stmt({ Stmt(a++) })))"},
		{"x = ({...a}) => {a++}", "Stmt(x=(Params(Binding({ ...Binding(a) })) => Stmt({ Stmt(a++) })))"},
		{"x = ([{...a}]) => {a++}", "Stmt(x=(Params(Binding([ Binding({ ...Binding(a) }) ])) => Stmt({ Stmt(a++) })))"},
		{"x = ([{a: b}]) => {a++}", "Stmt(x=(Params(Binding([ Binding({ a: Binding(b) }) ])) => Stmt({ Stmt(a++) })))"},
		{"x = (a = 5) => {a++}", "Stmt(x=(Params(Binding(a = 5)) => Stmt({ Stmt(a++) })))"},

		// expression precedence
		{"!!a", "Stmt(!(!a))"},
		{"x = a.b.c", "Stmt(x=((a.b).c))"},
		{"x = a+b+c", "Stmt(x=((a+b)+c))"},
		{"x = a**b**c", "Stmt(x=(a**(b**c)))"},
		{"a++ < b", "Stmt((a++)<b)"},
		{"a&&b&&c", "Stmt((a&&b)&&c)"},
		{"a||b||c", "Stmt((a||b)||c)"},
		{"new new a(b)", "Stmt(new (new a(b)))"},
		{"new super.a(b)", "Stmt(new (super.a)(b))"},
		{"new new.target(a)", "Stmt(new (new.target)(a))"},
		{"new import.meta(a)", "Stmt(new (import.meta)(a))"},
		{"a||b?c:d", "Stmt((a||b) ? c : d)"},
		{"a??b?c:d", "Stmt((a??b) ? c : d)"},
		{"a==b==c", "Stmt((a==b)==c)"},
		{"new a?.b", "Stmt((new a)?.b)"},
		{"new a++", "Stmt((new a)++)"},
		{"new a--", "Stmt((new a)--)"},
		{"a<<b<<c", "Stmt((a<<b)<<c)"},
		{"a&b&c", "Stmt((a&b)&c)"},
		{"a|b|c", "Stmt((a|b)|c)"},
		{"a^b^c", "Stmt((a^b)^c)"},
		{"a,b,c", "Stmt((a,b),c)"},

		// regular expressions
		{"/abc/", "Stmt(/abc/)"},
		{"return /abc/;", "Stmt(return /abc/)"},
		{"a/b/g", "Stmt((a/b)/g)"},
		{"{}/1/g", "Stmt({ }) Stmt(/1/g)"},
		{"i(0)/1/g", "Stmt(((i(0))/1)/g)"},
		{"if(0)/1/g", "Stmt(if 0 Stmt(/1/g))"},
		{"a.if(0)/1/g", "Stmt((((a.if)(0))/1)/g)"},
		{"this/1/g", "Stmt((this/1)/g)"},
		{"switch(a){case /1/g:}", "Stmt(switch a Clause(case /1/g))"},
		{"(a+b)/1/g", "Stmt((((a+b))/1)/g)"},
		{"f(); function foo() {} /42/i", "Stmt(f()) Decl(function foo Params() Stmt({ })) Stmt(/42/i)"},
		{"x = function() {} /42/i", "Stmt(x=((Decl(function Params() Stmt({ }))/42)/i))"},
		{"x = function foo() {} /42/i", "Stmt(x=((Decl(function foo Params() Stmt({ }))/42)/i))"},
		{"x = /foo/", "Stmt(x=/foo/)"},
		{"x = (/foo/)", "Stmt(x=(/foo/))"},
		{"x = {a: /foo/}", "Stmt(x={a: /foo/})"},
		{"x = (a) / foo", "Stmt(x=((a)/foo))"},
		{"do { /foo/ } while (a)", "Stmt(do Stmt({ Stmt(/foo/) }) while a)"},
		{"if (true) /foo/", "Stmt(if true Stmt(/foo/))"},
		{"/abc/ ? /def/ : /geh/", "Stmt(/abc/ ? /def/ : /geh/)"},
		{"yield * /abc/", "Stmt(yield*/abc/)"},

		// variable reuse
		{"var a; var a", "Decl(var Binding(a)) Decl(var Binding(a))"},
		{"var a; {let a}", "Decl(var Binding(a)) Stmt({ Decl(let Binding(a)) })"},
		{"{let a} var a", "Stmt({ Decl(let Binding(a)) }) Decl(var Binding(a))"},
		{"function a(b,b){}", "Decl(function a Params(Binding(b), Binding(b)) Stmt({ }))"},
		{"function a(b){var b}", "Decl(function a Params(Binding(b)) Stmt({ Decl(var Binding(b)) }))"},
		{"a=function(b){var b}", "Stmt(a=Decl(function Params(Binding(b)) Stmt({ Decl(var Binding(b)) })))"},
		{"a=function b(){var b}", "Stmt(a=Decl(function b Params() Stmt({ Decl(var Binding(b)) })))"},
		{"a=function b(){let b}", "Stmt(a=Decl(function b Params() Stmt({ Decl(let Binding(b)) })))"},
		{"a=>{var a}", "Stmt(Params(Binding(a)) => Stmt({ Decl(var Binding(a)) }))"},
		{"var a;function a(){}", "Decl(var Binding(a)) Decl(function a Params() Stmt({ }))"},
		{"try{}catch(a){var a}", "Stmt(try Stmt({ }) catch Binding(a) Stmt({ Decl(var Binding(a)) }))"},

		// ASI
		{"return a", "Stmt(return a)"},
		{"return; a", "Stmt(return) Stmt(a)"},
		{"return\na", "Stmt(return) Stmt(a)"},
		{"return /*comment*/ a", "Stmt(return a)"},
		{"return /*com\nment*/ a", "Stmt(return) Stmt(a)"},
		{"return //comment\n a", "Stmt(return) Stmt(a)"},
		{"a?.b\n`c`", "Stmt((a?.b)`c`)"},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(parse.NewInputString(tt.js))
			if err != io.EOF {
				test.Error(t, err)
			}
			test.String(t, ast.String(), tt.expected)
		})
	}

	// coverage
	for i := 0; ; i++ {
		if OpPrec(i).String() == fmt.Sprintf("Invalid(%d)", i) {
			break
		}
	}
	for i := 0; ; i++ {
		if DeclType(i).String() == fmt.Sprintf("Invalid(%d)", i) {
			break
		}
	}
}

func TestParseError(t *testing.T) {
	var tests = []struct {
		js  string
		err string
	}{
		{"{a", "unexpected EOF"},
		{"if", "expected '(' instead of EOF in if statement"},
		{"if(a", "expected ')' instead of EOF in if statement"},
		{"if(a)let b", "unexpected 'b' in expression"},
		{"if(a)const b", "unexpected 'const' in statement"},
		{"with", "expected '(' instead of EOF in with statement"},
		{"with(a", "expected ')' instead of EOF in with statement"},
		{"do a++", "expected 'while' instead of EOF in do-while statement"},
		{"do a++ while", "unexpected 'while' in expression"},
		{"do a++; while", "expected '(' instead of EOF in do-while statement"},
		{"do a++; while(a", "expected ')' instead of EOF in do-while statement"},
		{"while", "expected '(' instead of EOF in while statement"},
		{"while(a", "expected ')' instead of EOF in while statement"},
		{"for", "expected '(' instead of EOF in for statement"},
		{"for(a", "expected 'in', 'of', or ';' instead of EOF in for statement"},
		{"for(a;a", "expected ';' instead of EOF in for statement"},
		{"for(a;a;a", "expected ')' instead of EOF in for statement"},
		{"for await", "expected '(' instead of 'await' in for statement"},
		{"async function a(){ for await(a;", "expected 'of' instead of ';' in for statement"},
		{"async function a(){ for await(a in", "expected 'of' instead of 'in' in for statement"},
		{"for(var a of b", "expected ')' instead of EOF in for statement"},
		{"switch", "expected '(' instead of EOF in switch statement"},
		{"switch(a", "expected ')' instead of EOF in switch statement"},
		{"switch(a)", "expected '{' instead of EOF in switch statement"},
		{"switch(a){bad:5}", "expected 'case' or 'default' instead of 'bad' in switch statement"},
		{"switch(a){case", "unexpected EOF in expression"},
		{"switch(a){case a", "expected ':' instead of EOF in switch statement"},
		{"switch(a){case a:", "unexpected EOF in switch statement"},
		{"try", "expected '{' instead of EOF in try statement"},
		{"try{", "unexpected EOF"},
		{"try{}", "expected 'catch' or 'finally' instead of EOF in try statement"},
		{"try{}catch(a", "expected ')' instead of EOF in try-catch statement"},
		{"try{}catch(a,", "expected ')' instead of ',' in try-catch statement"},
		{"try{}catch", "expected '{' instead of EOF in try-catch statement"},
		{"try{}finally", "expected '{' instead of EOF in try-finally statement"},
		{"function", "expected 'Identifier' or '(' instead of EOF in function declaration"},
		{"async function", "expected 'Identifier' or '(' instead of EOF in function declaration"},
		{"function a", "expected '(' instead of EOF in function declaration"},
		{"function a(b", "unexpected EOF in function declaration"},
		{"function a(b,", "unexpected EOF in function declaration"},
		{"function a(...b", "expected ')' instead of EOF in function declaration"},
		{"function a()", "expected '{' instead of EOF in function declaration"},
		{"class A", "expected '{' instead of EOF in class declaration"},
		{"class A{", "unexpected EOF in class declaration"},
		{"class A extends a b {}", "expected '{' instead of 'b' in class declaration"},
		{"class A{+", "expected 'Identifier', 'String', 'Numeric', or '[' instead of '+' in method definition"},
		{"class A{[a", "expected ']' instead of EOF in method definition"},
		{"var [...a", "expected ']' instead of EOF in array binding pattern"},
		{"var [a", "expected ',' or ']' instead of EOF in array binding pattern"},
		{"var {[a", "expected ']' instead of EOF in object binding pattern"},
		{"var {+", "expected 'Identifier', 'String', 'Numeric', or '[' instead of '+' in object binding pattern"},
		{"var {a", "expected ',' or '}' instead of EOF in object binding pattern"},
		{"var {...a", "expected '}' instead of EOF in object binding pattern"},
		{"var 0", "unexpected '0' in binding"},
		{"x={", "expected '}' instead of EOF in object literal"},
		{"x={[a", "expected ']' instead of EOF in object literal"},
		{"x={[a]", "expected ':' or '(' instead of EOF in object literal"},
		{"x={+", "expected 'Identifier', 'String', 'Numeric', or '[' instead of '+' in object literal"},
		{"x={async\na", "unexpected 'a' in object literal"},
		{"class a extends ||", "unexpected '||' in expression"},
		{"class a extends =", "unexpected '=' in expression"},
		{"class a extends ?", "unexpected '?' in expression"},
		{"class a extends =>", "unexpected '=>' in expression"},
		{"x=a?b", "expected ':' instead of EOF in conditional expression"},
		{"x=(a", "unexpected EOF in expression"},
		{"x+(a", "expected ')' instead of EOF in expression"},
		{"x={a", "unexpected EOF in object literal"},
		{"x=a[b", "expected ']' instead of EOF in index expression"},
		{"x=async a", "expected '=>' instead of EOF in arrow function"},
		{"x=async (a", "unexpected EOF in arrow function"},
		{"x=async (a,", "unexpected EOF in arrow function"},
		{"x=async function", "expected 'Identifier' or '(' instead of EOF in function declaration"},
		{"x=async function *", "expected 'Identifier' or '(' instead of EOF in function declaration"},
		{"x=async function a", "expected '(' instead of EOF in function declaration"},
		{"x=?.?.b", "unexpected '?.' in expression"},
		{"x=a?.?.b", "expected 'Identifier', '(', '[', or 'Template' instead of '?.' in optional chaining expression"},
		{"x=a?..b", "expected 'Identifier', '(', '[', or 'Template' instead of '.' in optional chaining expression"},
		{"x=a?.[b", "expected ']' instead of EOF in optional chaining expression"},
		{"`tmp${", "unexpected EOF in expression"},
		{"`tmp${x", "expected 'Template' instead of EOF in template literal"},
		{"`tmpl` x `tmpl`", "unexpected 'x' in expression"},
		{"x=5=>", "unexpected '=>' in expression"},
		{"x=new.bad", "expected 'target' instead of 'bad' in new.target expression"},
		{"x=import.bad", "expected 'meta' instead of 'bad' in import.meta expression"},
		{"x=super", "expected '[', '(', or '.' instead of EOF in super expression"},
		{"x=super(a", "expected ')' instead of EOF in arguments"},
		{"x=super[a", "expected ']' instead of EOF in index expression"},
		{"x=super.", "expected 'Identifier' instead of EOF in dot expression"},
		{"x=new super(b)", "expected '[' or '.' instead of '(' in super expression"},
		{"x=import", "expected '(' instead of EOF in import expression"},
		{"x=import(5", "expected ')' instead of EOF in arguments"},
		{"x=new import(b)", "unexpected '(' in expression"},
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

		// no declarations
		{"if(a) function f(){}", "unexpected 'function' in statement"},
		{"if(a) async function f(){}", "unexpected 'async' in statement"},
		{"if(a) class c{}", "unexpected 'class' in statement"},

		// yield, async, await
		{"yield a = 5", "unexpected 'a' in expression"},
		{"function*a() { yield: var a", "unexpected ':' in expression"},
		{"function*a() { x = b + yield c", "unexpected 'yield' in expression"},
		{"function a(b = yield c){}", "unexpected 'c' in function declaration"},
		{"function*a(){ (yield) => yield }", "unexpected ')' in expression"},
		{"x = await\n=> a++", "unexpected '=>' in expression"},
		{"async function a() { class a extends await", "unexpected 'await' in expression"},
		{"async function a() { await: var a", "unexpected ':' in expression"},
		{"async function a() { let await", "unexpected 'await' in binding"},
		{"async function a() { let\nawait", "unexpected 'await' in binding"},
		{"async function a() { x = new await c", "unexpected 'await' in expression"},
		{"async function a() { x = await =>", "unexpected '=>' in expression"},
		{"async function a(){ (await) => await }", "unexpected ')' in expression"},
		{"async+a b", "unexpected 'b' in expression"},
		{"(async\nfunction(){})", "unexpected 'function' in expression"},
		{"a + async b", "unexpected 'b' in expression"},
		{"async await => 5", "unexpected 'await' in arrow function"},

		// specific cases
		{"{a, if: b, do(){}, ...d}", "unexpected 'if' in expression"}, // block stmt
		{"let {if = 5}", "expected ':' instead of '=' in object binding pattern"},
		{"let {...}", "expected 'Identifier' instead of '}' in object binding pattern"},
		{"let {...[]}", "expected 'Identifier' instead of '[' in object binding pattern"},
		{"let {...{}}", "expected 'Identifier' instead of '{' in object binding pattern"},
		{"for", "expected '(' instead of EOF in for statement"},
		{"for b", "expected '(' instead of 'b' in for statement"},
		{"for (a b)", "expected 'in', 'of', or ';' instead of 'b' in for statement"},
		{"for (var a in b;) {}", "expected ')' instead of ';' in for statement"},
		{"if (a) 1 else 3", "unexpected 'else' in expression"},
		{"x = [...]", "unexpected ']' in expression"},
		{"x = {...}", "unexpected '}' in expression"},
		{"let\nawait 0", "unexpected '0' in let declaration"},
		{"const\nawait 0", "unexpected '0' in const declaration"},
		{"var\nawait 0", "unexpected '0' in var statement"},

		// expression to arrow function parameters
		{"x = ()", "expected '=>' instead of EOF in arrow function"},
		{"x = [x] => a", "unexpected '=>' in expression"},
		{"x = ((x)) => a", "unexpected '=>' in expression"},
		{"x = ([...x, y]) => a", "unexpected '=>' in expression"},
		{"x = ({...x, y}) => a", "unexpected '=>' in expression"},
		{"x = ({b(){}}) => a", "unexpected '=>' in expression"},
		{"x = (a, b, ...c)", "expected '=>' instead of EOF in arrow function"},
		{"x = (a+b) =>", "unexpected '=>' in expression"},
		{"x = ([...a, b]) =>", "unexpected '=>' in expression"},
		{"x = ([...5]) =>", "unexpected '=>' in expression"},
		{"x = ([5]) =>", "unexpected '=>' in expression"},
		{"x = ({...a, b}) =>", "unexpected '=>' in expression"},
		{"x = ({...5}) =>", "unexpected '=>' in expression"},
		{"x = ({5: 5}) =>", "unexpected '=>' in expression"},
		{"x = ({[4+5]: 5}) =>", "unexpected '=>' in expression"},

		// expression precedence
		{"x = a + yield b", "unexpected 'b' in expression"},
		{"a??b||c", "unexpected '||' in expression"},
		{"a??b&&c", "unexpected '&&' in expression"},
		{"a||b??c", "unexpected '??' in expression"},
		{"a&&b??c", "unexpected '??' in expression"},
		{"x = a++--", "unexpected '--' in expression"},
		{"x = a\n++", "unexpected EOF in expression"},
		{"x = a++?", "unexpected EOF in expression"},
		{"a+b =", "unexpected '=' in expression"},
		{"!a**b", "unexpected '**' in expression"},
		{"new !a", "unexpected '!' in expression"},
		{"new +a", "unexpected '+' in expression"},
		{"new -a", "unexpected '-' in expression"},
		{"new ++a", "unexpected '++' in expression"},
		{"new --a", "unexpected '--' in expression"},
		{"a=>{return a} < b", "unexpected '<' in expression"},
		{"a=>{return a} == b", "unexpected '==' in expression"},
		{"a=>{return a} . b", "unexpected '.' in expression"},
		{"a=>{return a} (", "unexpected '(' in expression"},
		{"a=>{return a} [", "unexpected '[' in expression"},
		{"a=>{return a} `", "unexpected '`' in expression"},
		{"a=>{return a} ++", "unexpected '++' in expression"},
		{"a=>{return a} --", "unexpected '--' in expression"},
		{"a=>{return a} * b", "unexpected '*' in expression"},
		{"a=>{return a} + b", "unexpected '+' in expression"},
		{"a=>{return a} << b", "unexpected '<<' in expression"},
		{"a=>{return a} & b", "unexpected '&' in expression"},
		{"a=>{return a} | b", "unexpected '|' in expression"},
		{"a=>{return a} ^ b", "unexpected '^' in expression"},
		{"a=>{return a} ? b", "unexpected '?' in expression"},
		{"a=>{return a} => b=>b", "unexpected '=>' in expression"},
		{"class a extends b=>b", "expected '{' instead of '=>' in class declaration"},

		// regular expressions
		{"x = x / foo /", "unexpected EOF in expression"},
		{"bar (true) /foo/", "unexpected EOF in expression"},
		{"yield /abc/", "unexpected EOF in expression"},

		// variable reuse
		{"let a; var a", "identifier 'a' has already been declared"},
		{"let a; {var a}", "identifier 'a' has already been declared"},
		{"var a; let a", "identifier 'a' has already been declared"},
		{"{var a} let a", "identifier 'a' has already been declared"},
		{"var a; const a", "identifier 'a' has already been declared"},
		{"var a; class a{}", "identifier 'a' has already been declared"},
		{"function a(b){let b}", "identifier 'b' has already been declared"},
		{"a=function(b){let b}", "identifier 'b' has already been declared"},
		{"a=>{let a}", "identifier 'a' has already been declared"},
		{"let a;function a(){}", "identifier 'a' has already been declared"},
		{"try{}catch(a){let a}", "identifier 'a' has already been declared"},
		{"let {a, a}", "identifier 'a' has already been declared"},
		{"let {a, ...a}", "identifier 'a' has already been declared"},

		// other
		{"\x00", "unexpected 0x00"},
		{"@", "unexpected '@'"},
		{"\u200F", "unexpected U+200F"},
		{"\u2010", "unexpected '\u2010'"},
		{"a=\u2010", "unexpected '\u2010' in expression"},
		{"/", "unexpected EOF or newline in regular expression"},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			_, err := Parse(parse.NewInputString(tt.js))
			test.That(t, err != io.EOF && err != nil)

			e := err.Error()
			if len(tt.err) < len(err.Error()) {
				e = e[:len(tt.err)]
			}
			test.String(t, e, tt.err)
		})
	}
}

type ScopeVars struct {
	bound, uses, unbound string
	scopes               int
}

func NewScopeVars(unbounds VarArray) *ScopeVars {
	unboundsArray := []string{}
	for _, v := range unbounds {
		if 0 < v.Uses && v.Decl == NoDecl {
			unboundsArray = append(unboundsArray, string(v.Name))
		}
	}
	sort.Strings(unboundsArray)
	return &ScopeVars{
		unbound: strings.Join(unboundsArray, ","),
	}
}

func (sv *ScopeVars) String() string {
	return "bound:" + sv.bound + " uses:" + sv.uses + " unbound:" + sv.unbound
}

func (sv *ScopeVars) AddScope(scope Scope) {
	if sv.scopes != 0 {
		sv.bound += "/"
		sv.uses += "/"
	}
	sv.scopes++

	bounds := []string{}
	for _, v := range scope.Declared {
		bounds = append(bounds, string(v.Name))
	}
	sort.Strings(bounds)
	sv.bound += strings.Join(bounds, ",")

	uses := []string{}
	for _, v := range scope.Undeclared {
		uses = append(uses, string(v.Name))
	}
	sort.Strings(uses)
	sv.uses += strings.Join(uses, ",")
}

func (sv *ScopeVars) AddExpr(iexpr IExpr) {
	switch expr := iexpr.(type) {
	case *FuncDecl:
		sv.AddScope(expr.Body.Scope)
		for _, item := range expr.Params.List {
			if item.Binding != nil {
				sv.AddBinding(item.Binding)
			}
			if item.Default != nil {
				sv.AddExpr(item.Default)
			}
		}
		if expr.Params.Rest != nil {
			sv.AddBinding(expr.Params.Rest)
		}
		for _, item := range expr.Body.List {
			sv.AddStmt(item)
		}
	case *ClassDecl:
		for _, method := range expr.Methods {
			sv.AddScope(method.Body.Scope)
		}
	case *ArrowFunc:
		sv.AddScope(expr.Body.Scope)
		for _, item := range expr.Params.List {
			if item.Binding != nil {
				sv.AddBinding(item.Binding)
			}
			if item.Default != nil {
				sv.AddExpr(item.Default)
			}
		}
		if expr.Params.Rest != nil {
			sv.AddBinding(expr.Params.Rest)
		}
		for _, item := range expr.Body.List {
			sv.AddStmt(item)
		}
	case *UnaryExpr:
		sv.AddExpr(expr.X)
	case *BinaryExpr:
		sv.AddExpr(expr.X)
		sv.AddExpr(expr.Y)
	case *GroupExpr:
		sv.AddExpr(expr.X)
	}
}

func (sv *ScopeVars) AddBinding(ibinding IBinding) {
	switch binding := ibinding.(type) {
	case *BindingArray:
		for _, item := range binding.List {
			if item.Binding != nil {
				sv.AddBinding(item.Binding)
			}
			if item.Default != nil {
				sv.AddExpr(item.Default)
			}
		}
		if binding.Rest != nil {
			sv.AddBinding(binding.Rest)
		}
	case *BindingObject:
		for _, item := range binding.List {
			if item.Key.IsComputed() {
				sv.AddExpr(item.Key.Computed)
			}
			if item.Value.Binding != nil {
				sv.AddBinding(item.Value.Binding)
			}
			if item.Value.Default != nil {
				sv.AddExpr(item.Value.Default)
			}
		}
	}
}

func (sv *ScopeVars) AddStmt(istmt IStmt) {
	switch stmt := istmt.(type) {
	case *BlockStmt:
		sv.AddScope(stmt.Scope)
		for _, item := range stmt.List {
			sv.AddStmt(item)
		}
	case *FuncDecl:
		sv.AddScope(stmt.Body.Scope)
		for _, item := range stmt.Params.List {
			if item.Binding != nil {
				sv.AddBinding(item.Binding)
			}
			if item.Default != nil {
				sv.AddExpr(item.Default)
			}
		}
		if stmt.Params.Rest != nil {
			sv.AddBinding(stmt.Params.Rest)
		}
		for _, item := range stmt.Body.List {
			sv.AddStmt(item)
		}
	case *ClassDecl:
		for _, method := range stmt.Methods {
			sv.AddScope(method.Body.Scope)
		}
	case *ReturnStmt:
		sv.AddExpr(stmt.Value)
	case *ThrowStmt:
		sv.AddExpr(stmt.Value)
	case *IfStmt:
		sv.AddStmt(stmt.Body)
		if stmt.Else != nil {
			sv.AddStmt(stmt.Else)
		}
	case *TryStmt:
		if 0 < len(stmt.Body.List) {
			sv.AddStmt(&stmt.Body)
		}
		if stmt.Catch != nil {
			sv.AddStmt(stmt.Catch)
		}
		if stmt.Finally != nil {
			sv.AddStmt(stmt.Finally)
		}
	case *VarDecl:
		for _, item := range stmt.List {
			if item.Default != nil {
				sv.AddExpr(item.Default)
			}
		}
	case *ExprStmt:
		sv.AddExpr(stmt.Value)
	}
}

func TestParseScope(t *testing.T) {
	// vars registers all bound and unbound variables per scope. Unbound variables are not defined in the scope and are either defined in a parent scope or in global. Bound variables are variables that are defined in this scope. Divided by | on the left are bound vars and on the right unbound. Each scope is separated by /, and the variables are separated by a comma.
	// var and function declarations are function-scoped
	// const, let, and class declarations are block-scoped
	// unbound variables are registered at function-scope, not for every block-scope!
	var tests = []struct {
		js                   string
		bound, uses, unbound string
	}{
		{"var a; b;", "a", "b", "b"},
		{"var {a:b, c=d, ...e};", "b,c,e", "d", "d"},
		{"var [a, b=c, ...d];", "a,b,d", "c", "c"},
		{"x={a:b, c=d, ...e};", "", "b,c,d,e,x", "b,c,d,e,x"},
		{"x=[a, b=c, ...d];", "", "a,b,c,d,x", "a,b,c,d,x"},
		{"yield = 5", "", "yield", "yield"},
		{"await = 5", "", "await", "await"},
		{"function a(b,c){var d; e = 5; a}", "a/b,c,d", "e/a,e", "e"},
		{"!function a(b,c){var d; e = 5; a}", "/a,b,c,d", "e/e", "e"},
		{"function a(b=c){var c}", "a/b,c", "c/c", "c"},
		{"class a{b(){}}", "a/", "/", ""}, // classes are not tracked
		{"!class a{b(){}}", "/", "/", ""},
		{"a => a%5", "/a", "/", ""},
		{"a => a%b", "/a", "b/b", "b"},
		{"(a) + (a)", "", "a", "a"},
		{"(a,b)", "", "a,b", "a,b"},
		{"(a,b) + (a,b)", "", "a,b", "a,b"},
		{"(a) + (a => a%5)", "/a", "a/", "a"},
		{"(a=b) => {var c; d = 5}", "/a,c", "b,d/b,d", "b,d"},
		{"(a,b=a) => {}", "/a,b", "/", ""},
		{"(a=b) => {var b}", "/a,b", "b/b", "b"},
		{"({[a+b]:c}) => {}", "/c", "a,b/a,b", "a,b"},
		{"({a:b, c=d, ...e}=f) => 5", "/b,c,e", "d,f/d,f", "d,f"},
		{"([a, b=c, ...d]=e) => 5", "/a,b,d", "c,e/c,e", "c,e"},
		{"(a) + ((b,c) => {var d; e = 5; return e})", "/b,c,d", "a,e/e", "a,e"},
		{"(a) + ((a,b) => {var c; d = 5; return d})", "/a,b,c", "a,d/d", "a,d"},
		{"{(a) + ((a,b) => {var c; d = 5; return d})}", "//a,b,c", "a,d/a,d/d", "a,d"},
		{"(a=(b=>b/a)) => a", "/a/b", "//a", ""},
		{"(a=(b=>b/c)) => a", "/a/b", "c/c/c", "c"},
		{"(a=(function b(){})) => a", "/a/b", "//", ""},
		{"label: a", "", "a", "a"},
		{"yield => yield%5", "/yield", "/", ""},
		{"await => await%5", "/await", "/", ""},
		{"function*a(){b => yield%5}", "a//b", "yield/yield/yield", "yield"},
		{"async function a(){b => await%5}", "a//b", "await/await/await", "await"},
		{"let a; {let b = a}", "a/b", "/a", ""},
		{"let a; {var b = a}", "a,b/", "/a,b", ""},
		{"let a; {class b{}}", "a/b", "/", ""},
		{"a = 5; var a;", "a", "", ""},
		{"a = 5; let a;", "a", "", ""},
		{"a = 5; {var a}", "a/", "/a", ""},
		{"a = 5; {let a}", "/a", "a/", "a"},
		{"{a = 5} var a", "a/", "/a", ""},
		{"{a = 5} let a", "a/", "/a", ""},
		{"var a; {a = 5}", "a/", "/a", ""},
		{"let a; {a = 5}", "a/", "/a", ""},
		{"{var a} a = 5", "a/", "/a", ""},
		{"{let a} a = 5", "/a", "a/", "a"},
		{"!function(){throw new Error()}", "/", "Error/Error", "Error"},
		{"function(){return a}", "/", "a/a", "a"},
		{"function(){return a} var a;", "a/", "/a", ""},
		{"function(){return a} if(5){var a}", "a//", "/a/a", ""},
		{"try{}catch(a){let b; c}", "/a,b", "c/c", "c"},
		{"try{}catch(a){var b; c}", "b/a", "c/b,c", "c"},
		{"var a;try{}catch(a){var a}", "a/a", "/a", ""},
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(parse.NewInputString(tt.js))
			if err != io.EOF {
				test.Error(t, err)
			}

			vars := NewScopeVars(ast.Undeclared)
			vars.AddScope(ast.Scope)
			for _, istmt := range ast.List {
				vars.AddStmt(istmt)
			}
			test.String(t, vars.String(), "bound:"+tt.bound+" uses:"+tt.uses+" unbound:"+tt.unbound)
		})
	}
}

func TestParseRef(t *testing.T) {
	var tests = []struct {
		js   string
		refs string
	}{
		{"a; a", "a=1"},
		{"var a; {var a}", "a=1"},
		{"var a; {let a}", "a=1,a=2"},
		{"function a(b,c=b){}", "a=1,b=2,c=3"},
		{"function a(b=c,c){}", "a=1,b=2,c=3,c=4"},
		{"function a(b=c){var c}", "a=1,b=2,c=3,c=4"},
		{"function a(b){var b}", "a=1,b=2"},
		{"function a(b,b){}", "a=1,b=2"},
		{"a=function(b,c=b){}", "a=1,b=2,c=3"},
		{"a=function(b=c,c){}", "a=1,b=2,c=3,c=4"},
		{"a=function(b=c){var c}", "a=1,b=2,c=3,c=4"},
		{"a=function(b){var b}", "a=1,b=2"},
		{"a=function(b,b){}", "a=1,b=2"},
		{"(a) + (a)", "a=1"},
		{"(b,c=b)=>{}", "b=1,c=2"},
		{"(b=c,c)=>{}", "b=1,c=2,c=3"},
		{"(b=c)=>{var c}", "b=1,c=2,c=3"},
		{"a=>{var a}", "a=1"},
		{"(b,b)=>{}", "b=1"},
		{"try{}catch(a){var a}", "a=1,a=2"},
		{"var a;try{}catch(a){a}", "a=1,a=2"},
		{"{a} {a} var a", "a=1,a=1"},                // second block must add a new var in case the block contains a var decl
		{"(a),(a)", "a=1,a=1"},                      // second parens could have been arrow function, so must have added new var
		{"var a,b,c;(a = b[c])", "a=1,b=2,c=3,a=1"}, // parens could have been arrow function, so must have added new var
	}
	for _, tt := range tests {
		t.Run(tt.js, func(t *testing.T) {
			ast, err := Parse(parse.NewInputString(tt.js))
			if err != io.EOF {
				test.Error(t, err)
			}

			s := ""
			for _, v := range ast.Vars[1:] {
				if 0 < v.Uses {
					if len(s) != 0 {
						s += ","
					}
					s += fmt.Sprintf("%s=%d", string(v.Name), v.Ref)
				}
			}
			test.String(t, s, tt.refs)
		})
	}
}

func TestScope(t *testing.T) {
	js := "let a,b; b = 5; var c;"
	ast, err := Parse(parse.NewInputString(js))
	if err != io.EOF {
		test.Error(t, err)
	}
	scope := ast.Scope

	// test count
	test.T(t, scope.Count(LexicalDecl), 2)

	// test VarRef
	a := scope.Declared[0]
	test.T(t, a, a.Ref.Var(ast))
	test.T(t, a.Name, a.Ref.Name(ast))
	test.T(t, string(a.Name), a.Ref.String(ast))

	// test output
	test.T(t, scope.String(), "Scope{Declared: [Var{1 1 LexicalDecl a}, Var{2 2 LexicalDecl b}, Var{3 1 VariableDecl c}], Undeclared: []}")

	// test sort
	sort.Sort(scope.Declared)
	test.T(t, scope.String(), "Scope{Declared: [Var{2 2 LexicalDecl b}, Var{1 1 LexicalDecl a}, Var{3 1 VariableDecl c}], Undeclared: []}")
}

func TestParseInputError(t *testing.T) {
	_, err := Parse(parse.NewInput(test.NewErrorReader(0)))
	test.T(t, err, test.ErrPlain)

	_, err = Parse(parse.NewInput(test.NewErrorReader(1)))
	test.T(t, err, test.ErrPlain)
}
