package js

type OpPrec int

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Operator_Precedence
const (
	OpEnd OpPrec = iota
	OpComma
	OpYield
	OpAssign
	OpCond
	OpOr
	OpAnd
	OpNullish
	OpBitOr
	OpBitXor
	OpBitAnd
	OpEquals
	OpCompare
	OpShift
	OpAdd
	OpMul
	OpExp
	OpPrefix
	OpPostfix
	OpNew
	OpCall
	OpGroup
)

var Keywords = map[string]TokenType{
	"async":      AsyncToken,
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
	"implements": ImplementsToken,
	"import":     ImportToken,
	"in":         InToken,
	"instanceof": InstanceofToken,
	"interface":  InterfaceToken,
	"let":        LetToken,
	"new":        NewToken,
	"null":       NullToken,
	"package":    PackageToken,
	"private":    PrivateToken,
	"protected":  ProtectedToken,
	"public":     PublicToken,
	"return":     ReturnToken,
	"static":     StaticToken,
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
}
