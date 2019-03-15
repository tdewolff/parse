package parse

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestPosition(t *testing.T) {
	var newlineTests = []struct {
		offset int
		buf    string
		line   int
		col    int
		err    error
	}{
		{0, "x", 1, 1, nil},
		{1, "xx", 1, 2, nil},
		{2, "x\nx", 2, 1, nil},
		{2, "\n\nx", 3, 1, nil},
		{3, "\nxxx", 2, 3, nil},
		{2, "\r\nx", 2, 1, nil},
		{1, "\rx", 2, 1, nil},

		// edge cases
		{0, "", 1, 1, io.EOF},
		{0, "\n", 1, 1, nil},
		{1, "\r\n", 1, 2, nil},
		{-1, "x", 1, 2, io.EOF}, // continue till the end
		{0, "\x00a", 1, 1, io.EOF},
	}
	for _, tt := range newlineTests {
		t.Run(fmt.Sprint(tt.buf, " ", tt.offset), func(t *testing.T) {
			r := bytes.NewBufferString(tt.buf)
			line, col, _, err := Position(r, tt.offset)
			test.T(t, err, tt.err)
			test.T(t, line, tt.line, "line")
			test.T(t, col, tt.col, "column")
		})
	}
}

func TestPositionContext(t *testing.T) {
	var newlineTests = []struct {
		offset  int
		buf     string
		context string
	}{
		{10, "01234567890123456789012345678901234567890123456789012345678901234567890123456789", "012345678901234567890123456789012345678901234567890123456..."}, // 80 characters -> 60 characters
		{40, "01234567890123456789012345678901234567890123456789012345678901234567890123456789", "...01234567890123456789012345678901234567890..."},
		{60, "012345678901234567890123456789012345678901234567890123456789012345678901234567890", "...78901234567890123456789012345678901234567890"},
		{60, "012345678901234567890123456789012345678901234567890123456789012345678901234567890123", "...01234567890123456789012345678901234567890123"},
		{60, "0123456789012345678901234567890123456789012345678901234567890123456789012345678901234", "...01234567890123456789012345678901234567890..."},
	}
	for _, tt := range newlineTests {
		t.Run(fmt.Sprint(tt.buf, " ", tt.offset), func(t *testing.T) {
			r := bytes.NewBufferString(tt.buf)
			_, _, context, _ := Position(r, tt.offset)
			i := strings.IndexByte(context, '\n')
			context = context[7:i]
			test.T(t, context, tt.context)
		})
	}
}
