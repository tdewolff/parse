package parse // import "github.com/tdewolff/parse"

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertDataURI(t *testing.T, x, e1, e2 string, eerr error) {
	s1, s2, err := DataURI([]byte(x))
	assert.Equal(t, eerr, err, "err must match in "+x)
	assert.Equal(t, e1, string(s1), "mediatype part must match in "+x)
	assert.Equal(t, e2, string(s2), "data part must match in "+x)
}

func assertQuoteEntity(t *testing.T, s string, equote byte, en int) {
	quote, n := QuoteEntity([]byte(s))
	assert.Equal(t, en, n, "must match length in "+s)
	assert.Equal(t, equote, quote, "must match quote in "+s)
}

////////////////////////////////////////////////////////////////

func TestParseNumber(t *testing.T) {
	assert.Equal(t, 1, Number([]byte("5")))
	assert.Equal(t, 4, Number([]byte("0.51")))
	assert.Equal(t, 7, Number([]byte("0.5e-99")))
	assert.Equal(t, 3, Number([]byte("0.5e-")))
	assert.Equal(t, 5, Number([]byte("+50.0")))
	assert.Equal(t, 2, Number([]byte(".0")))
	assert.Equal(t, 1, Number([]byte("0.")))
	assert.Equal(t, 0, Number([]byte("")))
	assert.Equal(t, 0, Number([]byte("+")))
	assert.Equal(t, 0, Number([]byte(".")))
	assert.Equal(t, 0, Number([]byte("a")))
}

func TestParseInt(t *testing.T) {
	assert.Equal(t, 5, Int([]byte("5")))
	assert.Equal(t, 99, Int([]byte("99")))
	assert.Equal(t, 990, Int([]byte("0990")))
	assert.Equal(t, -5, Int([]byte("-5")))
}

func TestParseDataURI(t *testing.T) {
	assertDataURI(t, "www.domain.com", "", "", ErrBadDataURI)
	assertDataURI(t, "data:,", "text/plain", "", nil)
	assertDataURI(t, "data:text/xml,", "text/xml", "", nil)
	assertDataURI(t, "data:,text", "text/plain", "text", nil)
	assertDataURI(t, "data:;base64,dGV4dA==", "text/plain", "text", nil)
	assertDataURI(t, "data:image/svg+xml,", "image/svg+xml", "", nil)
	assertDataURI(t, "data:;base64,()", "", "", base64.CorruptInputError(0))
}

func TestParseQuoteEntity(t *testing.T) {
	assertQuoteEntity(t, "&#34;", '"', 5)
	assertQuoteEntity(t, "&#039;", '\'', 6)
	assertQuoteEntity(t, "&#x0022;", '"', 8)
	assertQuoteEntity(t, "&#x27;", '\'', 6)
	assertQuoteEntity(t, "&quot;", '"', 6)
	assertQuoteEntity(t, "&apos;", '\'', 6)
	assertQuoteEntity(t, "&gt;", 0x00, 0)
	assertQuoteEntity(t, "&amp;", 0x00, 0)
}
