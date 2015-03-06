package js // import "github.com/tdewolff/parse/js"

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
	for i := 0; i < 10; i++ {
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
	assertTokens(t, " ")
	assertTokens(t, "5.2 .4 0x0F 5e9", NumericToken, NumericToken, NumericToken, NumericToken)
	assertTokens(t, "a = 'string'", IdentifierToken, PunctuatorToken, StringToken)
	assertTokens(t, "/*comment*/ // comment", CommentToken, CommentToken)
	assertTokens(t, "null true false", NullToken, BoolToken, BoolToken)
	assertTokens(t, "{ } ( ) [ ]", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ". ; , < > <=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">= == != === !== /", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "+ - * % ++ --", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "<< >> >>> & | ^", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "! ~ && || ? :", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "= += -= *= %= <<=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">>= >>>= &= |= ^= /=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "a = /.*/g;", IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken)

	assertTokens(t, "a = /[a-z/]/g", IdentifierToken, PunctuatorToken, RegexpToken)
}
