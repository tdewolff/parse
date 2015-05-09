package html // import "github.com/tdewolff/parse/html"

import "github.com/tdewolff/parse"

// Token is a single token unit with an attribute value (if given) and hash of the data.
type Token struct {
	TokenType
	Data    []byte
	AttrVal []byte
	Hash    Hash
}

// TokenBuffer is a buffer that allows for token look-ahead.
type TokenBuffer struct {
	l *Lexer

	buf []Token
	pos int
}

// NewTokenBuffer returns a new TokenBuffer.
func NewTokenBuffer(l *Lexer) *TokenBuffer {
	return &TokenBuffer{
		l:   l,
		buf: make([]Token, 0, 8),
	}
}

func (z *TokenBuffer) read(p []Token) int {
	for i := 0; i < len(p); i++ {
		tt, data := z.l.Next()
		if !z.l.IsEOF() {
			data = parse.Copy(data)
		}

		var attrVal []byte
		var hash Hash
		if tt == AttributeToken {
			attrVal = z.l.AttrVal()
			if !z.l.IsEOF() {
				attrVal = parse.Copy(attrVal)
			}
			hash = ToHash(data)
		} else if tt == StartTagToken || tt == EndTagToken {
			hash = ToHash(data)
		}
		p[i] = Token{tt, data, attrVal, hash}
		if tt == ErrorToken {
			return i + 1
		}
	}
	return len(p)
}

// Peek returns the ith element and possibly does an allocation.
// Peeking past an error will panic.
func (z *TokenBuffer) Peek(i int) *Token {
	end := z.pos + i
	if end >= len(z.buf) {
		c := cap(z.buf)
		d := len(z.buf) - z.pos
		var buf []Token
		if 2*d > c {
			buf = make([]Token, d, 2*c)
		} else {
			buf = z.buf[:d]
		}
		copy(buf, z.buf[z.pos:])

		n := z.read(buf[d:cap(buf)])
		end -= z.pos
		z.pos, z.buf = 0, buf[:d+n]
	}
	return &z.buf[end]
}

// Shift returns the first element and advances position.
func (z *TokenBuffer) Shift() *Token {
	t := z.Peek(0)
	z.pos++
	return t
}
