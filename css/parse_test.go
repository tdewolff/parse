package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/test"
)

func assertParse(t *testing.T, isInline bool, input, expected string) {
	output := ""
	p := NewParser(bytes.NewBufferString(input), isInline)
	for {
		gt, _, data := p.Next()
		if gt == ErrorGrammar {
			err := p.Err()
			if err != nil {
				assert.Equal(t, io.EOF, err, "parser must not return error '"+err.Error()+"' in "+input)
			}
			break
		} else if gt == AtRuleGrammar || gt == BeginAtRuleGrammar || gt == BeginRulesetGrammar || gt == DeclarationGrammar {
			if gt == DeclarationGrammar {
				data = append(data, ":"...)
			}
			for _, val := range p.Values() {
				data = append(data, val.Data...)
			}
			if gt == BeginAtRuleGrammar || gt == BeginRulesetGrammar {
				data = append(data, "{"...)
			} else if gt == AtRuleGrammar || gt == DeclarationGrammar {
				data = append(data, ";"...)
			}
		}
		output += string(data)
	}
	assert.Equal(t, expected, output, "parsed string must match expected result in "+input)
}

func assertParseError(t *testing.T, isInline bool, input string, expected error) {
	p := NewParser(bytes.NewBufferString(input), isInline)
	for {
		gt, _, _ := p.Next()
		if gt == ErrorGrammar {
			assert.Equal(t, expected, p.Err(), "parser must return error '"+expected.Error()+"' in "+input)
			break
		}
	}
}

////////////////////////////////////////////////////////////////

