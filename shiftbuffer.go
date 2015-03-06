package parse // import "github.com/tdewolff/parse"

import (
	"errors"
	"io"
)

// MinBuf and MaxBuf are the initial and maximal internal buffer size.
var MinBuf = 1024
var MaxBuf = 4096

// ErrBufferExceeded is returned when the internal buffer exceeds 4096 bytes, a string or comment must thus be smaller than 4kB!
var ErrBufferExceeded = errors.New("max buffer exceeded")

////////////////////////////////////////////////////////////////

// ShiftBuffer is a buffered reader that allows peeking forward and shifting, taking an io.Reader.
type ShiftBuffer struct {
	r   io.Reader
	err error

	buf []byte
	pos int
	n   int
}

// NewShiftBufferReader returns a new ShiftBuffer for a given io.Reader.
func NewShiftBuffer(r io.Reader) *ShiftBuffer {
	// If reader has the bytes in memory already, use that instead!
	if fr, ok := r.(interface {
		Bytes() []byte
	}); ok {
		return &ShiftBuffer{
			err: io.EOF,
			buf: fr.Bytes(),
		}
	}

	b := &ShiftBuffer{
		r:   r,
		buf: make([]byte, 0, MinBuf),
	}
	b.Peek(0)
	return b
}

// Err returns the error.
func (z ShiftBuffer) Err() error {
	if z.err == io.EOF && z.pos+z.n < len(z.buf) {
		return nil
	}
	return z.err
}

// IsEOF returns true when it has encountered EOF and thus loaded the last buffer in memory.
func (z ShiftBuffer) IsEOF() bool {
	return z.err == io.EOF
}

// Move advances the 0 position of read.
func (z *ShiftBuffer) Move(n int) {
	z.n += n
}

// MoveTo sets the 0 position of read.
func (z *ShiftBuffer) MoveTo(n int) {
	z.n = n
}

// Pos returns the 0 position of read.
func (z ShiftBuffer) Pos() int {
	return z.n
}

// Len returns the length of the buffer.
func (z ShiftBuffer) Len() int {
	return len(z.buf) - z.pos
}

// Peek returns the ith byte and possible does a reallocation
func (z *ShiftBuffer) Peek(i int) byte {
	if z.pos+z.n+i >= len(z.buf) {
		if z.err != nil {
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := z.n + i
		var buf1 []byte
		if 2*d > c {
			if 2*c > MaxBuf {
				z.err = ErrBufferExceeded
				return 0
			}
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = z.buf[:d]
		}
		copy(buf1, z.buf[z.pos:z.pos+d])

		// Read in to fill the buffer till capacity
		var n int
		n, z.err = z.r.Read(buf1[d:cap(buf1)])
		z.pos, z.buf = 0, buf1[:d+n]
		if n == 0 {
			return 0
		}
	}
	return z.buf[z.pos+z.n+i]
}

// PeekRune returns the rune of the ith byte.
func (z *ShiftBuffer) PeekRune(i int) rune {
	// from unicode/utf8
	c := z.Peek(i)
	if c < 0xC0 {
		return rune(c)
	} else if c < 0xE0 {
		return rune(c&0x1F)<<6 | rune(z.Peek(i+1)&0x3F)
	} else if c < 0xF0 {
		return rune(c&0x0F)<<12 | rune(z.Peek(i+1)&0x3F)<<6 | rune(z.Peek(i+2)&0x3F)
	} else {
		return rune(c&0x07)<<18 | rune(z.Peek(i+1)&0x3F)<<12 | rune(z.Peek(i+2)&0x3F)<<6 | rune(z.Peek(i+3)&0x3F)
	}
}

// Buffered returns the bytes of the current selection.
func (z ShiftBuffer) Buffered() []byte {
	return z.buf[z.pos : z.pos+z.n]
}

// Shift returns the bytes of the current selection and advances the position.
func (z *ShiftBuffer) Shift() []byte {
	b := z.buf[z.pos : z.pos+z.n]
	z.pos += z.n
	z.n = 0
	return b
}
