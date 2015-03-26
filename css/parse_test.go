package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/parse"
)

////////////////////////////////////////////////////////////////

func assertParse(t *testing.T, input, expected string) {
	stylesheet, err := Parse(&ReaderMockup{bytes.NewBufferString(input)})
	assert.Nil(t, err, "parser must not return error")

	b := &bytes.Buffer{}
	stylesheet.WriteTo(b)
	assert.Equal(t, expected, b.String(), "parsed string must match expected result in "+input)
}

func TestParser(t *testing.T) {
	assertParse(t, " <!-- x : y ; --> ", "<!--x:y;-->")
	assertParse(t, "color: red;", "color:red;")
	assertParse(t, "color : red;", "color:red;")
	assertParse(t, "color: red; border: 0;", "color:red;border:0;")
	assertParse(t, "color: red !important;", "color:red!important;")
	assertParse(t, "color: red ! important;", "color:red!important;")
	assertParse(t, "white-space: -moz-pre-wrap;", "white-space:-moz-pre-wrap;")
	assertParse(t, "display: -moz-inline-stack;", "display:-moz-inline-stack;")
	assertParse(t, "x: 10px / 1em;", "x:10px/1em;")
	assertParse(t, "x: 1em/1.5em \"Times New Roman\", Times, serif;", "x:1em/1.5em \"Times New Roman\",Times,serif;")
	assertParse(t, "x: hsla(100,50%, 75%, 0.5);", "x:hsla(100,50%,75%,0.5);")
	assertParse(t, "x: hsl(100,50%, 75%);", "x:hsl(100,50%,75%);")
	assertParse(t, "x: rgba(255, 238 , 221, 0.3);", "x:rgba(255,238,221,0.3);")
	assertParse(t, "x: 50vmax;", "x:50vmax;")
	assertParse(t, "color: linear-gradient(to right, black, white);", "color:linear-gradient(to right,black,white);")
	assertParse(t, "color: calc(100%/2 - 1em);", "color:calc(100%/2 - 1em);")
	assertParse(t, "color: calc(100%/2--1em);", "color:calc(100%/2 - -1em);")
	assertParse(t, "@media print, screen { }", "@media print,screen;")
	assertParse(t, "@media { @viewport {} }", "@media{@viewport;}")
	assertParse(t, "@keyframes 'diagonal-slide' {  from { left: 0; top: 0; } to { left: 100px; top: 100px; } }", "@keyframes 'diagonal-slide'{from{left:0;top:0;}to{left:100px;top:100px;}}")
	assertParse(t, "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}", "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}")
	assertParse(t, ".foo { color: #fff;}", ".foo{color:#fff;}")
	assertParse(t, ".foo { *color: #fff;}", ".foo{color:#fff;}")
	assertParse(t, ".foo { _color: #fff;}", ".foo{_color:#fff;}")
	assertParse(t, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	assertParse(t, "a { color: red; ; } ; ;", "a{color:red;}")
	assertParse(t, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")

	// extraordinary
	assertParse(t, "color: red;;", "color:red;")
	assertParse(t, "@import;;", "@import;")
	assertParse(t, "@import;@import;", "@import;@import;")
	assertParse(t, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	assertParse(t, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")
	assertParse(t, "{x:y;}", "{x:y;}")
	assertParse(t, "a{}", "a{}")
	assertParse(t, "a,{x:y;}", "a{x:y;}")
	assertParse(t, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	assertParse(t, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	assertParse(t, "a{x:; z:q;}", "a{x:;z:q;}")
	assertParse(t, "@import { @media f; x:y; }", "@import{@media f;x:y;}")
	assertParse(t, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	assertParse(t, "color:#c0c0c0", "color:#c0c0c0;")
	assertParse(t, "background:URL(x.png);", "background:URL(x.png);")
	assertParse(t, "@document regexp('https:.*') { p { color: red; } }", "@document regexp('https:.*'){p{color:red;}}")
	assertParse(t, "@media all and ( max-width:400px ) { }", "@media all and (max-width:400px);")
	assertParse(t, "@media (max-width:400px) { }", "@media (max-width:400px);")
	assertParse(t, "@media (max-width:400px)", "@media (max-width:400px);")
	assertParse(t, "a[x={]{x:y;}", "a[x={]{x:y;}")
	assertParse(t, "a[x=,]{x:y;}", "a[x=,]{x:y;}")
	assertParse(t, "a[x=+]{x:y;}", "a[x=+]{x:y;}")
	assertParse(t, ".cla .ss > #id { x:y; }", ".cla .ss>#id{x:y;}")
	assertParse(t, ".cla /*a*/ /*b*/ .ss{}", ".cla .ss{}")
	assertParse(t, "filter: progid : DXImageTransform.Microsoft.BasicImage(rotation=1);", "filter:progid:DXImageTransform.Microsoft.BasicImage(rotation=1);")
	assertParse(t, "a{x:f(a(),b);}", "a{x:f(a(),b);}")
	assertParse(t, "/*a*/\n/*c*/\nkey: value;", "key:value;")
	assertParse(t, "a{x:y!z;}", "a{x:y!z;}")

	// issues
	assertParse(t, "@media print {.class{width:5px;}}", "@media print{.class{width:5px;}}")                    // #6
	assertParse(t, ".class{width:calc((50% + 2em)/2 + 14px);}}", ".class{width:calc((50% + 2em)/2 + 14px);}}") // #7
	assertParse(t, ".class [c=y]{}", ".class [c=y]{}")                                                         // #16
}

func TestParserSmall(t *testing.T) {
	parse.MinBuf = 4
	parse.MaxBuf = 4
	z := NewParser(&ReaderMockup{bytes.NewBufferString("a:b; c:d;")})
	gt, _ := z.Next()
	assert.Equal(t, DeclarationGrammar, gt, "first grammar must be DeclarationGrammar")
	gt, _ = z.Next()
	assert.Equal(t, DeclarationGrammar, gt, "second grammar must be DeclarationGrammar")
	gt, _ = z.Next()
	assert.Equal(t, ErrorGrammar, gt, "third grammar must be DeclarationGrammar")
}
