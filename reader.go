package parse // import "github.com/tdewolff/parse"

import (
	"errors"
	"io"
)

// minBuf and maxBuf are the initial and maximal internal buffer size.
var minBuf = 1024
var maxBuf = 4096

// ErrBufferExceeded is returned when the internal buffer exceeds 4096 bytes, a string or comment must thus be smaller than 4kB!
var ErrBufferExceeded = errors.New("max buffer exceeded")

// ShiftBuffer is an interface for tokenisation readers, which enables peeking and shifting the buffer.
type ShiftBuffer interface {
	Read(int) byte
	Move(int)
	MoveTo(int)
	Len() int
	Bytes() []byte
	Shift() []byte
	Err() error
}

type defaultShiftBuffer struct {
	buf []byte
	pos int
	n   int
}

// Move advances the 0 position of read.
func (z *defaultShiftBuffer) Move(n int) {
	z.n += n
}

// MoveTo sets the 0 position of read.
func (z *defaultShiftBuffer) MoveTo(n int) {
	z.n = n
}

// Len returns the 0 position of read.
func (z defaultShiftBuffer) Len() int {
	return z.n
}

// Bytes returns the bytes of the current selection.
func (z defaultShiftBuffer) Bytes() []byte {
	return z.buf[z.pos : z.pos+z.n]
}

// Shift returns the bytes of the current selection and advances the position.
func (z *defaultShiftBuffer) Shift() []byte {
	b := z.buf[z.pos : z.pos+z.n]
	z.pos += z.n
	z.n = 0
	return b
}

////////////////////////////////////////////////////////////////

// ShiftBufferReader is a buffered reader for tokenisation, taking an io.Reader.
type ShiftBufferReader struct {
	defaultShiftBuffer
	r       io.Reader
	readErr error
}

// NewShiftBufferReader returns a new ShiftBufferReader.
func NewShiftBufferReader(r io.Reader) *ShiftBufferReader {
	return &ShiftBufferReader{
		defaultShiftBuffer{
			buf: make([]byte, 0, minBuf),
		},
		r,
		nil,
	}
}

// Read returns the ith byte and possible does a reallocation
func (z *ShiftBufferReader) Read(i int) byte {
	if z.pos+z.n+i >= len(z.buf) {
		if z.readErr != nil {
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := z.n + i
		var buf1 []byte
		if 2*d > c {
			if 2*c > maxBuf {
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

// Err returns the error.
func (z ShiftBufferReader) Err() error {
	if z.readErr == io.EOF {
		if z.pos+z.n >= len(z.buf) {
			return io.EOF
		}
	} else if z.readErr != nil {
		return z.readErr
	}
	return nil
}

////////////////////////////////////////////////////////////////

// ShiftBufferBytes is a buffered reader for tokenisation, taking a []byte.
type ShiftBufferBytes struct {
	defaultShiftBuffer
}

// NewShiftBufferBytes returns a new ShiftBufferBytes.
func NewShiftBufferBytes(b []byte) *ShiftBufferBytes {
	return &ShiftBufferBytes{
		defaultShiftBuffer{
			buf: b,
		},
	}
}

// Read returns the ith byte and possible does a reallocation
func (z *ShiftBufferBytes) Read(i int) byte {
	if z.pos+z.n+i >= len(z.buf) {
		return 0
	}
	return z.buf[z.pos+z.n+i]
}

// Err returns the error.
func (z ShiftBufferBytes) Err() error {
	if z.pos+z.n >= len(z.buf) {
		return io.EOF
	}
	return nil
}
