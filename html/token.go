package html // import "github.com/tdewolff/parse/html"

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
	r *buffer.Shifter

	rawTag  Hash
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
		if c == 0 {
			return ErrorToken, []byte{}
		} else if c != '>' && (c != '/' || z.r.Peek(1) != '>') {
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

	if z.rawTag != 0 {
		if rawText := z.shiftRawText(); len(rawText) > 0 {
			z.rawTag = 0
			return TextToken, rawText
		}
		z.rawTag = 0
	}

	for {
		c = z.r.Peek(0)
		if c == '<' {
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
		} else if c == 0 {
			if z.r.Pos() > 0 {
				return TextToken, z.r.Shift()
			} else {
				return ErrorToken, []byte{}
			}
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) AttrVal() []byte {
	return z.attrVal
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
			if c == '<' {
				if z.r.Peek(1) == '/' {
					nPos := z.r.Pos()
					z.r.Move(2)
					for {
						if c = z.r.Peek(0); !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
							break
						}
						z.r.Move(1)
					}
					if h := ToHash(parse.ToLower(parse.Copy(z.r.Bytes()[nPos+2:]))); h == z.rawTag {
						z.r.MoveTo(nPos)
						return z.r.Shift()
					}
				} else if z.rawTag == Script && z.r.Peek(1) == '!' && z.r.Peek(2) == '-' && z.r.Peek(3) == '-' {
					z.r.Move(4)
					inScript := false
					for {
						c := z.r.Peek(0)
						if c == '-' && z.r.Peek(1) == '-' && z.r.Peek(2) == '>' {
							z.r.Move(3)
							break
						} else if c == '<' {
							isEnd := z.r.Peek(1) == '/'
							if isEnd {
								z.r.Move(2)
							} else {
								z.r.Move(1)
							}
							nPos := z.r.Pos()
							for {
								if c = z.r.Peek(0); !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
									break
								}
								z.r.Move(1)
							}
							if h := ToHash(parse.ToLower(parse.Copy(z.r.Bytes()[nPos:]))); h == Script {
								if !isEnd {
									inScript = true
								} else {
									if !inScript {
										z.r.MoveTo(nPos - 2)
										return z.r.Shift()
									}
									inScript = false
								}
							}
						} else if c == 0 {
							return z.r.Shift()
						}
						z.r.Move(1)
					}
				} else {
					z.r.Move(1)
				}
			} else if c == 0 {
				return z.r.Shift()
			} else {
				z.r.Move(1)
			}
		}
	}
}

func (z *Tokenizer) readMarkup() (TokenType, []byte) {
	if z.at('-', '-') {
		z.r.Move(2)
		z.r.Skip()
		for {
			if z.r.Peek(0) == 0 {
				return CommentToken, z.r.Shift()
			} else if z.at('-', '-', '>') {
				comment := z.r.Shift()
				z.r.Move(3)
				z.r.Skip()
				return CommentToken, comment
			} else if z.at('-', '-', '!', '>') {
				comment := z.r.Shift()
				z.r.Move(4)
				z.r.Skip()
				return CommentToken, comment
			}
			z.r.Move(1)
		}
	} else if z.at('[', 'C', 'D', 'A', 'T', 'A', '[') {
		z.r.Move(7)
		for {
			if z.r.Peek(0) == 0 {
				return TextToken, z.r.Shift()
			} else if z.at(']', ']', '>') {
				z.r.Move(3)
				return TextToken, z.r.Shift()
			}
			z.r.Move(1)
		}
	} else {
		z.r.Skip()
		if z.atCaseInsensitive('d', 'o', 'c', 't', 'y', 'p', 'e') {
			z.r.Move(7)
			if z.r.Peek(0) == ' ' {
				z.r.Move(1)
			}
			z.r.Skip()
			for {
				if c := z.r.Peek(0); c == '>' || c == 0 {
					doctype := z.r.Shift()
					z.r.Move(1)
					z.r.Skip()
					return DoctypeToken, doctype
				}
				z.r.Move(1)
			}
		}
	}
	bogus := z.shiftBogusComment()
	return CommentToken, bogus
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
		if c := z.r.Peek(0); c == ' ' || c == '>' || c == '/' && z.r.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	nameEnd := z.r.Pos()
	z.moveWhitespace() // before attribute name state
	name := parse.ToLower(z.r.Shift()[:nameEnd])
	if h := ToHash(name); h == Textarea || h == Title || h == Style || h == Xmp || h == Iframe || h == Script || h == Plaintext || h == Svg || h == Math {
		z.rawTag = h
	}
	return name
}

func (z *Tokenizer) shiftAttribute() []byte {
	for { // attribute name state
		if c := z.r.Peek(0); c == ' ' || c == '=' || c == '>' || c == '/' && z.r.Peek(1) == '>' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
	nameEnd := z.r.Pos()
	z.moveWhitespace() // after attribute name state
	if z.r.Peek(0) == '=' {
		z.r.Move(1)
		z.moveWhitespace() // before attribute value state
		attrPos := z.r.Pos()
		delim := z.r.Peek(0)
		if delim == '"' || delim == '\'' { // attribute value single- and double-quoted state
			z.r.Move(1)
			for {
				c := z.r.Peek(0)
				if c == delim {
					z.r.Move(1)
					break
				} else if c == 0 {
					break
				}
				z.r.Move(1)
			}
		} else { // attribute value unquoted state
			for {
				if c := z.r.Peek(0); c == ' ' || c == '>' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == 0 {
					break
				}
				z.r.Move(1)
			}
		}
		attrEnd := z.r.Pos()
		z.moveWhitespace() // before attribute name state or after attribute quoted value state
		z.attrVal = z.r.Bytes()[attrPos:attrEnd]
	} else {
		z.attrVal = nil
	}
	return parse.ToLower(z.r.Shift()[:nameEnd])
}

func (z *Tokenizer) shiftEndTag() []byte {
	for {
		c := z.r.Peek(0)
		if c == '>' {
			name := parse.ToLower(z.r.Shift())
			z.r.Move(1)
			z.r.Skip()
			return name
		} else if c == 0 {
			return z.r.Shift()
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) moveWhitespace() {
	for {
		if c := z.r.Peek(0); c != ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f' || c == 0 {
			break
		}
		z.r.Move(1)
	}
}

func (z *Tokenizer) at(b ...byte) bool {
	for i, c := range b {
		if z.r.Peek(i) != c {
			return false
		}
	}
	return true
}

func (z *Tokenizer) atCaseInsensitive(b ...byte) bool {
	for i, c := range b {
		if z.r.Peek(i) != c && (z.r.Peek(i)+('a'-'A')) != c {
			return false
		}
	}
	return true
}
