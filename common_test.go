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

func assertDimension(t *testing.T, s string, enum int, eunit int) {
	num, unit := Dimension([]byte(s))
	assert.Equal(t, enum, num, "must match number length in "+s)
	assert.Equal(t, eunit, unit, "must match unit length in "+s)
}

func assertInt(t *testing.T, s string, ei int64) {
	i, _ := Int([]byte(s))
	assert.Equal(t, ei, i, "must match integer in "+s)
}

func assertFloat(t *testing.T, s string, ef float64) {
	f, _ := Float([]byte(s))
	assert.Equal(t, ef, f, "must match float in "+s)
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

func TestParseDimension(t *testing.T) {
	assertDimension(t, "5px", 1, 2)
	assertDimension(t, "5px ", 1, 2)
	assertDimension(t, "5%", 1, 1)
	assertDimension(t, "5em", 1, 2)
	assertDimension(t, "px", 0, 0)
	assertDimension(t, "1", 1, 0)
	assertDimension(t, "1~", 1, 0)
}

func TestParseInt(t *testing.T) {
	assertInt(t, "5", 5)
	assertInt(t, "99", 99)
	assertInt(t, "999", 999)
	assertInt(t, "-5", -5)
	assertInt(t, "+5", 5)
	assertInt(t, "9223372036854775807", 9223372036854775807)
	assertInt(t, "9223372036854775808", 0)
	assertInt(t, "-9223372036854775807", -9223372036854775807)
	assertInt(t, "-9223372036854775808", -9223372036854775808)
	assertInt(t, "-9223372036854775809", 0)
	assertInt(t, "18446744073709551620", 0)
	assertInt(t, "a", 0)
}

func TestParseFloat(t *testing.T) {
	assertFloat(t, "5", 5)
	assertFloat(t, "5.1", 5.1)
	assertFloat(t, "-5.1", -5.1)
	assertFloat(t, "5.1e-2", 5.1e-2)
	assertFloat(t, "5.1e+2", 5.1e+2)
	assertFloat(t, "0.0e1", 0.0e1)
	assertFloat(t, "18446744073709551620", 18446744073709551620.0)
	assertFloat(t, "1e23", 1e23)
	// TODO: hard to test due to float imprecision
	// assertFloat(t, "1.7976931348623e+308", 1.7976931348623e+308)
	// assertFloat(t, "4.9406564584124e-308", 4.9406564584124e-308)
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
