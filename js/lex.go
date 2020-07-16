// Package js is an ECMAScript5.1 lexer following the specifications at http://www.ecma-international.org/ecma-262/5.1/.
package js

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
)

var identifierStart = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Other_ID_Start}
var identifierContinue = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue}

// IsIdentifierStart returns true if the byte-slice start is a continuation of an identifier
func IsIdentifierStart(b []byte) bool {
	r, _ := utf8.DecodeRune(b)
	return r == '$' || r == '\\' || r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierContinue, r)
}

// IsIdentifierEnd returns true if the byte-slice end is a start or continuation of an identifier
func IsIdentifierEnd(b []byte) bool {
	r, _ := utf8.DecodeLastRune(b)
	return r == '$' || r == '\\' || r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierContinue, r)
}

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint16 // from LSB to MSB: 8 bits for tokens per category, 1 bit for numeric, 1 bit for punctuator, 1 bit for operator, 1 bit for identifier, 4 bits unused

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	WhitespaceToken
	LineTerminatorToken // \r \n \r\n
	CommentToken
	CommentLineTerminatorToken
	StringToken
	TemplateToken
	TemplateStartToken
	TemplateMiddleToken
	TemplateEndToken
	RegExpToken
)

const (
	NumericToken TokenType = 0x0100 + iota
	DecimalToken
	BinaryToken
	OctalToken
	HexadecimalToken
	BigIntToken
)

const (
	PunctuatorToken   TokenType = 0x0200 + iota
	OpenBraceToken              // {
	CloseBraceToken             // }
	OpenParenToken              // (
	CloseParenToken             // )
	OpenBracketToken            // [
	CloseBracketToken           // ]
	DotToken                    // .
	SemicolonToken              // ;
	CommaToken                  // ,
	QuestionToken               // ?
	ColonToken                  // :
	ArrowToken                  // =>
	EllipsisToken               // ...
)

const (
	OperatorToken TokenType = 0x0600 + iota
	EqToken                 // =
	EqEqToken               // ==
	EqEqEqToken             // ===
	NotToken                // !
	NotEqToken              // !=
	NotEqEqToken            // !==
	LtToken                 // <
	LtEqToken               // <=
	LtLtToken               // <<
	LtLtEqToken             // <<=
	GtToken                 // >
	GtEqToken               // >=
	GtGtToken               // >>
	GtGtEqToken             // >>=
	GtGtGtToken             // >>>
	GtGtGtEqToken           // >>>=
	AddToken                // +
	AddEqToken              // +=
	IncrToken               // ++
	SubToken                // -
	SubEqToken              // -=
	DecrToken               // --
	MulToken                // *
	MulEqToken              // *=
	ExpToken                // **
	ExpEqToken              // **=
	DivToken                // /
	DivEqToken              // /=
	ModToken                // %
	ModEqToken              // %=
	BitAndToken             // &
	BitOrToken              // |
	BitXorToken             // ^
	BitNotToken             // ~
	BitAndEqToken           // &=
	BitOrEqToken            // |=
	BitXorEqToken           // ^=
	AndToken                // &&
	OrToken                 // ||
	NullishToken            // ??
	OptChainToken           // ?.

	// unused in lexer
	PosToken      // +a
	NegToken      // -a
	PreIncrToken  // ++a
	PreDecrToken  // --a
	PostIncrToken // a++
	PostDecrToken // a--
)

const (
	ReservedToken TokenType = 0x0800 + iota
	AwaitToken
	BreakToken
	CaseToken
	CatchToken
	ClassToken
	ConstToken
	ContinueToken
	DebuggerToken
	DefaultToken
	DeleteToken
	DoToken
	ElseToken
	EnumToken
	ExportToken
	ExtendsToken
	FalseToken
	FinallyToken
	ForToken
	FunctionToken
	IfToken
	ImportToken
	InToken
	InstanceofToken
	NewToken
	NullToken
	ReturnToken
	SuperToken
	SwitchToken
	ThisToken
	ThrowToken
	TrueToken
	TryToken
	TypeofToken
	VarToken
	VoidToken
	WhileToken
	WithToken
	YieldToken
)

