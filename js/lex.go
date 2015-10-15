// Package js is an ECMAScript5.1 lexer following the specifications at http://www.ecma-international.org/ecma-262/5.1/.
package js // import "github.com/tdewolff/parse/js"

import (
	"io"
	"strconv"
	"unicode"

	"github.com/tdewolff/buffer"
)

var identifierStart = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl}
var identifierPart = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc}

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken          TokenType = iota // extra token when errors occur
	UnknownToken                         // extra token when no token can be matched
	WhitespaceToken                      // space \t \v \f
	LineTerminatorToken                  // \r \n \r\n
	CommentToken
	IdentifierToken
	PunctuatorToken /* { } ( ) [ ] . ; , < > <= >= == != === !==  + - * % ++ -- << >>
	   >>> & | ^ ! ~ && || ? : = += -= *= %= <<= >>= >>>= &= |= ^= / /= */
	NumericToken
	StringToken
	RegexpToken
)

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
	case CommentToken:
		return "Comment"
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
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Lexer is the state for the lexer.
type Lexer struct {
	r *buffer.Lexer

	regexpState bool

	Free func(int)
}

// NewLexer returns a new Lexer for a given io.Reader.
func NewLexer(r io.Reader) *Lexer {
	l := &Lexer{
		r: buffer.NewLexer(r),
	}
	l.Free = l.r.Free
	return l
}

