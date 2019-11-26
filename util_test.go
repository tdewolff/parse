package parse

import (
	"bytes"
	"math/rand"
	"regexp"
	"testing"

	"github.com/tdewolff/test"
)

func helperRand(n, m int, chars []byte) [][]byte {
	r := make([][]byte, n)
	for i := range r {
		for j := 0; j < m; j++ {
			r[i] = append(r[i], chars[rand.Intn(len(chars))])
		}
	}
	return r
}

////////////////////////////////////////////////////////////////

var wsSlices [][]byte

func init() {
	wsSlices = helperRand(10000, 50, []byte("abcdefg \n\r\f\t"))
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

func TestReplaceMultipleWhitespaceRandom(t *testing.T) {
	wsRegexp := regexp.MustCompile("[ \t\f]+")
	wsNewlinesRegexp := regexp.MustCompile("[ ]*[\r\n][ \r\n]*")
	for _, e := range wsSlices {
		reference := wsRegexp.ReplaceAll(e, []byte(" "))
		reference = wsNewlinesRegexp.ReplaceAll(reference, []byte("\n"))
		test.Bytes(t, ReplaceMultipleWhitespace(e), reference, "in: "+string(e))
	}
}

func TestReplaceMultipleWhitespace(t *testing.T) {
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a")), []byte(" a"))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("a  ")), []byte("a "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("a  b  ")), []byte("a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a  b  ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte(" a b  ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a b ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("   a")), []byte(" a"))
}

func TestTrim(t *testing.T) {
	test.Bytes(t, TrimWhitespace([]byte("a")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte(" a")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte("a ")), []byte("a"))
	test.Bytes(t, TrimWhitespace([]byte(" ")), []byte(""))
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

func BenchmarkReplaceMultipleWhitespace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			ReplaceMultipleWhitespace(e)
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
