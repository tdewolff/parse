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
	assertTokens(t, " \t\v\f\u00A0\uFEFF\u2000") // WhitespaceToken
	assertTokens(t, "\n\r\r\n\u2028\u2029", LineTerminatorToken)
	assertTokens(t, "5.2 .4 0x0F 5e9", NumericToken, NumericToken, NumericToken, NumericToken)
	assertTokens(t, "a = 'string'", IdentifierToken, PunctuatorToken, StringToken)
	assertTokens(t, "/*comment*/ //comment", CommentToken, CommentToken)
	assertTokens(t, "{ } ( ) [ ]", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ". ; , < > <=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">= == != === !== /", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "+ - * % ++ --", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "<< >> >>> & | ^", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "! ~ && || ? :", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "= += -= *= %= <<=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">>= >>>= &= |= ^= /=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "a = /.*/g;", IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken)

	assertTokens(t, "/*co\nmm/*ent*/ //co//mment\n//comment", CommentToken, CommentToken, LineTerminatorToken, CommentToken)
	assertTokens(t, "$ _\u200C \\u2000 \u200C", IdentifierToken, IdentifierToken, IdentifierToken, UnknownToken)
	assertTokens(t, ">>>=>>>>=", PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "/ /=", PunctuatorToken, PunctuatorToken)
	assertTokens(t, "010xF", NumericToken, NumericToken, IdentifierToken)
	assertTokens(t, "50e+-0", NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken)
	assertTokens(t, "'str\\i\\'ng'", StringToken)
	assertTokens(t, "'str\\\ni\\\\u00A0ng'", StringToken)
	assertTokens(t, "a = /[a-z/]/g", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a=/=/g1", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a = /'\\\\/\n", IdentifierToken, PunctuatorToken, RegexpToken, LineTerminatorToken)
}
