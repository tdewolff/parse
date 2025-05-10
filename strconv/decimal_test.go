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
		{"5", 5.0},
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
		{"+1", 0, 0.0},
		{"-1", 2, -1.0},
		{".", 0, 0.0},
		{"1e1", 1, 1.0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseDecimal([]byte(tt.f))
			test.T(t, n, tt.n)
			test.T(t, f, tt.expected)
		})
	}
}

func TestAppendDecimal(t *testing.T) {
	tests := []struct {
		f        float64
		n        int
		expected string
	}{
		{0.0, 0, "0"},
		{1.0, 2, "1"},
		{-1.0, 2, "-1"},
		{1.2, 2, "1.2"},
		{1.23, 2, "1.23"},
		{1.234, 2, "1.23"},
		{1.235, 2, "1.24"},
		{0.1, 2, "0.1"},
		{0.01, 2, "0.01"},
		{0.001, 2, "0"},
		{0.005, 2, "0.01"},
		{-75.8077501, 6, "-75.80775"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			b := AppendDecimal(nil, tt.f, tt.n)
			test.T(t, string(b), tt.expected)
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
