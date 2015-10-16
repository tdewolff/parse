// Package xml is an XML1.0 lexer following the specifications at http://www.w3.org/TR/xml/.
package xml // import "github.com/tdewolff/parse/xml"

import (
	"io"
	"strconv"

	"github.com/tdewolff/buffer"
)

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	CommentToken
	DOCTYPEToken
	CDATAToken
	StartTagToken
	StartTagPIToken
	StartTagCloseToken
	StartTagCloseVoidToken
	StartTagClosePIToken
	EndTagToken
	AttributeToken
	TextToken
)

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	switch tt {
	case ErrorToken:
		return "Error"
	case CommentToken:
		return "Comment"
	case DOCTYPEToken:
		return "DOCTYPE"
	case CDATAToken:
		return "CDATA"
	case StartTagToken:
		return "StartTag"
	case StartTagPIToken:
		return "StartTagPI"
	case StartTagCloseToken:
		return "StartTagClose"
	case StartTagCloseVoidToken:
		return "StartTagCloseVoid"
	case StartTagClosePIToken:
		return "StartTagClosePI"
	case EndTagToken:
		return "EndTag"
	case AttributeToken:
		return "Attribute"
	case TextToken:
		return "Text"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Lexer is the state for the lexer.
type Lexer struct {
	r *buffer.Lexer

	inTag   bool
	attrVal []byte
}

// NewLexer returns a new Lexer for a given io.Reader.
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{
		r: buffer.NewLexer(r),
	}
}

// Err returns the error encountered during lexing, this is often io.EOF but also other errors can be returned.
func (l Lexer) Err() error {
	return l.r.Err()
}

//
func (l *Lexer) Free(n int) {
	l.r.Free(n)
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (l *Lexer) Next() (TokenType, []byte, int) {
	var c byte
	if l.inTag {
		l.attrVal = nil
		c = l.r.Peek(0)
		if c == 0 {
			return ErrorToken, nil, l.r.ShiftLen()
		} else if c != '>' && (c != '/' && c != '?' || l.r.Peek(1) != '>') {
			return AttributeToken, l.shiftAttribute(), l.r.ShiftLen()
		}
		l.inTag = false
		if c == '/' {
			l.r.Move(2)
			return StartTagCloseVoidToken, l.r.Shift(), l.r.ShiftLen()
		} else if c == '?' {
			l.r.Move(2)
			return StartTagClosePIToken, l.r.Shift(), l.r.ShiftLen()
		} else {
			l.r.Move(1)
			return StartTagCloseToken, l.r.Shift(), l.r.ShiftLen()
		}
	}

	for {
		c = l.r.Peek(0)
		if c == '<' {
			if l.r.Pos() > 0 {
				return TextToken, l.r.Shift(), l.r.ShiftLen()
			}
			c = l.r.Peek(1)
			if c == '/' {
				l.r.Move(2)
				l.r.Skip()
				return EndTagToken, l.shiftEndTag(), l.r.ShiftLen()
			} else if c == '!' {
				l.r.Move(2)
				if l.at('-', '-') {
					l.r.Move(2)
					return CommentToken, l.shiftCommentText(), l.r.ShiftLen()
				} else if l.at('[', 'C', 'D', 'A', 'T', 'A', '[') {
					l.r.Move(7)
					return CDATAToken, l.shiftCDATAText(), l.r.ShiftLen()
				} else if l.at('D', 'O', 'C', 'T', 'Y', 'P', 'E') {
					l.r.Move(8)
					return DOCTYPEToken, l.shiftDOCTYPEText(), l.r.ShiftLen()
				}
				l.r.Move(-2)
			} else if c == '?' {
				l.r.Move(2)
				l.inTag = true
				return StartTagPIToken, l.shiftStartTag(), l.r.ShiftLen()
			}
			l.r.Move(1)
			l.inTag = true
			return StartTagToken, l.shiftStartTag(), l.r.ShiftLen()
		} else if c == 0 {
			if l.r.Pos() > 0 {
				return TextToken, l.r.Shift(), l.r.ShiftLen()
			}
			return ErrorToken, nil, l.r.ShiftLen()
		}
		l.r.Move(1)
	}
}

// AttrVal returns the attribute value when an AttributeToken was returned from Next.
func (l *Lexer) AttrVal() []byte {
	return l.attrVal
}

////////////////////////////////////////////////////////////////

// The following functions follow the specifications at http://www.w3.org/html/wg/drafts/html/master/syntax.html

func (l *Lexer) shiftDOCTYPEText() []byte {
	l.r.Skip()
	inString := false
	inBrackets := false
	for {
		c := l.r.Peek(0)
		if c == '"' {
			inString = !inString
		} else if (c == '[' || c == ']') && !inString {
			inBrackets = (c == '[')
		} else if c == '>' && !inString && !inBrackets {
			doctype := l.r.Shift()
			l.r.Move(1)
			l.r.Skip()
			return doctype
		} else if c == 0 {
			return l.r.Shift()
		}
		l.r.Move(1)
	}
}

func (l *Lexer) shiftCDATAText() []byte {
	l.r.Skip()
	for {
		c := l.r.Peek(0)
		if c == ']' && l.r.Peek(1) == ']' && l.r.Peek(2) == '>' {
			cdata := l.r.Shift()
			l.r.Move(3)
			l.r.Skip()
			return cdata
		} else if c == 0 {
			return l.r.Shift()
		}
		l.r.Move(1)
	}
}

func (l *Lexer) shiftCommentText() []byte {
	l.r.Skip()
	for {
		c := l.r.Peek(0)
		if c == '-' && l.r.Peek(1) == '-' && l.r.Peek(2) == '>' {
			comment := l.r.Shift()
			l.r.Move(3)
			l.r.Skip()
			return comment
		} else if c == 0 {
			return l.r.Shift()
		}
		l.r.Move(1)
	}
}

func (l *Lexer) shiftStartTag() []byte {
	l.r.Skip()
	for {
		if c := l.r.Peek(0); c == ' ' || c == '>' || (c == '/' || c == '?') && l.r.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
			break
		}
		l.r.Move(1)
	}
	nameEnd := l.r.Pos()
	l.moveWhitespace() // before attribute name state
	return l.r.Shift()[:nameEnd]
}

