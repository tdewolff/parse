package strconv

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/tdewolff/test"
)

func TestParseFloat(t *testing.T) {
	floatTests := []struct {
		f        string
		expected float64
	}{
		{"5", 5},
		{"5.1", 5.1},
		{"-5.1", -5.1},
		{"5.1e-2", 5.1e-2},
		{"5.1e+2", 5.1e+2},
		{"0.0e1", 0.0e1},
		{"18446744073709551620", 18446744073709551620.0},
		{"1e23", 1e23},
		// mantissa truncated in the integer part, with a dot
		{"123456789123456789123.5", 1.234567891234568e+20},
		{"123456789123456789123.", 1.234567891234568e+20},
		{"-123456789123456789123.5", -1.234567891234568e+20},
		{"123456789123456789123.5e3", 1.234567891234568e+23},
		{"99999999999999999999.0", 1e20},
		{"1234567891234567891234.25", 1.234567891234568e+21},
		// uint64 accumulator at its limit
		{"18446744073709551615", 18446744073709551615.0},
		{"18446744073709551616", 1.8446744073709552e+19},
		{"18446744073709551616.5", 1.8446744073709552e+19},
		{"-18446744073709551616", -1.8446744073709552e+19},
		{"184467440737095516165", 1.844674407370955e+20},
		// 15 digits with 22 < exp, so the int*10^k path bails out
		{"993349238352373e23", 9.93349238352373e+37},
		// math.Pow10 is 0 below 1e-323 and infinite above 1e308
		{"0.0000000000000000000001610832207361e330", 1.610832207361e+308},
		{"5e-324", 5e-324},
		{"9e-324", 1e-323},
		{"0e309", 0},
		// TODO: hard to test due to float imprecision
		// {"1.7976931348623e+308", 1.7976931348623e+308)
		// {"4.9406564584124e-308", 4.9406564584124e-308)
	}
	for _, tt := range floatTests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseFloat([]byte(tt.f))
			test.T(t, n, len(tt.f))
			test.T(t, f, tt.expected)
		})
	}
}

// Literals that need scaling in more than one step, where a single math.Pow10
// leaves its domain. Compared against the stdlib within a few ULP.
func TestParseFloatScaling(t *testing.T) {
	floatTests := []string{
		"1" + strings.Repeat("0", 41) + "e-330",
		"1" + strings.Repeat("0", 60) + "e-360",
		"0." + strings.Repeat("0", 100) + "1e400",
		"0." + strings.Repeat("0", 200) + "5e450",
		"-1" + strings.Repeat("0", 41) + "e-330",
		"-0e309",
	}
	for _, tt := range floatTests {
		t.Run(fmt.Sprint(len(tt)), func(t *testing.T) {
			expected, err := strconv.ParseFloat(tt, 64)
			test.Error(t, err)
			f, n := ParseFloat([]byte(tt))
			test.T(t, n, len(tt))
			if math.Abs(f-expected) > 1e-14*math.Abs(expected) {
				test.Fail(t, fmt.Sprintf("%v != %v for %v", f, expected, tt))
			}
		})
	}
}

func TestParseFloatError(t *testing.T) {
	floatTests := []struct {
		f        string
		n        int
		expected float64
	}{
		{"e1", 0, 0},
		{".", 0, 0},
		{"1e", 1, 1},
		{"1e+", 1, 1},
		{"1e+1", 4, 10},
	}
	for _, tt := range floatTests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f, n := ParseFloat([]byte(tt.f))
			test.T(t, n, tt.n)
			test.T(t, f, tt.expected)
		})
	}
}

