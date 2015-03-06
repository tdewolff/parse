package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTable(t *testing.T) {
	assert.Equal(t, ToHash([]byte("font")), Font, "'font' must resolve to hash.Font")
	assert.Equal(t, "font", Font.String(), "hash.Font must resolve to 'font'")
	assert.Equal(t, "margin-left", Margin_Left.String(), "hash.Margin_Left must resolve to 'margin-left'")
}
