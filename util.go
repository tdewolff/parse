package parse // import "github.com/tdewolff/parse"

func Copy(src []byte) (dst []byte) {
	dst = make([]byte, len(src))
	copy(dst, src)
	return
}

func ToLower(src []byte) []byte {
	for i, c := range src {
		if c >= 'A' && c <= 'Z' {
			src[i] = c + ('a' - 'A')
		}
	}
	return src
}

func CopyToLower(src []byte) []byte {
	dst := Copy(src)
	for i, c := range dst {
		if c >= 'A' && c <= 'Z' {
			dst[i] = c + ('a' - 'A')
		}
	}
	return dst
}

func Equal(s, match []byte) bool {
	if len(s) != len(match) {
		return false
	}
	for i, c := range match {
		if s[i] != c {
			return false
		}
	}
	return true
}

func EqualCaseInsensitive(s, match []byte) bool {
	if len(s) != len(match) {
		return false
	}
	for i, c := range match {
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

func IsAllWhitespace(b []byte) bool {
	for _, c := range b {
		if !IsWhitespace(c) {
			return false
		}
	}
	return true
}

// Trim removes any character from the front and the end that matches the function.
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

// ReplaceMultiple replaces any character serie that matches the function into a single character.
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

func NormalizeContentType(b []byte) []byte {
	j := 0
	start := 0
	inString := false
	for i, c := range b {
		if !inString && IsWhitespace(c) {
			if start != 0 {
				j += copy(b[j:], b[start:i])
			} else {
				j += i
			}
			start = i + 1
		} else if c == '"' {
			inString = !inString
		}
	}
	if start != 0 {
		j += copy(b[j:], b[start:])
		return ToLower(b[:j])
	}
	return ToLower(b)
}