func TestAppendFloat(t *testing.T) {
	floatTests := []struct {
		f        float64
		prec     int
		expected string
	}{
		{0, 6, "0"},
		{1, 6, "1"},
		{9, 6, "9"},
		{9.99999, 6, "9.99999"},
		{123, 6, "123"},
		{0.123456, 6, ".123456"},
		{0.066, 6, ".066"},
		{0.0066, 6, ".0066"},
		{12e2, 6, "1200"},
		{12e3, 6, "12e3"},
		{0.1, 6, ".1"},
		{0.001, 6, ".001"},
		{0.0001, 6, "1e-4"},
		{-1, 6, "-1"},
		{-123, 6, "-123"},
		{-123.456, 6, "-123.456"},
		{-12e3, 6, "-12e3"},
		{-0.1, 6, "-.1"},
		{-0.0001, 6, "-1e-4"},
		{0.000100009, 10, "100009e-9"},
		{0.0001000009, 10, "1.000009e-4"},
		{1e18, 0, "1e18"},
		//{1e19, 0, "1e19"},
		//{1e19, 18, "1e19"},
		{1e1, 0, "10"},
		{1e2, 1, "100"},
		{1e3, 2, "1e3"},
		{1e10, -1, "1e10"},
		{1e15, -1, "1e15"},
		{1e-5, 6, "1e-5"},
		{math.NaN(), 0, ""},
		{math.Inf(1), 0, ""},
		{math.Inf(-1), 0, ""},
		{0, 19, "0"},
		{0.000923361977200859392, -1, "9.23361977200859392e-4"},
		{1234, 2, "1.23e3"},
		{12345, 2, "1.23e4"},
		{12.345, 2, "12.3"},
		{12.345, 3, "12.34"},
	}
	for _, tt := range floatTests {
		t.Run(fmt.Sprint(tt.f), func(t *testing.T) {
			f := AppendFloat([]byte{}, tt.f, tt.prec)
			test.String(t, string(f), tt.expected)
		})
	}

	b := make([]byte, 0, 22)
	AppendFloat(b, 12.34, -1)
	test.String(t, string(b[:5]), "12.34", "in buffer")
}

func FuzzParseFloat(f *testing.F) {
	f.Add("5")
	f.Add("99")
	f.Add("18446744073709551615")
	f.Add("5")
	f.Add("5.1")
	f.Add("-5.1")
	f.Add("5.1e-2")
	f.Add("5.1e+2")
	f.Add("0.0e1")
	f.Add("18446744073709551620")
	f.Add("1e23")
	f.Fuzz(func(t *testing.T, s string) {
		ParseFloat([]byte(s))
	})
}

func FuzzAppendFloat(f *testing.F) {
	f.Add(0.0, 6)
	f.Add(1.0, 6)
	f.Add(9.0, 6)
	f.Add(9.99999, 6)
	f.Add(123.0, 6)
	f.Add(0.123456, 6)
	f.Add(0.066, 6)
	f.Add(0.0066, 6)
	f.Add(12e2, 6)
	f.Add(12e3, 6)
	f.Add(0.1, 6)
	f.Add(0.001, 6)
	f.Add(0.0001, 6)
	f.Add(-1.0, 6)
	f.Add(-123.0, 6)
	f.Add(-123.456, 6)
	f.Add(-12e3, 6)
	f.Add(-0.1, 6)
	f.Add(-0.0001, 6)
	f.Add(0.000100009, 10)
	f.Add(0.0001000009, 10)
	f.Add(1e18, 0)
	f.Add(1e1, 0)
	f.Add(1e2, 1)
	f.Add(1e3, 2)
	f.Add(1e10, -1)
	f.Add(1e15, -1)
	f.Add(1e-5, 6)
	f.Add(math.NaN(), 0)
	f.Add(math.Inf(1), 0)
	f.Add(math.Inf(-1), 0)
	f.Add(0.0, 19)
	f.Add(0.000923361977200859392, -1)
	f.Add(1234.0, 2)
	f.Add(12345.0, 2)
	f.Add(12.345, 2)
	f.Add(12.345, 3)
	f.Fuzz(func(t *testing.T, f float64, prec int) {
		AppendFloat([]byte{}, f, prec)
	})
}

////////////////////////////////////////////////////////////////