const (
	IdentifierToken TokenType = 0x1000 + iota
	AsyncToken
	ImplementsToken
	InterfaceToken
	LetToken
	PackageToken
	PrivateToken
	ProtectedToken
	PublicToken
	StaticToken
	OfToken
	GetToken
	SetToken
	TargetToken
	MetaToken
	AsToken
	FromToken
)

func IsNumeric(tt TokenType) bool {
	return tt&0x0100 != 0
}

func IsPunctuator(tt TokenType) bool {
	return tt&0x0200 != 0
}

func IsOperator(tt TokenType) bool {
	return tt&0x0400 != 0
}

// IsIdentifierName matches IdentifierName, i.e. any identifier
func IsIdentifierName(tt TokenType) bool {
	return tt&0x1800 != 0
}

// IsReservedWord matches ReservedWord
func IsReservedWord(tt TokenType) bool {
	return tt&0x0800 != 0
}

// IsIdentifier matches Identifier, i.e. IdentifierName but not ReservedWord. Does not match yield or await.
func IsIdentifier(tt TokenType) bool {
	return tt&0x1000 != 0
}

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	s := tt.Bytes()
	if s == nil {
		return "Invalid(" + strconv.Itoa(int(tt)) + ")"
	}
	return string(s)
}

var opBytes = [][]byte{
	[]byte("Operator"),
	[]byte("="),
	[]byte("=="),
	[]byte("==="),
	[]byte("!"),
	[]byte("!="),
	[]byte("!=="),
	[]byte("<"),
	[]byte("<="),
	[]byte("<<"),
	[]byte("<<="),
	[]byte(">"),
	[]byte(">="),
	[]byte(">>"),
	[]byte(">>="),
	[]byte(">>>"),
	[]byte(">>>="),
	[]byte("+"),
	[]byte("+="),
	[]byte("++"),
	[]byte("-"),
	[]byte("-="),
	[]byte("--"),
	[]byte("*"),
	[]byte("*="),
	[]byte("**"),
	[]byte("**="),
	[]byte("/"),
	[]byte("/="),
	[]byte("%"),
	[]byte("%="),
	[]byte("&"),
	[]byte("|"),
	[]byte("^"),
	[]byte("~"),
	[]byte("&="),
	[]byte("|="),
	[]byte("^="),
	[]byte("&&"),
	[]byte("||"),
	[]byte("??"),
	[]byte("?."),
	[]byte("+"),
	[]byte("-"),
	[]byte("++"),
	[]byte("--"),
	[]byte("++"),
	[]byte("--"),
}

