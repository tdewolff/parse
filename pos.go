package parse

import (
	"io"

	"github.com/tdewolff/buffer"
)

// Pos returns the line and column number for a certain position in a file. It is useful for recovering the position in a file that caused an error.
// It only treates \n, \r, and \r\n as newlines, which might be different from some languages also recognizing \f, \u2028, and \u2029 to be newlines.
func Pos(r io.Reader, pos int) (line, col int, err error) {
	l := buffer.NewMemLexer(r)

	line = 1
	for {
		c := l.Peek(0)
		if c == 0 {
			col = l.Pos() + 1
			err = l.Err()
			if err == nil {
				err = io.EOF
			}
			return
		}

		if pos == l.Pos() {
			col = l.Pos() + 1
			return
		}

		if c == '\n' {
			l.Move(1)
			line++
			pos -= l.Pos()
			l.Skip()
		} else if c == '\r' {
			if l.Peek(1) == '\n' {
				if pos == l.Pos()+1 {
					l.Move(1)
					continue
				}
				l.Move(2)
			} else {
				l.Move(1)
			}
			line++
			pos -= l.Pos()
			l.Skip()
		} else {
			l.Move(1)
		}
	}
}
