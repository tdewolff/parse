package parse

import (
	"fmt"
	"io"

	"github.com/tdewolff/parse/v2/buffer"
)

// Error is a parsing error returned by parser. It contains a message and an offset at which the error occurred.
type Error struct {
	Message string
	Line    int
	Column  int
	Context string
}

// NewError creates a new error
func NewError(msg string, r io.Reader, offset int) *Error {
	line, column, context := Position(r, offset)
	return &Error{
		Message: msg,
		Line:    line,
		Column:  column,
		Context: context,
	}
}

// NewErrorLexer creates a new error from an active Lexer.
func NewErrorLexer(msg string, l *buffer.Lexer) *Error {
	r := buffer.NewReader(l.Bytes())
	offset := l.Offset()
	return NewError(msg, r, offset)
}

// Positions returns the line, column, and context of the error.
// Context is the entire line at which the error occurred.
func (e *Error) Position() (int, int, string) {
	return e.Line, e.Column, e.Context
}

// Error returns the error string, containing the context and line + column number.
func (e *Error) Error() string {
	return fmt.Sprintf("%s on line %d and column %d\n%s", e.Message, e.Line, e.Column, e.Context)
}
