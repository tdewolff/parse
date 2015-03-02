package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertParse(t *testing.T, input string, expected string) {
	p, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Error(err)
		return
	}

	b := &bytes.Buffer{}
	p.Serialize(b)

	assert.Equal(t, expected, b.String(), "parsed string must match expected result in "+input)
}

func TestParser(t *testing.T) {
	assertParse(t, "<!-- x:y; -->", "<!--x:y;-->")
	assertParse(t, "color: red;", "color:red;")
	assertParse(t, "color: red; border: 0;", "color:red;border:0;")
	assertParse(t, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	assertParse(t, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")
	assertParse(t, "color: rgb(1,2,3);", "color:rgb(1,2,3);")
	assertParse(t, "@media condition { x:y; .f { z:q; } }", "@media condition {x:y; .f{z:q;}}")

	assertParse(t, "color: red;;", "color:red;")
	assertParse(t, "@import;;", "@import;")
	assertParse(t, "@import;@import;", "@import;@import;")
	assertParse(t, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	assertParse(t, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")

	assertParse(t, "{x:y;}", "{x:y;}")
	assertParse(t, "a{}", "")
	assertParse(t, "a,{x:y;}", "a{x:y;}")
	assertParse(t, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	assertParse(t, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	assertParse(t, "a{x:; z:q;}", "a{z:q;}")
	assertParse(t, "@import { @media f; x:y; }", "@import {@media f; x:y;}")

	assertParse(t, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	assertParse(t, "color:#c0c0c0", "color:#c0c0c0;")
	assertParse(t, "a: b:c(d=1);", "a:b : c(d=1);")
	assertParse(t, "background:URL(x.png);", "background:URL(x.png);")

	// hacks
	assertParse(t, "*zoom:5;", "*zoom:5;")
	assertParse(t, "a{*zoom:5;}", "")

	// coverage
	assertParse(t, "a('';{})['';()]{x:y;}", "a('';{})['';()]{x:y;}")
}
