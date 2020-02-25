// Package js is an ECMAScript5.1 lexer following the specifications at http://www.ecma-international.org/ecma-262/5.1/.
package js

import (
	"io"
	"strconv"
	"unicode"

	"github.com/tdewolff/parse/v2/buffer"
)

var identifierStart = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Other_ID_Start}
var identifierContinue = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue}

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken          TokenType = iota // extra token when errors occur
	UnknownToken                         // extra token when no token can be matched
	WhitespaceToken                      // space \t \v \f
	LineTerminatorToken                  // \r \n \r\n
	SingleLineCommentToken
	MultiLineCommentToken // token for comments with line terminators (not just any /*block*/)
	NumericToken
	StringToken
	RegexpToken
	TemplateToken

	PunctuatorToken = 0x40 // < > <= >= == != === !==  + - * % ++ -- << >> >>> & | ^ ! ~ && || ? : = += -= *= %= <<= >>= >>>= &= |= ^= / /= >=
	OpenBraceToken
	CloseBraceToken
	OpenParenToken
	CloseParenToken
	OpenBracketToken
	CloseBracketToken
	DotToken
	SemicolonToken
	CommaToken

	// keywords
	IdentifierToken = 0x80
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
	ImplementsToken
	ImportToken
	InToken
	InstanceofToken
	InterfaceToken
	LetToken
	NewToken
	NullToken
	PackageToken
	PrivateToken
	ProtectedToken
	PublicToken
	ReturnToken
	StaticToken
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

func IsPunctuator(tt TokenType) bool {
	return tt&0x40 != 0
}

func IsIdentifier(tt TokenType) bool {
	return tt&0x80 != 0
}

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	switch tt {
	case ErrorToken:
		return "Error"
	case UnknownToken:
		return "Unknown"
	case WhitespaceToken:
		return "Whitespace"
	case LineTerminatorToken:
		return "LineTerminator"
	case SingleLineCommentToken:
		return "SingleLineComment"
	case MultiLineCommentToken:
		return "MultiLineComment"
	case IdentifierToken:
		return "Identifier"
	case PunctuatorToken:
		return "Punctuator"
	case NumericToken:
		return "Numeric"
	case StringToken:
		return "String"
	case RegexpToken:
		return "Regexp"
	case TemplateToken:
		return "Template"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Lexer is the state for the lexer.
type Lexer struct {
	r                  *buffer.Lexer
	prevLineTerminator bool
	level              int
	templateLevels     []int
}

// NewLexer returns a new Lexer for a given io.Reader.
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		r:                  buffer.NewLexer(r),
		prevLineTerminator: true,
		level:              0,
		templateLevels:     []int{},
	}
}

// Err returns the error encountered during lexing, this is often io.EOF but also other errors can be returned.
func (l *Lexer) Err() error {
	return l.r.Err()
}

// Restore restores the NULL byte at the end of the buffer.
func (l *Lexer) Restore() {
	l.r.Restore()
}

