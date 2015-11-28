package js // import "github.com/tdewolff/parse/js"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertTokens(t *testing.T, s string, tokentypes ...TokenType) {
	stringify := helperStringify(t, s)
	l := NewLexer(bytes.NewBufferString(s))
	i := 0
	for {
		tt, _ := l.Next()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, l.Err(), "error must be EOF in "+stringify)
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
	l := NewLexer(bytes.NewBufferString(input))
	for i := 0; i < 10; i++ {
		tt, data := l.Next()
		if tt == ErrorToken {
			s += tt.String() + "('" + l.Err().Error() + "')"
			break
		} else if tt == WhitespaceToken {
			continue
		} else {
			s += tt.String() + "('" + string(data) + "') "
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
	assertTokens(t, ">>= >>>= &= |= ^= =>", PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "a = /.*/g;", IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken)

	assertTokens(t, "/*co\nm\u2028m/*ent*/ //co//mment\u2029//comment", CommentToken, CommentToken, LineTerminatorToken, CommentToken)
	assertTokens(t, "$ _\u200C \\u2000 \u200C", IdentifierToken, IdentifierToken, IdentifierToken, UnknownToken)
	assertTokens(t, ">>>=>>>>=", PunctuatorToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "/", PunctuatorToken)
	assertTokens(t, "/=", PunctuatorToken)
	assertTokens(t, "010xF", NumericToken, NumericToken, IdentifierToken)
	assertTokens(t, "50e+-0", NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken)
	assertTokens(t, "'str\\i\\'ng'", StringToken)
	assertTokens(t, "'str\\\\'abc", StringToken, IdentifierToken)
	assertTokens(t, "'str\\\ni\\\\u00A0ng'", StringToken)
	assertTokens(t, "a = /[a-z/]/g", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a=/=/g1", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "a = /'\\\\/\n", IdentifierToken, PunctuatorToken, RegexpToken, LineTerminatorToken)
	assertTokens(t, "a=/\\//g1", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "new RegExp(a + /\\d{1,2}/.source)", IdentifierToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken, PunctuatorToken)

	assertTokens(t, "0b0101 0o0707 0b17", NumericToken, NumericToken, NumericToken, NumericToken)
	assertTokens(t, "`template`", TemplateToken)
	assertTokens(t, "`a${x+y}b`", TemplateToken, IdentifierToken, PunctuatorToken, IdentifierToken, TemplateToken)
	assertTokens(t, "`temp\nlate`", TemplateToken)

	// early endings
	assertTokens(t, "'string", StringToken)
	assertTokens(t, "'\n '\u2028", UnknownToken, LineTerminatorToken, UnknownToken, LineTerminatorToken)
	assertTokens(t, "'str\\\U00100000ing\\0'", StringToken)
	assertTokens(t, "'strin\\00g'", StringToken)
	assertTokens(t, "/*comment", CommentToken)
	assertTokens(t, "a=/regexp", IdentifierToken, PunctuatorToken, RegexpToken)
	assertTokens(t, "\\u002", UnknownToken, IdentifierToken)

	// coverage
	assertTokens(t, "Ø a〉", IdentifierToken, IdentifierToken, UnknownToken)
	assertTokens(t, "0xg 0.f", NumericToken, IdentifierToken, NumericToken, PunctuatorToken, IdentifierToken)
	assertTokens(t, "0bg 0og", NumericToken, IdentifierToken, NumericToken, IdentifierToken)
	assertTokens(t, "\u00A0\uFEFF\u2000")
	assertTokens(t, "\u2028\u2029", LineTerminatorToken)
	assertTokens(t, "\\u0029ident", IdentifierToken)
	assertTokens(t, "\\u{0029FEF}ident", IdentifierToken)
	assertTokens(t, "\\u{}", UnknownToken, IdentifierToken, PunctuatorToken, PunctuatorToken)
	assertTokens(t, "\\ugident", UnknownToken, IdentifierToken)
	assertTokens(t, "'str\u2028ing'", UnknownToken, IdentifierToken, LineTerminatorToken, IdentifierToken, StringToken)
	assertTokens(t, "a=/\\\n", IdentifierToken, PunctuatorToken, PunctuatorToken, UnknownToken, LineTerminatorToken)
	assertTokens(t, "a=/x/\u200C\u3009", IdentifierToken, PunctuatorToken, RegexpToken, UnknownToken)
	assertTokens(t, "a=/x\n", IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken, LineTerminatorToken)

	// TODO: small buffer
	// buffer.MinBuf = 2
	// assertTokens(t, `"*(?:'((?:\\\\.|[^\\\\'])*)'|\"((?:\\\\.|[^\\\\\"])*)\"|("`, StringToken)

	assert.Equal(t, "Whitespace", WhitespaceToken.String())
	assert.Equal(t, "Invalid(100)", TokenType(100).String())
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("var x = 'lorem ipsum';"))
	out := ""
	for {
		tt, data := l.Next()
		if tt == ErrorToken {
			break
		}
		out += string(data)
		l.Free(len(data))
	}
	fmt.Println(out)
	// Output: var x = 'lorem ipsum';
}
