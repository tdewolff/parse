package css // import "github.com/tdewolff/parse/css"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertSplitNumberDimension(t *testing.T, x, e1, e2 string) {
	s1, s2, ok := SplitNumberDimension([]byte(x))
	if !ok && e1 == "" && e2 == "" {
		return
	}
	assert.Equal(t, true, ok, "ok must be true in "+x)
	assert.Equal(t, e1, string(s1), "number part must match in "+x)
	assert.Equal(t, e2, string(s2), "dimension part must match in "+x)
}

////////////////////////////////////////////////////////////////

func TestSplitNumberDimension(t *testing.T) {
	assertSplitNumberDimension(t, "5em", "5", "em")
	assertSplitNumberDimension(t, "+5em", "+5", "em")
	assertSplitNumberDimension(t, "-5.01em", "-5.01", "em")
	assertSplitNumberDimension(t, ".2em", ".2", "em")
	assertSplitNumberDimension(t, ".2e-51em", ".2e-51", "em")
	assertSplitNumberDimension(t, "5%", "5", "%")
	assertSplitNumberDimension(t, "5&%", "", "")
}

func TestIsIdent(t *testing.T) {
	assert.True(t, IsIdent([]byte("color")))
	assert.False(t, IsIdent([]byte("4.5")))
}

func TestIsUrlUnquoted(t *testing.T) {
	assert.True(t, IsUrlUnquoted([]byte("http://x")))
	assert.False(t, IsUrlUnquoted([]byte(")")))
}
