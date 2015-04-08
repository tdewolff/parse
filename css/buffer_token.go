package css

import "github.com/tdewolff/parse"

type tokenBuffer struct {
	tokenizer *Tokenizer
	copy      bool

	buf []TokenNode
	pos int
}

func newTokenBuffer(tokenizer *Tokenizer) *tokenBuffer {
	return &tokenBuffer{
		tokenizer: tokenizer,
		buf:       make([]TokenNode, 0, 8),
	}
}

func (z *tokenBuffer) EnableLookback() {
	z.copy = true
}

func (z *tokenBuffer) Read(p []TokenNode) int {
	prevWS := len(z.buf) > 0 && z.buf[len(z.buf)-1].TokenType == WhitespaceToken
	for i := 0; i < len(p); i++ {
		tt, data := z.tokenizer.Next()
		// ignore comments and multiple whitespace
		for tt == CommentToken || tt == WhitespaceToken && prevWS {
			tt, data = z.tokenizer.Next()
		}
		prevWS = tt == WhitespaceToken
		// copy necessary for whenever the tokenizer overwrites its buffer
		// checking if buffer has EOF optimizes for small files and files already in memory
		if !z.tokenizer.IsEOF() {
			data = parse.Copy(data)
		}
		p[i] = TokenNode{tt, data}
		if tt == ErrorToken {
			return i + 1
		}
	}
	return len(p)
}

// Peek returns the ith element and possibly does an allocation.
// Peeking past an error will panic.
func (z *tokenBuffer) Peek(i int) *TokenNode {
	end := z.pos + i
	if end >= len(z.buf) {
		c := cap(z.buf)
		d := len(z.buf) - z.pos
		var buf []TokenNode
		if 2*d > c || z.copy {
			buf = make([]TokenNode, d, 2*c)
		} else {
			buf = z.buf[:d]
		}
		copy(buf, z.buf[z.pos:])

		n := z.Read(buf[d:cap(buf)])
		end -= z.pos
		z.pos, z.buf = 0, buf[:d+n]
	}
	return &z.buf[end]
}

// Shift returns the first element and advances position.
func (z *tokenBuffer) Shift() *TokenNode {
	t := z.Peek(0)
	z.pos++
	return t
}
