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
		{"18446744073709551620", 18446744073709551620.0},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseDecimal([]byte(tt.f))
			test.T(t, n, len(tt.f))
			test.T(t, f, tt.expected)
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
