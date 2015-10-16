package parse // import "github.com/tdewolff/parse"

// Copy returns a copy of the given byte slice.
func Copy(src []byte) (dst []byte) {
	dst = make([]byte, len(src))
	copy(dst, src)
	return
}

// ToLower converts all characters in the byte slice from A-Z to a-z.
func ToLower(src []byte) []byte {
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c >= 'A' && c <= 'Z' {
			src[i] = c + ('a' - 'A')
		}
	}
	return src
}

// Equal returns true when s matches the target.
func Equal(s, target []byte) bool {
	if len(s) != len(target) {
		return false
	}
	for i := 0; i < len(target); i++ {
		if s[i] != target[i] {
			return false
		}
	}
	return true
}

// EqualFold returns true when s matches case-insensitively the targetLower (which must be lowercase).
func EqualFold(s, targetLower []byte) bool {
	if len(s) != len(targetLower) {
		return false
	}
	for i := 0; i < len(targetLower); i++ {
		c := targetLower[i]
		if s[i] != c && (c < 'A' && c > 'Z' || s[i]+('a'-'A') != c) {
			return false
		}
	}
	return true
}

// Trim removes any character from the start and end for which the function returns true.
func Trim(b []byte, f func(byte) bool) []byte {
	n := len(b)
	start := n
	for i := 0; i < n; i++ {
		if !f(b[i]) {
			start = i
			break
		}
	}
	end := n
	for i := n - 1; i >= start; i-- {
		if !f(b[i]) {
			end = i + 1
			break
		}
	}
	return b[start:end]
}

var whitespaceTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, true, true, true, true, true, false, false, // tab, new line, vertical tab, form feed, carriage return
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	true, false, false, false, false, false, false, false, // space
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}

// IsWhitespace returns true for space, \n, \r, \t, \f.
func IsWhitespace(c byte) bool {
	return whitespaceTable[c]
}

// IsAllWhitespace returns true when the entire byte slice consists of space, \n, \r, \t, \f.
func IsAllWhitespace(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if !IsWhitespace(b[i]) {
			return false
		}
	}
	return true
}

// ReplaceMultipleWhitespace replaces character series of space, \n, \t, \f, \r into a single space.
func ReplaceMultipleWhitespace(b []byte) []byte {
	j := 0
	start := 0
	prevMatch := false
	for i := 0; i < len(b); i++ {
		if IsWhitespace(b[i]) {
			if !prevMatch {
				prevMatch = true
				b[i] = ' '
			} else {
				if start < i {
					if start != 0 {
						j += copy(b[j:], b[start:i])
					} else {
						j += i
					}
				}
				start = i + 1
			}
		} else {
			prevMatch = false
		}
	}
	if start != 0 {
		j += copy(b[j:], b[start:])
		return b[:j]
	}
	return b
}