// Bytes returns the string representation of a TokenType.
func (tt TokenType) Bytes() []byte {
	if IsOperator(tt) && tt <= PostDecrToken {
		return opBytes[tt-OperatorToken]
	}

	switch tt {
	case ErrorToken:
		return []byte("Error")
	case WhitespaceToken:
		return []byte("Whitespace")
	case LineTerminatorToken:
		return []byte("LineTerminator")
	case CommentToken:
		return []byte("Comment")
	case CommentLineTerminatorToken:
		return []byte("CommentLineTerminator")
	case StringToken:
		return []byte("String")
	case TemplateToken:
		return []byte("Template")
	case TemplateStartToken:
		return []byte("TemplateStart")
	case TemplateMiddleToken:
		return []byte("TemplateMiddle")
	case TemplateEndToken:
		return []byte("TemplateEnd")
	case RegExpToken:
		return []byte("RegExp")
	case NumericToken:
		return []byte("Numeric")
	case DecimalToken:
		return []byte("Decimal")
	case BinaryToken:
		return []byte("Binary")
	case OctalToken:
		return []byte("Octal")
	case HexadecimalToken:
		return []byte("Hexadecimal")
	case BigIntToken:
		return []byte("BigInt")
	case PunctuatorToken:
		return []byte("Punctuator")
	case OpenBraceToken:
		return []byte("{")
	case CloseBraceToken:
		return []byte("}")
	case OpenParenToken:
		return []byte("(")
	case CloseParenToken:
		return []byte(")")
	case OpenBracketToken:
		return []byte("[")
	case CloseBracketToken:
		return []byte("]")
	case DotToken:
		return []byte(".")
	case SemicolonToken:
		return []byte(";")
	case CommaToken:
		return []byte(",")
	case QuestionToken:
		return []byte("?")
	case ColonToken:
		return []byte(":")
	case ArrowToken:
		return []byte("=>")
	case EllipsisToken:
		return []byte("...")
	case IdentifierToken:
		return []byte("Identifier")
	case AwaitToken:
		return []byte("await")
	case BreakToken:
		return []byte("break")
	case CaseToken:
		return []byte("case")
	case CatchToken:
		return []byte("catch")
	case ClassToken:
		return []byte("class")
	case ConstToken:
		return []byte("const")
	case ContinueToken:
		return []byte("continue")
	case DebuggerToken:
		return []byte("debugger")
	case DefaultToken:
		return []byte("default")
	case DeleteToken:
		return []byte("delete")
	case DoToken:
		return []byte("do")
	case ElseToken:
		return []byte("else")
	case EnumToken:
		return []byte("enum")
	case ExportToken:
		return []byte("export")
	case ExtendsToken:
		return []byte("extends")
	case FalseToken:
		return []byte("false")
	case FinallyToken:
		return []byte("finally")
	case ForToken:
		return []byte("for")
	case FunctionToken:
		return []byte("function")
	case IfToken:
		return []byte("if")
	case ImportToken:
		return []byte("import")
	case InToken:
		return []byte("in")
	case InstanceofToken:
		return []byte("instanceof")
	case NewToken:
		return []byte("new")
	case NullToken:
		return []byte("null")
	case ReturnToken:
		return []byte("return")
	case SuperToken:
		return []byte("super")
	case SwitchToken:
		return []byte("switch")
	case ThisToken:
		return []byte("this")
	case ThrowToken:
		return []byte("throw")
	case TrueToken:
		return []byte("true")
	case TryToken:
		return []byte("try")
	case TypeofToken:
		return []byte("typeof")
	case VarToken:
		return []byte("var")
	case VoidToken:
		return []byte("void")
	case WhileToken:
		return []byte("while")
	case WithToken:
		return []byte("with")
	case YieldToken:
		return []byte("yield")
	case LetToken:
		return []byte("let")
	case StaticToken:
		return []byte("static")
	case ImplementsToken:
		return []byte("implements")
	case InterfaceToken:
		return []byte("interface")
	case PackageToken:
		return []byte("package")
	case PrivateToken:
		return []byte("private")
	case ProtectedToken:
		return []byte("protected")
	case PublicToken:
		return []byte("public")
	case AsToken:
		return []byte("as")
	case AsyncToken:
		return []byte("async")
	case FromToken:
		return []byte("from")
	case GetToken:
		return []byte("get")
	case MetaToken:
		return []byte("meta")
	case OfToken:
		return []byte("of")
	case SetToken:
		return []byte("set")
	case TargetToken:
		return []byte("target")
	}
	return nil
}

////////////////////////////////////////////////////////////////

// Lexer is the state for the lexer.
type Lexer struct {
	r                  *parse.Input
	err                error
	prevLineTerminator bool
	level              int
	templateLevels     []int
}

// NewLexer returns a new Lexer for a given io.Reader.
func NewLexer(r *parse.Input) *Lexer {
	return &Lexer{
		r:                  r,
		prevLineTerminator: true,
		level:              0,
		templateLevels:     []int{},
	}
}

// Err returns the error encountered during lexing, this is often io.EOF but also other errors can be returned.
func (l *Lexer) Err() error {
	if l.err != nil {
		return l.err
	}
	return l.r.Err()
}