func (l *Lexer) shiftAttribute() []byte {
	for { // attribute name state
		if c := l.r.Peek(0); c == ' ' || c == '=' || c == '>' || (c == '/' || c == '?') && l.r.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
			break
		}
		l.r.Move(1)
	}
	nameEnd := l.r.Pos()
	l.moveWhitespace() // after attribute name state
	if l.r.Peek(0) == '=' {
		l.r.Move(1)
		l.moveWhitespace() // before attribute value state
		attrPos := l.r.Pos()
		delim := l.r.Peek(0)
		if delim == '"' || delim == '\'' { // attribute value single- and double-quoted state
			l.r.Move(1)
			for {
				c := l.r.Peek(0)
				if c == delim {
					l.r.Move(1)
					break
				} else if c == 0 {
					break
				}
				l.r.Move(1)
				if c == '\t' || c == '\n' || c == '\r' {
					l.r.Lexeme()[l.r.Pos()-1] = ' '
				}
			}
		} else { // attribute value unquoted state
			for {
				if c := l.r.Peek(0); c == ' ' || c == '>' || (c == '/' || c == '?') && l.r.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
					break
				}
				l.r.Move(1)
			}
		}
		attrEnd := l.r.Pos()
		l.moveWhitespace() // before attribute name state or after attribute quoted value state
		l.attrVal = l.r.Lexeme()[attrPos:attrEnd]
	} else {
		l.attrVal = nil
	}
	return l.r.Shift()[:nameEnd]
}

func (l *Lexer) shiftEndTag() []byte {
	for {
		c := l.r.Peek(0)
		if c == '>' {
			name := l.r.Shift()
			l.r.Move(1)
			l.r.Skip()
			return name
		} else if c == 0 {
			return l.r.Shift()
		}
		l.r.Move(1)
	}
}

////////////////////////////////////////////////////////////////

func (l *Lexer) moveWhitespace() {
	for {
		c := l.r.Peek(0)
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' || c == 0 {
			break
		}
		l.r.Move(1)
	}
}

func (l *Lexer) at(b ...byte) bool {
	for i, c := range b {
		if l.r.Peek(i) != c {
			return false
		}
	}
	return true
}
