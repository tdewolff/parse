package json // import "github.com/tdewolff/parse/json"

// TODO: optimize and use skipping after obtaining a value like HTML/XML do (colon and comma characters)

import (
	"errors"
	"io"
	"strconv"

	"github.com/tdewolff/buffer"
)

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	WhitespaceToken
	LiteralToken
	NumberToken
	StringToken
	StartObjectToken // {
	EndObjectToken   // }
	StartArrayToken  // [
	EndArrayToken    // ]
)

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	switch tt {
	case ErrorToken:
		return "Error"
	case WhitespaceToken:
		return "Whitespace"
	case LiteralToken:
		return "Literal"
	case NumberToken:
		return "Number"
	case StringToken:
		return "String"
	case StartObjectToken:
		return "StartObject"
	case EndObjectToken:
		return "EndObject"
	case StartArrayToken:
		return "StartArray"
	case EndArrayToken:
		return "EndArray"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

type State uint32

const (
	ValueState State = iota // extra token when errors occur
	ObjectKeyState
	ObjectValueState
	ArrayState
)

func (state State) String() string {
	switch state {
	case ValueState:
		return "Value"
	case ObjectKeyState:
		return "ObjectKey"
	case ObjectValueState:
		return "ObjectValue"
	case ArrayState:
		return "Array"
	}
	return "Invalid(" + strconv.Itoa(int(state)) + ")"
}

////////////////////////////////////////////////////////////////

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r     *buffer.Shifter
	state []State
	err   error

	needComma bool
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r:     buffer.NewShifter(r),
		state: []State{ValueState},
	}
}

// Err returns the error encountered during tokenization, this is often io.EOF but also other errors can be returned.
func (z Tokenizer) Err() error {
	if z.err != nil {
		return z.err
	}
	return z.r.Err()
}

// IsEOF returns true when it has encountered EOF and thus loaded the last buffer in memory.
func (z Tokenizer) IsEOF() bool {
	return z.r.IsEOF()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, []byte) {
	z.skipWhitespace()
	if z.r.Peek(0) == ',' {
		if z.state[len(z.state)-1] != ArrayState && z.state[len(z.state)-1] != ObjectKeyState {
			z.err = errors.New("Unexpected ','")
			return ErrorToken, []byte{}
		}
		z.r.Move(1)
		z.skipWhitespace()
		z.needComma = false
	}
	z.r.Skip()

	c := z.r.Peek(0)
	state := z.state[len(z.state)-1]
	if z.needComma && c != '}' && c != ']' && c != 0 {
		z.err = errors.New("Expected ','")
		return ErrorToken, []byte{}
	} else if c == '{' {
		z.state = append(z.state, ObjectKeyState)
		z.r.Move(1)
		return StartObjectToken, z.r.Shift()
	} else if c == '}' {
		if state != ObjectKeyState {
			z.err = errors.New("Unexpected '}'")
			return ErrorToken, []byte{}
		}
		z.needComma = true
		z.state = z.state[:len(z.state)-1]
		if z.state[len(z.state)-1] == ObjectValueState {
			z.state[len(z.state)-1] = ObjectKeyState
		}
		z.r.Move(1)
		return EndObjectToken, z.r.Shift()
	} else if c == '[' {
		z.state = append(z.state, ArrayState)
		z.r.Move(1)
		return StartArrayToken, z.r.Shift()
	} else if c == ']' {
		z.needComma = true
		if state != ArrayState {
			z.err = errors.New("Unexpected ']'")
			return ErrorToken, []byte{}
		}
		z.state = z.state[:len(z.state)-1]
		if z.state[len(z.state)-1] == ObjectValueState {
			z.state[len(z.state)-1] = ObjectKeyState
		}
		z.r.Move(1)
		return EndArrayToken, z.r.Shift()
	} else if state == ObjectKeyState {
		if c != '"' || !z.consumeStringToken() {
			z.err = errors.New("Expected object key to be a string")
			return ErrorToken, []byte{}
		}
		n := z.r.Pos()
		z.skipWhitespace()
		if c := z.r.Peek(0); c != ':' {
			z.err = errors.New("Unexpected '" + string(c) + "', expected ':'")
			return ErrorToken, []byte{}
		}
		z.r.Move(1)
		z.state[len(z.state)-1] = ObjectValueState
		return StringToken, z.r.Shift()[1 : n-1]
	} else {
		z.needComma = true
		if state == ObjectValueState {
			z.state[len(z.state)-1] = ObjectKeyState
		}
		if c == '"' && z.consumeStringToken() {
			n := z.r.Pos()
			return StringToken, z.r.Shift()[1 : n-1]
		} else if z.consumeNumberToken() {
			return NumberToken, z.r.Shift()
		} else if z.consumeLiteralToken() {
			return LiteralToken, z.r.Shift()
		}
	}
	return ErrorToken, []byte{}
}

func (z *Tokenizer) State() State {
	return z.state[len(z.state)-1]
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://json.org/
*/

func (z *Tokenizer) skipWhitespace() {
	for {
		if c := z.r.Peek(0); c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
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
