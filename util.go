package parse // import "github.com/tdewolff/parse"
import "unsafe"

// Copy returns a copy of the given byte slice.
func Copy(src []byte) (dst []byte) {
	dst = make([]byte, len(src))
	copy(dst, src)
	return
}

// ToLower converts all characters in the byte slice from A-Z to a-z.
func ToLower(src []byte) []byte {
	for i, c := range src {
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
	for i, c := range target {
		if s[i] != c {
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
	for i, c := range targetLower {
		if s[i] != c && (c < 'A' && c > 'Z' || s[i]+('a'-'A') != c) {
			return false
		}
	}
	return true
}

// IsWhitespace returns true for space, \n, \t, \f, \r.
func IsWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

// IsAllWhitespace returns true when the entire byte slice consists of space, \n, \t, \f, \r.
func IsAllWhitespace(b []byte) bool {
	for _, c := range b {
		if !IsWhitespace(c) {
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

// ReplaceMultiple replaces any character serie for which the function return true into a single character given by r.
func ReplaceMultiple(b []byte, f func(byte) bool, r byte) []byte {
	j := 0
	start := 0
	prevMatch := false
	for i, c := range b {
		if f(c) {
			if !prevMatch {
				prevMatch = true
				b[i] = r
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

func UnsafeToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
