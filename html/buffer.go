package html // import "github.com/tdewolff/parse/html"

// Token is a single token unit with an attribute value (if given) and hash of the data.
type Token struct {
	TokenType
	Hash    Hash
	Data    []byte
	AttrVal []byte
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

func (z *TokenBuffer) read(t *Token) {
	tt, data, n := z.l.Next()
	var attrVal []byte
	var hash Hash
	if tt == AttributeToken {
		attrVal = z.l.AttrVal()
		hash = ToHash(data)
	} else if tt == StartTagToken || tt == EndTagToken {
		hash = ToHash(data)
	}
	t.TokenType = tt
	t.Data = data
	t.AttrVal = attrVal
	t.Hash = hash
	t.n = n
}

// Peek returns the ith element and possibly does an allocation.
// Peeking past an error will panic.
func (z *TokenBuffer) Peek(pos int) *Token {
	pos += z.pos
	if pos >= len(z.buf) {
		c := cap(z.buf)
		d := len(z.buf) - z.pos
		var buf []Token
		if 2*d > c {
			buf = make([]Token, d, 2*c)
		} else {
			buf = z.buf[:d]
		}
		copy(buf, z.buf[z.pos:])

		readinBuf := buf[d:cap(buf)]
		n := len(readinBuf)
		for i := 0; i < n; i++ {
			z.read(&readinBuf[i])
			if readinBuf[i].TokenType == ErrorToken {
				n = i + 1
				break
			}
		}
		pos -= z.pos
		z.pos, z.buf = 0, buf[:d+n]
	}
	return &z.buf[pos]
}

// Shift returns the first element and advances position.
func (z *TokenBuffer) Shift() *Token {
	if z.pos == len(z.buf) {
		z.buf = z.buf[:1]
		z.pos = 1
		t := &z.buf[0]
		z.read(t)
		z.l.Free(t.n)
		return t
	}
	t := z.Peek(0)
	z.l.Free(t.n)
	z.pos++
	return t
}
