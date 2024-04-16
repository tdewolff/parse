package parse

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/tdewolff/test"
)

func helperRandChars(n, m int, chars string) [][]byte {
	r := make([][]byte, n)
	for i := range r {
		for j := 0; j < m; j++ {
			r[i] = append(r[i], chars[rand.Intn(len(chars))])
		}
	}
	return r
}

func helperRandStrings(n, m int, ss []string) [][]byte {
	r := make([][]byte, n)
	for i := range r {
		for j := 0; j < m; j++ {
			r[i] = append(r[i], []byte(ss[rand.Intn(len(ss))])...)
		}
	}
	return r
}

////////////////////////////////////////////////////////////////

var wsSlices [][]byte

func init() {
	wsSlices = helperRandChars(10000, 50, "abcdefg \n\r\f\t")
}

func TestCopy(t *testing.T) {
	foo := []byte("abc")
	bar := Copy(foo)
	foo[0] = 'b'
	test.String(t, string(foo), "bbc")
	test.String(t, string(bar), "abc")
}

func TestToLower(t *testing.T) {
	foo := []byte("Abc")
	bar := ToLower(foo)
	bar[1] = 'B'
	test.String(t, string(foo), "aBc")
	test.String(t, string(bar), "aBc")
}

func TestEqualFold(t *testing.T) {
	test.That(t, EqualFold([]byte("Abc"), []byte("abc")))
	test.That(t, !EqualFold([]byte("Abcd"), []byte("abc")))
	test.That(t, !EqualFold([]byte("Bbc"), []byte("abc")))
	test.That(t, !EqualFold([]byte("[]"), []byte("{}"))) // same distance in ASCII as 'a' and 'A'
}

func TestWhitespace(t *testing.T) {
	test.That(t, IsAllWhitespace([]byte("\t \r\n\f")))
	test.That(t, !IsAllWhitespace([]byte("\t \r\n\fx")))
}

func TestTrim(t *testing.T) {
	test.Bytes(t, TrimWhitespace([]byte("a")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte(" a")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte("a ")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte(" ")), []byte(""))
}

func TestPrintable(t *testing.T) {
	var tests = []struct {
		s         string
		printable string
	}{
		{"a", "a"},
		{"\x00", "0x00"},
		{"\x7F", "0x7F"},
		{"\u0800", "ࠀ"},
		{"\u200F", "U+200F"},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			printable := ""
			for _, r := range tt.s {
				printable += Printable(r)
			}
			test.T(t, printable, tt.printable)
		})
	}
}

////////////////////////////////////////////////////////////////

func BenchmarkBytesTrim(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			bytes.TrimSpace(e)
		}
	}
}

func BenchmarkTrim(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			TrimWhitespace(e)
		}
	}
}

func BenchmarkWhitespaceTable(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if IsWhitespace(c) {
					n++
				}
			}
		}
	}
}

func BenchmarkWhitespaceIf1(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if c == ' ' {
					n++
				}
			}
		}
	}
}

func BenchmarkWhitespaceIf2(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if c == ' ' || c == '\n' {
					n++
				}
			}
		}
	}
}

func BenchmarkWhitespaceIf3(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if c == ' ' || c == '\n' || c == '\r' {
					n++
				}
			}
		}
	}
}

func BenchmarkWhitespaceIf4(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
					n++
				}
			}
		}
	}
}

func BenchmarkWhitespaceIf5(b *testing.B) {
	n := 0
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			for _, c := range e {
				if c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '\f' {
					n++
				}
			}
		}
	}
}
