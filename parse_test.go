package css

import (
	"bytes"
	"testing"
)

func helperTestParseString(t *testing.T, input string, expected string) {
	p, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Error(err)
		return
	}

	if p.String() != expected {
		t.Error(p.String(), "!=", expected)
	}
}

func TestParser(t *testing.T) {
	helperTestParseString(t, "<!-- x:y; -->", "<!--x:y;-->")
	helperTestParseString(t, "color: red;", "color:red;")
	helperTestParseString(t, "color: red; border: 0;", "color:red;border:0;")
	helperTestParseString(t, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	helperTestParseString(t, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")
	helperTestParseString(t, "color: rgb(1,2,3);", "color:rgb(1,2,3);")
	helperTestParseString(t, "@media condition { x:y; .f { z:q; } }", "@media condition {x:y; .f{z:q;}}")

	helperTestParseString(t, "color: red;;", "color:red;")
	helperTestParseString(t, "@import;;", "@import;")
	helperTestParseString(t, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	helperTestParseString(t, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")

	helperTestParseString(t, "{x:y;}", "{x:y;}")
	helperTestParseString(t, "a{}", "")
	helperTestParseString(t, "a,{x:y;}", "a{x:y;}")
	helperTestParseString(t, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	helperTestParseString(t, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	helperTestParseString(t, "a{x:; z:q;}", "a{z:q;}")
	helperTestParseString(t, "@import { @media; x:y; }", "@import {@media; x:y;}")

	helperTestParseString(t, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	helperTestParseString(t, "color:#c0c0c0", "color:#c0c0c0;")
	helperTestParseString(t, "a: b:c(d=1);", "a:b : c(d=1);")
}
