package parse

import (
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