// Offset returns the current position in the input stream.
func (l *Lexer) Offset() int {
	return l.r.Offset()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (l *Lexer) Next() (TokenType, []byte) {
	prevLineTerminator := l.prevLineTerminator
	l.prevLineTerminator = false

	c := l.r.Peek(0)
	switch c {
	case '(':
		l.level++
		l.r.Move(1)
		return OpenParenToken, l.r.Shift()
	case ')':
		l.level--
		l.r.Move(1)
		return CloseParenToken, l.r.Shift()
	case '{':
		l.level++
		l.r.Move(1)
		return OpenBraceToken, l.r.Shift()
	case '}':
		l.level--
		if len(l.templateLevels) != 0 && l.level == l.templateLevels[len(l.templateLevels)-1] {
			l.consumeTemplateToken()
			return TemplateToken, l.r.Shift()
		}
		l.r.Move(1)
		return CloseBraceToken, l.r.Shift()
	case ']':
		l.r.Move(1)
		return CloseBracketToken, l.r.Shift()
	case '[':
		l.r.Move(1)
		return OpenBracketToken, l.r.Shift()
	case ';':
		l.r.Move(1)
		return SemicolonToken, l.r.Shift()
	case ',':
		l.r.Move(1)
		return CommaToken, l.r.Shift()
	case '~', '?', ':':
		l.r.Move(1)
		return PunctuatorToken, l.r.Shift()
	case '<', '-':
		if l.consumeHTMLLikeCommentToken(prevLineTerminator) {
			return SingleLineCommentToken, l.r.Shift()
		} else if l.consumePunctuatorToken() {
			return PunctuatorToken, l.r.Shift()
		}
	case '>', '=', '!', '+', '*', '%', '&', '|', '^':
		if l.consumePunctuatorToken() {
			return PunctuatorToken, l.r.Shift()
		}
	case '/':
		if tt := l.consumeCommentToken(); tt != UnknownToken {
			return tt, l.r.Shift()
		} else if l.consumePunctuatorToken() {
			return PunctuatorToken, l.r.Shift()
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		if l.consumeNumericToken() {
			return NumericToken, l.r.Shift()
		} else if c == '.' {
			l.r.Move(1)
			return DotToken, l.r.Shift()
		}
	case '\'', '"':
		if l.consumeStringToken() {
			return StringToken, l.r.Shift()
		}
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
	case '`':
		l.templateLevels = append(l.templateLevels, l.level)
		l.consumeTemplateToken()
		return TemplateToken, l.r.Shift()
	default:
		if tt := l.consumeIdentifierToken(); tt != UnknownToken {
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

	_, n := l.r.PeekRune(0)
	l.r.Move(n)
	return UnknownToken, l.r.Shift()
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
	c := l.r.Peek(0)
	if c == '/' {
		c = l.r.Peek(1)
		if c == '/' {
			// single line comment
			l.r.Move(2)
			l.consumeSingleLineComment()
			return SingleLineCommentToken
		} else if c == '*' {
			// block comment (potentially multiline)
			tt := SingleLineCommentToken
			l.r.Move(2)
			for {
				c := l.r.Peek(0)
				if c == '*' && l.r.Peek(1) == '/' {
					l.r.Move(2)
					break
				} else if c == 0 && l.r.Err() != nil {
					break
				} else if l.consumeLineTerminator() {
					tt = MultiLineCommentToken
					l.prevLineTerminator = true
				} else {
					l.r.Move(1)
				}
			}
			return tt
		}
	}
	return UnknownToken
}

func (l *Lexer) consumePunctuatorToken() bool {
	c := l.r.Peek(0)
	if c == '!' || c == '=' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '&' || c == '|' || c == '^' {
		l.r.Move(1)
		if l.r.Peek(0) == '=' {
			l.r.Move(1)
			if (c == '!' || c == '=') && l.r.Peek(0) == '=' {
				l.r.Move(1)
			}
		} else if (c == '+' || c == '-' || c == '&' || c == '|') && l.r.Peek(0) == c {
			l.r.Move(1)
		} else if c == '=' && l.r.Peek(0) == '>' {
			l.r.Move(1)
		}
	} else { // c == '<' || c == '>'
		l.r.Move(1)
		if l.r.Peek(0) == c {
			l.r.Move(1)
			if c == '>' && l.r.Peek(0) == '>' {
				l.r.Move(1)
			}
		}
		if l.r.Peek(0) == '=' {
			l.r.Move(1)
		}
	}
	return true
}

func (l *Lexer) consumeIdentifierToken() TokenType {
	c := l.r.Peek(0)
	if identifierTable[c] && (c < '0' || c > '9') {
		if c >= 0xC0 {
			if r, n := l.r.PeekRune(0); unicode.IsOneOf(identifierStart, r) {
				l.r.Move(n)
			} else {
				return UnknownToken
			}
		} else {
			l.r.Move(1)
		}
	} else if !l.consumeUnicodeEscape() {
		return UnknownToken
	}
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
	return IdentifierToken
}

func (l *Lexer) consumeNumericToken() bool {
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
			} else {
				l.r.Move(-1) // return just the zero
			}
			return true
		} else if l.r.Peek(0) == 'b' || l.r.Peek(0) == 'B' {
			l.r.Move(1)
			if l.consumeBinaryDigit() {
				for l.consumeBinaryDigit() {
				}
			} else {
				l.r.Move(-1) // return just the zero
			}
			return true
		} else if l.r.Peek(0) == 'o' || l.r.Peek(0) == 'O' {
			l.r.Move(1)
			if l.consumeOctalDigit() {
				for l.consumeOctalDigit() {
				}
			} else {
				l.r.Move(-1) // return just the zero
			}
			return true
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
			return true
		} else {
			l.r.Rewind(mark)
			return false
		}
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
			return true
		}
		for l.consumeDigit() {
		}
	}
	return true
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

func (l *Lexer) consumeTemplateToken() {
	// assume to be on ` or } when already within template
	l.r.Move(1)
	for {
		c := l.r.Peek(0)
		if c == '`' {
			l.templateLevels = l.templateLevels[:len(l.templateLevels)-1]
			l.r.Move(1)
			return
		} else if c == '$' && l.r.Peek(1) == '{' {
			l.level++
			l.r.Move(2)
			return
		} else if c == '\\' {
			l.r.Move(1)
			if c := l.r.Peek(0); c != 0 {
				l.r.Move(1)
			}
			continue
		} else if c == 0 && l.r.Err() != nil {
			return
		}
		l.r.Move(1)
	}
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

	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,

	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,
	true, true, true, true, true, true, true, true,
}
