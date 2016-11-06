package js // import "github.com/tdewolff/parse/js"

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestHashTable(t *testing.T) {
	test.That(t, ToHash([]byte("break")) == Break, "'break' must resolve to hash.Break")
	test.That(t, ToHash([]byte("var")) == Var, "'var' must resolve to hash.Var")
	test.String(t, Break.String(), "break")
	test.That(t, ToHash([]byte("")) == Hash(0), "empty string must resolve to zero")
	test.String(t, Hash(0xffffff).String(), "")
	test.That(t, ToHash([]byte("breaks")) == Hash(0), "'breaks' must resolve to zero")
	test.That(t, ToHash([]byte("sdf")) == Hash(0), "'sdf' must resolve to zero")
	test.That(t, ToHash([]byte("uio")) == Hash(0), "'uio' must resolve to zero")
}
