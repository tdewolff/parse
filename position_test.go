package parse

import (
	"bytes"
	"fmt"
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
	}{
		{0, "x", 1, 1},
		{1, "xx", 1, 2},
		{2, "x\nx", 2, 1},
		{2, "\n\nx", 3, 1},
		{3, "\nxxx", 2, 3},
		{2, "\r\nx", 2, 1},
		{1, "\rx", 2, 1},
		{3, "\u2028x", 2, 1},
		{3, "\u2029x", 2, 1},

		// edge cases
		{0, "", 1, 1},
		{0, "\nx", 1, 1},
		{1, "\r\nx", 1, 2},
		{-1, "x", 1, 2}, // continue till the end
		{0, "\x00a", 1, 1},
		{1, "x\u2028x", 1, 2},
		{2, "x\u2028x", 1, 3},
		{3, "x\u2028x", 1, 4},
	}
	for _, tt := range newlineTests {
		t.Run(fmt.Sprint(tt.buf, " ", tt.offset), func(t *testing.T) {
			r := bytes.NewBufferString(tt.buf)
			line, col, _ := Position(r, tt.offset)
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
			_, _, context := Position(r, tt.offset)
			i := strings.IndexByte(context, '\n')
			context = context[7:i]
			test.T(t, context, tt.context)
		})
	}
}
