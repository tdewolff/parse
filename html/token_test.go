package html // import "github.com/tdewolff/parse/html"

import (
	"bytes"
	"io"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/buffer"
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
	s := ""
	z := NewTokenizer(&ReaderMockup{bytes.NewBufferString(input)})
	for i := 0; i < 10; i++ {
		tt, text := z.Next()
		if tt == ErrorToken {
			s += tt.String() + "('" + z.Err().Error() + "')"
			break
		} else {
			s += tt.String() + "('" + string(text) + "') "
		}
	}
	return s
}

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

func TestTokenizer(t *testing.T) {
	assertTokens(t, "<html></html>", StartTagToken, StartTagCloseToken, EndTagToken)
	assertTokens(t, "<img/>", StartTagToken, StartTagVoidToken)
	assertTokens(t, "<!-- comment -->", CommentToken)
	assertTokens(t, "<p>text</p>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<input type='button'/>", StartTagToken, AttributeToken, StartTagVoidToken)
	assertTokens(t, "<input type='=/>' \r\n\t\f value=\"'\" name=x checked />", StartTagToken, AttributeToken, AttributeToken, AttributeToken, AttributeToken, StartTagVoidToken)
	assertTokens(t, "<!doctype html>", DoctypeToken)
	assertTokens(t, "<?bogus>", CommentToken)
	assertTokens(t, "</0bogus>", CommentToken)
	assertTokens(t, "< ", TextToken)
	assertTokens(t, "</", TextToken)

	// raw
	assertTokens(t, "<title><p></p></title>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<TITLE><p></p></TITLE>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<plaintext></plaintext>", StartTagToken, StartTagCloseToken, TextToken)
	assertTokens(t, "<script></script>", StartTagToken, StartTagCloseToken, EndTagToken)
	assertTokens(t, "<script>var x='</script>';</script>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken, TextToken, EndTagToken)
	assertTokens(t, "<script><!--var x='</script>';--></script>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken, TextToken, EndTagToken)
	assertTokens(t, "<script><!--var x='<script></script>';--></script>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<script><!--var x='<script>';--></script>", StartTagToken, StartTagCloseToken, TextToken, EndTagToken)
	assertTokens(t, "<![CDATA[ test ]]>", TextToken)

	buffer.MinBuf = 4
	assert.Equal(t, "StartTag('ab') StartTagClose('>') Error('EOF')", helperStringify(t, "<ab   >"), "buffer reallocation must keep tagname valid")
}