func TestParser(t *testing.T) {
	assertParse(t, true, " x : y ; ", "x:y;")
	assertParse(t, true, "color: red;", "color:red;")
	assertParse(t, true, "color : red;", "color:red;")
	assertParse(t, true, "color: red; border: 0;", "color:red;border:0;")
	assertParse(t, true, "color: red !important;", "color:red!important;")
	assertParse(t, true, "color: red ! important;", "color:red!important;")
	assertParse(t, true, "white-space: -moz-pre-wrap;", "white-space:-moz-pre-wrap;")
	assertParse(t, true, "display: -moz-inline-stack;", "display:-moz-inline-stack;")
	assertParse(t, true, "x: 10px / 1em;", "x:10px/1em;")
	assertParse(t, true, "x: 1em/1.5em \"Times New Roman\", Times, serif;", "x:1em/1.5em \"Times New Roman\",Times,serif;")
	assertParse(t, true, "x: hsla(100,50%, 75%, 0.5);", "x:hsla(100,50%,75%,0.5);")
	assertParse(t, true, "x: hsl(100,50%, 75%);", "x:hsl(100,50%,75%);")
	assertParse(t, true, "x: rgba(255, 238 , 221, 0.3);", "x:rgba(255,238,221,0.3);")
	assertParse(t, true, "x: 50vmax;", "x:50vmax;")
	assertParse(t, true, "color: linear-gradient(to right, black, white);", "color:linear-gradient(to right,black,white);")
	assertParse(t, true, "color: calc(100%/2 - 1em);", "color:calc(100%/2 - 1em);")
	assertParse(t, true, "color: calc(100%/2--1em);", "color:calc(100%/2--1em);")
	assertParse(t, false, "<!-- @charset; -->", "<!--@charset;-->")
	assertParse(t, false, "@media print, screen { }", "@media print,screen{}")
	assertParse(t, false, "@media { @viewport ; }", "@media{@viewport;}")
	assertParse(t, false, "@keyframes 'diagonal-slide' {  from { left: 0; top: 0; } to { left: 100px; top: 100px; } }", "@keyframes 'diagonal-slide'{from{left:0;top:0;}to{left:100px;top:100px;}}")
	assertParse(t, false, "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}", "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}")
	assertParse(t, false, ".foo { color: #fff;}", ".foo{color:#fff;}")
	assertParse(t, false, ".foo { *color: #fff;}", ".foo{*color:#fff;}")
	assertParse(t, false, ".foo { ; _color: #fff;}", ".foo{_color:#fff;}")
	assertParse(t, false, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	assertParse(t, false, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")

	// extraordinary
	assertParse(t, true, "color: red;;", "color:red;")
	assertParse(t, true, "color:#c0c0c0", "color:#c0c0c0;")
	assertParse(t, true, "background:URL(x.png);", "background:URL(x.png);")
	assertParse(t, true, "filter: progid : DXImageTransform.Microsoft.BasicImage(rotation=1);", "filter:progid:DXImageTransform.Microsoft.BasicImage(rotation=1);")
	assertParse(t, true, "/*a*/\n/*c*/\nkey: value;", "key:value;")
	assertParse(t, true, "@-moz-charset;", "@-moz-charset;")
	assertParse(t, false, "@import;@import;", "@import;@import;")
	assertParse(t, false, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	assertParse(t, false, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")
	//assertParse(t, false, "{x:y;}", "{x:y;}")
	assertParse(t, false, "a{}", "a{}")
	assertParse(t, false, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	assertParse(t, false, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	assertParse(t, false, "a{x:; z:q;}", "a{x:;z:q;}")
	assertParse(t, false, "@font-face { x:y; }", "@font-face{x:y;}")
	assertParse(t, false, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	assertParse(t, false, "@document regexp('https:.*') { p { color: red; } }", "@document regexp('https:.*'){p{color:red;}}")
	assertParse(t, false, "@media all and ( max-width:400px ) { }", "@media all and (max-width:400px){}")
	assertParse(t, false, "@media (max-width:400px) { }", "@media(max-width:400px){}")
	assertParse(t, false, "@media (max-width:400px)", "@media(max-width:400px);")
	assertParse(t, false, "@font-face { ; font:x; }", "@font-face{font:x;}")
	assertParse(t, false, "@-moz-font-face { ; font:x; }", "@-moz-font-face{font:x;}")
	assertParse(t, false, "@unknown abc { {} lala }", "@unknown abc{{}lala}")
	assertParse(t, false, "a[x={}]{x:y;}", "a[x={}]{x:y;}")
	assertParse(t, false, "a[x=,]{x:y;}", "a[x=,]{x:y;}")
	assertParse(t, false, "a[x=+]{x:y;}", "a[x=+]{x:y;}")
	assertParse(t, false, ".cla .ss > #id { x:y; }", ".cla .ss>#id{x:y;}")
	assertParse(t, false, ".cla /*a*/ /*b*/ .ss{}", ".cla .ss{}")
	assertParse(t, false, "a{x:f(a(),b);}", "a{x:f(a(),b);}")
	assertParse(t, false, "a{x:y!z;}", "a{x:y!z;}")
	assertParse(t, false, "[class*=\"column\"]+[class*=\"column\"]:last-child{a:b;}", "[class*=\"column\"]+[class*=\"column\"]:last-child{a:b;}")
	assertParse(t, false, "@media { @viewport }", "@media{@viewport;}")
	assertParse(t, false, "table { @unknown }", "table{@unknown;}")

	// early endings
	assertParse(t, true, "~color:red;", "")
	assertParse(t, false, "selector{", "selector{")
	assertParse(t, false, "@media{selector{", "@media{selector{")
	assertParseError(t, false, "selector", ErrBadQualifiedRule)
	assertParseError(t, true, "color 0", ErrBadDeclaration)

	// issues
	assertParse(t, false, "@media print {.class{width:5px;}}", "@media print{.class{width:5px;}}")                  // #6
	assertParse(t, false, ".class{width:calc((50% + 2em)/2 + 14px);}", ".class{width:calc((50% + 2em)/2 + 14px);}") // #7
	assertParse(t, false, ".class [c=y]{}", ".class [c=y]{}")                                                       // tdewolff/minify#16
	assertParse(t, false, "table{font-family:Verdana}", "table{font-family:Verdana;}")                              // tdewolff/minify#22

	// go-fuzz
	assertParse(t, false, "@-webkit-", "@-webkit-;")

	assert.Equal(t, "Error", ErrorGrammar.String())
	assert.Equal(t, "AtRule", AtRuleGrammar.String())
	assert.Equal(t, "BeginAtRule", BeginAtRuleGrammar.String())
	assert.Equal(t, "EndAtRule", EndAtRuleGrammar.String())
	assert.Equal(t, "BeginRuleset", BeginRulesetGrammar.String())
	assert.Equal(t, "EndRuleset", EndRulesetGrammar.String())
	assert.Equal(t, "Declaration", DeclarationGrammar.String())
	assert.Equal(t, "Token", TokenGrammar.String())
	assert.Equal(t, "Invalid(100)", GrammarType(100).String())
}

func TestReader(t *testing.T) {
	input := "x:a;"
	p := NewParser(test.NewPlainReader(bytes.NewBufferString(input)), true)
	for {
		gt, _, _ := p.Next()
		if gt == ErrorGrammar {
			break
		}
	}
}

////////////////////////////////////////////////////////////////

func ExampleNewParser() {
	p := NewParser(bytes.NewBufferString("color: red;"), true) // false because this is the content of an inline style attribute
	out := ""
	for {
		gt, _, data := p.Next()
		if gt == ErrorGrammar {
			break
		} else if gt == AtRuleGrammar || gt == BeginAtRuleGrammar || gt == BeginRulesetGrammar || gt == DeclarationGrammar {
			out += string(data)
			if gt == DeclarationGrammar {
				out += ":"
			}
			for _, val := range p.Values() {
				out += string(val.Data)
			}
			if gt == BeginAtRuleGrammar || gt == BeginRulesetGrammar {
				out += "{"
			} else if gt == AtRuleGrammar || gt == DeclarationGrammar {
				out += ";"
			}
		} else {
			out += string(data)
		}
	}
	fmt.Println(out)
	// Output: color:red;
}
