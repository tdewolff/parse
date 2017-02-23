package html // import "github.com/tdewolff/parse/html"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/tdewolff/test"
)

func helperStringify(t *testing.T, input string) string {
	s := ""
	l := NewLexer(bytes.NewBufferString(input))
	for i := 0; i < 10; i++ {
		tt, data := l.Next()
		if tt == ErrorToken {
			if l.Err() != nil {
				s += tt.String() + "('" + l.Err().Error() + "')"
			} else {
				s += tt.String() + "(nil)"
			}
			break
		} else if tt == AttributeToken {
			s += tt.String() + "('" + string(data) + "=" + string(l.AttrVal()) + "') "
		} else {
			s += tt.String() + "('" + string(data) + "') "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

type TTs []TokenType

func TestTokens(t *testing.T) {
	var tokenTests = []struct {
		html     string
		expected []TokenType
	}{
		{"<html></html>", TTs{StartTagToken, StartTagCloseToken, EndTagToken}},
		{"<img/>", TTs{StartTagToken, StartTagVoidToken}},
		{"<!-- comment -->", TTs{CommentToken}},
		{"<!-- comment --!>", TTs{CommentToken}},
		{"<p>text</p>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken}},
		{"<input type='button'/>", TTs{StartTagToken, AttributeToken, StartTagVoidToken}},
		{"<input  type='button'  value=''/>", TTs{StartTagToken, AttributeToken, AttributeToken, StartTagVoidToken}},
		{"<input type='=/>' \r\n\t\f value=\"'\" name=x checked />", TTs{StartTagToken, AttributeToken, AttributeToken, AttributeToken, AttributeToken, StartTagVoidToken}},
		{"<!doctype>", TTs{DoctypeToken}},
		{"<!doctype html>", TTs{DoctypeToken}},
		{"<?bogus>", TTs{CommentToken}},
		{"</0bogus>", TTs{CommentToken}},
		{"<!bogus>", TTs{CommentToken}},
		{"< ", TTs{TextToken}},
		{"</", TTs{TextToken}},

		// raw tags
		{"<title><p></p></title>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken}},
		{"<TITLE><p></p></TITLE>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken}},
		{"<plaintext></plaintext>", TTs{StartTagToken, StartTagCloseToken, TextToken}},
		{"<script></script>", TTs{StartTagToken, StartTagCloseToken, EndTagToken}},
		{"<script>var x='</script>';</script>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken, TextToken, EndTagToken}},
		{"<script><!--var x='</script>';--></script>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken, TextToken, EndTagToken}},
		{"<script><!--var x='<script></script>';--></script>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken}},
		{"<script><!--var x='<script>';--></script>", TTs{StartTagToken, StartTagCloseToken, TextToken, EndTagToken}},
		{"<![CDATA[ test ]]>", TTs{TextToken}},
		{"<svg>text</svg>", TTs{SvgToken}},
		{"<math>text</math>", TTs{MathToken}},
		{`<svg>text<x a="</svg>"></x></svg>`, TTs{SvgToken}},
		{"<a><svg>text</svg></a>", TTs{StartTagToken, StartTagCloseToken, SvgToken, EndTagToken}},

		// early endings
		{"<!-- comment", TTs{CommentToken}},
		{"<? bogus comment", TTs{CommentToken}},
		{"<foo", TTs{StartTagToken}},
		{"</foo", TTs{EndTagToken}},
		{"<foo x", TTs{StartTagToken, AttributeToken}},
		{"<foo x=", TTs{StartTagToken, AttributeToken}},
		{"<foo x='", TTs{StartTagToken, AttributeToken}},
		{"<foo x=''", TTs{StartTagToken, AttributeToken}},
		{"<!DOCTYPE note SYSTEM", TTs{DoctypeToken}},
		{"<![CDATA[ test", TTs{TextToken}},
		{"<script>", TTs{StartTagToken, StartTagCloseToken}},
		{"<script><!--", TTs{StartTagToken, StartTagCloseToken, TextToken}},
		{"<script><!--var x='<script></script>';-->", TTs{StartTagToken, StartTagCloseToken, TextToken}},

		// go-fuzz
		{"</>", TTs{EndTagToken}},
	}
	for _, tt := range tokenTests {
		stringify := helperStringify(t, tt.html)
		l := NewLexer(bytes.NewBufferString(tt.html))
		i := 0
		for {
			token, _ := l.Next()
			if token == ErrorToken {
				test.That(t, i == len(tt.expected), "when error occurred we must be at the end in "+stringify)
				test.Error(t, l.Err(), io.EOF, "in "+stringify)
				break
			}
			test.That(t, i < len(tt.expected), "index", i, "must not exceed expected token types size", len(tt.expected), "in "+stringify)
			if i < len(tt.expected) {
				test.That(t, token == tt.expected[i], "token types must match at index "+strconv.Itoa(i)+" in "+stringify)
			}
			i++
		}
	}

	test.String(t, TokenType(100).String(), "Invalid(100)")
}

func TestTags(t *testing.T) {
	var tagTests = []struct {
		html     string
		expected string
	}{
		{"<foo:bar.qux-norf/>", "foo:bar.qux-norf"},
		{"<foo?bar/qux>", "foo?bar/qux"},
		{"<!DOCTYPE note SYSTEM \"Note.dtd\">", " note SYSTEM \"Note.dtd\""},
		{"</foo >", "foo"},

		// early endings
		{"<foo ", "foo"},
	}
	for _, tt := range tagTests {
		stringify := helperStringify(t, tt.html)
		l := NewLexer(bytes.NewBufferString(tt.html))
		for {
			token, _ := l.Next()
			if token == ErrorToken {
				test.That(t, false, "when error occurred we must be at the end in "+stringify)
				test.Error(t, l.Err(), io.EOF, "in "+stringify)
				break
			} else if token == StartTagToken || token == EndTagToken || token == DoctypeToken {
				test.String(t, string(l.Text()), tt.expected, "in "+stringify)
				break
			}
		}
	}
}

func TestAttributes(t *testing.T) {
	var attributeTests = []struct {
		attr     string
		expected []string
	}{
		{"<foo a=\"b\" />", []string{"a", "\"b\""}},
		{"<foo \nchecked \r\n value\r=\t'=/>\"' />", []string{"checked", "", "value", "'=/>\"'"}},
		{"<foo bar=\" a \n\t\r b \" />", []string{"bar", "\" a \n\t\r b \""}},
		{"<foo a/>", []string{"a", ""}},
		{"<foo /=/>", []string{"/", "/"}},

		// early endings
		{"<foo x", []string{"x", ""}},
		{"<foo x=", []string{"x", ""}},
		{"<foo x='", []string{"x", "'"}},
	}
	for _, tt := range attributeTests {
		stringify := helperStringify(t, tt.attr)
		l := NewLexer(bytes.NewBufferString(tt.attr))
		i := 0
		for {
			token, _ := l.Next()
			if token == ErrorToken {
				test.That(t, i == len(tt.expected), "when error occurred we must be at the end in "+stringify)
				test.Error(t, l.Err(), io.EOF, "in "+stringify)
				break
			} else if token == AttributeToken {
				test.That(t, i+1 < len(tt.expected), "index", i+1, "must not exceed expected attributes size", len(tt.expected), "in "+stringify)
				if i+1 < len(tt.expected) {
					test.String(t, string(l.Text()), tt.expected[i], "attribute keys must match at index "+strconv.Itoa(i)+" in "+stringify)
					test.String(t, string(l.AttrVal()), tt.expected[i+1], "attribute keys must match at index "+strconv.Itoa(i)+" in "+stringify)
					i += 2
				}
			}
		}
	}
}

func TestErrors(t *testing.T) {
	var errorTests = []struct {
		html string
		err  error
	}{
		{"a\x00b", ErrBadNull},
	}
	for _, tt := range errorTests {
		stringify := helperStringify(t, tt.html)
		l := NewLexer(bytes.NewBufferString(tt.html))
		for {
			token, _ := l.Next()
			if token == ErrorToken {
				test.Error(t, l.Err(), tt.err, "in "+stringify)
				break
			}
		}
	}
}

////////////////////////////////////////////////////////////////

var J int
var ss = [][]byte{
	[]byte(" style"),
	[]byte("style"),
	[]byte(" \r\n\tstyle"),
	[]byte("      style"),
	[]byte(" x"),
	[]byte("x"),
}

func BenchmarkWhitespace1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, s := range ss {
			j := 0
			for {
				if c := s[j]; c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
					j++
				} else {
					break
				}
			}
			J += j
		}
	}
}

func BenchmarkWhitespace2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, s := range ss {
			j := 0
			for {
				if c := s[j]; c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
					j++
					continue
				}
				break
			}
			J += j
		}
	}
}

func BenchmarkWhitespace3(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, s := range ss {
			j := 0
			for {
				if c := s[j]; c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f' {
					break
				}
				j++
			}
			J += j
		}
	}
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("<span class='user'>John Doe</span>"))
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
	// Output: <span class='user'>John Doe</span>
}
