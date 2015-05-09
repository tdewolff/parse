package js // import "github.com/tdewolff/parse/js"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/buffer"
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
	assertTokens(t, " \t\v\f\u00A0\uFEFF\u2000") // WhitespaceToken
	assertTokens(t, "\n\r\r\n\u2028\u2029", LineTerminatorToken)
	assertTokens(t, "5.2 .04 0x0F 5e99", NumericToken, NumericToken, NumericToken, NumericToken)
	assertTokens(t, "a = 'string'", IdentifierToken, PunctuatorToken, StringToken)
	assertTokens(t, "/*comment*/ //comment", CommentToken, CommentToken)
	assertTokens(t, "{ } ( ) [ ]", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ". ; , < > <=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">= == != === !==", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "+ - * % ++ --", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "<< >> >>> & | ^", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "! ~ && || ? :", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "= += -= *= %= <<=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, ">>= >>>= &= |= ^=", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "a = /.*/g;", IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken)

	assertTokens(t, "/*co\nm\u2028m/*ent*/ //co//mment\u2029//comment", CommentToken, CommentToken, LineTerminatorToken, CommentToken)
	assertTokens(t, "$ _\u200C \\u2000 \u200C", IdentifierToken, IdentifierToken, IdentifierToken, UnknownToken)
	assertTokens(t, ">>>=>>>>=", PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "/", PunctuatorToken)
	assertTokens(t, "/=", PunctuatorToken)
	assertTokens(t, "010xF", NumericToken, NumericToken, IdentifierToken)
	assertTokens(t, "50e+-0", NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken)
	assertTokens(t, "'str\\i\\'ng'", StringToken)
	assertTokens(t, "'str\\\ni\\\\u00A0ng'", StringToken)
	assertTokens(t, "a = /[a-z/]/g", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a=/=/g1", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a = /'\\\\/\n", IdentifierToken, PunctuatorToken, RegexpToken, LineTerminatorToken)
	assertTokens(t, "a=/\\//g1", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "new RegExp(a + /\\d{1,2}/.source)", IdentifierToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken, PunctuatorToken)

	// early endings
	assertTokens(t, "'string", StringToken)
	assertTokens(t, "'\n '\u2028", UnknownToken, LineTerminatorToken, UnknownToken, LineTerminatorToken)
	assertTokens(t, "'str\\\U00100000ing\\0'", StringToken)
	assertTokens(t, "'strin\\00g'", StringToken, NumericToken, IdentifierToken, StringToken)
	assertTokens(t, "/*comment", CommentToken)
	assertTokens(t, "a=/regexp", IdentifierToken, PunctuatorToken, RegexpToken)

	// coverage
	assertTokens(t, "Ø a〉", IdentifierToken, IdentifierToken, UnknownToken)
	assertTokens(t, "0xg 0.f", NumericToken, IdentifierToken, NumericToken, PunctuatorToken, IdentifierToken)
	assertTokens(t, "\u00A0\uFEFF\u2000")
	assertTokens(t, "\u2028\u2029", LineTerminatorToken)
	assertTokens(t, "\\u0029ident", IdentifierToken)
	assertTokens(t, "\\ugident", UnknownToken, IdentifierToken)
	assertTokens(t, "'str\u2028ing'", UnknownToken, IdentifierToken, LineTerminatorToken, IdentifierToken, StringToken)
	assertTokens(t, "a=/\\\n", IdentifierToken, PunctuatorToken, PunctuatorToken, UnknownToken, LineTerminatorToken)
	assertTokens(t, "a=/x/\u200C\u3009", IdentifierToken, PunctuatorToken, RegexpToken, UnknownToken)

	// small buffer
	buffer.MinBuf = 2
	assertTokens(t, `"*(?:'((?:\\\\.|[^\\\\'])*)'|\"((?:\\\\.|[^\\\\\"])*)\"|("`, StringToken)

	assert.Equal(t, "Whitespace", WhitespaceToken.String())
	assert.Equal(t, "Invalid(100)", TokenType(100).String())
}

////////////////////////////////////////////////////////////////

func ExampleNewTokenizer() {
	p := NewTokenizer(bytes.NewBufferString("var x = 'lorem ipsum';"))
	out := ""
	for {
		tt, data := p.Next()
		if tt == ErrorToken {
			break
		}
		out += string(data)
	}
	fmt.Println(out)
	// Output: var x = 'lorem ipsum';
}
