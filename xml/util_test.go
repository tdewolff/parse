package xml // import "github.com/tdewolff/parse/xml"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertAttrVal(t *testing.T, input, expected string) {
	s := []byte(input)
	if len(s) > 1 && (s[0] == '"' || s[0] == '\'') && s[0] == s[len(s)-1] {
		s = s[1 : len(s)-1]
	}
	buf := make([]byte, len(s))
	assert.Equal(t, expected, string(EscapeAttrVal(&buf, []byte(s))))
}

func assertCDATAVal(t *testing.T, input, expected string, eUse bool) {
	s := []byte(input)
	var buf []byte
	text, use := EscapeCDATAVal(&buf, s)
	assert.Equal(t, eUse, use)
	assert.Equal(t, expected, string(text))
}

////////////////////////////////////////////////////////////////

func TestAttrVal(t *testing.T) {
	assertAttrVal(t, "xyz", "\"xyz\"")
	assertAttrVal(t, "", "\"\"")
	assertAttrVal(t, "x&amp;z", "\"x&amp;z\"")
	assertAttrVal(t, "x'z", "\"x'z\"")
	assertAttrVal(t, "x\"z", "'x\"z'")
	assertAttrVal(t, "a'b=\"\"", "'a&#39;b=\"\"'")
	assertAttrVal(t, "'x&#39;\"&#39;z'", "\"x'&#34;'z\"")
	assertAttrVal(t, "\"x&#34;'&#34;z\"", "'x\"&#39;\"z'")
	assertAttrVal(t, "a&#39;b=\"\"", "'a&#39;b=\"\"'")
}

func TestCDATAVal(t *testing.T) {
	assertCDATAVal(t, "<![CDATA[<b>]]>", "&lt;b>", true)
	assertCDATAVal(t, "<![CDATA[abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz]]>", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", true)
	assertCDATAVal(t, "<![CDATA[ <b> ]]>", " &lt;b> ", true)
	assertCDATAVal(t, "<![CDATA[<<<<<]]>", "<![CDATA[<<<<<]]>", false)
	assertCDATAVal(t, "<![CDATA[&]]>", "&amp;", true)
	assertCDATAVal(t, "<![CDATA[&&&&]]>", "<![CDATA[&&&&]]>", false)
	assertCDATAVal(t, "<![CDATA[ a ]]>", " a ", true)
}
