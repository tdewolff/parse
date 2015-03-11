package parse // import "github.com/tdewolff/parse"

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Don't implement Bytes() to test for buffer exceeding.
type ReaderMockup struct {
	r io.Reader
}

func (r *ReaderMockup) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

////////////////////////////////////////////////////////////////

func TestShiftBuffer(t *testing.T) {
	var s = `Lorem ipsum dolor sit amet, consectetur adipiscing elit.`
	var b = NewShiftBuffer(bytes.NewBufferString(s))

	assert.Equal(t, true, b.IsEOF(), "buffer must be fully in memory")
	assert.Equal(t, 0, b.Pos(), "buffer must start at position 0")
	assert.Equal(t, byte('L'), b.Peek(0), "first character must be 'L'")
	assert.Equal(t, byte('o'), b.Peek(1), "second character must be 'o'")

	b.Move(1)
	assert.Equal(t, byte('o'), b.Peek(0), "must be 'o' at position 1")
	assert.Equal(t, byte('r'), b.Peek(1), "must be 'r' at position 1")
	b.MoveTo(6)
	assert.Equal(t, byte('i'), b.Peek(0), "must be 'i' at position 6")
	assert.Equal(t, byte('p'), b.Peek(1), "must be 'p' at position 7")

	assert.Equal(t, []byte("Lorem "), b.Bytes(), "buffered string must now read 'Lorem ' when at position 6")
	assert.Equal(t, []byte("Lorem "), b.Shift(), "shift must return the buffered string")
	assert.Equal(t, 0, b.Pos(), "after shifting position must be 0")
	assert.Equal(t, byte('i'), b.Peek(0), "must be 'i' at position 0 after shifting")
	assert.Equal(t, byte('p'), b.Peek(1), "must be 'p' at position 1 after shifting")
	assert.Nil(t, b.Err(), "error must be nil at this point")

	b.Move(len(s) - len("Lorem ") - 1)
	assert.Nil(t, b.Err(), "error must be nil just before the end of the buffer")
	b.Move(1)
	assert.Equal(t, io.EOF, b.Err(), "error must be EOF when past the buffer")
	b.Move(-1)
	assert.Nil(t, b.Err(), "error must be nil just before the end of the buffer, even when it has been past the buffer")
}

func TestShiftBufferSmall(t *testing.T) {
	MinBuf = 4
	MaxBuf = 8

	var s = `abcdefgh`
	var b = NewShiftBuffer(&ReaderMockup{bytes.NewBufferString(s)})

	b.Move(4)
	assert.Equal(t, false, b.IsEOF(), "buffer must not be fully in memory")
	assert.Equal(t, byte('e'), b.Peek(0), "first character must be 'e' at position 4")
	b.Move(4)
	assert.Equal(t, byte(0), b.Peek(0), "first character past max buffer size must give error and return 0")
	assert.Equal(t, ErrBufferExceeded, b.Err(), "error must be ErrBufferExceeded when past the max buffer size")
	assert.Equal(t, byte(0), b.Peek(0), "peek when readErr != nil must return 0")
}

func TestShiftBufferRunes(t *testing.T) {
	var b = NewShiftBuffer(bytes.NewBufferString("aæ†"))
	assert.Equal(t, 'a', b.PeekRune(0), "first character must be rune 'a'")
	assert.Equal(t, 'æ', b.PeekRune(1), "second character must be rune 'æ'")
	assert.Equal(t, '†', b.PeekRune(3), "fourth character must be rune '†'")
	// Can't test 4 byte unicode codepoints my editor
}

func TestShiftBufferZeroLen(t *testing.T) {
	var b = NewShiftBuffer(&ReaderMockup{bytes.NewBufferString("")})
	assert.Equal(t, byte(0), b.Peek(0), "first character must yield error")
}