package xml // import "github.com/tdewolff/parse/xml"

import (
	"io"
	"strconv"

	"github.com/tdewolff/buffer"
	"github.com/tdewolff/parse"
)

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	CommentToken
	CDATAToken
	StartTagToken
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
	case CDATAToken:
		return "CDATA"
	case StartTagToken:
		return "StartTag"
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

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r *buffer.Shifter

	inTag   bool
	attrVal []byte
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r: buffer.NewShifter(r),
	}
}

// Err returns the error encountered during tokenization, this is often io.EOF but also other errors can be returned.
func (z Tokenizer) Err() error {
	return z.r.Err()
}

// IsEOF returns true when it has encountered EOF and thus loaded the last buffer in memory.
func (z Tokenizer) IsEOF() bool {
	return z.r.IsEOF()
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, []byte) {
	var c byte
	if z.inTag {
		z.attrVal = nil
		c = z.r.Peek(0)
		if c != '>' && (c != '/' && c != '?' || z.r.Peek(1) != '>') {
			return AttributeToken, z.shiftAttribute()
		}
		z.inTag = false
		if c == '/' {
			z.r.Move(2)
			return StartTagCloseVoidToken, z.r.Shift()
		} else if c == '?' {
			z.r.Move(2)
			return StartTagClosePIToken, z.r.Shift()
		} else {
			z.r.Move(1)
			return StartTagCloseToken, z.r.Shift()
		}
	}

	for {
		c = z.r.Peek(0)
		if c == 0 {
			if z.r.Pos() > 0 {
				return TextToken, z.r.Shift()
			}
			return ErrorToken, []byte{}
		} else if c == '<' {
			if z.r.Pos() > 0 {
				return TextToken, z.r.Shift()
			}
			c = z.r.Peek(1)
			if c == '/' {
				z.r.Move(2)
				z.r.Skip()
				return EndTagToken, z.shiftEndTag()
			} else if c == '!' {
				z.r.Move(2)
				if z.at('-', '-') {
					z.r.Move(2)
					z.r.Skip()
					return CommentToken, z.shiftCommentText()
				} else if z.at('[', 'C', 'D', 'A', 'T', 'A', '[') {
					z.r.Move(7)
					z.r.Skip()
					return CDATAToken, z.shiftCDATAText()
				}
				z.r.Move(-2)
			}
			z.r.Move(1)
			z.r.Skip()
			z.inTag = true
			return StartTagToken, z.shiftStartTag()
		}
		z.r.Move(1)
	}
	return ErrorToken, []byte{}
}

func (z *Tokenizer) AttrVal() []byte {
	return z.attrVal
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.w3.org/html/wg/drafts/html/master/syntax.html
*/

func (z *Tokenizer) shiftCDATAText() []byte {
	for {
		if z.r.Peek(0) == 0 {
			return TextToken, z.r.Shift()
		} else if z.at(']', ']', '>') {
			z.r.Move(3)
			return TextToken, z.r.Shift()
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) shiftCommentText() []byte {
	for {
		if z.r.Peek(0) == 0 {
			return z.r.Shift()
		} else if z.at('-', '-', '>') {
			comment := z.r.Shift()
			z.r.Move(3)
			z.r.Skip()
			return comment
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) shiftStartTag() []byte {
	for {
		if c := z.r.Peek(0); c == ' ' || c == '>' || c == '/' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	name := parse.ToLower(z.r.Shift())
	if h := ToHash(name); h == Textarea || h == Title || h == Style || h == Xmp || h == Iframe || h == Script || h == Plaintext || h == Svg || h == Math {
		z.rawTag = h
	}
	z.skipWhitespace() // before attribute name state
	return name
}

func (z *Tokenizer) shiftAttribute() []byte {
	for { // attribute name state
		if c := z.r.Peek(0); c == ' ' || c == '=' || c == '>' || c == '/' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	name := parse.ToLower(z.r.Shift())
	z.skipWhitespace() // after attribute name state
	if z.r.Peek(0) == '=' {
		z.r.Move(1)
		z.skipWhitespace() // before attribute value state
		delim := z.r.Peek(0)
		if delim == '"' || delim == '\'' { // attribute value single- and double-quoted state
			z.r.Move(1)
			for {
				if z.r.Peek(0) == delim {
					break
				}
				z.r.Move(1)
			}
			z.r.Move(1)
		} else { // attribute value unquoted state
			for {
				if c := z.r.Peek(0); c == ' ' || c == '>' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
					break
				}
				z.r.Move(1)
			}
		}
		z.attrVal = z.r.Shift()
		z.skipWhitespace() // before attribute name state or after attribute quoted value state
	} else {
		z.attrVal = nil
	}
	return name
}

func (z *Tokenizer) shiftEndTag() []byte {
	for {
		c := z.r.Peek(0)
		if c == 0 {
			return z.r.Shift()
		} else if c == '>' {
			name := parse.ToLower(z.r.Shift())
			z.r.Move(1)
			z.r.Skip()
			return name
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) skipWhitespace() {
	for {
		c := z.r.Peek(0)
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	z.r.Skip()
}

func (z *Tokenizer) at(b ...byte) bool {
	for i, c := range b {
		if z.r.Peek(i) != c {
			return false
		}
	}
	return true
}
