// Package parse contains a collection of parsers for various formats in its subpackages.
package parse // import "github.com/tdewolff/parse"

import (
	"encoding/base64"
	"errors"
	"math"
	"net/url"
)

// ErrBadDataURI is returned by DataURI when the byte slice does not start with 'data:' or is too short.
var ErrBadDataURI = errors.New("not a data URI")

// Number returns the number of bytes that parse as a number of the regex format (+|-)?([0-9]+(\.[0-9]+)?|\.[0-9]+)((e|E)(+|-)?[0-9]+)?.
func Number(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	i := 0
	if b[i] == '+' || b[i] == '-' {
		i++
		if i >= len(b) {
			return 0
		}
	}
	firstDigit := (b[i] >= '0' && b[i] <= '9')
	if firstDigit {
		i++
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
	}
	if i < len(b) && b[i] == '.' {
		i++
		if i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
			for i < len(b) && b[i] >= '0' && b[i] <= '9' {
				i++
			}
		} else if firstDigit {
			// . could belong to the next token
			i--
			return i
		} else {
			return 0
		}
	} else if !firstDigit {
		return 0
	}
	iOld := i
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		i++
		if i < len(b) && (b[i] == '+' || b[i] == '-') {
			i++
		}
		if i >= len(b) || b[i] < '0' || b[i] > '9' {
			// e could belong to next token
			return iOld
		}
		for i < len(b) && b[i] >= '0' && b[i] <= '9' {
			i++
		}
	}
	return i
}

// Dimension parses a byte-slice and returns the length of the number and its unit.
func Dimension(b []byte) (int, int) {
	num := Number(b)
	if num == 0 || num == len(b) {
		return num, 0
	} else if b[num] == '%' {
		return num, 1
	} else if b[num] >= 'a' && b[num] <= 'z' || b[num] >= 'A' && b[num] <= 'Z' {
		i := num + 1
		for i < len(b) && (b[i] >= 'a' && b[i] <= 'z' || b[i] >= 'A' && b[i] <= 'Z') {
			i++
		}
		return num, i - num
	}
	return num, 0
}

// Int parses a byte-slice and returns the integer it represents.
// If an invalid character is encountered, it will stop there.
func Int(b []byte) (int64, bool) {
	neg := false
	if len(b) > 0 && (b[0] == '+' || b[0] == '-') {
		neg = b[0] == '-'
		b = b[1:]
	}
	n := uint64(0)
	for i := 0; i < len(b); i++ {
		c := b[i]
		if n > math.MaxUint64/10 {
			return 0, false
		} else if c >= '0' && c <= '9' {
			n *= 10
			n += uint64(c - '0')
		} else {
			break
		}
	}
	if !neg && n > uint64(math.MaxInt64) || n > uint64(math.MaxInt64)+1 {
		return 0, false
	} else if neg {
		return -int64(n), true
	}
	return int64(n), true
}

var float64pow10 = []float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
	1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
	1e20, 1e21, 1e22,
}

// Float parses a byte-slice and returns the float it represents.
// If an invalid character is encountered, it will stop there.
func Float(b []byte) (float64, bool) {
	i := 0
	neg := false
	if i < len(b) && (b[i] == '+' || b[i] == '-') {
		neg = b[i] == '-'
		i++
	}
	dot := -1
	trunk := -1
	n := uint64(0)
	for ; i < len(b); i++ {
		c := b[i]
		if c >= '0' && c <= '9' {
			if trunk == -1 {
				if n > math.MaxUint64/10 {
					trunk = i
				} else {
					n *= 10
					n += uint64(c - '0')
				}
			}
		} else if dot == -1 && c == '.' {
			dot = i
		} else {
			break
		}
	}
	f := float64(n)
	if neg {
		f = -f
	}
	mantExp := int64(0)
	if dot != -1 {
		if trunk == -1 {
			trunk = i
		}
		mantExp = int64(trunk - dot - 1)
	} else if trunk != -1 {
		mantExp = int64(trunk - i)
	}
	expExp := int64(0)
	if i < len(b) && (b[i] == 'e' || b[i] == 'E') {
		i++
		if e, ok := Int(b[i:]); ok {
			expExp = e
		}
	}
	exp := expExp - mantExp
	// copied from strconv/atof.go
	if exp == 0 {
		return f, true
	} else if exp > 0 && exp <= 15+22 { // int * 10^k
		// If exponent is big but number of digits is not,
		// can move a few zeros into the integer part.
		if exp > 22 {
			f *= float64pow10[exp-22]
			exp = 22
		}
		if f <= 1e15 && f >= -1e15 {
			return f * float64pow10[exp], true
		}
	} else if exp < 0 && exp >= -22 { // int / 10^k
		return f / float64pow10[-exp], true
	}
	f *= math.Pow10(int(-mantExp))
	return f * math.Pow10(int(expExp)), true
}

