package js

import "strconv"

type OpPrec int

const (
	OpExpr     OpPrec = iota // a,b
	OpAssign                 // a?b:c, yield x, ()=>x, async ()=>x, a=b, a+=b, ...
	OpCoalesce               // a??b
	OpOr                     // a||b
	OpAnd                    // a&&b
	OpBitOr                  // a|b
	OpBitXor                 // a^b
	OpBitAnd                 // a&b
	OpEquals                 // a==b, a!=b, a===b, a!==b
	OpCompare                // a<b, a>b, a<=b, a>=b, a instanceof b, a in b
	OpShift                  // a<<b, a>>b, a>>>b
	OpAdd                    // a+b, a-b
	OpMul                    // a*b, a/b, a%b
	OpExp                    // a**b
	OpUnary                  // ++x, --x, delete x, void x, typeof x, +x, -x, ~x, !x, await x
	OpUpdate                 // x++, x--
	OpLHS                    // a?.b
	OpCall                   // a(b), super(a), import(a)
	OpNew                    // new a
	OpMember                 // a[b], a.b, a`b`, super[x], super.x, new.target, import.meta, new a(b)
	OpPrimary                // literal, function, class, parenthesized
)

func (prec OpPrec) String() string {
	switch prec {
	case OpExpr:
		return "OpExpr"
	case OpAssign:
		return "OpAssign"
	case OpCoalesce:
		return "OpCoalesce"
	case OpOr:
		return "OpOr"
	case OpAnd:
		return "OpAnd"
	case OpBitOr:
		return "OpBitOr"
	case OpBitXor:
		return "OpBitXor"
	case OpBitAnd:
		return "OpBitAnd"
	case OpEquals:
		return "OpEquals"
	case OpCompare:
		return "OpCompare"
	case OpShift:
		return "OpShift"
	case OpAdd:
		return "OAdd"
	case OpMul:
		return "OpMul"
	case OpExp:
		return "OpExp"
	case OpUnary:
		return "OpUnary"
	case OpUpdate:
		return "OpUpdate"
	case OpLHS:
		return "OpLHS"
	case OpCall:
		return "OpCall"
	case OpNew:
		return "OpNew"
	case OpMember:
		return "OpMember"
	case OpPrimary:
		return "OpPrimary"
	}
	return "Invalid(" + strconv.Itoa(int(prec)) + ")"
}

var Keywords = map[string]TokenType{
	// reserved
	"await":      AwaitToken,
	"break":      BreakToken,
	"case":       CaseToken,
	"catch":      CatchToken,
	"class":      ClassToken,
	"const":      ConstToken,
	"continue":   ContinueToken,
	"debugger":   DebuggerToken,
	"default":    DefaultToken,
	"delete":     DeleteToken,
	"do":         DoToken,
	"else":       ElseToken,
	"enum":       EnumToken,
	"export":     ExportToken,
	"extends":    ExtendsToken,
	"false":      FalseToken,
	"finally":    FinallyToken,
	"for":        ForToken,
	"function":   FunctionToken,
	"if":         IfToken,
	"import":     ImportToken,
	"in":         InToken,
	"instanceof": InstanceofToken,
	"new":        NewToken,
	"null":       NullToken,
	"return":     ReturnToken,
	"super":      SuperToken,
	"switch":     SwitchToken,
	"this":       ThisToken,
	"throw":      ThrowToken,
	"true":       TrueToken,
	"try":        TryToken,
	"typeof":     TypeofToken,
	"var":        VarToken,
	"void":       VoidToken,
	"while":      WhileToken,
	"with":       WithToken,
	"yield":      YieldToken,

	// strict mode
	"let":        LetToken,
	"static":     StaticToken,
	"implements": ImplementsToken,
	"interface":  InterfaceToken,
	"package":    PackageToken,
	"private":    PrivateToken,
	"protected":  ProtectedToken,
	"public":     PublicToken,

	// extra
	"as":     AsToken,
	"async":  AsyncToken,
	"from":   FromToken,
	"get":    GetToken,
	"meta":   MetaToken,
	"of":     OfToken,
	"set":    SetToken,
	"target": TargetToken,
}

var Globals = map[string]struct{}{
	"Infinity":           struct{}{},
	"NaN":                struct{}{},
	"undefined":          struct{}{},
	"globalThis":         struct{}{},
	"eval":               struct{}{},
	"isFinite":           struct{}{},
	"isNaN":              struct{}{},
	"parseFloat":         struct{}{},
	"parseInt":           struct{}{},
	"decodeURI":          struct{}{},
	"decodeURIComponent": struct{}{},
	"encodeURI":          struct{}{},
	"encodeURIComponent": struct{}{},
	"Object":             struct{}{},
	"Function":           struct{}{},
	"Boolean":            struct{}{},
	"Symbol":             struct{}{},
	"Error":              struct{}{},
	"AggregateError":     struct{}{},
	"EvalError":          struct{}{},
	"InternalError":      struct{}{},
	"RangeError":         struct{}{},
	"ReferenceError":     struct{}{},
	"SyntaxError":        struct{}{},
	"TypeError":          struct{}{},
	"URIError":           struct{}{},
	"Number":             struct{}{},
	"BigInt":             struct{}{},
	"Math":               struct{}{},
	"Date":               struct{}{},
	"String":             struct{}{},
	"RegExp":             struct{}{},
	"Array":              struct{}{},
	"Int8Array":          struct{}{},
	"Uint8Array":         struct{}{},
	"Uint8ClampedArray":  struct{}{},
	"Int16Array":         struct{}{},
	"Uint16Array":        struct{}{},
	"Int32Array":         struct{}{},
	"Uint32Array":        struct{}{},
	"Float32Array":       struct{}{},
	"Float64Array":       struct{}{},
	"BigInt64Array":      struct{}{},
	"BigUint64Array":     struct{}{},
	"Map":                struct{}{},
	"Set":                struct{}{},
	"WeakMap":            struct{}{},
	"WeakSet":            struct{}{},
	"ArrayBuffer":        struct{}{},
	"SharedArrayBuffer":  struct{}{},
	"Atomics":            struct{}{},
	"DataView":           struct{}{},
	"JSON":               struct{}{},
	"Promise":            struct{}{},
	"Generator":          struct{}{},
	"GeneratorFunction":  struct{}{},
	"AsyncFunction":      struct{}{},
	"Reflect":            struct{}{},
	"Proxy":              struct{}{},
	"Intl":               struct{}{},
	"WebAssembly":        struct{}{},
	"arguments":          struct{}{},
	"escape":             struct{}{},
	"unescape":           struct{}{},

	// DOM
	"document": struct{}{},
	"window":   struct{}{},
	// TODO: setTimeout etc
}
