package css // import "github.com/tdewolff/parse/css"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIdent(t *testing.T) {
	assert.True(t, IsIdent([]byte("color")))
	assert.False(t, IsIdent([]byte("4.5")))
}

func TestIsUrlUnquoted(t *testing.T) {
	assert.True(t, IsUrlUnquoted([]byte("http://x")))
	assert.False(t, IsUrlUnquoted([]byte(")")))
}

func TestHsl2Rgb(t *testing.T) {
	r, g, b := Hsl2Rgb(0.0, 1.0, 0.5)
	assert.Equal(t, r, 1.0)
	assert.Equal(t, g, 0.0)
	assert.Equal(t, b, 0.0)

	r, g, b = Hsl2Rgb(1.0, 1.0, 0.5)
	assert.Equal(t, r, 1.0)
	assert.Equal(t, g, 0.0)
	assert.Equal(t, b, 0.0)

	r, g, b = Hsl2Rgb(0.66, 0.0, 1.0)
	assert.Equal(t, r, 1.0)
	assert.Equal(t, g, 1.0)
	assert.Equal(t, b, 1.0)
}
