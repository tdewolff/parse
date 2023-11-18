// +build gofuzz

package fuzz

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/js"
)

// Fuzz is a fuzz test.
func Fuzz(data []byte) int {
	if !utf8.Valid(data) {
		return 0
	}

	o := js.Options{}
	input := parse.NewInputBytes(data)
	if ast, err := js.Parse(input, o); err == nil {
		src := ast.JSString()
		input2 := parse.NewInputString(src)
		if ast2, err := js.Parse(input2, o); err != nil {
			if !strings.HasPrefix(err.Error(), "too many nested") {
				panic(err)
			}
		} else if src2 := ast2.JSString(); src != src2 {
			fmt.Println("JS1:", src)
			fmt.Println("JS2:", src2)
			panic("ASTs not equal")
		}
		return 1
	}
	return 0
}
