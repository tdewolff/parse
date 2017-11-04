package parse

import (
	"bytes"
	"io"
	"testing"

	"github.com/tdewolff/test"
)

func TestPosition(t *testing.T) {
	var newlineTests = []struct {
		pos  int
		s    string
		line int
		col  int
		err  error
	}{
		{0, "x", 1, 1, nil},
		{1, "xx", 1, 2, nil},
		{2, "x\nx", 2, 1, nil},
		{2, "\n\nx", 3, 1, nil},
		{3, "\nxxx", 2, 3, nil},
		{2, "\r\nx", 2, 1, nil},

		// edge cases
		{0, "", 1, 1, io.EOF},
		{0, "\n", 1, 1, nil},
		{1, "\r\n", 1, 2, nil},
		{-1, "x", 1, 2, io.EOF},
	}
	for _, nt := range newlineTests {
		t.Run(nt.s, func(t *testing.T) {
			r := bytes.NewBufferString(nt.s)
			line, col, err := Pos(r, nt.pos)

			test.Error(t, err, nt.err)
			test.V(t, "line", line, nt.line)
			test.V(t, "column", col, nt.col)
		})
	}
}