// Mediatype parses a given mediatype and splits the mimetype from the parameters.
// It works similar to mime.ParseMediaType but is faster.
func Mediatype(b []byte) ([]byte, map[string]string) {
	i := 0
	for i < len(b) && b[i] == ' ' {
		i++
	}
	b = b[i:]
	n := len(b)
	mimetype := b
	var params map[string]string
	for i := 3; i < n; i++ { // mimetype is at least three characters long
		if b[i] == ';' || b[i] == ' ' {
			mimetype = b[:i]
			if b[i] == ' ' {
				i++
				for i < n && b[i] == ' ' {
					i++
				}
				if i < n && b[i] != ';' {
					break
				}
			}
			params = map[string]string{}
			s := string(b)
		PARAM:
			i++
			for i < n && s[i] == ' ' {
				i++
			}
			start := i
			for i < n && s[i] != '=' && s[i] != ';' && s[i] != ' ' {
				i++
			}
			key := s[start:i]
			for i < n && s[i] == ' ' {
				i++
			}
			if i < n && s[i] == '=' {
				i++
				for i < n && s[i] == ' ' {
					i++
				}
				start = i
				for i < n && s[i] != ';' && s[i] != ' ' {
					i++
				}
			} else {
				start = i
			}
			params[key] = s[start:i]
			for i < n && s[i] == ' ' {
				i++
			}
			if i < n && s[i] == ';' {
				goto PARAM
			}
			break
		}
	}
	return mimetype, params
}

// DataURI parses the given data URI and returns the mediatype, data and ok.
func DataURI(dataURI []byte) ([]byte, []byte, error) {
	if len(dataURI) > 5 && Equal(dataURI[:5], []byte("data:")) {
		dataURI = dataURI[5:]
		inBase64 := false
		var mediatype []byte
		i := 0
		for j := 0; j < len(dataURI); j++ {
			c := dataURI[j]
			if c == '=' || c == ';' || c == ',' {
				if c != '=' && Equal(TrimWhitespace(dataURI[i:j]), []byte("base64")) {
					if len(mediatype) > 0 {
						mediatype = mediatype[:len(mediatype)-1]
					}
					inBase64 = true
					i = j
				} else if c != ',' {
					mediatype = append(append(mediatype, TrimWhitespace(dataURI[i:j])...), c)
					i = j + 1
				} else {
					mediatype = append(mediatype, TrimWhitespace(dataURI[i:j])...)
				}
				if c == ',' {
					if len(mediatype) == 0 || mediatype[0] == ';' {
						mediatype = []byte("text/plain")
					}
					data := dataURI[j+1:]
					if inBase64 {
						decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
						n, err := base64.StdEncoding.Decode(decoded, data)
						if err != nil {
							return nil, nil, err
						}
						data = decoded[:n]
					} else if unescaped, err := url.QueryUnescape(string(data)); err == nil {
						data = []byte(unescaped)
					}
					return mediatype, data, nil
				}
			}
		}
	}
	return nil, nil, ErrBadDataURI
}

// QuoteEntity parses the given byte slice and returns the quote that got matched (' or ") and its entity length.
func QuoteEntity(b []byte) (quote byte, n int) {
	if len(b) < 5 || b[0] != '&' {
		return 0, 0
	}
	if b[1] == '#' {
		if b[2] == 'x' {
			i := 3
			for i < len(b) && b[i] == '0' {
				i++
			}
			if i+2 < len(b) && b[i] == '2' && b[i+2] == ';' {
				if b[i+1] == '2' {
					return '"', i + 3 // &#x22;
				} else if b[i+1] == '7' {
					return '\'', i + 3 // &#x27;
				}
			}
		} else {
			i := 2
			for i < len(b) && b[i] == '0' {
				i++
			}
			if i+2 < len(b) && b[i] == '3' && b[i+2] == ';' {
				if b[i+1] == '4' {
					return '"', i + 3 // &#34;
				} else if b[i+1] == '9' {
					return '\'', i + 3 // &#39;
				}
			}
		}
	} else if len(b) >= 6 && b[5] == ';' {
		if EqualFold(b[1:5], []byte{'q', 'u', 'o', 't'}) {
			return '"', 6 // &quot;
		} else if EqualFold(b[1:5], []byte{'a', 'p', 'o', 's'}) {
			return '\'', 6 // &apos;
		}
	}
	return 0, 0
}
