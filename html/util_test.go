package html

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestEscapeAttrVal(t *testing.T) {
	var escapeAttrValTests = []struct {
		attrVal  string
		expected string
	}{
		{`xyz`, `xyz`},
		{``, ``},
		{`x/z`, `x/z`},
		{`x'z`, `"x'z"`},
		{`x"z`, `'x"z'`},
		{`'x"z'`, `'x"z'`},
		{`'x'"'z'`, `"x'&#34;'z"`},
		{`"x"'"z"`, `'x"&#39;"z'`},
		{`"x'z"`, `"x'z"`},
		{`'x'z'`, `"x'z"`},
		{`a'b=""`, `'a&#39;b=""'`},
		{`x<z`, `"x<z"`},
		{`'x"'"z'`, `'x"&#39;"z'`},
	}
	var buf []byte
	for _, tt := range escapeAttrValTests {
		t.Run(tt.attrVal, func(t *testing.T) {
			b := []byte(tt.attrVal)
			var quote byte
			if 0 < len(b) && (b[0] == '\'' || b[0] == '"') {
				quote = b[0]
			}
			if len(b) > 1 && (b[0] == '"' || b[0] == '\'') && b[0] == b[len(b)-1] {
				b = b[1 : len(b)-1]
			}
			val := EscapeAttrVal(&buf, b, quote, false)
			test.String(t, string(val), tt.expected)
		})
	}
}

func TestEscapeAttrValXML(t *testing.T) {
	var escapeAttrValTests = []struct {
		attrVal  string
		expected string
	}{
		{`"xyz"`, `"xyz"`},
		{`'xyz'`, `'xyz'`},
		{`xyz`, `xyz`},
		{``, ``},
	}
	var buf []byte
	for _, tt := range escapeAttrValTests {
		t.Run(tt.attrVal, func(t *testing.T) {
			b := []byte(tt.attrVal)
			var quote byte
			if 0 < len(b) && (b[0] == '\'' || b[0] == '"') {
				quote = b[0]
			}
			if len(b) > 1 && (b[0] == '"' || b[0] == '\'') && b[0] == b[len(b)-1] {
				b = b[1 : len(b)-1]
			}
			val := EscapeAttrVal(&buf, b, quote, true)
			test.String(t, string(val), tt.expected)
		})
	}
}
