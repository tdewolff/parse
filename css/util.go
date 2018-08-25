package css // import "github.com/tdewolff/parse/css"

func isEscape(b []byte) bool {
	return len(b) > 1 && b[0] == '\\' && b[1] != '\n' && b[1] != '\f' && b[1] != '\r'
}

// IsIdent returns true if the bytes are a valid identifier.
// Must be identical to consumeIdentToken
func IsIdent(b []byte) bool {
	i := 0
	if i < len(b) && b[i] == '-' {
		i++
	}
	if i >= len(b) {
		return false
	}
	c := b[i]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80 || isEscape(b[i:])) {
		return false
	} else {
		i++
	}
	for i < len(b) {
		c := b[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80 || isEscape(b[i:])) {
			return false
		}
		i++
	}
	return true
}

// IsURLUnquoted returns true if the bytes are a valid unquoted URL.
func IsURLUnquoted(b []byte) bool {
	i := 0
	for i < len(b) {
		c := b[i]
		if c == '"' || c == '\'' || c == '(' || c == ')' || c == ' ' || c <= 0x1F || c == 0x7F || c == '\\' && !isEscape(b[i:]) {
			return false
		}
		i++
	}
	return true
}

// HSL2RGB converts HSL to RGB with all of range [0,1]
// from http://www.w3.org/TR/css3-color/#hsl-color
func HSL2RGB(h, s, l float64) (float64, float64, float64) {
	m2 := l * (s + 1)
	if l > 0.5 {
		m2 = l + s - l*s
	}
	m1 := l*2 - m2
	return hue2rgb(m1, m2, h+1.0/3.0), hue2rgb(m1, m2, h), hue2rgb(m1, m2, h-1.0/3.0)
}

func hue2rgb(m1, m2, h float64) float64 {
	if h < 0.0 {
		h += 1.0
	}
	if h > 1.0 {
		h -= 1.0
	}
	if h*6.0 < 1.0 {
		return m1 + (m2-m1)*h*6.0
	} else if h*2.0 < 1.0 {
		return m2
	} else if h*3.0 < 2.0 {
		return m1 + (m2-m1)*(2.0/3.0-h)*6.0
	}
	return m1
}
