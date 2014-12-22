package css

import (
	"bytes"
	"io"
	"testing"
)

func helperTestTokens(t *testing.T, input string, tokentypes ...TokenType) {
	z := NewTokenizer(bytes.NewBufferString(input))
	i := 0
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			if z.Err() != io.EOF {
				t.Error(z.Err(), helperStringToken(t, input))
			}
			if i < len(tokentypes) {
				t.Error("too few tokens for '"+input+"', expected", len(tokentypes), "!=", i, helperStringToken(t, input))
			}
			break
		} else if tt == WhitespaceToken {
			continue
		}
		if i >= len(tokentypes) {
			t.Error("too many tokens for '"+input+"', expected", len(tokentypes), helperStringToken(t, input))
			break
		}
		if tt != tokentypes[i] {
			t.Error(tt, "!=", tokentypes[i], " for '"+input+"' at token position", i, helperStringToken(t, input))
			break
		}
		i++
	}
	return
}

func helperTestTokenError(t *testing.T, input string, expErr error) {
	z := NewTokenizer(bytes.NewBufferString(input))
	for {
		tt, _ := z.Next()
		if tt == ErrorToken {
			if z.Err() != expErr {
				t.Error(z.Err(), "!=", expErr, "for", string(z.buf), "in", input)
			}
			break
		}
	}
	return
}

