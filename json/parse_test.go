package json // import "github.com/tdewolff/parse/json"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertGrammars(t *testing.T, s string, grammartypes ...GrammarType) {
	stringify := helperStringify(t, s)
	p := NewParser(bytes.NewBufferString(s))
	assert.True(t, p.IsEOF(), "parser must have buffer fully in memory in "+stringify)
	i := 0
	for {
		tt, _ := p.Next()
		if tt == ErrorGrammar {
			assert.Equal(t, io.EOF, p.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(grammartypes), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == WhitespaceGrammar {
			continue
		}
		assert.False(t, i >= len(grammartypes), "index must not exceed grammartypes size in "+stringify)
		if i < len(grammartypes) {
			assert.Equal(t, grammartypes[i], tt, "grammartypes must match at index "+strconv.Itoa(i)+" in "+stringify)
		}
		i++
	}
	return
}

func assertGrammarsError(t *testing.T, input string, expected error) {
	stringify := helperStringify(t, input)
	p := NewParser(bytes.NewBufferString(input))
	for {
		tt, _ := p.Next()
		if tt == ErrorGrammar {
			assert.Equal(t, expected, p.Err(), "parser must return error '"+expected.Error()+"' in "+stringify)
			break
		}
	}
}

func assertStates(t *testing.T, s string, states ...State) {
	stringify := helperStringify(t, s)
	p := NewParser(bytes.NewBufferString(s))
	i := 0
	for {
		tt, _ := p.Next()
		state := p.State()
		if tt == ErrorGrammar {
			assert.Equal(t, io.EOF, p.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(states), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == WhitespaceGrammar {
			continue
		}
		assert.False(t, i >= len(states), "index must not exceed states size in "+stringify)
		if i < len(states) {
			assert.Equal(t, states[i], state, "states must match at index "+strconv.Itoa(i)+" in "+stringify)
		}
		i++
	}
	return
}

func helperStringify(t *testing.T, input string) string {
	s := ""
	p := NewParser(bytes.NewBufferString(input))
	for i := 0; i < 10; i++ {
		tt, text := p.Next()
		if tt == ErrorGrammar {
			s += tt.String() + "('" + p.Err().Error() + "')"
			break
		} else if tt == WhitespaceGrammar {
			continue
		} else {
			s += tt.String() + "('" + string(text) + "') "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

func TestGrammars(t *testing.T) {
	assertGrammars(t, " \t\n\r") // WhitespaceGrammar
	assertGrammars(t, "null", LiteralGrammar)
	assertGrammars(t, "[]", StartArrayGrammar, EndArrayGrammar)
	assertGrammars(t, "[15.2, 0.4, 5e9, -4E-3]", StartArrayGrammar, NumberGrammar, NumberGrammar, NumberGrammar, NumberGrammar, EndArrayGrammar)
	assertGrammars(t, "[true, false, null]", StartArrayGrammar, LiteralGrammar, LiteralGrammar, LiteralGrammar, EndArrayGrammar)
	assertGrammars(t, `["", "abc", "\"", "\\"]`, StartArrayGrammar, StringGrammar, StringGrammar, StringGrammar, StringGrammar, EndArrayGrammar)
	assertGrammars(t, "{}", StartObjectGrammar, EndObjectGrammar)
	assertGrammars(t, `{"a": "b", "c": "d"}`, StartObjectGrammar, StringGrammar, StringGrammar, StringGrammar, StringGrammar, EndObjectGrammar)
	assertGrammars(t, `{"a": [1, 2], "b": {"c": 3}}`, StartObjectGrammar, StringGrammar, StartArrayGrammar, NumberGrammar, NumberGrammar, EndArrayGrammar, StringGrammar, StartObjectGrammar, StringGrammar, NumberGrammar, EndObjectGrammar, EndObjectGrammar)
	assertGrammars(t, "[null,]", StartArrayGrammar, LiteralGrammar, EndArrayGrammar)

	// early endings
	assertGrammars(t, "\"a", StringGrammar)
	assertGrammars(t, "\"a\\", StringGrammar)

	assert.Equal(t, "Whitespace", WhitespaceGrammar.String())
	assert.Equal(t, "Invalid(100)", GrammarType(100).String())
	assert.Equal(t, "Value", ValueState.String())
	assert.Equal(t, "ObjectKey", ObjectKeyState.String())
	assert.Equal(t, "ObjectValue", ObjectValueState.String())
	assert.Equal(t, "Array", ArrayState.String())
	assert.Equal(t, "Invalid(100)", State(100).String())
}

func TestGrammarsError(t *testing.T) {
	assertGrammarsError(t, "true, false", ErrBadComma)
	assertGrammarsError(t, "[true false]", ErrNoComma)
	assertGrammarsError(t, "]", ErrBadArrayEnding)
	assertGrammarsError(t, "}", ErrBadObjectEnding)
	assertGrammarsError(t, "{0: 1}", ErrBadObjectKey)
	assertGrammarsError(t, "{\"a\" 1}", ErrBadObjectDeclaration)
	assertGrammarsError(t, "1.", ErrNoComma)
	assertGrammarsError(t, "1e+", ErrNoComma)
}

func TestStates(t *testing.T) {
	assertStates(t, "null", ValueState)
	assertStates(t, "[null]", ArrayState, ArrayState, ValueState)
	assertStates(t, "{\"\":null}", ObjectKeyState, ObjectValueState, ObjectKeyState, ValueState)
}

////////////////////////////////////////////////////////////////

func ExampleNewParser() {
	p := NewParser(bytes.NewBufferString(`{"key": 5}`))
	out := ""
	for {
		state := p.State()
		tt, data := p.Next()
		if tt == ErrorGrammar {
			break
		}
		if state == ObjectKeyState && tt != EndObjectGrammar {
			out += "\""
		}
		out += string(data)
		if state == ObjectKeyState && tt != EndObjectGrammar {
			out += "\":"
		}
		// not handling comma insertion
	}
	fmt.Println(out)
	// Output: {"key":5}
}
