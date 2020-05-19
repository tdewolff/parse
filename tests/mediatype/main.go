// +build gofuzz
package fuzz

import "github.com/tdewolff/parse/v2"

func Fuzz(data []byte) int {
	_, _ = parse.Mediatype(data)
	return 1
}
