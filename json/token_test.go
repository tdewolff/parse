package json // import "github.com/tdewolff/parse/json"

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertTokens(t *testing.T, s string, tokentypes ...TokenType) {
	z := NewTokenizer(bytes.NewBufferString(s))
	i := 0
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, z.Err(), "error must be EOF in "+helperStringify(t, s))
			assert.Equal(t, len(tokentypes), i, "when error occurred we must be at the end in "+helperStringify(t, s))
			break
		} else if tt == WhitespaceToken {
			continue
		}
		if i >= len(tokentypes) {
			assert.False(t, i >= len(tokentypes), "index must not exceed tokentypes size in "+helperStringify(t, s))
			break
		}
		if tt != tokentypes[i] {
			assert.Equal(t, tokentypes[i], tt, "tokentypes must match at index "+strconv.Itoa(i)+" in "+helperStringify(t, s))
			break
		}
		i++
	}
	return
}

func helperStringify(t *testing.T, input string) string {
	s := "\n["
	z := NewTokenizer(bytes.NewBufferString(input))
	for i := 0; i < 100; i++ {
		tt, text := z.Next()
		if tt == ErrorToken {
			s += tt.String() + "('" + z.Err().Error() + "')]"
			break
		} else if tt == WhitespaceToken {
			continue
		} else {
			s += tt.String() + "('" + string(text) + "'), "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

func TestTokenizer(t *testing.T) {
	assertTokens(t, " \t\n\r") // WhitespaceToken
	assertTokens(t, "null", LiteralToken)
	assertTokens(t, "[]", StartArrayToken, EndArrayToken)
	assertTokens(t, "[15.2, 0.4, 5e9, -4E-3]", StartArrayToken, NumberToken, NumberToken, NumberToken, NumberToken, EndArrayToken)
	assertTokens(t, "[true, false, null]", StartArrayToken, LiteralToken, LiteralToken, LiteralToken, EndArrayToken)
	assertTokens(t, `["", "abc", "\"", "\\"]`, StartArrayToken, StringToken, StringToken, StringToken, StringToken, EndArrayToken)
	assertTokens(t, "{}", StartObjectToken, EndObjectToken)
	assertTokens(t, `{"a": "b", "c": "d"}`, StartObjectToken, StringToken, StringToken, StringToken, StringToken, EndObjectToken)
	assertTokens(t, `{"a": [1, 2], "b": {"c": 3}}`, StartObjectToken, StringToken, StartArrayToken, NumberToken, NumberToken, EndArrayToken, StringToken, StartObjectToken, StringToken, NumberToken, EndObjectToken, EndObjectToken)
}
