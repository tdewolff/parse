package json // import "github.com/tdewolff/parse/json"

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertTokens(t *testing.T, s string, tokentypes ...TokenType) {
	stringify := helperStringify(t, s)
	z := NewTokenizer(bytes.NewBufferString(s))
	assert.True(t, z.IsEOF(), "tokenizer must have buffer fully in memory in "+stringify)
	i := 0
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, z.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(tokentypes), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == WhitespaceToken {
			continue
		}
		assert.False(t, i >= len(tokentypes), "index must not exceed tokentypes size in "+stringify)
		if i < len(tokentypes) {
			assert.Equal(t, tokentypes[i], tt, "tokentypes must match at index "+strconv.Itoa(i)+" in "+stringify)
		}
		i++
	}
	return
}

func assertTokensError(t *testing.T, input string, expected error) {
	stringify := helperStringify(t, input)
	z := NewTokenizer(bytes.NewBufferString(input))
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, expected, z.Err(), "tokenizer must return error '"+expected.Error()+"' in "+stringify)
			break
		}
	}
}

func assertStates(t *testing.T, s string, states ...State) {
	stringify := helperStringify(t, s)
	z := NewTokenizer(bytes.NewBufferString(s))
	i := 0
	for {
		tt, _ := z.Next()
		state := z.State()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, z.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(states), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == WhitespaceToken {
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
	z := NewTokenizer(bytes.NewBufferString(input))
	for i := 0; i < 10; i++ {
		tt, text := z.Next()
		if tt == ErrorToken {
			s += tt.String() + "('" + z.Err().Error() + "')"
			break
		} else if tt == WhitespaceToken {
			continue
		} else {
			s += tt.String() + "('" + string(text) + "') "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

func TestTokens(t *testing.T) {
	assertTokens(t, " \t\n\r") // WhitespaceToken
	assertTokens(t, "null", LiteralToken)
	assertTokens(t, "[]", StartArrayToken, EndArrayToken)
	assertTokens(t, "[15.2, 0.4, 5e9, -4E-3]", StartArrayToken, NumberToken, NumberToken, NumberToken, NumberToken, EndArrayToken)
	assertTokens(t, "[true, false, null]", StartArrayToken, LiteralToken, LiteralToken, LiteralToken, EndArrayToken)
	assertTokens(t, `["", "abc", "\"", "\\"]`, StartArrayToken, StringToken, StringToken, StringToken, StringToken, EndArrayToken)
	assertTokens(t, "{}", StartObjectToken, EndObjectToken)
	assertTokens(t, `{"a": "b", "c": "d"}`, StartObjectToken, StringToken, StringToken, StringToken, StringToken, EndObjectToken)
	assertTokens(t, `{"a": [1, 2], "b": {"c": 3}}`, StartObjectToken, StringToken, StartArrayToken, NumberToken, NumberToken, EndArrayToken, StringToken, StartObjectToken, StringToken, NumberToken, EndObjectToken, EndObjectToken)
	assertTokens(t, "[null,]", StartArrayToken, LiteralToken, EndArrayToken)

	// early endings
	assertTokens(t, "\"a", StringToken)
	assertTokens(t, "\"a\\", StringToken)

	assert.Equal(t, "Whitespace", WhitespaceToken.String())
	assert.Equal(t, "Invalid(100)", TokenType(100).String())
	assert.Equal(t, "Value", ValueState.String())
	assert.Equal(t, "ObjectKey", ObjectKeyState.String())
	assert.Equal(t, "ObjectValue", ObjectValueState.String())
	assert.Equal(t, "Array", ArrayState.String())
	assert.Equal(t, "Invalid(100)", State(100).String())
}

func TestTokensError(t *testing.T) {
	assertTokensError(t, "true, false", ErrBadComma)
	assertTokensError(t, "[true false]", ErrNoComma)
	assertTokensError(t, "]", ErrBadArrayEnding)
	assertTokensError(t, "}", ErrBadObjectEnding)
	assertTokensError(t, "{0: 1}", ErrBadObjectKey)
	assertTokensError(t, "{\"a\" 1}", ErrBadObjectDeclaration)
	assertTokensError(t, "1.", ErrNoComma)
	assertTokensError(t, "1e+", ErrNoComma)
}

func TestStates(t *testing.T) {
	assertStates(t, "null", ValueState)
	assertStates(t, "[null]", ArrayState, ArrayState, ValueState)
	assertStates(t, "{\"\":null}", ObjectKeyState, ObjectValueState, ObjectKeyState, ValueState)
}
