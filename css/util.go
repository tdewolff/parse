package css // import "github.com/tdewolff/parse/css"

import "bytes"

// SplitNumberDimension splits the data of a dimension token into the number and dimension parts.
func SplitNumberDimension(b []byte) ([]byte, []byte, bool) {
	split := len(b)
	for i := len(b) - 1; i >= 0; i-- {
		c := b[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '%' {
			split = i + 1
			break
		}
	}
	for i := split - 1; i >= 0; i-- {
		c := b[i]
		if (c < '0' || c > '9') && c != '.' && c != '+' && c != '-' && c != 'e' && c != 'E' {
			return nil, nil, false
		}
	}
	return b[:split], b[split:], true
}

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
