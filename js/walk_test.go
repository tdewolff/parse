package js

import (
	"bytes"
	"testing"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/test"
)

type walker struct{}

func (w *walker) Enter(n INode) IVisitor {
	switch n := n.(type) {
	case *Var:
		if bytes.Equal(n.Data, []byte("x")) {
			n.Data = []byte("obj")
		}
	}

	return w
}

func (w *walker) Exit(n INode) {}

func TestWalk(t *testing.T) {
	js := `
	if (true) {
		for (i = 0; i < 1; i++) {
			x.y = i
		}
	}`

	ast, err := Parse(parse.NewInputString(js))
	if err != nil {
		t.Fatal(err)
	}

	Walk(&walker{}, ast)

	t.Run("TestWalk", func(t *testing.T) {
		test.String(t, ast.Raw(), "if (true) { for (i = 0; i < 1; i++) { obj.y = i; }; }; ")
	})
}
