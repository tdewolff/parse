package json // import "github.com/tdewolff/parse/json"

import (
	"io"
	"strconv"

	"github.com/tdewolff/buffer"
)

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken   TokenType = iota // extra token when errors occur
	UnknownToken                  // extra token when no token can be matched
	WhitespaceToken
	LiteralToken
	PunctuatorToken /* { } [ ] , : */
	NumberToken
	StringToken
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
	case LiteralToken:
		return "Literal"
	case PunctuatorToken:
		return "Punctuator"
	case NumberToken:
		return "Number"
	case StringToken:
		return "String"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r *buffer.Shifter
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
	c := z.r.Peek(0)
	switch c {
	case ' ', '\t', '\r', '\n':
		z.r.Move(1)
		z.consumeWhitespaceToken()
		return WhitespaceToken, z.r.Shift()
	case '[', ']', '{', '}', ',', ':':
		z.r.Move(1)
		return PunctuatorToken, z.r.Shift()
	case '"':
		if z.consumeStringToken() {
			return StringToken, z.r.Shift()
		}
	default:
		if z.consumeNumberToken() {
			return NumberToken, z.r.Shift()
		} else if z.consumeLiteralToken() {
			return LiteralToken, z.r.Shift()
		} else if z.Err() != nil {
			return ErrorToken, []byte{}
		}
	}
	z.r.Move(1)
	return UnknownToken, z.r.Shift()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://json.org/
*/

func (z *Tokenizer) consumeWhitespaceToken() bool {
	for {
		if c := z.r.Peek(0); c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return true
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) consumeLiteralToken() bool {
	c := z.r.Peek(0)
	if c == 't' && z.r.Peek(1) == 'r' && z.r.Peek(2) == 'u' && z.r.Peek(3) == 'e' {
		z.r.Move(4)
		return true
	} else if c == 'f' && z.r.Peek(1) == 'a' && z.r.Peek(2) == 'l' && z.r.Peek(3) == 's' && z.r.Peek(4) == 'e' {
		z.r.Move(5)
		return true
	} else if c == 'n' && z.r.Peek(1) == 'u' && z.r.Peek(2) == 'l' && z.r.Peek(3) == 'l' {
		z.r.Move(4)
		return true
	}
	return false
}

func (z *Tokenizer) consumeNumberToken() bool {
	nOld := z.r.Pos()
	if z.r.Peek(0) == '-' {
		z.r.Move(1)
	}
	c := z.r.Peek(0)
	if c >= '1' && c <= '9' {
		z.r.Move(1)
		for {
			if c := z.r.Peek(0); c < '0' || c > '9' {
				break
			}
			z.r.Move(1)
		}
	} else if c != '0' {
		z.r.MoveTo(nOld)
		return false
	} else {
		z.r.Move(1) // 0
	}
	if c := z.r.Peek(0); c == '.' {
		z.r.Move(1)
		if c := z.r.Peek(0); c < '0' || c > '9' {
			z.r.Move(-1)
			return true
		}
		for {
			if c := z.r.Peek(0); c < '0' || c > '9' {
				break
			}
			z.r.Move(1)
		}
	}
	nOld = z.r.Pos()
	if c := z.r.Peek(0); c == 'e' || c == 'E' {
		z.r.Move(1)
		if c := z.r.Peek(0); c == '+' || c == '-' {
			z.r.Move(1)
		}
		if c := z.r.Peek(0); c < '0' || c > '9' {
			z.r.MoveTo(nOld)
			return true
		}
		for {
			if c := z.r.Peek(0); c < '0' || c > '9' {
				break
			}
			z.r.Move(1)
		}
	}
	return true
}

func (z *Tokenizer) consumeStringToken() bool {
	// assume to be on "
	z.r.Move(1)
	for {
		c := z.r.Peek(0)
		if c == 0 {
			break
		} else if c == '"' {
			z.r.Move(1)
			break
		} else if c == '\\' {
			if z.r.Peek(1) != 0 {
				z.r.Move(2)
				continue
			} else {
				z.r.Move(1)
				break
			}
		}
		z.r.Move(1)
	}
	return true
}
