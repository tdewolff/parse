package html

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashTable(t *testing.T) {
	assert.Equal(t, "address", Address.String(), "hash.Address must resolve to 'address'")
	assert.Equal(t, "accept-charset", Accept_Charset.String(), "hash.Accept_Charset must resolve to 'accept-charset'")
}