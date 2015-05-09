package xml // import "github.com/tdewolff/parse/xml"

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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
		}
		assert.False(t, i >= len(tokentypes), "index must not exceed tokentypes size in "+stringify)
		if i < len(tokentypes) {
			assert.Equal(t, tokentypes[i], tt, "tokentypes must match at index "+strconv.Itoa(i)+" in "+stringify)
		}
		i++
	}
	return
}

func assertTags(t *testing.T, s string, tags ...string) {
	stringify := helperStringify(t, s)
	z := NewTokenizer(bytes.NewBufferString(s))
	i := 0
	for {
		tt, data := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, z.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(tags), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == StartTagToken || tt == StartTagPIToken || tt == EndTagToken || tt == DOCTYPEToken {
			assert.False(t, i >= len(tags), "index must not exceed tags size in "+stringify)
			if i < len(tags) {
				assert.Equal(t, tags[i], string(data), "tags must match at index "+strconv.Itoa(i)+" in "+stringify)
				i++
			}
		}
	}
	return
}

func assertAttributes(t *testing.T, s string, attributes ...string) {
	stringify := helperStringify(t, s)
	z := NewTokenizer(bytes.NewBufferString(s))
	i := 0
	for {
		tt, data := z.Next()
		if tt == ErrorToken {
			assert.Equal(t, io.EOF, z.Err(), "error must be EOF in "+stringify)
			assert.Equal(t, len(attributes), i, "when error occurred we must be at the end in "+stringify)
			break
		} else if tt == AttributeToken {
			assert.False(t, i+1 >= len(attributes), "index must not exceed attributes size in "+stringify)
			if i+1 < len(attributes) {
				assert.Equal(t, attributes[i], string(data), "attribute keys must match at index "+strconv.Itoa(i)+" in "+stringify)
				assert.Equal(t, attributes[i+1], string(z.AttrVal()), "attribute values must match at index "+strconv.Itoa(i)+" in "+stringify)
				i += 2
			}
		}
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
		} else if tt == AttributeToken {
			s += tt.String() + "('" + string(text) + "=" + string(z.AttrVal()) + "') "
		} else {
			s += tt.String() + "('" + string(text) + "') "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

func TestTokens(t *testing.T) {
	assertTokens(t, "")
	assertTokens(t, "<!-- comment -->", CommentToken)
	assertTokens(t, "<!-- comment \n multi \r line -->", CommentToken)
	assertTokens(t, "<foo/>", StartTagToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo \t\r\n/>", StartTagToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo:bar.qux-norf/>", StartTagToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo></foo>", StartTagToken, StartTagCloseToken, EndTagToken)
	assertTokens(t, "<foo>text</foo>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<foo/> text", StartTagToken, StartTagCloseVoidToken, TextToken)
	assertTokens(t, "<a> <b> <c>text</c> </b> </a>", StartTagToken, StartTagCloseToken, TextToken, StartTagToken, StartTagCloseToken, TextToken, StartTagToken, StartTagCloseToken, TextToken, EndTagToken, TextToken, EndTagToken, TextToken, EndTagToken)
	assertTokens(t, "<foo a='a' b=\"b\" c=c/>", StartTagToken, AttributeToken, AttributeToken, AttributeToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo a=\"\"/>", StartTagToken, AttributeToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo a-b=\"\"/>", StartTagToken, AttributeToken, StartTagCloseVoidToken)
	assertTokens(t, "<foo \nchecked \r\n value\r=\t'=/>\"' />", StartTagToken, AttributeToken, AttributeToken, StartTagCloseVoidToken)
	assertTokens(t, "<?xml?>", StartTagPIToken, StartTagClosePIToken)
	assertTokens(t, "<?xml a=\"a\" ?>", StartTagPIToken, AttributeToken, StartTagClosePIToken)
	assertTokens(t, "<?xml a=a?>", StartTagPIToken, AttributeToken, StartTagClosePIToken)
	assertTokens(t, "<![CDATA[ test ]]>", CDATAToken)
	assertTokens(t, "<!DOCTYPE>", DOCTYPEToken)
	assertTokens(t, "<!DOCTYPE note SYSTEM \"Note.dtd\">", DOCTYPEToken)
	assertTokens(t, `<!DOCTYPE note [<!ENTITY nbsp "&#xA0;"><!ENTITY writer "Writer: Donald Duck."><!ENTITY copyright "Copyright:]> W3Schools.">]>`, DOCTYPEToken)
	assertTokens(t, "<!foo>", StartTagToken, StartTagCloseToken)

	// early endings
	assertTokens(t, "<!-- comment", CommentToken)
	assertTokens(t, "<foo", StartTagToken)
	assertTokens(t, "</foo", EndTagToken)
	assertTokens(t, "<foo x", StartTagToken, AttributeToken)
	assertTokens(t, "<foo x=", StartTagToken, AttributeToken)
	assertTokens(t, "<foo x='", StartTagToken, AttributeToken)
	assertTokens(t, "<foo x=''", StartTagToken, AttributeToken)
	assertTokens(t, "<?xml", StartTagPIToken)
	assertTokens(t, "<![CDATA[ test", CDATAToken)
	assertTokens(t, "<!DOCTYPE note SYSTEM", DOCTYPEToken)

	assert.Equal(t, "Invalid(100)", TokenType(100).String())
}

func TestTags(t *testing.T) {
	assertTags(t, "<foo:bar.qux-norf/>", "foo:bar.qux-norf")
	assertTags(t, "<?xml?>", "xml")
	assertTags(t, "<foo?bar/qux>", "foo?bar/qux")
	assertTags(t, "<!DOCTYPE note SYSTEM \"Note.dtd\">", "note SYSTEM \"Note.dtd\"")

	// early endings
	assertTags(t, "<foo ", "foo")
}

func TestAttributes(t *testing.T) {
	assertAttributes(t, "<foo a=\"b\" />", "a", "\"b\"")
	assertAttributes(t, "<foo \nchecked \r\n value\r=\t'=/>\"' />", "checked", "", "value", "'=/>\"'")
	assertAttributes(t, "<foo bar=\" a \n\t\r b \" />", "bar", "\" a     b \"")
	assertAttributes(t, "<?xml a=b?>", "a", "b")
	assertAttributes(t, "<foo /=? >", "/", "?")

	// early endings
	assertAttributes(t, "<foo x", "x", "")
	assertAttributes(t, "<foo x=", "x", "")
	assertAttributes(t, "<foo x='", "x", "'")
}
