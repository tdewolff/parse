package svg // import "github.com/tdewolff/parse/svg"

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestHashTable(t *testing.T) {
	test.That(t, ToHash([]byte("svg")) == Svg, "'svg' must resolve to hash.Svg")
	test.That(t, ToHash([]byte("width")) == Width, "'width' must resolve to hash.Width")
	test.String(t, Svg.String(), "svg")
	test.That(t, ToHash([]byte("")) == Hash(0), "empty string must resolve to zero")
	test.String(t, Hash(0xffffff).String(), "")
	test.That(t, ToHash([]byte("svgs")) == Hash(0), "'svgs' must resolve to zero")
	test.That(t, ToHash([]byte("uopi")) == Hash(0), "'uopi' must resolve to zero")
}
