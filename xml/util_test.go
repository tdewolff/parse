package xml // import "github.com/tdewolff/parse/xml"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertAttrVal(t *testing.T, input, expected string) {
	buf := make([]byte, len(input))
	assert.Equal(t, expected, string(EscapeAttrVal(&buf, []byte(input))))
}

////////////////////////////////////////////////////////////////

func TestAttrVal(t *testing.T) {
	assertAttrVal(t, "xyz", "\"xyz\"")
	assertAttrVal(t, "", "\"\"")
	assertAttrVal(t, "x&amp;z", "\"x&amp;z\"")
	assertAttrVal(t, "x'z", "\"x'z\"")
	assertAttrVal(t, "x\"z", "'x\"z'")
	assertAttrVal(t, "a'b=\"\"", "'a&#39;b=\"\"'")
}
