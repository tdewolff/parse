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
