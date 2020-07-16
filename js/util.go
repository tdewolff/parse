package js

import "github.com/tdewolff/parse/v2"

func ParseIdentifierName(b []byte) (TokenType, bool) {
	// TODO: optimize: check for first character first? don't use NewLexer?
	l := NewLexer(parse.NewInputBytes(b))
	tt := l.consumeIdentifierToken()
	l.r.Restore()
	return tt, l.r.Pos() == len(b)
}

func ParseNumericLiteral(b []byte) (TokenType, bool) {
	if len(b) == 0 || (b[0] < '0' || '9' < b[0]) && b[0] != '.' {
		return 0, false
	}
	// TODO: optimize
	l := NewLexer(parse.NewInputBytes(b))
	tt := l.consumeNumericToken()
	l.r.Restore()
	return tt, l.r.Pos() == len(b)
}
