package parse

import (
	"io"
	"testing"

	"github.com/tdewolff/test"
)

func TestBinaryReaderShortRead(t *testing.T) {
	// A truncated buffer (fewer bytes remaining than the fixed-width read
	// needs) must be treated as EOF and return 0, not panic on an
	// out-of-range index into the partial slice returned by ReadBytes.
	test.T(t, NewBinaryReaderBytes([]byte{1}).ReadUint16(), uint16(0))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2}).ReadUint24(), uint32(0))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2, 3}).ReadUint32(), uint32(0))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2, 3, 4, 5, 6, 7}).ReadUint64(), uint64(0))
}

func TestBinaryReaderFullRead(t *testing.T) {
	// Complete reads keep working (big-endian is the default byte order).
	test.T(t, NewBinaryReaderBytes([]byte{1, 2}).ReadUint16(), uint16(0x0102))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2, 3}).ReadUint24(), uint32(0x010203))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2, 3, 4}).ReadUint32(), uint32(0x01020304))
	test.T(t, NewBinaryReaderBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8}).ReadUint64(), uint64(0x0102030405060708))
}

func TestBinaryReaderSeekEnd(t *testing.T) {
	// io.SeekEnd counts back from the end, so a negative offset moves earlier.
	buf := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	for _, tt := range []struct {
		off  int64
		want int64
	}{{0, 8}, {-1, 7}, {-4, 4}, {-8, 0}} {
		r := NewBinaryReaderBytes(buf)
		pos, err := r.Seek(tt.off, io.SeekEnd)
		test.T(t, err, nil)
		test.T(t, pos, tt.want)
	}

	r := NewBinaryReaderBytes(buf)
	_, err := r.Seek(-4, io.SeekEnd)
	test.T(t, err, nil)
	test.T(t, r.ReadUint32(), uint32(0x05060708))
}

func TestBinaryReaderInt24(t *testing.T) {
	for _, v := range []int32{-1, -2, -8388608, 0, 1, 8388607} {
		w := NewBinaryWriter(nil)
		w.WriteInt24(v)
		test.T(t, NewBinaryReaderBytes(w.Bytes()).ReadInt24(), v)
	}
}
