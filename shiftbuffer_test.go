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

	s := `abcdefgh`
	b := NewShiftBuffer(&ReaderMockup{bytes.NewBufferString(s)})

	b.Move(4)
	assert.Equal(t, false, b.IsEOF(), "buffer must not be fully in memory")
	assert.Equal(t, byte('e'), b.Peek(0), "first character must be 'e' at position 4")
	b.Move(4)
	assert.Equal(t, byte(0), b.Peek(0), "first character past max buffer size must give error and return 0")
	assert.Equal(t, ErrBufferExceeded, b.Err(), "error must be ErrBufferExceeded when past the max buffer size")
	assert.Equal(t, byte(0), b.Peek(0), "peek when readErr != nil must return 0")

	b = NewShiftBuffer(&ReaderMockup{bytes.NewBufferString(s)})
	assert.Equal(t, byte('f'), b.Peek(5), "first character must be 'f' at position 5")
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

////////////////////////////////////////////////////////////////

var c = 0
var haystack = []byte("abcdefghijklmnopqrstuvwxyz")

func BenchmarkBytesEqual(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if bytes.Equal([]byte("wxyz"), haystack[j:j+4]) {
			c++
		}
	}
}

func BenchmarkBytesEqual2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if bytes.Equal([]byte{'w', 'x', 'y', 'z'}, haystack[j:j+4]) {
			c++
		}
	}
}

func BenchmarkBytesEqual3(b *testing.B) {
	match := []byte{'w', 'x', 'y', 'z'}
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if bytes.Equal(match, haystack[j:j+4]) {
			c++
		}
	}
}

func BenchmarkBytesEqual4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if bytesEqual(haystack[j:j+4], 'w', 'x', 'y', 'z') {
			c++
		}
	}
}

func bytesEqual(stack []byte, match ...byte) bool {
	return bytes.Equal(stack, match)
}

func BenchmarkCharsEqual(b *testing.B) {
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if haystack[j] == 'w' && haystack[j+1] == 'x' && haystack[j+2] == 'y' && haystack[j+3] == 'z' {
			c++
		}
	}
}

func BenchmarkCharsLoopEqual(b *testing.B) {
	match := []byte("wxyz")
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		equal := true
		for k := 0; k < 4; k++ {
			if haystack[j+k] != match[k] {
				equal = false
				break
			}
		}
		if equal {
			c++
		}
	}
}

func BenchmarkCharsFuncEqual(b *testing.B) {
	match := []byte("wxyz")
	for i := 0; i < b.N; i++ {
		j := i % (len(haystack)-3)
		if at(match, haystack[j:]) {
			c++
		}
	}
}

func at(match []byte, stack []byte) bool {
	if len(stack) < len(match) {
		return false
	}
	for i, c := range match {
		if stack[i] != c {
			return false
		}
	}
	return true
}