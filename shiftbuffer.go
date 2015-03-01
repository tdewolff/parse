package parse // import "github.com/tdewolff/parse"

import (
	"errors"
	"io"
)

// minBuf and maxBuf are the initial and maximal internal buffer size.
var MinBuf = 1024
var MaxBuf = 4096

// ErrBufferExceeded is returned when the internal buffer exceeds 4096 bytes, a string or comment must thus be smaller than 4kB!
var ErrBufferExceeded = errors.New("max buffer exceeded")

////////////////////////////////////////////////////////////////

// ShiftBuffer is a buffered reader that allows peeking forward and shifting, taking an io.Reader.
type ShiftBuffer struct {
	r       io.Reader
	readErr error

	buf []byte
	pos int
	n   int

	minBuf, maxBuf int
}

// NewShiftBufferReader returns a new ShiftBuffer.
func NewShiftBuffer(r io.Reader) *ShiftBuffer {
	// If reader has the bytes in memory already, use that instead!
	if fr, ok := r.(interface {
		Bytes() []byte
	}); ok {
		return &ShiftBuffer{
			readErr: io.EOF,
			buf:     fr.Bytes(),
		}
	}
	return &ShiftBuffer{
		r:   r,
		buf: make([]byte, 0, MinBuf),
	}
}

// Err returns the error.
func (z ShiftBuffer) Err() error {
	return z.readErr
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
	return len(z.buf)
}

// Peek returns the ith byte and possible does a reallocation
func (z *ShiftBuffer) Peek(i int) byte {
	if z.pos+z.n+i >= len(z.buf) {
		if z.readErr != nil {
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := z.n + i
		var buf1 []byte
		if 2*d > c {
			if 2*c > MaxBuf {
				z.readErr = ErrBufferExceeded
				return 0
			}
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = z.buf[:d]
		}
		copy(buf1, z.buf[z.pos:z.pos+d])

		// Read in to fill the buffer till capacity
		var n int
		n, z.readErr = z.r.Read(buf1[d:cap(buf1)])
		z.pos, z.buf = 0, buf1[:d+n]
		if n == 0 {
			return 0
		}
	}
	return z.buf[z.pos+z.n+i]
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
