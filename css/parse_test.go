package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/parse"
)

func assertParse(t *testing.T, isStylesheet bool, input, expected string) {
	output := ""
	p := NewParser(bytes.NewBufferString(input), isStylesheet)
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
			data = parse.Copy(data)
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

func TestParser(t *testing.T) {
	assertParse(t, false, " x : y ; ", "x:y;")
	assertParse(t, false, "color: red;", "color:red;")
	assertParse(t, false, "color : red;", "color:red;")
	assertParse(t, false, "color: red; border: 0;", "color:red;border:0;")
	assertParse(t, false, "color: red !important;", "color:red!important;")
	assertParse(t, false, "color: red ! important;", "color:red!important;")
	assertParse(t, false, "white-space: -moz-pre-wrap;", "white-space:-moz-pre-wrap;")
	assertParse(t, false, "display: -moz-inline-stack;", "display:-moz-inline-stack;")
	assertParse(t, false, "x: 10px / 1em;", "x:10px/1em;")
	assertParse(t, false, "x: 1em/1.5em \"Times New Roman\", Times, serif;", "x:1em/1.5em \"Times New Roman\",Times,serif;")
	assertParse(t, false, "x: hsla(100,50%, 75%, 0.5);", "x:hsla(100,50%,75%,0.5);")
	assertParse(t, false, "x: hsl(100,50%, 75%);", "x:hsl(100,50%,75%);")
	assertParse(t, false, "x: rgba(255, 238 , 221, 0.3);", "x:rgba(255,238,221,0.3);")
	assertParse(t, false, "x: 50vmax;", "x:50vmax;")
	assertParse(t, false, "color: linear-gradient(to right, black, white);", "color:linear-gradient(to right,black,white);")
	assertParse(t, false, "color: calc(100%/2 - 1em);", "color:calc(100%/2 - 1em);")
	assertParse(t, false, "color: calc(100%/2--1em);", "color:calc(100%/2--1em);")
	assertParse(t, true, "@media print, screen { }", "@media print,screen{}")
	assertParse(t, true, "@media { @viewport ; }", "@media{@viewport;}")
	assertParse(t, true, "@keyframes 'diagonal-slide' {  from { left: 0; top: 0; } to { left: 100px; top: 100px; } }", "@keyframes 'diagonal-slide'{from{left:0;top:0;}to{left:100px;top:100px;}}")
	assertParse(t, true, "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}", "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}")
	assertParse(t, true, ".foo { color: #fff;}", ".foo{color:#fff;}")
	assertParse(t, true, ".foo { *color: #fff;}", ".foo{}")
	assertParse(t, true, ".foo { _color: #fff;}", ".foo{_color:#fff;}")
	assertParse(t, true, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	assertParse(t, true, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")

	// extraordinary
	assertParse(t, false, "color: red;;", "color:red;")
	assertParse(t, true, "@import;@import;", "@import;@import;")
	assertParse(t, true, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	assertParse(t, true, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")
	//assertParse(t, true, "{x:y;}", "{x:y;}")
	assertParse(t, true, "a{}", "a{}")
	assertParse(t, true, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	assertParse(t, true, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	assertParse(t, true, "a{x:; z:q;}", "a{x:;z:q;}")
	assertParse(t, true, "@font-face { x:y; }", "@font-face{x:y;}")
	assertParse(t, true, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	assertParse(t, false, "color:#c0c0c0", "color:#c0c0c0;")
	assertParse(t, false, "background:URL(x.png);", "background:URL(x.png);")
	assertParse(t, true, "@document regexp('https:.*') { p { color: red; } }", "@document regexp('https:.*'){p{color:red;}}")
	assertParse(t, true, "@media all and ( max-width:400px ) { }", "@media all and (max-width:400px){}")
	assertParse(t, true, "@media (max-width:400px) { }", "@media(max-width:400px){}")
	assertParse(t, true, "@media (max-width:400px)", "@media(max-width:400px);")
	assertParse(t, true, "a[x={}]{x:y;}", "a[x={}]{x:y;}")
	assertParse(t, true, "a[x=,]{x:y;}", "a[x=,]{x:y;}")
	assertParse(t, true, "a[x=+]{x:y;}", "a[x=+]{x:y;}")
	assertParse(t, true, ".cla .ss > #id { x:y; }", ".cla .ss>#id{x:y;}")
	assertParse(t, true, ".cla /*a*/ /*b*/ .ss{}", ".cla .ss{}")
	assertParse(t, false, "filter: progid : DXImageTransform.Microsoft.BasicImage(rotation=1);", "filter:progid:DXImageTransform.Microsoft.BasicImage(rotation=1);")
	assertParse(t, true, "a{x:f(a(),b);}", "a{x:f(a(),b);}")
	assertParse(t, false, "/*a*/\n/*c*/\nkey: value;", "key:value;")
	assertParse(t, true, "a{x:y!z;}", "a{x:y!z;}")
	assertParse(t, true, "[class*=\"column\"]+[class*=\"column\"]:last-child{a:b;}", "[class*=\"column\"]+[class*=\"column\"]:last-child{a:b;}")

	// issues
	assertParse(t, true, "@media print {.class{width:5px;}}", "@media print{.class{width:5px;}}")                  // #6
	assertParse(t, true, ".class{width:calc((50% + 2em)/2 + 14px);}", ".class{width:calc((50% + 2em)/2 + 14px);}") // #7
	assertParse(t, true, ".class [c=y]{}", ".class [c=y]{}")                                                       // #16
}
