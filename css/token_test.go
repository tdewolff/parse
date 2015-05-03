package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/buffer"
)

// Don't implement Bytes() to test for buffer exceeding.
type ReaderMockup struct {
	r io.Reader
}

func (r *ReaderMockup) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

////////////////////////////////////////////////////////////////

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

func assertTokenError(t *testing.T, s string, e error) {
	z := NewTokenizer(&ReaderMockup{bytes.NewBufferString(s)})
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, e, z.Err(), "errors must match in "+s)
			break
		}
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

func assertSplitNumberDimension(t *testing.T, x, e1, e2 string) {
	s1, s2, ok := SplitNumberDimension([]byte(x))
	if !ok && e1 == "" && e2 == "" {
		return
	}
	assert.Equal(t, true, ok, "ok must be true in "+x)
	assert.Equal(t, e1, string(s1), "number part must match in "+x)
	assert.Equal(t, e2, string(s2), "dimension part must match in "+x)
}

func assertSplitDataURI(t *testing.T, x, e1, e2 string, eok bool) {
	s1, s2, ok := SplitDataURI([]byte(x))
	assert.Equal(t, eok, ok, "ok must match in "+x)
	assert.Equal(t, e1, string(s1), "mediatype part must match in "+x)
	assert.Equal(t, e2, string(s2), "data part must match in "+x)
}

////////////////////////////////////////////////////////////////

func TestTokenizer(t *testing.T) {
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
}

func TestTokenizerSmall(t *testing.T) {
	// small buffer
	// buffer.MinBuf = 2
	// buffer.MaxBuf = 4
	// assertTokenError(t, "\"abcd", buffer.ErrExceeded)
	// assertTokenError(t, "ident", buffer.ErrExceeded)
	// assertTokenError(t, "\\ABCD", buffer.ErrExceeded)
	// assertTokenError(t, "/*comment", buffer.ErrExceeded)
	// assertTokenError(t, " \t \t ", buffer.ErrExceeded)
	// assertTokenError(t, "#abcd", buffer.ErrExceeded)
	// assertTokenError(t, "12345", buffer.ErrExceeded)
	// assertTokenError(t, "1.234", buffer.ErrExceeded)
	// assertTokenError(t, "U+ABC", buffer.ErrExceeded)
	// assertTokenError(t, "U+A-B", buffer.ErrExceeded)
	// assertTokenError(t, "U+???", buffer.ErrExceeded)
	// assertTokenError(t, "url((", buffer.ErrExceeded)
	// assertTokenError(t, "id\u554a", buffer.ErrExceeded)

	buffer.MinBuf = 5
	//buffer.MaxBuf = 20
	assertTokens(t, "ab,d,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)
	assertTokens(t, "ab,cd,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)
}

func TestSplitNumberDimension(t *testing.T) {
	assertSplitNumberDimension(t, "5em", "5", "em")
	assertSplitNumberDimension(t, "+5em", "+5", "em")
	assertSplitNumberDimension(t, "-5.01em", "-5.01", "em")
	assertSplitNumberDimension(t, ".2em", ".2", "em")
	assertSplitNumberDimension(t, ".2e-51em", ".2e-51", "em")
	assertSplitNumberDimension(t, "5%", "5", "%")
	assertSplitNumberDimension(t, "5&%", "", "")
}

func TestSplitDataURI(t *testing.T) {
	assertSplitDataURI(t, "url(www.domain.com)", "", "", false)
	assertSplitDataURI(t, "url(data:,)", "text/plain", "", true)
	assertSplitDataURI(t, "url(data:text/xml,)", "text/xml", "", true)
	assertSplitDataURI(t, "url(data:,text)", "text/plain", "text", true)
	assertSplitDataURI(t, "url(data:;base64,dGV4dA==)", "text/plain", "text", true)
	assertSplitDataURI(t, "url(data:image/svg+xml,)", "image/svg+xml", "", true)
}

func TestIsIdent(t *testing.T) {
	assert.True(t, IsIdent([]byte("color")))
	assert.False(t, IsIdent([]byte("4.5")))
}

func TestIsUrlUnquoted(t *testing.T) {
	assert.True(t, IsUrlUnquoted([]byte("http://x")))
	assert.False(t, IsUrlUnquoted([]byte(")")))
}
