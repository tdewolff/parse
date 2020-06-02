// +build gofuzz
package fuzz

import "github.com/tdewolff/parse/v2"

func Fuzz(data []byte) int {
	_ = parse.Number(data)
	return 1
}
