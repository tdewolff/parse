package parse // import "github.com/tdewolff/parse"

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertParseNumber(t *testing.T, s string, en int, eok bool) {
	n, ok := ParseNumber([]byte(s))
	assert.Equal(t, eok, ok, "must match ok in "+s)
	assert.Equal(t, en, n, "must match length in "+s)
}

func assertParseDataURI(t *testing.T, x, e1, e2 string, eerr error) {
	s1, s2, err := ParseDataURI([]byte(x))
	assert.Equal(t, eerr, err, "err must match in "+x)
	assert.Equal(t, e1, string(s1), "mediatype part must match in "+x)
	assert.Equal(t, e2, string(s2), "data part must match in "+x)
}

func assertParseQuoteEntity(t *testing.T, s string, equote byte, en int, eok bool) {
	quote, n, ok := ParseQuoteEntity([]byte(s))
	assert.Equal(t, eok, ok, "must match ok in "+s)
	assert.Equal(t, en, n, "must match length in "+s)
	assert.Equal(t, equote, quote, "must match quote in "+s)
}

////////////////////////////////////////////////////////////////

func TestNormalizeContentType(t *testing.T) {
	assert.Equal(t, "text/html", string(NormalizeContentType([]byte("text/html"))))
	assert.Equal(t, "text/html;charset=utf-8", string(NormalizeContentType([]byte("text/html; charset=UTF-8"))))
	assert.Equal(t, "text/html;charset=utf-8;param=\" ; \"", string(NormalizeContentType([]byte("text/html; charset=UTF-8 ; param = \" ; \""))))
	assert.Equal(t, "text/html,text/css", string(NormalizeContentType([]byte("text/html, text/css"))))
}

func TestParseNumber(t *testing.T) {
	assertParseNumber(t, "5", 1, true)
	assertParseNumber(t, "0.51", 4, true)
	assertParseNumber(t, "0.5e-99", 7, true)
	assertParseNumber(t, "0.5e-", 3, true)
	assertParseNumber(t, "+50.0", 5, true)
	assertParseNumber(t, ".0", 2, true)
	assertParseNumber(t, "0.", 1, true)
	assertParseNumber(t, "", 0, false)
	assertParseNumber(t, "+", 0, false)
	assertParseNumber(t, ".", 0, false)
	assertParseNumber(t, "a", 0, false)
}

func TestParseDataURI(t *testing.T) {
	assertParseDataURI(t, "www.domain.com", "", "", ErrBadDataURI)
	assertParseDataURI(t, "data:,", "text/plain", "", nil)
	assertParseDataURI(t, "data:text/xml,", "text/xml", "", nil)
	assertParseDataURI(t, "data:,text", "text/plain", "text", nil)
	assertParseDataURI(t, "data:;base64,dGV4dA==", "text/plain", "text", nil)
	assertParseDataURI(t, "data:image/svg+xml,", "image/svg+xml", "", nil)
	assertParseDataURI(t, "data:;base64,()", "", "", base64.CorruptInputError(0))
}

func TestParseQuoteEntity(t *testing.T) {
	assertParseQuoteEntity(t, "&#34;", '"', 5, true)
	assertParseQuoteEntity(t, "&#039;", '\'', 6, true)
	assertParseQuoteEntity(t, "&#x0022;", '"', 8, true)
	assertParseQuoteEntity(t, "&#x27;", '\'', 6, true)
	assertParseQuoteEntity(t, "&quot;", '"', 6, true)
	assertParseQuoteEntity(t, "&apos;", '\'', 6, true)
	assertParseQuoteEntity(t, "&gt;", 0x00, 0, false)
	assertParseQuoteEntity(t, "&amp;", 0x00, 0, false)
}
