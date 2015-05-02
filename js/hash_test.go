package js // import "github.com/tdewolff/parse/js"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTable(t *testing.T) {
	assert.Equal(t, ToHash([]byte("break")), Break, "'break' must resolve to hash.Break")
	assert.Equal(t, "break", Break.String(), "hash.Break must resolve to 'break'")
}
