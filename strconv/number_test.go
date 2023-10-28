package strconv

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestParseNumber(t *testing.T) {
	tests := []struct {
		s   string
		num int64
		dec int
		n   int
	}{
		{"5", 5, 0, 1},
		{"-5", -5, 0, 2},
		{"5,0", 50, 1, 3},
		{"5,0a", 50, 1, 3},
		{"-1000,00", -100000, 2, 8},
		{"9223372036854775807", 9223372036854775807, 0, 19},
		{"-9223372036854775807", -9223372036854775807, 0, 20},
		{"-9223372036854775808", -9223372036854775808, 0, 20},
		{"92233720368547758070", 9223372036854775807, 0, 19},
		{"-92233720368547758080", -9223372036854775808, 0, 20},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			num, dec, n := ParseNumber([]byte(tt.s), '.', ',')
			test.T(t, []interface{}{num, dec, n}, []interface{}{tt.num, tt.dec, tt.n})
		})
	}
}

func TestAppendNumber(t *testing.T) {
	tests := []struct {
		num int64
		dec int
		s   string
	}{
		{0, 0, "0"},
		{0, -1, "0"},
		{0, 2, "0,00"},
		{1, 2, "0,01"},
		{-1, 2, "-0,01"},
		{100, 2, "1,00"},
		{-100, 2, "-1,00"},
		{1000, 0, "1.000"},
		{100000, 2, "1.000,00"},
		{123456789012, 2, "1.234.567.890,12"},
		{9223372036854775807, 2, "92.233.720.368.547.758,07"},
		{-9223372036854775808, 2, "-92.233.720.368.547.758,08"},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			num := AppendNumber(make([]byte, 0, 4), tt.num, tt.dec, 3, '.', ',')
			test.String(t, string(num), tt.s)
		})
	}

	// coverage
	test.String(t, string(AppendNumber(make([]byte, 0, 7), 12345, 1, 3, -1, -1)), "1.234,5")
}

func FuzzParseNumber(f *testing.F) {
	f.Add("5")
	f.Add("-5")
	f.Add("5,0")
	f.Add("5,0a")
	f.Add("-1000,00")
	f.Add("9223372036854775807")
	f.Add("-9223372036854775807")
	f.Add("-9223372036854775808")
	f.Add("92233720368547758070")
	f.Add("-92233720368547758080")
	f.Fuzz(func(t *testing.T, s string) {
		ParseNumber([]byte(s), '.', ',')
	})
}

func FuzzAppendNumber(f *testing.F) {
	f.Add(int64(0), 0)
	f.Add(int64(0), -1)
	f.Add(int64(0), 2)
	f.Add(int64(1), 2)
	f.Add(int64(-1), 2)
	f.Add(int64(100), 2)
	f.Add(int64(-100), 2)
	f.Add(int64(1000), 0)
	f.Add(int64(100000), 2)
	f.Add(int64(123456789012), 2)
	f.Add(int64(9223372036854775807), 2)
	f.Add(int64(-9223372036854775808), 2)
	f.Fuzz(func(t *testing.T, num int64, dec int) {
		AppendNumber([]byte{}, num, dec, 3, '.', ',')
	})
}
