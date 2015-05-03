package parse // import "github.com/tdewolff/parse"

import (
	"bytes"
	"math/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func helperRand(n, m int, chars []byte) [][]byte {
	r := make([][]byte, n)
	for i := range r {
		for j := 0; j < m; j++ {
			r[i] = append(r[i], chars[rand.Intn(len(chars))])
		}
	}
	return r
}

////////////////////////////////////////////////////////////////

func TestReplaceMultipleWhitespace(t *testing.T) {
	multipleWhitespaceRegexp := regexp.MustCompile("\\s+")
	for _, e := range wsSlices {
		reference := multipleWhitespaceRegexp.ReplaceAll(e, []byte(" "))
		assert.Equal(t, string(reference), string(ReplaceMultiple(e, IsWhitespace, ' ')), "must remove all multiple whitespace")
	}
}

func TestNormalizeContentType(t *testing.T) {
	assert.Equal(t, "text/html", string(NormalizeContentType([]byte("text/html"))))
	assert.Equal(t, "text/html;charset=utf-8", string(NormalizeContentType([]byte("text/html; charset=UTF-8"))))
	assert.Equal(t, "text/html,text/css", string(NormalizeContentType([]byte("text/html, text/css"))))
}

func TestTrim(t *testing.T) {
	assert.Equal(t, "a", string(Trim([]byte("a"), IsWhitespace)))
	assert.Equal(t, "a", string(Trim([]byte(" a"), IsWhitespace)))
	assert.Equal(t, "a", string(Trim([]byte("a "), IsWhitespace)))
	assert.Equal(t, "", string(Trim([]byte(" "), IsWhitespace)))
}

////////////////////////////////////////////////////////////////

var wsSlices [][]byte

func TestMain(t *testing.T) {
	wsSlices = helperRand(100, 20, []byte("abcdefg \n\r\f\t"))
}

func BenchmarkBytesTrim(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			e = bytes.TrimSpace(e)
		}
	}
}

func BenchmarkTrim(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			e = Trim(e, IsWhitespace)
		}
	}
}