// RegExp reparses the input stream for a regular expression. It is assumed that we just received DivToken or DivEqToken with Next(). This function will go back and read that as a regular expression.
func (l *Lexer) RegExp() (TokenType, []byte) {
	if 0 < l.r.Offset() && l.r.Peek(-1) == '/' {
		l.r.Move(-1)
	} else if 1 < l.r.Offset() && l.r.Peek(-1) == '=' && l.r.Peek(-2) == '/' {
		l.r.Move(-2)
	} else {
		l.err = parse.NewErrorLexer(l.r, "expected '/' or '/='")
		return ErrorToken, nil
	}
	l.r.Skip() // trick to set start = pos

	if l.consumeRegExpToken() {
		return RegExpToken, l.r.Shift()
	}
	l.err = parse.NewErrorLexer(l.r, "unexpected EOF or newline")
	return ErrorToken, nil
}

var counts [256]int

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (l *Lexer) Next() (TokenType, []byte) {
	prevLineTerminator := l.prevLineTerminator
	l.prevLineTerminator = false

	c := l.r.Peek(0)
	//if identifierStartTable[c] {
	//	tt := l.consumeIdentifierToken()
	//	return tt, l.r.Shift()
	//}
	switch c {
	case ' ', '\t', '\v', '\f':
		l.r.Move(1)
		for l.consumeWhitespaceByte() || l.consumeWhitespaceRune() {
		}
		l.prevLineTerminator = prevLineTerminator
		return WhitespaceToken, l.r.Shift()
	case '\n', '\r':
		l.r.Move(1)
		for l.consumeLineTerminator() {
		}
		l.prevLineTerminator = true
		return LineTerminatorToken, l.r.Shift()
	case '>', '=', '!', '+', '*', '%', '&', '|', '^', '?':
		if tt := l.consumeOperatorToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		if tt := l.consumeNumericToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		} else if c == '.' {
			l.r.Move(1)
			if l.r.Peek(0) == '.' && l.r.Peek(1) == '.' {
				l.r.Move(2)
				return EllipsisToken, l.r.Shift()
			}
			return DotToken, l.r.Shift()
		}
	case ',':
		l.r.Move(1)
		return CommaToken, l.r.Shift()
	case ';':
		l.r.Move(1)
		return SemicolonToken, l.r.Shift()
	case '(':
		l.level++
		l.r.Move(1)
		return OpenParenToken, l.r.Shift()
	case ')':
		l.level--
		l.r.Move(1)
		return CloseParenToken, l.r.Shift()
	case '/':
		if tt := l.consumeCommentToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		} else if tt := l.consumeOperatorToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		}
	case '{':
		l.level++
		l.r.Move(1)
		return OpenBraceToken, l.r.Shift()
	case '}':
		l.level--
		if len(l.templateLevels) != 0 && l.level == l.templateLevels[len(l.templateLevels)-1] {
			return l.consumeTemplateToken(), l.r.Shift()
		}
		l.r.Move(1)
		return CloseBraceToken, l.r.Shift()
	case ':':
		l.r.Move(1)
		return ColonToken, l.r.Shift()
	case '\'', '"':
		if l.consumeStringToken() {
			return StringToken, l.r.Shift()
		}
	case ']':
		l.r.Move(1)
		return CloseBracketToken, l.r.Shift()
	case '[':
		l.r.Move(1)
		return OpenBracketToken, l.r.Shift()
	case '<', '-':
		if l.consumeHTMLLikeCommentToken(prevLineTerminator) {
			return CommentToken, l.r.Shift()
		} else if tt := l.consumeOperatorToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		}
	case '~':
		l.r.Move(1)
		return BitNotToken, l.r.Shift()
	case '`':
		l.templateLevels = append(l.templateLevels, l.level)
		return l.consumeTemplateToken(), l.r.Shift()
	default:
		if tt := l.consumeIdentifierToken(); tt != ErrorToken {
			return tt, l.r.Shift()
		} else if c >= 0xC0 {
			if l.consumeWhitespaceByte() || l.consumeWhitespaceRune() {
				for l.consumeWhitespaceByte() || l.consumeWhitespaceRune() {
				}
				l.prevLineTerminator = prevLineTerminator
				return WhitespaceToken, l.r.Shift()
			} else if l.consumeLineTerminator() {
				for l.consumeLineTerminator() {
				}
				l.prevLineTerminator = true
				return LineTerminatorToken, l.r.Shift()
			}
		} else if c == 0 && l.r.Err() != nil {
			return ErrorToken, nil
		}
	}

	if r, _ := l.r.PeekRune(0); unicode.IsGraphic(r) {
		l.err = parse.NewErrorLexer(l.r, "unexpected '%c'", r)
	} else if r < 128 {
		l.err = parse.NewErrorLexer(l.r, "unexpected 0x%02X", c)
	} else {
		l.err = parse.NewErrorLexer(l.r, "unexpected %U", r)
	}
	return ErrorToken, l.r.Shift()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.ecma-international.org/ecma-262/5.1/
