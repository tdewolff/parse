// Package js is an ECMAScript5.1 tokenizer. It is implemented using the specifications at http://www.ecma-international.org/ecma-262/5.1/.
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

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r *buffer.Shifter

	regexpState bool
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r: buffer.NewShifter(r),
	}
}

// Err returns the error encountered during tokenization, this is often io.EOF but also other errors can be returned.
func (z Tokenizer) Err() error {
	return z.r.Err()
}

// IsEOF returns true when it has encountered EOF and thus loaded the last buffer in memory.
func (z Tokenizer) IsEOF() bool {
	return z.r.IsEOF()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, []byte) {
	tt := UnknownToken
	c := z.r.Peek(0)
	switch c {
	case '(', ')', '[', ']', '{', '}', ';', ',', '~', '?', ':':
		z.r.Move(1)
		tt = PunctuatorToken
	case '<', '>', '=', '!', '+', '-', '*', '%', '&', '|', '^':
		if z.consumeLongPunctuatorToken() {
			tt = PunctuatorToken
		}
	case '/':
		if z.consumeCommentToken() {
			return CommentToken, z.r.Shift()
		} else if z.regexpState && z.consumeRegexpToken() {
			tt = RegexpToken
		} else if z.consumeLongPunctuatorToken() {
			tt = PunctuatorToken
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		if z.consumeNumericToken() {
			tt = NumericToken
		} else if c == '.' {
			z.r.Move(1)
			tt = PunctuatorToken
		}
	case '\'', '"':
		if z.consumeStringToken() {
			tt = StringToken
		}
	case ' ', '\t', '\v', '\f':
		z.r.Move(1)
		for z.consumeWhitespace() {
		}
		return WhitespaceToken, z.r.Shift()
	case '\n', '\r':
		z.r.Move(1)
		for z.consumeLineTerminator() {
		}
		tt = LineTerminatorToken
	default:
		if z.consumeIdentifierToken() {
			tt = IdentifierToken
		} else if c >= 0xC0 {
			if z.consumeWhitespace() {
				for z.consumeWhitespace() {
				}
				return WhitespaceToken, z.r.Shift()
			} else if z.consumeLineTerminator() {
				for z.consumeLineTerminator() {
				}
				tt = LineTerminatorToken
			}
		} else if z.Err() != nil {
			return ErrorToken, []byte{}
		}
	}

	// differentiate between divisor and regexp state, because the '/' character is ambiguous!
	// ErrorToken, WhitespaceToken and CommentToken are already returned
	if tt == LineTerminatorToken || tt == PunctuatorToken && RegexpStateByte[c] {
		z.regexpState = true
	} else {
		z.regexpState = false
	}
	if tt == UnknownToken {
		_, n := z.r.PeekRune(0)
		z.r.Move(n)
	}
	return tt, z.r.Shift()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.ecma-international.org/ecma-262/5.1/
*/

func (z *Tokenizer) consumeWhitespace() bool {
	c := z.r.Peek(0)
	if c == ' ' || c == '\t' || c == '\v' || c == '\f' {
		z.r.Move(1)
		return true
	} else if c >= 0xC0 {
		if r, n := z.r.PeekRune(0); r == '\u00A0' || r == '\uFEFF' || unicode.Is(unicode.Zs, r) {
			z.r.Move(n)
			return true
		}
	}
	return false
}

func (z *Tokenizer) consumeLineTerminator() bool {
	c := z.r.Peek(0)
	if c == '\n' {
		z.r.Move(1)
		return true
	} else if c == '\r' {
		if z.r.Peek(1) == '\n' {
			z.r.Move(2)
		} else {
			z.r.Move(1)
		}
		return true
	} else if c >= 0xC0 {
		if r, n := z.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
			z.r.Move(n)
			return true
		}
	}
	return false
}

func (z *Tokenizer) consumeDigit() bool {
	c := z.r.Peek(0)
	if c >= '0' && c <= '9' {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeHexDigit() bool {
	c := z.r.Peek(0)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeEscape() bool {
	// assume to be on \
	z.r.Move(1)
	c := z.r.Peek(0)
	if c == '0' {
		z.r.Move(1)
		if !z.consumeDigit() {
			return true
		}
		z.r.Move(-1)
		return false
	} else if z.consumeLineTerminator() {
		return true
	} else if c >= 0xC0 {
		_, n := z.r.PeekRune(0)
		z.r.Move(n)
		return true
	} else {
		z.r.Move(1)
		return true
	}
}

func (z *Tokenizer) consumeUnicodeEscape() bool {
	if z.r.Peek(0) != '\\' || z.r.Peek(1) != 'u' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(2)
	for k := 0; k < 4; k++ {
		if !z.consumeHexDigit() {
			z.r.MoveTo(nOld)
			return false
		}
	}
	return true
}

////////////////////////////////////////////////////////////////

func (z *Tokenizer) consumeCommentToken() bool {
	if z.r.Peek(0) != '/' || z.r.Peek(1) != '/' && z.r.Peek(1) != '*' {
		return false
	}
	if z.r.Peek(1) == '/' {
		z.r.Move(2)
		// single line
		for {
			c := z.r.Peek(0)
			if c == '\r' || c == '\n' || c == 0 {
				break
			} else if c >= 0xC0 {
				nOld := z.r.Pos()
				if r, _ := z.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
					z.r.MoveTo(nOld)
					break
				}
			}
			z.r.Move(1)
		}
	} else {
		z.r.Move(2)
		// multi line
		for {
			c := z.r.Peek(0)
			if c == '*' && z.r.Peek(1) == '/' {
				z.r.Move(2)
				return true
			} else if c == 0 {
				break
			}
			z.r.Move(1)
		}
	}
	return true
}

func (z *Tokenizer) consumeLongPunctuatorToken() bool {
	c := z.r.Peek(0)
	if c == '!' || c == '=' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '&' || c == '|' || c == '^' {
		z.r.Move(1)
		if z.r.Peek(0) == '=' {
			z.r.Move(1)
			if (c == '!' || c == '=') && z.r.Peek(0) == '=' {
				z.r.Move(1)
			}
		} else if (c == '+' || c == '-' || c == '&' || c == '|') && z.r.Peek(0) == c {
			z.r.Move(1)
		}
	} else { // c == '<' || c == '>'
		z.r.Move(1)
		if z.r.Peek(0) == c {
			z.r.Move(1)
			if c == '>' && z.r.Peek(0) == '>' {
				z.r.Move(1)
			}
		}
		if z.r.Peek(0) == '=' {
			z.r.Move(1)
		}
	}
	return true
}

func (z *Tokenizer) consumeIdentifierToken() bool {
	c := z.r.Peek(0)
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '$' || c == '_' {
		z.r.Move(1)
	} else if c >= 0xC0 {
		if r, n := z.r.PeekRune(0); unicode.IsOneOf(identifierStart, r) {
			z.r.Move(n)
		} else {
			return false
		}
	} else if !z.consumeUnicodeEscape() {
		return false
	}
	for {
		c := z.r.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			z.r.Move(1)
		} else if c >= 0xC0 {
			if r, n := z.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierPart, r) {
				z.r.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
}

func (z *Tokenizer) consumeNumericToken() bool {
	// assume to be on 0 1 2 3 4 5 6 7 8 9 .
	nOld := z.r.Pos()
	c := z.r.Peek(0)
	if c == '0' {
		z.r.Move(1)
		if z.r.Peek(0) == 'x' || z.r.Peek(0) == 'X' {
			z.r.Move(1)
			if z.consumeHexDigit() {
				for z.consumeHexDigit() {
				}
			} else {
				z.r.Move(-1) // return just the zero
			}
			return true
		}
	} else if c != '.' {
		for z.consumeDigit() {
		}
	}
	if z.r.Peek(0) == '.' {
		z.r.Move(1)
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if c != '.' {
			// . could belong to the next token
			z.r.Move(-1)
			return true
		} else {
			z.r.MoveTo(nOld)
			return false
		}
	}
	nOld = z.r.Pos()
	c = z.r.Peek(0)
	if c == 'e' || c == 'E' {
		z.r.Move(1)
		c = z.r.Peek(0)
		if c == '+' || c == '-' {
			z.r.Move(1)
		}
		if !z.consumeDigit() {
			// e could belong to the next token
			z.r.MoveTo(nOld)
			return true
		}
		for z.consumeDigit() {
		}
	}
	return true
}

func (z *Tokenizer) consumeStringToken() bool {
	// assume to be on ' or "
	nOld := z.r.Pos()
	delim := z.r.Peek(0)
	z.r.Move(1)
	for {
		c := z.r.Peek(0)
		if c == delim {
			z.r.Move(1)
			break
		} else if c == '\\' {
			if !z.consumeEscape() {
				break
			}
			continue
		} else if c == '\n' || c == '\r' {
			z.r.MoveTo(nOld)
			return false
		} else if c >= 0xC0 {
			if r, _ := z.r.PeekRune(0); r == '\u2028' || r == '\u2029' {
				z.r.MoveTo(nOld)
				return false
			}
		} else if c == 0 {
			break
		}
		z.r.Move(1)
	}
	return true
}

func (z *Tokenizer) consumeRegexpToken() bool {
	// assume to be on / and not /*
	nOld := z.r.Pos()
	z.r.Move(1)
	inClass := false
	for {
		c := z.r.Peek(0)
		if !inClass && c == '/' {
			z.r.Move(1)
			break
		} else if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		} else if c == '\\' {
			z.r.Move(1)
			if z.consumeLineTerminator() {
				z.r.MoveTo(nOld)
				return false
			}
		} else if z.consumeLineTerminator() {
			z.r.MoveTo(nOld)
			return false
		} else if c == 0 {
			return true
		}
		z.r.Move(1)
	}
	// flags
	for {
		c := z.r.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			z.r.Move(1)
		} else if c >= 0xC0 {
			if r, n := z.r.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(identifierPart, r) {
				z.r.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
}
