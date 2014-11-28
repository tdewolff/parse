package css

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func helperTokens(t *testing.T, input string, tokentypes ...TokenType) {
	z := NewTokenizer(bytes.NewBufferString(input))
	i := 0
	for {
		tt := z.Next()
		if tt.Type == ErrorToken {
			if z.Err() != io.EOF {
				t.Error(z.Err())
				helperPrint(input)
			}
			if i < len(tokentypes) {
				t.Error("too few tokens for '"+input+"', expected", len(tokentypes), "!=", i)
				helperPrint(input)
				return
			}
			return
		} else if tt.Type == WhitespaceToken {
			continue
		}
		if i >= len(tokentypes) {
			t.Error("too many tokens for '"+input+"', expected", len(tokentypes))
			helperPrint(input)
			return
		}
		if tt.Type != tokentypes[i] {
			t.Error(tt.Type, "!=", tokentypes[i], " for '"+input+"' at token position", i)
			helperPrint(input)
			return
		}
		i++
	}
	return
}

func helperPrint(input string) {
	fmt.Print("[")
	z := NewTokenizer(bytes.NewBufferString(input))
	for {
		tt := z.Next()
		switch tt.Type {
		case ErrorToken:
			fmt.Println("]\n")
			if z.Err() != io.EOF {
				fmt.Println(tt.Type, "line", z.Line(), "in", "'"+tt.Data+"'", z.Err())
			}
			return
		case WhitespaceToken:
		default:
			fmt.Print(tt.Type.String()+"('"+tt.Data+"'), ")
		}
	}
}

func TestTokenizer(t *testing.T) {
	helperTokens(t, "")
	helperTokens(t, "color: red;", IdentToken, ColonToken, IdentToken, SemicolonToken)
	helperTokens(t, "background: url(\"http://x\");", IdentToken, ColonToken, UrlToken, SemicolonToken)
	helperTokens(t, "color: rgb(4, 0%, 5em);", IdentToken, ColonToken, FunctionToken, NumberToken, CommaToken, PercentageToken, CommaToken, DimensionToken, BracketToken, SemicolonToken)
	helperTokens(t, "body { \"string\" }", IdentToken, BracketToken, StringToken, BracketToken)
	helperTokens(t, ".class { }", DelimToken, IdentToken, BracketToken, BracketToken)
	helperTokens(t, "#class { }", HashToken, BracketToken, BracketToken)
	helperTokens(t, "@media print { }", AtKeywordToken, IdentToken, BracketToken, BracketToken)
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
	helperTokens(t, "'foo\u0345bar'", StringToken)
	helperTokens(t, "\\000026B", IdentToken)
	helperTokens(t, "\\26 B", IdentToken)

	// helperTokens(t, "\\\n", DelimToken)
}
