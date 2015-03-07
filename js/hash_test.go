package js

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTable(t *testing.T) {
	assert.Equal(t, ToHash([]byte("break")), Font, "'break' must resolve to hash.Break")
	assert.Equal(t, "break", Break.String(), "hash.Break must resolve to 'break'")
}
