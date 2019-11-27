package parse

import (
	"bytes"
	"testing"

	"github.com/tdewolff/parse/v2/buffer"
	"github.com/tdewolff/test"
)

func TestError(t *testing.T) {
	err := NewError("message", bytes.NewBufferString("buffer"), 3)

	line, column, context := err.Position()
	test.T(t, line, 1, "line")
	test.T(t, column, 4, "column")
	test.T(t, "\n"+context, "\n    1: buffer\n          ^", "context")

	test.T(t, err.Error(), "message on line 1 and column 4\n    1: buffer\n          ^", "error")
}

func TestErrorLexer(t *testing.T) {
	l := buffer.NewLexer(bytes.NewBufferString("buffer"))
	l.Move(3)
	err := NewErrorLexer("message", l)

	line, column, context := err.Position()
	test.T(t, line, 1, "line")
	test.T(t, column, 4, "column")
	test.T(t, "\n"+context, "\n    1: buffer\n          ^", "context")

	test.T(t, err.Error(), "message on line 1 and column 4\n    1: buffer\n          ^", "error")
}
