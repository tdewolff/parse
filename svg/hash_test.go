package svg // import "github.com/tdewolff/parse/svg"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTable(t *testing.T) {
	assert.Equal(t, ToHash([]byte("svg")), Svg, "'svg' must resolve to hash.Svg")
	assert.Equal(t, "svg", Svg.String(), "hash.Svg must resolve to 'svg'")
	assert.Equal(t, Hash(0), ToHash([]byte("")), "empty string must resolve to zero")
	assert.Equal(t, "", Hash(0xffffff).String(), "Hash(0xffffff) must resolve to empty string")
}
