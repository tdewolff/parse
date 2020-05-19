// +build gofuzz
package fuzz

import "github.com/tdewolff/parse/v2"

func Fuzz(data []byte) int {
	newData := parse.ReplaceEntities(data, map[string][]byte{
		"test":  []byte("&t;"),
		"test3": []byte("&test;"),
		"test5": []byte("&#5;"),
		"quot":  []byte("\""),
		"apos":  []byte("'"),
	}, map[byte][]byte{
		'\'': []byte("&#34;"),
		'"':  []byte("&#39;"),
	})
	if len(newData) > len(data) {
		panic("output longer than input")
	}
	return 1
}
