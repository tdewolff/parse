package parse

import (
	"fmt"
	"io"

	"github.com/tdewolff/parse/buffer"
)

type Error struct {
	Message string
	r       io.Reader
	Offset  int
	line    int
	column  int
	context string
}

func NewError(msg string, r io.Reader, offset int) *Error {
	return &Error{
		Message: msg,
		r:       r,
		Offset:  offset,
	}
}

func NewErrorLexer(msg string, l *buffer.Lexer) *Error {
	r := buffer.NewReader(l.Bytes())
	offset := l.Offset()
	return NewError(msg, r, offset)
}

func (e *Error) Position() (int, int, string) {
	if e.line == 0 {
		e.line, e.column, e.context, _ = Position(e.r, e.Offset)
	}
	return e.line, e.column, e.context
}

func (e *Error) Error() string {
	line, column, context := e.Position()
	return fmt.Sprintf("parse error:%d:%d: %s\n%s", line, column, e.Message, context)
}
