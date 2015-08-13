package css // import "github.com/tdewolff/parse/css"

import "bytes"

// IsIdent returns true if the bytes are a valid identifier.
func IsIdent(b []byte) bool {
	l := NewLexer(bytes.NewBuffer(b))
	l.consumeIdentToken()
	return l.r.Pos() == len(b)
}

// IsUrlUnquoted returns true if the bytes are a valid unquoted URL.
func IsUrlUnquoted(b []byte) bool {
	l := NewLexer(bytes.NewBuffer(b))
	l.consumeUnquotedURL()
	return l.r.Pos() == len(b)
}

// Hsl2Rgb converts HSL to RGB with all of range [0,1]
// from http://www.w3.org/TR/css3-color/#hsl-color
func Hsl2Rgb(h, s, l float64) (float64, float64, float64) {
	m2 := l * (s + 1)
	if l > 0.5 {
		m2 = l + s - l*s
	}
	m1 := l*2 - m2
	return hue2rgb(m1, m2, h+1.0/3.0), hue2rgb(m1, m2, h), hue2rgb(m1, m2, h-1.0/3.0)
}

func hue2rgb(m1, m2, h float64) float64 {
	for h < 0.0 {
		h += 1.0
	}
	for h > 1.0 {
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
