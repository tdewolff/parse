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
