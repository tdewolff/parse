package html // import "github.com/tdewolff/parse/html"

// Token is a single token unit with an attribute value (if given) and hash of the data.
type Token struct {
	TokenType
	Data    []byte
	AttrVal []byte
	Hash    Hash
	n       int
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
		tt, data, n := z.l.Next()
		var attrVal []byte
		var hash Hash
		if tt == AttributeToken {
			attrVal = z.l.AttrVal()
			hash = ToHash(data)
		} else if tt == StartTagToken || tt == EndTagToken {
			hash = ToHash(data)
		}
		p[i].TokenType = tt
		p[i].Data = data
		p[i].AttrVal = attrVal
		p[i].Hash = hash
		p[i].n = n
		if tt == ErrorToken {
			return i + 1
		}
	}
	return len(p)
}

// Peek returns the ith element and possibly does an allocation.
// Peeking past an error will panic.
func (z *TokenBuffer) Peek(end int) *Token {
	end += z.pos
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
	z.l.r.Free(t.n)
	z.pos++
	return t
}
