package css // import "github.com/tdewolff/parse/css"

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
		tt, text := l.Next()
		if tt == ErrorToken {
			s += tt.String() + "('" + l.Err().Error() + "')"
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
	assertTokens(t, " ")
	assertTokens(t, "5.2 .4", NumberToken, NumberToken)
	assertTokens(t, "color: red;", IdentToken, ColonToken, IdentToken, SemicolonToken)
	assertTokens(t, "background: url(\"http://x\");", IdentToken, ColonToken, URLToken, SemicolonToken)
	assertTokens(t, "background: URL(x.png);", IdentToken, ColonToken, URLToken, SemicolonToken)
	assertTokens(t, "color: rgb(4, 0%, 5em);", IdentToken, ColonToken, FunctionToken, NumberToken, CommaToken, PercentageToken, CommaToken, DimensionToken, RightParenthesisToken, SemicolonToken)
	assertTokens(t, "body { \"string\" }", IdentToken, LeftBraceToken, StringToken, RightBraceToken)
	assertTokens(t, "body { \"str\\\"ing\" }", IdentToken, LeftBraceToken, StringToken, RightBraceToken)
	assertTokens(t, ".class { }", DelimToken, IdentToken, LeftBraceToken, RightBraceToken)
	assertTokens(t, "#class { }", HashToken, LeftBraceToken, RightBraceToken)
	assertTokens(t, "#class\\#withhash { }", HashToken, LeftBraceToken, RightBraceToken)
	assertTokens(t, "@media print { }", AtKeywordToken, IdentToken, LeftBraceToken, RightBraceToken)
	assertTokens(t, "/*comment*/", CommentToken)
	assertTokens(t, "/*com* /ment*/", CommentToken)
	assertTokens(t, "~= |= ^= $= *=", IncludeMatchToken, DashMatchToken, PrefixMatchToken, SuffixMatchToken, SubstringMatchToken)
	assertTokens(t, "||", ColumnToken)
	assertTokens(t, "<!-- -->", CDOToken, CDCToken)
	assertTokens(t, "U+1234", UnicodeRangeToken)
	assertTokens(t, "5.2 .4 4e-22", NumberToken, NumberToken, NumberToken)

	// unexpected ending
	assertTokens(t, "ident", IdentToken)
	assertTokens(t, "123.", NumberToken, DelimToken)
	assertTokens(t, "\"string", StringToken)
	assertTokens(t, "123/*comment", NumberToken, CommentToken)
	assertTokens(t, "U+1-", IdentToken, NumberToken, DelimToken)

	// unicode
	assertTokens(t, "fooδbar􀀀", IdentToken)
	assertTokens(t, "foo\\æ\\†", IdentToken)
	//assertTokens(t, "foo\x00bar", IdentToken)
	assertTokens(t, "'foo\u554abar'", StringToken)
	assertTokens(t, "\\000026B", IdentToken)
	assertTokens(t, "\\26 B", IdentToken)

	// hacks
	assertTokens(t, `\-\mo\z\-b\i\nd\in\g:\url(//business\i\nfo.co.uk\/labs\/xbl\/xbl\.xml\#xss);`, IdentToken, ColonToken, URLToken, SemicolonToken)
	assertTokens(t, "width/**/:/**/ 40em;", IdentToken, CommentToken, ColonToken, CommentToken, DimensionToken, SemicolonToken)
	assertTokens(t, ":root *> #quince", ColonToken, IdentToken, DelimToken, DelimToken, HashToken)
	assertTokens(t, "html[xmlns*=\"\"]:root", IdentToken, LeftBracketToken, IdentToken, SubstringMatchToken, StringToken, RightBracketToken, ColonToken, IdentToken)
	assertTokens(t, "body:nth-of-type(1)", IdentToken, ColonToken, FunctionToken, NumberToken, RightParenthesisToken)
	assertTokens(t, "color/*\\**/: blue\\9;", IdentToken, CommentToken, ColonToken, IdentToken, SemicolonToken)
	assertTokens(t, "color: blue !ie;", IdentToken, ColonToken, IdentToken, DelimToken, IdentToken, SemicolonToken)

	// escapes, null and replacement character
	assertTokens(t, "c\\\x00olor: white;", IdentToken, ColonToken, IdentToken, SemicolonToken)
	assertTokens(t, "null\\0", IdentToken)
	assertTokens(t, "eof\\", IdentToken)
	assertTokens(t, "\"a\x00b\"", StringToken)
	assertTokens(t, "a\\\x00b", IdentToken)
	assertTokens(t, "url(a\x00b)", BadURLToken) // null character cannot be unquoted
	assertTokens(t, "/*a\x00b*/", CommentToken)

	// coverage
	assertTokens(t, "  \n\r\n\r\"\\\r\n\\\r\"", StringToken)
	assertTokens(t, "U+?????? U+ABCD?? U+ABC-DEF", UnicodeRangeToken, UnicodeRangeToken, UnicodeRangeToken)
	assertTokens(t, "U+? U+A?", IdentToken, DelimToken, DelimToken, IdentToken, DelimToken, IdentToken, DelimToken)
	assertTokens(t, "-5.23 -moz", NumberToken, IdentToken)
	assertTokens(t, "()", LeftParenthesisToken, RightParenthesisToken)
	assertTokens(t, "url( //url  )", URLToken)
	assertTokens(t, "url( ", URLToken)
	assertTokens(t, "url( //url", URLToken)
	assertTokens(t, "url(\")a", URLToken)
	assertTokens(t, "url(a'\\\n)a", BadURLToken, IdentToken)
	assertTokens(t, "url(\"\n)a", BadURLToken, IdentToken)
	assertTokens(t, "url(a h)a", BadURLToken, IdentToken)
	assertTokens(t, "<!- | @4 ## /2", DelimToken, DelimToken, DelimToken, DelimToken, DelimToken, NumberToken, DelimToken, DelimToken, DelimToken, NumberToken)
	assertTokens(t, "\"s\\\n\"", StringToken)
	assertTokens(t, "\"a\\\"b\"", StringToken)
	assertTokens(t, "\"s\n", BadStringToken)
	//assertTokenError(t, "\\\n", ErrBadEscape)

	assert.Equal(t, "Whitespace", WhitespaceToken.String())
	assert.Equal(t, "Empty", EmptyToken.String())
	assert.Equal(t, "Invalid(100)", TokenType(100).String())
	assert.Equal(t, ErrorToken, NewLexer(bytes.NewBufferString("x")).consumeBracket())
}

func TestTokensSmall(t *testing.T) {
	assertTokens(t, "\"abcd", StringToken)
	assertTokens(t, "/*comment", CommentToken)
	assertTokens(t, "U+A-B", UnicodeRangeToken)
	assertTokens(t, "url((", BadURLToken)
	assertTokens(t, "id\u554a", IdentToken)

	buffer.MinBuf = 5
	assertTokens(t, "ab,d,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)
	assertTokens(t, "ab,cd,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("color: red;"))
	out := ""
	for {
		tt, data := l.Next()
		if tt == ErrorToken {
			break
		} else if tt == WhitespaceToken || tt == CommentToken {
			continue
		}
		out += string(data)
	}
	fmt.Println(out)
	// Output: color:red;
}
