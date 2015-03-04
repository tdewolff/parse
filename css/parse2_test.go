package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"os"
	"testing"
	"fmt"

	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////

func assertParse2(t *testing.T, input, expected string) {
	stylesheet, err := NewParser2(NewTokenizer(&ReaderMockup{bytes.NewBufferString(input)})).Parse()
	assert.Nil(t, err, "parser must not return error")

	b := &bytes.Buffer{}
	stylesheet.Serialize(b)
	assert.Equal(t, expected, b.String(), "parsed string must match expected result in "+input)

	if expected != b.String() {
		printParse2(t, input)
		fmt.Println(len(stylesheet.Nodes))
		for _, x := range stylesheet.Nodes {
			if y, ok := x.(*AtRuleNode); ok {
				//for _, z := range y.Rules {
					fmt.Print("\n--")
					y.Serialize(os.Stdout)
					fmt.Print("--\n")
				//}
			}
		}
	}
}

func printParse2(t *testing.T, input string) {
	i := 0
	p := NewParser2(NewTokenizer(bytes.NewBufferString(input)))
	for {
		gt, n := p.Next()
		if gt == ErrorGrammar {
			break
		}

		if i > 0 && (gt == AtRuleGrammar || gt == RulesetGrammar) {
			fmt.Print("\n    ")
		}

		fmt.Print(" "+gt.String()+"(")
		switch gt {
		default:
			n.Serialize(os.Stdout)
		}
		fmt.Print(")")
		i++
	}
	fmt.Print(".\n")
}

func TestParser2(t *testing.T) {
	assertParse2(t, " <!-- x : y ; --> ", "<!--x:y;-->")
	assertParse2(t, "color: red;", "color:red;")
	assertParse2(t, "color : red;", "color:red;")
	assertParse2(t, "color: red; border: 0;", "color:red;border:0;")
	assertParse2(t, "color: red !important;", "color:red !important;")
	assertParse2(t, "color: red ! important;", "color:red !important;")
	assertParse2(t, "white-space: -moz-pre-wrap;", "white-space:-moz-pre-wrap;")
	assertParse2(t, "display: -moz-inline-stack;", "display:-moz-inline-stack;")
	assertParse2(t, "x: 10px / 1em;", "x:10px/1em;")
	assertParse2(t, "x: 1em/1.5em \"Times New Roman\", Times, serif;", "x:1em/1.5em \"Times New Roman\",Times,serif;")
	assertParse2(t, "x: hsla(100,50%, 75%, 0.5);", "x:hsla(100,50%,75%,0.5);")
	assertParse2(t, "x: hsl(100,50%, 75%);", "x:hsl(100,50%,75%);")
	assertParse2(t, "x: rgba(255, 238 , 221, 0.3);", "x:rgba(255,238,221,0.3);")
	assertParse2(t, "x: 50vmax;", "x:50vmax;")
	assertParse2(t, "color: linear-gradient(to right, black, white);", "color:linear-gradient(to right,black,white);")
	assertParse2(t, "color: calc(100%/2 - 1em);", "color:calc(100%/2 - 1em);")
	assertParse2(t, "color: calc(100%/2--1em);", "color:calc(100%/2 - -1em);")
	assertParse2(t, "@media print, screen { }", "@media print,screen;")
	assertParse2(t, "@media { @viewport {} }", "@media{@viewport;}")
	assertParse2(t, "@keyframes 'diagonal-slide' {  from { left: 0; top: 0; } to { left: 100px; top: 100px; } }", "@keyframes 'diagonal-slide'{from{left:0;top:0;}to{left:100px;top:100px;}}")
	assertParse2(t, "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}", "@keyframes movingbox{0%{left:90%;}50%{left:10%;}100%{left:90%;}}")
	assertParse2(t, ".foo { color: #fff;}", ".foo{color:#fff;}")
	assertParse2(t, ".foo { *color: #fff;}", ".foo{color:#fff;}")
	assertParse2(t, ".foo { _color: #fff;}", ".foo{_color:#fff;}")
	assertParse2(t, "a { color: red; border: 0; }", "a{color:red;border:0;}")
	assertParse2(t, "a { color: red; ; } ; ;", "a{color:red;}")
	assertParse2(t, "a { color: red; border: 0; } b { padding: 0; }", "a{color:red;border:0;}b{padding:0;}")

	// extraordinary
	assertParse2(t, "color: red;;", "color:red;")
	assertParse2(t, "@import;;", "@import;")
	assertParse2(t, "@import;@import;", "@import;@import;")
	assertParse2(t, ".a .b#c, .d<.e { x:y; }", ".a .b#c,.d<.e{x:y;}")
	assertParse2(t, ".a[b~=c]d { x:y; }", ".a[b~=c]d{x:y;}")
	assertParse2(t, "{x:y;}", "{x:y;}")
	assertParse2(t, "a{}", "a{}")
	assertParse2(t, "a,{x:y;}", "a{x:y;}")
	assertParse2(t, "a,.b/*comment*/ {x:y;}", "a,.b{x:y;}")
	assertParse2(t, "a,.b/*comment*/.c {x:y;}", "a,.b.c{x:y;}")
	assertParse2(t, "a{x:; z:q;}", "a{x:;z:q;}")
	assertParse2(t, "@import { @media f; x:y; }", "@import{@media f;x:y;}")
	assertParse2(t, "a:not([controls]){x:y;}", "a:not([controls]){x:y;}")
	assertParse2(t, "color:#c0c0c0", "color:#c0c0c0;")
	assertParse2(t, "background:URL(x.png);", "background:URL(x.png);")
	assertParse2(t, "@document regexp('https:.*') { p { color: red; } }", "@document regexp('https:.*'){p{color:red;}}")
	assertParse2(t, "@media all and ( max-width:400px ) { }", "@media all and (max-width:400px);")
	assertParse2(t, "@media (max-width:400px) { }", "@media (max-width:400px);")
	assertParse2(t, "@media (max-width:400px)", "@media (max-width:400px);")
	assertParse2(t, "a[x={]{x:y;}", "a[x={]{x:y;}")
	assertParse2(t, "a[x=,]{x:y;}", "a[x=,]{x:y;}")
	assertParse2(t, "a[x=+]{x:y;}", "a[x=+]{x:y;}")
	assertParse2(t, ".cla .ss > #id { x:y; }", ".cla .ss>#id{x:y;}")
	assertParse2(t, ".cla /*a*/ /*b*/ .ss{}", ".cla .ss{}")
	assertParse2(t, "filter: progid : DXImageTransform.Microsoft.BasicImage(rotation=1);", "filter:progid:DXImageTransform.Microsoft.BasicImage(rotation=1);")
	assertParse2(t, "a{x:f(a(),b);}", "a{x:f(a(),b);}")

	// issues
	assertParse2(t, "@media print {.class{width:5px;}}", "@media print{.class{width:5px;}}") // #6
	assertParse2(t, ".class{width:calc((50% + 2em)/2 + 14px);}}", ".class{width:calc((50% + 2em)/2 + 14px);}}") // #7
}
