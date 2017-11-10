package parse

import (
	"fmt"
	"io"
	"strings"

	"github.com/tdewolff/buffer"
)

// Position returns the line and column number for a certain position in a file. It is useful for recovering the position in a file that caused an error.
// It only treates \n, \r, and \r\n as newlines, which might be different from some languages also recognizing \f, \u2028, and \u2029 to be newlines.
func Position(r io.Reader, offset int) (line, col int, context string, err error) {
	l := buffer.NewLexer(r)

	line = 1
	for {
		c := l.Peek(0)
		if c == 0 {
			col = l.Pos() + 1
			context = positionContext(l, line, col)
			err = l.Err()
			if err == nil {
				err = io.EOF
			}
			return
		}

		if offset == l.Pos() {
			col = l.Pos() + 1
			context = positionContext(l, line, col)
			return
		}

		if c == '\n' {
			l.Move(1)
			line++
			offset -= l.Pos()
			l.Skip()
		} else if c == '\r' {
			if l.Peek(1) == '\n' {
				if offset == l.Pos()+1 {
					l.Move(1)
					continue
				}
				l.Move(2)
			} else {
				l.Move(1)
			}
			line++
			offset -= l.Pos()
			l.Skip()
		} else {
			l.Move(1)
		}
	}
}

func positionContext(l *buffer.Lexer, line, col int) (context string) {
	for {
		c := l.Peek(0)
		if c == 0 || c == '\n' || c == '\r' {
			break
		}
		l.Move(1)
	}

	b := l.Lexeme()
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b[len(b)-1] = ' ' // if error occurs at \n in \r\n, replace \r by a space so it won't wrap
	}

	context += fmt.Sprintf("%5d: %s\n", line, string(b))
	context += fmt.Sprintf("%s^", strings.Repeat(" ", col+6))
	return
}
