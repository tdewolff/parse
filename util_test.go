package parse // import "github.com/tdewolff/parse"

import (
	"math/rand"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func helperRand(n, m int, chars []byte) []string {
	r := make([]string, n)
	for i := range r {
		for j := 0; j < m; j++ {
			r[i] += string(chars[rand.Intn(len(chars))])
		}
	}
	return r
}

func TestReplaceMultipleWhitespace(t *testing.T) {
	multipleWhitespaceRegexp := regexp.MustCompile("\\s+")
	array := helperRand(100, 20, []byte("abcdefg \n\r\f\t"))
	for _, e := range array {
		reference := multipleWhitespaceRegexp.ReplaceAll([]byte(e), []byte(" "))
		assert.Equal(t, string(reference), string(ReplaceMultiple([]byte(e), IsWhitespace, ' ')), "must remove all multiple whitespace")
	}
}

func TestNormalizeContentType(t *testing.T) {
	assert.Equal(t, "text/html", string(NormalizeContentType([]byte("text/html"))))
	assert.Equal(t, "text/html;charset=utf-8", string(NormalizeContentType([]byte("text/html; charset=UTF-8"))))
	assert.Equal(t, "text/html,text/css", string(NormalizeContentType([]byte("text/html, text/css"))))
}
