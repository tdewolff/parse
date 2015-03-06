package js // import "github.com/tdewolff/parse/js"

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"unicode"

	"github.com/tdewolff/parse"
)

// ErrBadEscape is returned when an escaped sequence contains a newline.
var ErrBadEscape = errors.New("bad escape")

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	WhitespaceToken      // space \t \v \f
	LineTerminatorToken  // \r \n \r\n
	CommentToken
	IdentifierToken
	PunctuatorToken
	BoolToken
	NullToken
	NumericToken
	StringToken
	RegexpToken
)

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	switch tt {
	case ErrorToken:
		return "Error"
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
	case NullToken:
		return "Null"
	case BoolToken:
		return "Bool"
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
	r    *parse.ShiftBuffer
	line int
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		parse.NewShiftBuffer(r),
		1,
	}
}

// Line returns the current line that is being tokenized (1 + number of \n, \r or \r\n encountered).
func (z Tokenizer) Line() int {
	return z.line
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
	if z.consumeWhitespaceToken() {
		return WhitespaceToken, z.r.Shift()
	} else if z.consumeLineTerminatorToken() {
		return LineTerminatorToken, z.r.Shift()
	} else if z.consumeIdentifierToken() {
		if bytes.Equal(z.r.Buffered(), []byte("null")) {
			return NullToken, z.r.Shift()
		} else if bytes.Equal(z.r.Buffered(), []byte("true")) || bytes.Equal(z.r.Buffered(), []byte("false")) {
			return BoolToken, z.r.Shift()
		}
		return IdentifierToken, z.r.Shift()
	} else if z.consumeNumericToken() {
		return NumericToken, z.r.Shift()
	} else if z.consumeStringToken() {
		return StringToken, z.r.Shift()
	} else if z.consumeCommentToken() {
		return CommentToken, z.r.Shift()
	} else if z.consumeRegexpToken() {
		return RegexpToken, z.r.Shift()
	} else if z.consumePunctuatorToken() {
		return PunctuatorToken, z.r.Shift()
	}
	return ErrorToken, []byte{}
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.ecma-international.org/ecma-262/5.1
*/

func (z *Tokenizer) consumeByte(c byte) bool {
	if z.r.Peek(0) == c {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeRune() bool {
	c := z.r.Peek(0)
	if c < 0xC0 {
		z.r.Move(1)
	} else if c < 0xE0 {
		z.r.Move(2)
	} else if c < 0xF0 {
		z.r.Move(3)
	} else {
		z.r.Move(4)
	}
	return true
}

func (z *Tokenizer) consumeWhitespace() bool {
	r := z.r.PeekRune(0)
	if r == ' ' || r == '\t' || r == '\v' || r == '\f' || r == '\u00A0' || r == '\uFEFF' || unicode.Is(unicode.Zs, r) {
		return z.consumeRune()
	}
	return false
}

func (z *Tokenizer) consumeLineTerminator() bool {
	r := z.r.PeekRune(0)
	if r == '\n' {
		z.line++
		z.r.Move(1)
		return true
	}
	if r == '\r' {
		z.line++
		if z.r.Peek(1) == '\n' {
			z.r.Move(2)
		} else {
			z.r.Move(1)
		}
		return true
	}
	if r == '\u2028' || r == '\u2029' {
		z.line++
		return z.consumeRune()
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
	if z.r.Peek(0) != '\\' {
		return false
	}
	if z.consumeHexEscape() || z.consumeUnicodeEscape() {
		return true
	}
	if z.r.Peek(1) == '0' {
		nOld := z.r.Pos()
		z.r.Move(2)
		if !z.consumeDigit() {
			return true
		}
		z.r.MoveTo(nOld)
		return false
	}
	return true
}
func (z *Tokenizer) consumeHexEscape() bool {
	if z.r.Peek(0) != '\\' && z.r.Peek(1) != 'x' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(2)
	for k := 0; k < 2; k++ {
		if !z.consumeHexDigit() {
			z.r.MoveTo(nOld)
			return false
		}
	}
	return true
}

func (z *Tokenizer) consumeUnicodeEscape() bool {
	if z.r.Peek(0) != '\\' && z.r.Peek(1) != 'u' {
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

func (z *Tokenizer) consumeWhitespaceToken() bool {
	if z.consumeWhitespace() {
		for z.consumeWhitespace() {
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeLineTerminatorToken() bool {
	if z.consumeLineTerminator() {
		for z.consumeLineTerminator() {
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeCommentToken() bool {
	if z.r.Peek(0) != '/' || z.r.Peek(1) != '/' && z.r.Peek(1) != '*' {
		return false
	}
	z.r.Move(2)
	if z.r.Peek(1) == '/' {
		// single line
		for {
			if z.r.Peek(0) == 0 || z.r.Peek(0) == '\r' || z.r.Peek(0) == '\n' {
				break
			}
			z.consumeRune()
		}
	} else {
		// multi line
		for {
			if z.r.Peek(0) == '*' && z.r.Peek(1) == '/' {
				z.r.Move(2)
				return true
			}
			if z.r.Peek(0) == 0 {
				break
			}
			z.consumeRune()
		}
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeIdentifierToken() bool {
	r := z.r.PeekRune(0)
	if r == '$' || r == '_' || unicode.IsOneOf([]*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl}, r) {
		z.consumeRune()
	} else if !z.consumeUnicodeEscape() {
		return false
	}
	rangeTable := []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc}
	for {
		r := z.r.PeekRune(0)
		if r != '$' && r != '_' && !unicode.IsOneOf(rangeTable, r) && r != '\u200C' && r != '\u200D' {
			break
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeNumericToken() bool {
	nOld := z.r.Pos()
	c := z.r.Peek(0)
	firstDigit := false
	if firstDigit = z.r.Peek(0) == '0'; firstDigit {
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
		firstDigit = true
	} else if firstDigit = z.consumeDigit(); firstDigit {
		for z.consumeDigit() {
		}
	}
	if z.r.Peek(0) == '.' {
		z.r.Move(1)
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if firstDigit {
			// . could belong to the next token
			z.r.Move(-1)
			return true
		} else {
			z.r.MoveTo(nOld)
			return false
		}
	} else if !firstDigit {
		z.r.MoveTo(nOld)
		return false
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
	delim := z.r.Peek(0)
	if delim != '"' && delim != '\'' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(1)
	for {
		if z.consumeLineTerminator() {
			z.r.MoveTo(nOld)
			return false
		}
		c := z.r.Peek(0)
		if c == 0 {
			break
		}
		if c == delim {
			z.r.Move(1)
			break
		}
		if c == '\\' {
			if !z.consumeEscape() {
				break
			}
			continue
		}
		z.consumeRune()
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeRegexpToken() bool {
	if z.r.Peek(0) != '/' || z.r.Peek(1) == '*' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(1)
	inClass := false
	for {
		if z.consumeLineTerminator() {
			z.r.MoveTo(nOld)
			return false
		}
		c := z.r.Peek(0)
		if c == 0 {
			break
		} else if !inClass && c == '/' {
			z.r.Move(1)
			break
		} else if c == '\\' && (z.r.Peek(1) == '/' || z.r.Peek(1) == '[') {
			z.r.Move(2)
			continue
		} else if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		}
		z.consumeRune()
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumePunctuatorToken() bool {
	c := z.r.Peek(0)
	if c == '{' || c == '}' || c == '(' || c == ')' || c == '[' || c == ']' || c == '.' || c == ';' || c == ',' || c == '~' || c == '?' || c == ':' {
		z.r.Move(1)
		return true
	}
	if c == '!' || c == '=' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '&' || c == '|' || c == '^' {
		z.r.Move(1)
		if z.r.Peek(0) == '=' {
			z.r.Move(1)
			if (c == '!' || c == '=') && z.r.Peek(0) == '=' {
				z.r.Move(1)
			}
		} else if (c == '&' || c == '|') && z.r.Peek(0) == c {
			z.r.Move(1)
		}
		return true
	}
	if c == '<' || c == '>' {
		if z.r.Peek(0) == c {
			z.r.Move(1)
			if c == '>' && z.r.Peek(0) == '>' {
				z.r.Move(1)
			}
		}
		if z.r.Peek(0) == '=' {
			z.r.Move(1)
		}
		return true
	}
	return false
}
