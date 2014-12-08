package css

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/kr/pretty"
)

func helperTestParseString(t *testing.T, input string, expected string) {
	p, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Error(err)
		return
	}
	if p.String() != expected {
		t.Error(p.String() + " != " + expected)
	}
}

func helperPrintParse(t *testing.T, input string) {
	p, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%# v\n", pretty.Formatter(p))
}

func TestParser(t *testing.T) {
	helperTestParseString(t, "color: red;", "color:red")
	helperTestParseString(t, "color: red; border: 0;", "[color:red border:0]")
	helperTestParseString(t, "a { color: red; border: 0; }", "a=[color:red border:0]")
	helperTestParseString(t, "a { color: red; border: 0; } b { padding: 0; }", "[a=[color:red border:0] b=padding:0]")

	helperTestParseString(t, "color: red;;", "color:red")
	helperTestParseString(t, "@import;;", "@import")
	helperTestParseString(t, ".a, .b {position:relative;}", "")
}
