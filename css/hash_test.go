package css // import "github.com/tdewolff/parse/css"

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestHashTable(t *testing.T) {
	test.That(t, ToHash([]byte("font")) == Font, "'font' must resolve to hash.Font")
	test.String(t, Font.String(), "font")
	test.String(t, Margin_Left.String(), "margin-left")
	test.That(t, ToHash([]byte("")) == Hash(0), "empty string must resolve to zero")
	test.String(t, Hash(0xffffff).String(), "")
	test.That(t, ToHash([]byte("fonts")) == Hash(0), "'fonts' must resolve to zero")
}