// Err returns the error encountered during lexing, this is often io.EOF but also other errors can be returned.
func (l Lexer) Err() error {
	return l.r.Err()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (l *Lexer) Next() (TokenType, []byte, int) {
	tt := UnknownToken
	c := l.r.Peek(0)
	switch c {
	case '(', ')', '[', ']', '{', '}', ';', ',', '~', '?', ':':
		l.r.Move(1)
		tt = PunctuatorToken
	case '<', '>', '=', '!', '+', '-', '*', '%', '&', '|', '^':
		if l.consumeLongPunctuatorToken() {
			tt = PunctuatorToken
		}
	case '/':
		if l.consumeCommentToken() {
			return CommentToken, l.r.Shift(), l.r.ShiftLen()
		} else if l.regexpState && l.consumeRegexpToken() {
			tt = RegexpToken
		} else if l.consumeLongPunctuatorToken() {
			tt = PunctuatorToken
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		if l.consumeNumericToken() {
			tt = NumericToken
		} else if c == '.' {
			l.r.Move(1)
			tt = PunctuatorToken
		}
	case '\'', '"':
		if l.consumeStringToken() {
			tt = StringToken
		}
	case ' ', '\t', '\v', '\f':
		l.r.Move(1)
		for l.consumeWhitespace() {
		}
		return WhitespaceToken, l.r.Shift(), l.r.ShiftLen()
	case '\n', '\r':
		l.r.Move(1)
		for l.consumeLineTerminator() {
		}
		tt = LineTerminatorToken
	default:
		if l.consumeIdentifierToken() {
			tt = IdentifierToken
		} else if c >= 0xC0 {
			if l.consumeWhitespace() {
				for l.consumeWhitespace() {
				}
				return WhitespaceToken, l.r.Shift(), l.r.ShiftLen()
			} else if l.consumeLineTerminator() {
				for l.consumeLineTerminator() {
				}
				tt = LineTerminatorToken
			}
		} else if l.Err() != nil {
			return ErrorToken, []byte{}, l.r.ShiftLen()
		}
	}

	// differentiate between divisor and regexp state, because the '/' character is ambiguous!
	// ErrorToken, WhitespaceToken and CommentToken are already returned
	if tt == LineTerminatorToken || tt == PunctuatorToken && regexpStateByte[c] {
		l.regexpState = true
	} else {
		l.regexpState = false
	}
	if tt == UnknownToken {
		_, n := l.r.PeekRune(0)
		l.r.Move(n)
	}
	return tt, l.r.Shift(), l.r.ShiftLen()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.ecma-international.org/ecma-262/5.1/
*/

func (l *Lexer) consumeWhitespace() bool {
	c := l.r.Peek(0)
	if c == ' ' || c == '\t' || c == '\v' || c == '\f' {
		l.r.Move(1)
		return true
	} else if c >= 0xC0 {
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
	c := l.r.Peek(0)
	if c >= '0' && c <= '9' {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeHexDigit() bool {
	c := l.r.Peek(0)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		l.r.Move(1)
		return true
	}
	return false
}

func (l *Lexer) consumeEscape() bool {
	// assume to be on \
	l.r.Move(1)
	c := l.r.Peek(0)
	if c == '0' {
		l.r.Move(1)
		if !l.consumeDigit() {
			return true
		}
		l.r.Move(-1)
		return false
	} else if l.consumeLineTerminator() {
		return true
	} else if c >= 0xC0 {
		_, n := l.r.PeekRune(0)
		l.r.Move(n)
		return true
	}
	l.r.Move(1)
	return true
}

func (l *Lexer) consumeUnicodeEscape() bool {
	if l.r.Peek(0) != '\\' || l.r.Peek(1) != 'u' {
		return false
	}
	mark := l.r.Pos()
	l.r.Move(2)
	for k := 0; k < 4; k++ {
		if !l.consumeHexDigit() {
			l.r.Rewind(mark)
			return false
		}
	}
	return true
}

////////////////////////////////////////////////////////////////

func (l *Lexer) consumeCommentToken() bool {
	if l.r.Peek(0) != '/' || l.r.Peek(1) != '/' && l.r.Peek(1) != '*' {
		return false
	}
	if l.r.Peek(1) == '/' {
		l.r.Move(2)
		// single line
		for {
			c := l.r.Peek(0)
			if c == '\r' || c == '\n' || c == 0 {
				break
			} else if c >= 0xC0 {
				mark := l.r.Pos()
				if r, _ := l.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
					l.r.Rewind(mark)
					break
				}
			}
			l.r.Move(1)
		}
	} else {
		l.r.Move(2)
		// multi line
		for {
			c := l.r.Peek(0)
			if c == '*' && l.r.Peek(1) == '/' {
				l.r.Move(2)
				return true
			} else if c == 0 {
				break
			}
			l.r.Move(1)
		}
	}
	return true
}

func (l *Lexer) consumeLongPunctuatorToken() bool {
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

func (l *Lexer) consumeIdentifierToken() bool {
	c := l.r.Peek(0)
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '$' || c == '_' {
		l.r.Move(1)
	} else if c >= 0xC0 {
		if r, n := l.r.PeekRune(0); unicode.IsOneOf(identifierStart, r) {
			l.r.Move(n)
		} else {
			return false
		}
	} else if !l.consumeUnicodeEscape() {
		return false
	}
	for {
		c := l.r.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			l.r.Move(1)
		} else if c >= 0xC0 {
			if r, n := l.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierPart, r) {
				l.r.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
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
			if !l.consumeEscape() {
				break
			}
			continue
		} else if c == '\n' || c == '\r' {
			l.r.Rewind(mark)
			return false
		} else if c >= 0xC0 {
			if r, _ := l.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
				l.r.Rewind(mark)
				return false
			}
		} else if c == 0 {
			break
		}
		l.r.Move(1)
	}
	return true
}

func (l *Lexer) consumeRegexpToken() bool {
	// assume to be on / and not /*
	mark := l.r.Pos()
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
			if l.consumeLineTerminator() {
				l.r.Rewind(mark)
				return false
			}
		} else if l.consumeLineTerminator() {
			l.r.Rewind(mark)
			return false
		} else if c == 0 {
			return true
		}
		l.r.Move(1)
	}
	// flags
	for {
		c := l.r.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			l.r.Move(1)
		} else if c >= 0xC0 {
			if r, n := l.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierPart, r) {
				l.r.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
}
