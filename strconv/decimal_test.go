package strconv

import (
	"fmt"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		f        string
		expected float64
	}{
		{"5", 5},
		{"5.1", 5.1},
		{"0.0000000000000000000000000005", 5e-28},
		{"18446744073709551620", 18446744073709551620.0},
		{"1000000000000000000000000.0000", 1e24},              // TODO
		{"1000000000000000000000000000000000000000000", 1e42}, // TODO
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseDecimal([]byte(tt.f))
			test.T(t, n, len(tt.f))
			test.Float(t, f, tt.expected)
		})
	}
}

func TestParseDecimalError(t *testing.T) {
	tests := []struct {
		f        string
		n        int
		expected float64
	}{
		{"+1", 0, 0},
		{"-1", 0, 0},
		{".", 0, 0},
		{"1e1", 1, 1},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseDecimal([]byte(tt.f))
			test.T(t, n, tt.n)
			test.T(t, f, tt.expected)
		})
	}
}

func FuzzParseDecimal(f *testing.F) {
	f.Add("5")
	f.Add("5.1")
	f.Add("18446744073709551620")
	f.Add("0.0000000000000000000000000005")
	f.Fuzz(func(t *testing.T, s string) {
		ParseDecimal([]byte(s))
	})
}
