package js

import "strconv"

type OpPrec int

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Operator_Precedence
const (
	OpEnd OpPrec = iota
	OpComma
	OpYield
	OpAssign
	OpCond
	OpNullish
	OpOr
	OpAnd
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
	OpLiteral
)

func (prec OpPrec) String() string {
	switch prec {
	case OpEnd:
		return "OpEnd"
	case OpComma:
		return "OpComma"
	case OpYield:
		return "OpYield"
	case OpAssign:
		return "OpAssign"
	case OpCond:
		return "OpCond"
	case OpNullish:
		return "OpNullish"
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
	case OpShift:
		return "OpShift"
	case OpAdd:
		return "OAdd"
	case OpMul:
		return "OpMul"
	case OpExp:
		return "OpExp"
	case OpPrefix:
		return "OpPrefix"
	case OpPostfix:
		return "OpPostfix"
	case OpNew:
		return "OpNew"
	case OpCall:
		return "OpCall"
	case OpGroup:
		return "OpGroup"
	case OpLiteral:
		return "OpLiteral"
	}
	return "Invalid(" + strconv.Itoa(int(prec)) + ")"
}

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
