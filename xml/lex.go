// Package xml is an XML1.0 lexer following the specifications at http://www.w3.org/TR/xml/.
package xml // import "github.com/tdewolff/parse/xml"

import (
	"io"
	"parse"
	"strconv"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/buffer"
)

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
type Parser struct {
	lexer *buffer.Lexer
	err   error

	inTag bool

	text    []byte
	attrVal []byte
}

// NewLexer returns a new Lexer for a given io.Reader.
func NewParser(r io.Reader) *Parser {
	return &Parser{
		lexer: buffer.NewLexer(r),
	}
}
// NewLexer returns a new Lexer for a given io.Reader.
func NewCustomLexerParser(lexer *buffer.NewLexer) *Parser {
	return &Lexer{
		lexer,
	}
}

// Err returns the error encountered during lexing, this is often io.EOF but also other errors can be returned.
func (l *Parser) Err() error {
	if l.err != nil {
		return l.err
	}
	return l.lexer.Err()
}

// Restore restores the NULL byte at the end of the buffer.
func (l *Parser) Restore() {
	l.lexer.Restore()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (l *Parser) Next() (TokenType, []byte) {
	l.text = nil
	var c byte
	if l.inTag {
		l.attrVal = nil
		for { // before attribute name state
			if c = l.lexer.Peek(0); c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				l.lexer.Move(1)
				continue
			}
			break
		}
		if c == 0 {
			if l.lexer.Err() == nil {
				l.err = parse.NewErrorLexer("unexpected null character", l.lexer)
			}
			return ErrorToken, nil
		} else if c != '>' && (c != '/' && c != '?' || l.lexer.Peek(1) != '>') {
			return AttributeToken, l.shiftAttribute()
		}
		start := l.lexer.Pos()
		l.inTag = false
		if c == '/' {
			l.lexer.Move(2)
			l.text = l.lexer.Lexeme()[start:]
			return StartTagCloseVoidToken, l.lexer.Shift()
		} else if c == '?' {
			l.lexer.Move(2)
			l.text = l.lexer.Lexeme()[start:]
			return StartTagClosePIToken, l.lexer.Shift()
		} else {
			l.lexer.Move(1)
			l.text = l.lexer.Lexeme()[start:]
			return StartTagCloseToken, l.lexer.Shift()
		}
	}

	for {
		c = l.lexer.Peek(0)
		if c == '<' {
			if l.lexer.Pos() > 0 {
				return TextToken, l.lexer.Shift()
			}
			c = l.lexer.Peek(1)
			if c == '/' {
				l.lexer.Move(2)
				return EndTagToken, l.shiftEndTag()
			} else if c == '!' {
				l.lexer.Move(2)
				if l.at('-', '-') {
					l.lexer.Move(2)
					return CommentToken, l.shiftCommentText()
				} else if l.at('[', 'C', 'D', 'A', 'T', 'A', '[') {
					l.lexer.Move(7)
					return CDATAToken, l.shiftCDATAText()
				} else if l.at('D', 'O', 'C', 'T', 'Y', 'P', 'E') {
					l.lexer.Move(7)
					return DOCTYPEToken, l.shiftDOCTYPEText()
				}
				l.lexer.Move(-2)
			} else if c == '?' {
				l.lexer.Move(2)
				l.inTag = true
				return StartTagPIToken, l.shiftStartTag()
			}
			l.lexer.Move(1)
			l.inTag = true
			return StartTagToken, l.shiftStartTag()
		} else if c == 0 {
			if l.lexer.Pos() > 0 {
				return TextToken, l.lexer.Shift()
			}
			if l.lexer.Err() == nil {
				l.err = parse.NewErrorLexer("unexpected null character", l.lexer)
			}
			return ErrorToken, nil
		}
		l.lexer.Move(1)
	}
}

// Text returns the textual representation of a token. This excludes delimiters and additional leading/trailing characters.
func (l *Parser) Text() []byte {
	return l.text
}

// AttrVal returns the attribute value when an AttributeToken was returned from Next.
func (l *Parser) AttrVal() []byte {
	return l.attrVal
}

////////////////////////////////////////////////////////////////

// The following functions follow the specifications at http://www.w3.org/html/wg/drafts/html/master/syntax.html