func helperStringToken(t *testing.T, input string) string {
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

func helperTestSplit(t *testing.T, s, q string) {
	s1, s2 := SplitDimensionToken(s)
	s = s1 + " " + s2
	if s != q {
		t.Error(s, "!=", q)
	}
}

func TestTokenizer(t *testing.T) {
	helperTestTokens(t, " ")
	helperTestTokens(t, "color: red;", IdentToken, ColonToken, IdentToken, SemicolonToken)
	helperTestTokens(t, "background: url(\"http://x\");", IdentToken, ColonToken, URLToken, SemicolonToken)
	helperTestTokens(t, "color: rgb(4, 0%, 5em);", IdentToken, ColonToken, FunctionToken, NumberToken, CommaToken, PercentageToken, CommaToken, DimensionToken, RightParenthesisToken, SemicolonToken)
	helperTestTokens(t, "body { \"string\" }", IdentToken, LeftBraceToken, StringToken, RightBraceToken)
	helperTestTokens(t, ".class { }", DelimToken, IdentToken, LeftBraceToken, RightBraceToken)
	helperTestTokens(t, "#class { }", HashToken, LeftBraceToken, RightBraceToken)
	helperTestTokens(t, "@media print { }", AtKeywordToken, IdentToken, LeftBraceToken, RightBraceToken)
	helperTestTokens(t, "/*comment*/", CommentToken)
	helperTestTokens(t, "/*com* /ment*/", CommentToken)
	helperTestTokens(t, "~= |= ^= $= *=", IncludeMatchToken, DashMatchToken, PrefixMatchToken, SuffixMatchToken, SubstringMatchToken)
	helperTestTokens(t, "||", ColumnToken)
	helperTestTokens(t, "<!-- -->", CDOToken, CDCToken)
	helperTestTokens(t, "U+1234", UnicodeRangeToken)
	helperTestTokens(t, "5.2 .4", NumberToken, NumberToken)

	// unexpected ending
	helperTestTokens(t, "ident", IdentToken)
	helperTestTokens(t, "123.", NumberToken, DelimToken)
	helperTestTokens(t, "\"string", StringToken)
	helperTestTokens(t, "123/*comment", NumberToken, CommentToken)
	helperTestTokens(t, "U+1-", IdentToken, NumberToken, DelimToken)

	// unicode
	helperTestTokens(t, "fooδbar", IdentToken)
	helperTestTokens(t, "foo\x00bar", IdentToken)
	helperTestTokens(t, "'foo\u554abar'", StringToken)
	helperTestTokens(t, "\\000026B", IdentToken)
	helperTestTokens(t, "\\26 B", IdentToken)

	// hacks
	helperTestTokens(t, `\-\mo\z\-b\i\nd\in\g:\url(//business\i\nfo.co.uk\/labs\/xbl\/xbl\.xml\#xss);`, IdentToken, ColonToken, URLToken, SemicolonToken)
	helperTestTokens(t, "width/**/:/**/ 40em;", IdentToken, CommentToken, ColonToken, CommentToken, DimensionToken, SemicolonToken)
	helperTestTokens(t, ":root *> #quince", ColonToken, IdentToken, DelimToken, DelimToken, HashToken)
	helperTestTokens(t, "html[xmlns*=\"\"]:root", IdentToken, LeftBracketToken, IdentToken, SubstringMatchToken, StringToken, RightBracketToken, ColonToken, IdentToken)
	helperTestTokens(t, "body:nth-of-type(1)", IdentToken, ColonToken, FunctionToken, NumberToken, RightParenthesisToken)
	helperTestTokens(t, "color/*\\**/: blue\\9;", IdentToken, CommentToken, ColonToken, IdentToken, SemicolonToken)
	helperTestTokens(t, "color: blue !ie;", IdentToken, ColonToken, IdentToken, DelimToken, IdentToken, SemicolonToken)

	// coverage
	helperTestTokens(t, "  \n\r\n\"\\\r\n\"", StringToken)
	helperTestTokens(t, "U+?????? U+ABCD?? U+ABC-DEF", UnicodeRangeToken, UnicodeRangeToken, UnicodeRangeToken)
	helperTestTokens(t, "U+? U+A?", IdentToken, DelimToken, DelimToken, IdentToken, DelimToken, IdentToken, DelimToken)
	helperTestTokens(t, "-5.23 -moz", NumberToken, IdentToken)
	helperTestTokens(t, "()", LeftParenthesisToken, RightParenthesisToken)
	helperTestTokens(t, "url( //url  )", URLToken)
	helperTestTokens(t, "url( ", URLToken)
	helperTestTokens(t, "url( //url", URLToken)
	helperTestTokens(t, "url(\")a", URLToken)
	helperTestTokens(t, "url(a')a", BadURLToken, IdentToken)
	helperTestTokens(t, "url(\"\n)a", BadURLToken, IdentToken)
	helperTestTokens(t, "url(a h)a", BadURLToken, IdentToken)
	helperTestTokens(t, "<!- | @4 ## /2", DelimToken, DelimToken, DelimToken, DelimToken, DelimToken, NumberToken, DelimToken, DelimToken, DelimToken, NumberToken)
	helperTestTokens(t, "\"s\\\n\"", StringToken)
	helperTestTokens(t, "\"a\\\"b\"", StringToken)
	helperTestTokens(t, "\"s\n", BadStringToken)
	helperTestTokenError(t, "\\\n", ErrBadEscape)

	// small buffer
	minBuf = 2
	maxBuf = 4
	helperTestTokenError(t, "\"abcd", ErrBufferExceeded)
	helperTestTokenError(t, "ident", ErrBufferExceeded)
	helperTestTokenError(t, "\\ABCD", ErrBufferExceeded)
	helperTestTokenError(t, "/*comment", ErrBufferExceeded)
	helperTestTokenError(t, " \t \t ", ErrBufferExceeded)
	helperTestTokenError(t, "#abcd", ErrBufferExceeded)
	helperTestTokenError(t, "12345", ErrBufferExceeded)
	helperTestTokenError(t, "1.234", ErrBufferExceeded)
	helperTestTokenError(t, "U+ABC", ErrBufferExceeded)
	helperTestTokenError(t, "U+A-B", ErrBufferExceeded)
	helperTestTokenError(t, "U+???", ErrBufferExceeded)
	helperTestTokenError(t, "url((", ErrBufferExceeded)
	helperTestTokenError(t, "id\u554a", ErrBufferExceeded)

	minBuf = 5
	maxBuf = 20
	helperTestTokens(t, "ab,d,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)
	helperTestTokens(t, "ab,cd,e", IdentToken, CommaToken, IdentToken, CommaToken, IdentToken)

	helperTestSplit(t, "5em", "5 em")
	helperTestSplit(t, "-5.01em", "-5.01 em")
	helperTestSplit(t, ".2em", ".2 em")
	helperTestSplit(t, ".2e-51em", ".2e-51 em")
}