*/

func (l *Lexer) consumeWhitespaceByte() bool {
	c := l.r.Peek(0)
	if c == ' ' || c == '\t' || c == '\v' || c == '\f' {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeWhitespaceRune() bool {
	c := l.r.Peek(0)
	if c >= 0xC0 {
		if r, n := l.r.PeekRune(0); r == '\u00A0' || r == '\uFEFF' || unicode.Is(unicode.Zs, r) {
			l.r.Move(n)
			return true
		}
	}
	return false
}

func (l *Lexer) isLineTerminator() bool {
	c := l.r.Peek(0)
	if c == '\n' || c == '\r' {
		return true
	} else if c == 0xE2 && l.r.Peek(1) == 0x80 && (l.r.Peek(2) == 0xA8 || l.r.Peek(2) == 0xA9) {
		return true
	}
	return false
}

func (l *Lexer) consumeLineTerminator() bool {
	c := l.r.Peek(0)
	if c == '\n' {
		l.r.Move(1)
		return true
	} else if c == '\r' {
		if l.r.Peek(1) == '\n' {
			l.r.Move(2)
		} else {
			l.r.Move(1)
		}
		return true
	} else if c >= 0xC0 {
		if r, n := l.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
			l.r.Move(n)
			return true
		}
	}
	return false
}

func (l *Lexer) consumeDigit() bool {
	if c := l.r.Peek(0); c >= '0' && c <= '9' {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeHexDigit() bool {
	if c := l.r.Peek(0); (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeBinaryDigit() bool {
	if c := l.r.Peek(0); c == '0' || c == '1' {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeOctalDigit() bool {
	if c := l.r.Peek(0); c >= '0' && c <= '7' {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeUnicodeEscape() bool {
	if l.r.Peek(0) != '\\' || l.r.Peek(1) != 'u' {
		return false
	}
	mark := l.r.Pos()
	l.r.Move(2)
	if c := l.r.Peek(0); c == '{' {
		l.r.Move(1)
		if l.consumeHexDigit() {
			for l.consumeHexDigit() {
			}
			if c := l.r.Peek(0); c == '}' {
				l.r.Move(1)
				return true
			}
		}
		l.r.Rewind(mark)
		return false
	} else if !l.consumeHexDigit() || !l.consumeHexDigit() || !l.consumeHexDigit() || !l.consumeHexDigit() {
		l.r.Rewind(mark)
		return false
	}
	return true
}

func (l *Lexer) consumeSingleLineComment() {
	for {
		c := l.r.Peek(0)
		if c == '\r' || c == '\n' || c == 0 && l.r.Err() != nil {
			break
		} else if c >= 0xC0 {
			if r, _ := l.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
				break
			}
		}
		l.r.Move(1)
	}
}

////////////////////////////////////////////////////////////////

func (l *Lexer) consumeHTMLLikeCommentToken(prevLineTerminator bool) bool {
	c := l.r.Peek(0)
	if c == '<' && l.r.Peek(1) == '!' && l.r.Peek(2) == '-' && l.r.Peek(3) == '-' {
		// opening HTML-style single line comment
		l.r.Move(4)
		l.consumeSingleLineComment()
		return true
	} else if prevLineTerminator && c == '-' && l.r.Peek(1) == '-' && l.r.Peek(2) == '>' {
		// closing HTML-style single line comment
		// (only if current line didn't contain any meaningful tokens)
		l.r.Move(3)
		l.consumeSingleLineComment()
		return true
	}
	return false
}

func (l *Lexer) consumeCommentToken() TokenType {
	c := l.r.Peek(1)
	if c == '/' {
		// single line comment
		l.r.Move(2)
		l.consumeSingleLineComment()
		return CommentToken
	} else if c == '*' {
		l.r.Move(2)
		tt := CommentToken
		for {
			c := l.r.Peek(0)
			if c == '*' && l.r.Peek(1) == '/' {
				l.r.Move(2)
				break
			} else if c == 0 && l.r.Err() != nil {
				break
			} else if l.consumeLineTerminator() {
				l.prevLineTerminator = true
				tt = CommentLineTerminatorToken
			} else {
				l.r.Move(1)
			}
		}
		return tt
	}
	return ErrorToken
}

var opTokens = map[byte]TokenType{
	'=': EqToken,
	'!': NotToken,
	'<': LtToken,
	'>': GtToken,
	'+': AddToken,
	'-': SubToken,
	'*': MulToken,
	'/': DivToken,
	'%': ModToken,
	'&': BitAndToken,
	'|': BitOrToken,
	'^': BitXorToken,
	'?': QuestionToken,
}

var opEqTokens = map[byte]TokenType{
	'=': EqEqToken,
	'!': NotEqToken,
	'<': LtEqToken,
	'>': GtEqToken,
	'+': AddEqToken,
	'-': SubEqToken,
	'*': MulEqToken,
	'/': DivEqToken,
	'%': ModEqToken,
	'&': BitAndEqToken,
	'|': BitOrEqToken,
	'^': BitXorEqToken,
}

var opOpTokens = map[byte]TokenType{
	'+': IncrToken,
	'-': DecrToken,
	'*': ExpToken,
	'&': AndToken,
	'|': OrToken,
	'?': NullishToken,
}

func (l *Lexer) consumeOperatorToken() TokenType {
	c := l.r.Peek(0)
	l.r.Move(1)
	if l.r.Peek(0) == '=' {
		l.r.Move(1)
		if l.r.Peek(0) == '=' && (c == '!' || c == '=') {
			l.r.Move(1)
			if c == '!' {
				return NotEqEqToken
			}
			return EqEqEqToken
		}
		return opEqTokens[c]
	} else if l.r.Peek(0) == c && (c == '+' || c == '-' || c == '*' || c == '&' || c == '|' || c == '?') {
		l.r.Move(1)
		if c == '*' && l.r.Peek(0) == '=' {
			l.r.Move(1)
			return ExpEqToken
		}
		return opOpTokens[c]
	} else if c == '?' && l.r.Peek(0) == '.' && (l.r.Peek(1) < '0' || l.r.Peek(1) > '9') {
		l.r.Move(1)
		return OptChainToken
	} else if c == '=' && l.r.Peek(0) == '>' {
		l.r.Move(1)
		return ArrowToken
	} else if c == '<' && l.r.Peek(0) == '<' {
		l.r.Move(1)
		if l.r.Peek(0) == '=' {
			l.r.Move(1)
			return LtLtEqToken
		}
		return LtLtToken
	} else if c == '>' && l.r.Peek(0) == '>' {
		l.r.Move(1)
		if l.r.Peek(0) == '>' {
			l.r.Move(1)
			if l.r.Peek(0) == '=' {
				l.r.Move(1)
				return GtGtGtEqToken
			}
			return GtGtGtToken
		} else if l.r.Peek(0) == '=' {
			l.r.Move(1)
			return GtGtEqToken
		}
		return GtGtToken
	}
	return opTokens[c]
}

func (l *Lexer) consumeUnicodeIdentifierToken() TokenType {
	c := l.r.Peek(0)
	if c >= 0xC0 {
		if r, n := l.r.PeekRune(0); unicode.IsOneOf(identifierStart, r) {
			l.r.Move(n)
		} else {
			return ErrorToken
		}
	} else if !l.consumeUnicodeEscape() {
		return ErrorToken
	}
	return l.consumeIdentifierToken()
}

func (l *Lexer) consumeIdentifierToken() TokenType {
	// assume to be passed identifierStart character
	c := l.r.Peek(0)
	if identifierStartTable[c] {
		if c >= 0xC0 {
			if r, n := l.r.PeekRune(0); unicode.IsOneOf(identifierStart, r) {
				l.r.Move(n)
			} else {
				return ErrorToken
			}
		} else {
			l.r.Move(1)
		}
	} else if !l.consumeUnicodeEscape() {
		return ErrorToken
	}
	for {
		c := l.r.Peek(0)
		if identifierTable[c] {
			l.r.Move(1)
		} else if c >= 0xC0 {
			if r, n := l.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierContinue, r) {
				l.r.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	if keyword, ok := Keywords[string(l.r.Lexeme())]; ok {
		return keyword
	}
	return IdentifierToken
}

func (l *Lexer) consumeNumericToken() TokenType {
	// assume to be on 0 1 2 3 4 5 6 7 8 9 .
	mark := l.r.Pos()
	c := l.r.Peek(0)
	if c == '0' {
		l.r.Move(1)
		if l.r.Peek(0) == 'x' || l.r.Peek(0) == 'X' {
			l.r.Move(1)
			if l.consumeHexDigit() {
				for l.consumeHexDigit() {
				}
				return HexadecimalToken
			}
			l.r.Move(-1) // return just the zero
			return DecimalToken
		} else if l.r.Peek(0) == 'b' || l.r.Peek(0) == 'B' {
			l.r.Move(1)
			if l.consumeBinaryDigit() {
				for l.consumeBinaryDigit() {
				}
				return BinaryToken
			}
			l.r.Move(-1) // return just the zero
			return DecimalToken
		} else if l.r.Peek(0) == 'o' || l.r.Peek(0) == 'O' {
			l.r.Move(1)
			if l.consumeOctalDigit() {
				for l.consumeOctalDigit() {
				}
				return OctalToken
			}
			l.r.Move(-1) // return just the zero
			return DecimalToken
		} else if l.r.Peek(0) == 'n' {
			l.r.Move(1)
			return BigIntToken
		}
	} else if c != '.' {
		for l.consumeDigit() {
		}
	}
	if l.r.Peek(0) == '.' {
		l.r.Move(1)
		if l.consumeDigit() {
			for l.consumeDigit() {
			}
		} else if c != '.' {
			// . could belong to the next token
			l.r.Move(-1)
			return DecimalToken
		} else {
			l.r.Rewind(mark)
			return ErrorToken
		}
	} else if l.r.Peek(0) == 'n' {
		l.r.Move(1)
		return BigIntToken
	}
	mark = l.r.Pos()
	c = l.r.Peek(0)
	if c == 'e' || c == 'E' {
		l.r.Move(1)
		c = l.r.Peek(0)
		if c == '+' || c == '-' {
			l.r.Move(1)
		}
		if !l.consumeDigit() {
			// e could belong to the next token
			l.r.Rewind(mark)
			return DecimalToken
		}
		for l.consumeDigit() {
		}
	}
	return DecimalToken
}

func (l *Lexer) consumeStringToken() bool {
	// assume to be on ' or "
	mark := l.r.Pos()
	delim := l.r.Peek(0)
	l.r.Move(1)
	for {
		c := l.r.Peek(0)
		if c == delim {
			l.r.Move(1)
			break
		} else if c == '\\' {
			l.r.Move(1)
			if !l.consumeLineTerminator() {
				if c := l.r.Peek(0); c == delim || c == '\\' {
					l.r.Move(1)
				}
			}
			continue
		} else if l.consumeLineTerminator() || c == 0 && l.r.Err() != nil {
			l.r.Rewind(mark)
			return false
		}
		l.r.Move(1)
	}
	return true
}

func (l *Lexer) consumeRegExpToken() bool {
	// assume to be on /
	l.r.Move(1)
	inClass := false
	for {
		c := l.r.Peek(0)
		if !inClass && c == '/' {
			l.r.Move(1)
			break
		} else if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		} else if c == '\\' {
			l.r.Move(1)
			if l.isLineTerminator() || l.r.Peek(0) == 0 && l.r.Err() != nil {
				return false
			}
		} else if l.isLineTerminator() || c == 0 && l.r.Err() != nil {
			return false
		}
		l.r.Move(1)
	}
	// flags
	for {
		c := l.r.Peek(0)
		if identifierTable[c] {
			if c >= 0xC0 {
				if r, n := l.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierContinue, r) {
					l.r.Move(n)
				} else {
					break
				}
			} else {
				l.r.Move(1)
			}
		} else {
			break
		}
	}
	return true
}

func (l *Lexer) consumeTemplateToken() TokenType {
	// assume to be on ` or } when already within template
	continuation := l.r.Peek(0) == '}'
	l.r.Move(1)
	for {
		c := l.r.Peek(0)
		if c == '`' {
			l.templateLevels = l.templateLevels[:len(l.templateLevels)-1]
			l.r.Move(1)
			if continuation {
				return TemplateEndToken
			}
			return TemplateToken
		} else if c == '$' && l.r.Peek(1) == '{' {
			l.level++
			l.r.Move(2)
			if continuation {
				return TemplateMiddleToken
			}
			return TemplateStartToken
		} else if c == '\\' {
			l.r.Move(1)
			if c := l.r.Peek(0); c != 0 {
				l.r.Move(1)
			}
			continue
		} else if c == 0 && l.r.Err() != nil {
			if continuation {
				return TemplateEndToken
			}
			return TemplateToken
		}
		l.r.Move(1)
	}
}

var identifierStartTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, true, false, false, false, // $
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, true, true, true, true, true, true, true, // A, B, C, D, E, F, G
	true, true, true, true, true, true, true, true, // H, I, J, K, L, M, N, O
	true, true, true, true, true, true, true, true, // P, Q, R, S, T, U, V, W
	true, true, true, false, false, false, false, true, // X, Y, Z, _

	false, true, true, true, true, true, true, true, // a, b, c, d, e, f, g
	true, true, true, true, true, true, true, true, // h, i, j, k, l, m, n, o
	true, true, true, true, true, true, true, true, // p, q, r, s, t, u, v, w
	true, true, true, false, false, false, false, false, // x, y, z

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}

var identifierTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, true, false, false, false, // $
	false, false, false, false, false, false, false, false,
	true, true, true, true, true, true, true, true, // 0, 1, 2, 3, 4, 5, 6, 7
	true, true, false, false, false, false, false, false, // 8, 9

	false, true, true, true, true, true, true, true, // A, B, C, D, E, F, G
	true, true, true, true, true, true, true, true, // H, I, J, K, L, M, N, O
	true, true, true, true, true, true, true, true, // P, Q, R, S, T, U, V, W
	true, true, true, false, false, false, false, true, // X, Y, Z, _

	false, true, true, true, true, true, true, true, // a, b, c, d, e, f, g
	true, true, true, true, true, true, true, true, // h, i, j, k, l, m, n, o
	true, true, true, true, true, true, true, true, // p, q, r, s, t, u, v, w
	true, true, true, false, false, false, false, false, // x, y, z

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}
