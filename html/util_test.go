package html // import "github.com/tdewolff/parse/html"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertAttrVal(t *testing.T, input, expected string) {
	s := []byte(input)
	orig := s
	if len(s) > 1 && (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		s = s[1 : len(s)-1]
	}
	buf := make([]byte, len(s))
	assert.Equal(t, expected, string(EscapeAttrVal(&buf, orig, s)))
}

////////////////////////////////////////////////////////////////

func TestAttrVal(t *testing.T) {
	assertAttrVal(t, "xyz", "xyz")
	assertAttrVal(t, "", "")
	assertAttrVal(t, "x&amp;z", "x&amp;z")
	assertAttrVal(t, "x/z", "x/z")
	assertAttrVal(t, "x'z", "\"x'z\"")
	assertAttrVal(t, "x\"z", "'x\"z'")
	assertAttrVal(t, "'x\"z'", "'x\"z'")
	assertAttrVal(t, "'x&#39;\"&#39;z'", "\"x'&#34;'z\"")
	assertAttrVal(t, "\"x&#34;'&#34;z\"", "'x\"&#39;\"z'")
	assertAttrVal(t, "\"x&#x27;z\"", "\"x'z\"")
	assertAttrVal(t, "'x&#x00022;z'", "'x\"z'")
	assertAttrVal(t, "'x\"&gt;'", "'x\"&gt;'")
	assertAttrVal(t, "You&#039;re encouraged to log in; however, it&#039;s not mandatory. [o]", "\"You're encouraged to log in; however, it's not mandatory. [o]\"")
}
