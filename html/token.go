package html // import "github.com/tdewolff/parse/html"

import (
	"io"
	"strconv"

	"github.com/tdewolff/parse"
)

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	CommentToken
	DoctypeToken
	StartTagToken
	StartTagCloseToken
	StartTagVoidToken
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
	case DoctypeToken:
		return "Doctype"
	case StartTagToken:
		return "StartTag"
	case StartTagCloseToken:
		return "StartTagClose"
	case StartTagVoidToken:
		return "StartTagVoid"
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
	r   *parse.ShiftBuffer

	inTag   bool
	rawTag  Hash
	attrVal []byte
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r:    parse.NewShiftBuffer(r),
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
		c = z.r.Peek(0)
		if c != '>' && c != '/' {
			return AttributeToken, z.shiftAttribute()
		}
		z.inTag = false
		if c == '/' {
			z.r.Move(2)
			return StartTagVoidToken, z.r.Shift()
		} else {
			z.r.Move(1)
			return StartTagCloseToken, z.r.Shift()
		}
	}

	if z.rawTag == 0 {
		for {
			c = z.r.Peek(0)
			if c == 0 {
				if z.r.Pos() > 0 {
					return TextToken, z.r.Shift()
				} else {
					return ErrorToken, []byte{}
				}
			} else if c == '<' {
				c = z.r.Peek(1)
				if z.r.Pos() > 0 {
					if c == '/' && z.r.Peek(2) != 0 || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '!' || c == '?' {
						return TextToken, z.r.Shift()
					}
				} else if c == '/' && z.r.Peek(2) != 0 {
					z.r.Move(2)
					z.r.Skip()
					if c = z.r.Peek(0); c != '>' && !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
						return CommentToken, z.shiftBogusComment()
					}
					return EndTagToken, z.shiftEndTag()
				} else if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
					z.r.Move(1)
					z.r.Skip()
					z.inTag = true
					return StartTagToken, z.shiftStartTag()
				} else if c == '!' {
					z.r.Move(2)
					z.r.Skip()
					return z.readMarkup()
				} else if c == '?' {
					z.r.Move(1)
					z.r.Skip()
					return CommentToken, z.shiftBogusComment()
				}
			}
			z.r.Move(1)
		}
	} else if rawText := z.shiftRawText(); len(rawText) > 0 {
		z.rawTag = 0
		return TextToken, rawText
	}
	return ErrorToken, []byte{}
}

func (z *Tokenizer) AttrVal() (val []byte, more bool) {
	return z.attrVal, z.r.Peek(0) != '>' && z.r.Peek(0) != '/'
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the specifications at http://www.w3.org/html/wg/drafts/html/master/syntax.html
*/

func (z *Tokenizer) shiftRawText() []byte {
	if z.rawTag == Plaintext {
		for {
			if z.r.Peek(0) == 0 {
				return z.r.Shift()
			}
			z.r.Move(1)
		}
	} else { // RCDATA, RAWTEXT and SCRIPT
		for {
			c := z.r.Peek(0)
			if c == 0 {
				return z.r.Shift()
			} else if c == '<' && z.r.Peek(1) == '/' {
				nPos := z.r.Pos()
				z.r.Move(2)
				for {
					if c = z.r.Peek(0); !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
						break
					}
					z.r.Move(1)
				}
				if h := ToHash(ToLower(z.r.Bytes()[nPos+2:])); h == z.rawTag {
					z.r.MoveTo(nPos)
					return z.r.Shift()
				}
			} else if z.rawTag == Script && z.at([]byte("<!--")) {
				z.r.Move(4)
				inScript := false
				for {
					if z.at([]byte("-->")) {
						z.r.Move(3)
						break
					} else if z.at([]byte("<script")) {
						z.r.Move(7)
						inScript = true
					} else if z.at([]byte("</script")) {
						if inScript {
							z.r.Move(8)
							inScript = false
						} else {
							return z.r.Shift()
						}
					}
					z.r.Move(1)
				}
			} else {
				z.r.Move(1)
			}
		}
	}
}

func (z *Tokenizer) readMarkup() (TokenType, []byte) {
	if z.at([]byte("--")) {
		z.r.Move(4)
		z.r.Skip()
		for {
			if z.r.Peek(0) == 0 {
				return CommentToken, z.r.Shift()
			} else if z.at([]byte("-->")) {
				comment := z.r.Shift()
				z.r.Move(3)
				z.r.Skip()
				return CommentToken, comment
			} else if z.at([]byte("--!>")) {
				comment := z.r.Shift()
				z.r.Move(4)
				z.r.Skip()
				return CommentToken, comment
			}
			z.r.Move(1)
		}
	} else if z.at([]byte("[CDATA[")) {
		z.r.Move(7)
		z.r.Skip()
		for {
			if z.r.Peek(0) == 0 {
				return TextToken, z.r.Shift()
			} else if z.at([]byte("]]>")) {
				text := z.r.Shift()
				z.r.Move(3)
				z.r.Skip()
				return TextToken, text
			}
			z.r.Move(1)
		}
	} else if startTag := z.shiftStartTag(); len(startTag) > 0 {
		z.inTag = true
		return DoctypeToken, startTag
	} else {
		return CommentToken, z.shiftBogusComment()
	}
}

func (z *Tokenizer) shiftBogusComment() []byte {
	for {
		if c := z.r.Peek(0); c == '>' || c == 0 {
			bogus := z.r.Shift()
			z.r.Move(1)
			z.r.Skip()
			return bogus
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
	name := ToLower(z.r.Shift())
	if h := ToHash(name); h == Textarea || h == Title || h == Style || h == Xmp || h == Iframe || h == Noembed || h == Noframes || h == Noscript || h == Script || h == Plaintext {
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
	attrName := ToLower(z.r.Shift())
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
	return attrName
}

func (z *Tokenizer) shiftEndTag() []byte {
	for {
		c := z.r.Peek(0)
		if c == 0 {
			return z.r.Shift()
		} else if c == '>' {
			endTag := z.r.Shift()
			z.r.Move(1)
			z.r.Skip()
			return endTag
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) skipWhitespace() {
	for {
		c := z.r.Peek(0)
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	z.r.Skip()
}

func (z *Tokenizer) at(b []byte) bool {
	for i, c := range b {
		if z.r.Peek(i) != c {
			return false
		}
	}
	return true
}

func ToLower(b []byte) []byte {
	for i, c := range b {
		if c < 0xC0 {
			b[i] = c | ('a' - 'A')
		}
	}
	return b
}
