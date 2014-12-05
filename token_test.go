package css

import (
	"bytes"
	"io"
	"testing"
)

func helperTokens(t *testing.T, input string, tokentypes ...TokenType) {
	z := NewTokenizer(bytes.NewBufferString(input))
	i := 0
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			if z.Err() != io.EOF {
				t.Error(z.Err(), helperString(t, input))
			}
			if i < len(tokentypes) {
				t.Error("too few tokens for '"+input+"', expected", len(tokentypes), "!=", i, helperString(t, input))
			}
			break
		} else if tt == WhitespaceToken {
			continue
		}
		if i >= len(tokentypes) {
			t.Error("too many tokens for '"+input+"', expected", len(tokentypes), helperString(t, input))
			break
		}
		if tt != tokentypes[i] {
			t.Error(tt, "!=", tokentypes[i], " for '"+input+"' at token position", i, helperString(t, input))
			break
		}
		i++
	}
	return
}

func helperTest(t *testing.T, input string, expErr error) {
	z := NewTokenizer(bytes.NewBufferString(input))
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			if z.Err() != expErr {
				t.Error(z.Err(), "!=", expErr, "for", string(z.buf))
			}
			break
		}
	}
	return
}

func helperString(t *testing.T, input string) string {
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

func TestTokenizer(t *testing.T) {
	helperTokens(t, " ")
	helperTokens(t, "color: red;", IdentToken, ColonToken, IdentToken, SemicolonToken)
	helperTokens(t, "background: url(\"http://x\");", IdentToken, ColonToken, URLToken, SemicolonToken)
	helperTokens(t, "color: rgb(4, 0%, 5em);", IdentToken, ColonToken, FunctionToken, NumberToken, CommaToken, PercentageToken, CommaToken, DimensionToken, RightParenthesisToken, SemicolonToken)
	helperTokens(t, "body { \"string\" }", IdentToken, LeftBraceToken, StringToken, RightBraceToken)
	helperTokens(t, ".class { }", DelimToken, IdentToken, LeftBraceToken, RightBraceToken)
	helperTokens(t, "#class { }", HashToken, LeftBraceToken, RightBraceToken)
	helperTokens(t, "@media print { }", AtKeywordToken, IdentToken, LeftBraceToken, RightBraceToken)
	helperTokens(t, "/*comment*/", CommentToken)
	helperTokens(t, "~= |= ^= $= *=", IncludeMatchToken, DashMatchToken, PrefixMatchToken, SuffixMatchToken, SubstringMatchToken)
	helperTokens(t, "||", ColumnToken)
	helperTokens(t, "<!-- -->", CDOToken, CDCToken)
	helperTokens(t, "U+1234", UnicodeRangeToken)
	helperTokens(t, "5.2 .4", NumberToken, NumberToken)

	// unexpected ending
	helperTokens(t, "ident", IdentToken)
	helperTokens(t, "123.", NumberToken, DelimToken)
	helperTokens(t, "\"string", StringToken)
	helperTokens(t, "123/*comment", NumberToken, CommentToken)
	helperTokens(t, "U+1-", IdentToken, NumberToken, DelimToken)

	// unicode
	helperTokens(t, "fooδbar", IdentToken)
	helperTokens(t, "foo\x00bar", IdentToken)
	helperTokens(t, "'foo\u554abar'", StringToken)
	helperTokens(t, "\\000026B", IdentToken)
	helperTokens(t, "\\26 B", IdentToken)

	// hacks
	helperTokens(t, `\-\mo\z\-b\i\nd\in\g:\url(//business\i\nfo.co.uk\/labs\/xbl\/xbl\.xml\#xss);`, IdentToken, ColonToken, URLToken, SemicolonToken)
	helperTokens(t, "width/**/:/**/ 40em;", IdentToken, CommentToken, ColonToken, CommentToken, DimensionToken, SemicolonToken)
	helperTokens(t, ":root *> #quince", ColonToken, IdentToken, DelimToken, DelimToken, HashToken)
	helperTokens(t, "html[xmlns*=\"\"]:root", IdentToken, LeftBracketToken, IdentToken, SubstringMatchToken, StringToken, RightBracketToken, ColonToken, IdentToken)
	helperTokens(t, "body:nth-of-type(1)", IdentToken, ColonToken, FunctionToken, NumberToken, RightParenthesisToken)
	helperTokens(t, "color/*\\**/: blue\\9;", IdentToken, CommentToken, ColonToken, IdentToken, SemicolonToken)
	helperTokens(t, "color: blue !ie;", IdentToken, ColonToken, IdentToken, DelimToken, IdentToken, SemicolonToken)

	// coverage
	helperTokens(t, "  \n\r\n\"\\\r\n\"", StringToken)
	helperTokens(t, "U+?????? U+ABCD?? U+ABC-DEF", UnicodeRangeToken, UnicodeRangeToken, UnicodeRangeToken)
	helperTokens(t, "U+? U+A?", IdentToken, DelimToken, DelimToken, IdentToken, DelimToken, IdentToken, DelimToken)
	helperTokens(t, "-5.23 -moz", NumberToken, IdentToken)
	helperTokens(t, "url( //url  )", URLToken)
	helperTokens(t, "url( ", URLToken)
	helperTokens(t, "url( //url", URLToken)
	helperTokens(t, "url(\")a", URLToken)
	helperTokens(t, "url(a')a", BadURLToken, IdentToken)
	helperTokens(t, "url(\"\n)a", BadURLToken, IdentToken)
	helperTokens(t, "url(a h)a", BadURLToken, IdentToken)
	helperTokens(t, "<!- | @4 ## /2", DelimToken, DelimToken, DelimToken, DelimToken, DelimToken, NumberToken, DelimToken, DelimToken, DelimToken, NumberToken)
	helperTokens(t, "\"s\\\n\"", StringToken)
	helperTokens(t, "\"a\\\"b\"", StringToken)
	helperTokens(t, "\"s\n", BadStringToken)
	helperTest(t, "\\\n", ErrBadEscape)

	// small buffer
	minBuf = 2
	maxBuf = 4
	helperTest(t, "\"abcd", ErrBufferExceeded)
	helperTest(t, "ident", ErrBufferExceeded)
	helperTest(t, "\\ABCD", ErrBufferExceeded)
	helperTest(t, "/*comment", ErrBufferExceeded)
	helperTest(t, " \t \t ", ErrBufferExceeded)
	helperTest(t, "#abcd", ErrBufferExceeded)
	helperTest(t, "12345", ErrBufferExceeded)
	helperTest(t, "1.234", ErrBufferExceeded)
	helperTest(t, "U+ABC", ErrBufferExceeded)
	helperTest(t, "U+A-B", ErrBufferExceeded)
	helperTest(t, "U+???", ErrBufferExceeded)
	helperTest(t, "url((", ErrBufferExceeded)
	helperTest(t, "id\u554a", ErrBufferExceeded)
}