// ParseFloat keeps at most 19-20 significant digits, so it is not bit-exact
// against the stdlib, but it must stay within a few ULP of it.
func TestParseFloatRandom(t *testing.T) {
	N := int(2e5)
	if testing.Short() {
		N = int(1e4)
	}
	r := rand.New(rand.NewSource(6))
	bad := 0
	for i := 0; i < N; i++ {
		var sb strings.Builder
		if r.Intn(4) == 0 {
			sb.WriteByte('-')
		}
		// vary the integer length across the uint64 accumulator limit
		sb.WriteByte(byte('1' + r.Intn(9)))
		for j := 1 + r.Intn(30); 0 < j; j-- {
			sb.WriteByte(byte('0' + r.Intn(10)))
		}
		if frac := r.Intn(14); 0 < frac || r.Intn(4) == 0 {
			sb.WriteByte('.')
			for ; 0 < frac; frac-- {
				sb.WriteByte(byte('0' + r.Intn(10)))
			}
		}
		if r.Intn(2) == 0 {
			sb.WriteString("e")
			sb.WriteString(strconv.Itoa(r.Intn(700) - 350))
		}
		s := sb.String()
		expected, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsInf(expected, 0) || math.Abs(expected) < 1e-280 {
			continue // below 1e-280 scaling runs into subnormals
		}
		f, n := ParseFloat([]byte(s))
		if n != len(s) || math.Abs(f-expected) > 1e-14*math.Abs(expected) {
			if bad++; bad < 10 {
				t.Errorf("ParseFloat(%v): %v (%v bytes) != %v", s, f, n, expected)
			}
		}
	}
	if 0 < bad {
		t.Errorf("%v of %v literals differ from strconv.ParseFloat", bad, N)
	}
}

func TestAppendFloatRandom(t *testing.T) {
	N := int(1e6)
	if testing.Short() {
		N = 0
	}
	r := rand.New(rand.NewSource(99))
	//prec := 10
	for i := 0; i < N; i++ {
		f := r.ExpFloat64()
		//f = math.Floor(f*float64(prec)) / float64(prec)

		b := AppendFloat([]byte{}, f, -1)
		f2, _ := strconv.ParseFloat(string(b), 64)
		if math.Abs(f-f2) > 1e-6 {
			fmt.Println("Bad:", f, "!=", f2, "in", string(b))
		}
	}
}

func BenchmarkFloatToBytes1(b *testing.B) {
	r := []byte{} //make([]byte, 10)
	f := 123.456
	for i := 0; i < b.N; i++ {
		r = strconv.AppendFloat(r[:0], f, 'g', 6, 64)
	}
}

func BenchmarkFloatToBytes2(b *testing.B) {
	r := make([]byte, 10)
	f := 123.456
	for i := 0; i < b.N; i++ {
		r = AppendFloat(r[:0], f, 6)
	}
}

func BenchmarkModf1(b *testing.B) {
	f := 123.456
	x := 0.0
	for i := 0; i < b.N; i++ {
		a, b := math.Modf(f)
		x += a + b
	}
}

func BenchmarkModf2(b *testing.B) {
	f := 123.456
	x := 0.0
	for i := 0; i < b.N; i++ {
		a := float64(int64(f))
		b := f - a
		x += a + b
	}
}

func BenchmarkPrintInt1(b *testing.B) {
	X := int64(123456789)
	n := LenInt(X)
	r := make([]byte, n)
	for i := 0; i < b.N; i++ {
		x := X
		j := n
		for x > 0 {
			j--
			r[j] = '0' + byte(x%10)
			x /= 10
		}
	}
}

func BenchmarkPrintInt2(b *testing.B) {
	X := int64(123456789)
	n := LenInt(X)
	r := make([]byte, n)
	for i := 0; i < b.N; i++ {
		x := X
		j := n
		for x > 0 {
			j--
			newX := x / 10
			r[j] = '0' + byte(x-10*newX)
			x = newX
		}
	}
}

func BenchmarkPrintInt3(b *testing.B) {
	X := int64(123456789)
	n := LenInt(X)
	r := make([]byte, n)
	for i := 0; i < b.N; i++ {
		x := X
		j := 0
		for j < n {
			pow := int64pow10[n-j-1]
			tmp := x / pow
			r[j] = '0' + byte(tmp)
			j++
			x -= tmp * pow
		}
	}
}