func (l *Parser) shiftDOCTYPEText() []byte {
	inString := false
	inBrackets := false
	for {
		c := l.lexer.Peek(0)
		if c == '"' {
			inString = !inString
		} else if (c == '[' || c == ']') && !inString {
			inBrackets = (c == '[')
		} else if c == '>' && !inString && !inBrackets {
			l.text = l.lexer.Lexeme()[9:]
			l.lexer.Move(1)
			return l.lexer.Shift()
		} else if c == 0 {
			l.text = l.lexer.Lexeme()[9:]
			return l.lexer.Shift()
		}
		l.lexer.Move(1)
	}
}

func (l *Parser) shiftCDATAText() []byte {
	for {
		c := l.lexer.Peek(0)
		if c == ']' && l.lexer.Peek(1) == ']' && l.lexer.Peek(2) == '>' {
			l.text = l.lexer.Lexeme()[9:]
			l.lexer.Move(3)
			return l.lexer.Shift()
		} else if c == 0 {
			l.text = l.lexer.Lexeme()[9:]
			return l.lexer.Shift()
		}
		l.lexer.Move(1)
	}
}

func (l *Parser) shiftCommentText() []byte {
	for {
		c := l.lexer.Peek(0)
		if c == '-' && l.lexer.Peek(1) == '-' && l.lexer.Peek(2) == '>' {
			l.text = l.lexer.Lexeme()[4:]
			l.lexer.Move(3)
			return l.lexer.Shift()
		} else if c == 0 {
			return l.lexer.Shift()
		}
		l.lexer.Move(1)
	}
}

func (l *Parser) shiftStartTag() []byte {
	nameStart := l.lexer.Pos()
	for {
		if c := l.lexer.Peek(0); c == ' ' || c == '>' || (c == '/' || c == '?') && l.lexer.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
			break
		}
		l.lexer.Move(1)
	}
	l.text = l.lexer.Lexeme()[nameStart:]
	return l.lexer.Shift()
}

func (l *Parser) shiftAttribute() []byte {
	nameStart := l.lexer.Pos()
	var c byte
	for { // attribute name state
		if c = l.lexer.Peek(0); c == ' ' || c == '=' || c == '>' || (c == '/' || c == '?') && l.lexer.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
			break
		}
		l.lexer.Move(1)
	}
	nameEnd := l.lexer.Pos()
	for { // after attribute name state
		if c = l.lexer.Peek(0); c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			l.lexer.Move(1)
			continue
		}
		break
	}
	if c == '=' {
		l.lexer.Move(1)
		for { // before attribute value state
			if c = l.lexer.Peek(0); c == ' ' || c == '\t' || c == '\n' || c == '\r' {
				l.lexer.Move(1)
				continue
			}
			break
		}
		attrPos := l.lexer.Pos()
		delim := c
		if delim == '"' || delim == '\'' { // attribute value single- and double-quoted state
			l.lexer.Move(1)
			for {
				c = l.lexer.Peek(0)
				if c == delim {
					l.lexer.Move(1)
					break
				} else if c == 0 {
					break
				}
				l.lexer.Move(1)
				if c == '\t' || c == '\n' || c == '\r' {
					l.lexer.Lexeme()[l.lexer.Pos()-1] = ' '
				}
			}
		} else { // attribute value unquoted state
			for {
				if c = l.lexer.Peek(0); c == ' ' || c == '>' || (c == '/' || c == '?') && l.lexer.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == 0 {
					break
				}
				l.lexer.Move(1)
			}
		}
		l.attrVal = l.lexer.Lexeme()[attrPos:]
	} else {
		l.lexer.Rewind(nameEnd)
		l.attrVal = nil
	}
	l.text = l.lexer.Lexeme()[nameStart:nameEnd]
	return l.lexer.Shift()
}

func (l *Parser) shiftEndTag() []byte {
	for {
		c := l.lexer.Peek(0)
		if c == '>' {
			l.text = l.lexer.Lexeme()[2:]
			l.lexer.Move(1)
			break
		} else if c == 0 {
			l.text = l.lexer.Lexeme()[2:]
			break
		}
		l.lexer.Move(1)
	}

	end := len(l.text)
	for end > 0 {
		if c := l.text[end-1]; c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			end--
			continue
		}
		break
	}
	l.text = l.text[:end]
	return l.lexer.Shift()
}

////////////////////////////////////////////////////////////////

func (l *Parser) at(b ...byte) bool {
	for i, c := range b {
		if l.lexer.Peek(i) != c {
			return false
		}
	}
	return true
}
