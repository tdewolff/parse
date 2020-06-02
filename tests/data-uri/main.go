// +build gofuzz
package fuzz

import "github.com/tdewolff/parse/v2"

func Fuzz(data []byte) int {
	_, _, _ = parse.DataURI(data)
	return 1
}
