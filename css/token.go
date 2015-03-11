package css // import "github.com/tdewolff/parse/css"

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/url"
	"strconv"

	"github.com/tdewolff/parse"
)

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values.
const (
	ErrorToken TokenType = iota // extra token when errors occur
	IdentToken
	FunctionToken  // rgb( rgba( ...
	AtKeywordToken // @abc
	HashToken      // #abc
	StringToken
	BadStringToken
	URLToken
	BadURLToken
	DelimToken            // any unmatched character
	NumberToken           // 5
	PercentageToken       // 5%
	DimensionToken        // 5em
	UnicodeRangeToken     // U+554A
	IncludeMatchToken     // ~=
	DashMatchToken        // |=
	PrefixMatchToken      // ^=
	SuffixMatchToken      // $=
	SubstringMatchToken   // *=
	ColumnToken           // ||
	WhitespaceToken       // space \t \r \n \f
	CDOToken              // <!--
	CDCToken              // -->
	ColonToken            // :
	SemicolonToken        // ;
	CommaToken            // ,
	LeftBracketToken      // [
	RightBracketToken     // ]
	LeftParenthesisToken  // (
	RightParenthesisToken // )
	LeftBraceToken        // {
	RightBraceToken       // }
	CommentToken          // extra token for comments
)

// String returns the string representation of a TokenType.
func (tt TokenType) String() string {
	switch tt {
	case ErrorToken:
		return "Error"
	case IdentToken:
		return "Ident"
	case FunctionToken:
		return "Function"
	case AtKeywordToken:
		return "AtKeyword"
	case HashToken:
		return "Hash"
	case StringToken:
		return "String"
	case BadStringToken:
		return "BadString"
	case URLToken:
		return "URL"
	case BadURLToken:
		return "BadURL"
	case DelimToken:
		return "Delim"
	case NumberToken:
		return "Number"
	case PercentageToken:
		return "Percentage"
	case DimensionToken:
		return "Dimension"
	case UnicodeRangeToken:
		return "UnicodeRange"
	case IncludeMatchToken:
		return "IncludeMatch"
	case DashMatchToken:
		return "DashMatch"
	case PrefixMatchToken:
		return "PrefixMatch"
	case SuffixMatchToken:
		return "SuffixMatch"
	case SubstringMatchToken:
		return "SubstringMatch"
	case ColumnToken:
		return "Column"
	case WhitespaceToken:
		return "Whitespace"
	case CDOToken:
		return "CDO"
	case CDCToken:
		return "CDC"
	case ColonToken:
		return "Colon"
	case SemicolonToken:
		return "Semicolon"
	case CommaToken:
		return "Comma"
	case LeftBracketToken:
		return "LeftBracket"
	case RightBracketToken:
		return "RightBracket"
	case LeftParenthesisToken:
		return "LeftParenthesis"
	case RightParenthesisToken:
		return "RightParenthesis"
	case LeftBraceToken:
		return "LeftBrace"
	case RightBraceToken:
		return "RightBrace"
	case CommentToken:
		return "Comment"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

////////////////////////////////////////////////////////////////

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r    *parse.ShiftBuffer
	line int
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		parse.NewShiftBuffer(r),
		1,
	}
}

// Line returns the current line that is being tokenized (1 + number of \n, \r or \r\n encountered).
func (z Tokenizer) Line() int {
	return z.line
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
	switch z.r.Peek(0) {
	case ' ', '\t', '\n', '\r', '\f':
		z.r.Move(1)
		for z.consumeWhitespace() {
		}
		return WhitespaceToken, z.r.Shift()
	case ':':
		z.r.Move(1)
		return ColonToken, z.r.Shift()
	case ';':
		z.r.Move(1)
		return SemicolonToken, z.r.Shift()
	case ',':
		z.r.Move(1)
		return CommaToken, z.r.Shift()
	case '(', ')', '[', ']', '{', '}':
		if t := z.consumeBracket(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '#':
		if z.consumeHashToken() {
			return HashToken, z.r.Shift()
		}
	case '"', '\'':
		if t := z.consumeString(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '.', '+':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '-':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		} else if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		} else if z.consumeCDCToken() {
			return CDCToken, z.r.Shift()
		}
	case '@':
		if z.consumeAtKeywordToken() {
			return AtKeywordToken, z.r.Shift()
		}
	case '$', '*', '^', '~':
		if t := z.consumeMatch(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '/':
		if z.consumeComment() {
			return CommentToken, z.r.Shift()
		}
	case '<':
		if z.consumeCDOToken() {
			return CDOToken, z.r.Shift()
		}
	case '\\':
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case 'u', 'U':
		if z.consumeUnicodeRangeToken() {
			return UnicodeRangeToken, z.r.Shift()
		} else if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '|':
		if t := z.consumeMatch(); t != ErrorToken {
			return t, z.r.Shift()
		} else if z.consumeColumnToken() {
			return ColumnToken, z.r.Shift()
		}
	default:
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		} else if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
	}
	if z.Err() != nil {
		return ErrorToken, []byte{}
	}
	// can't be rune for consumeIdentlike consumes that as an identifier
	z.r.Move(1)
	return DelimToken, z.r.Shift()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the railroad diagrams in http://www.w3.org/TR/css3-syntax/
*/

func (z *Tokenizer) consumeByte(c byte) bool {
	if z.r.Peek(0) == c {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeRune() bool {
	c := z.r.Peek(0)
	if c < 0xC0 {
		z.r.Move(1)
	} else if c < 0xE0 {
		z.r.Move(2)
	} else if c < 0xF0 {
		z.r.Move(3)
	} else {
		z.r.Move(4)
	}
	return true
}

func (z *Tokenizer) consumeComment() bool {
	if z.r.Peek(0) != '/' || z.r.Peek(1) != '*' {
		return false
	}
	z.r.Move(2)
	for {
		c := z.r.Peek(0)
		if c == 0 {
			break
		} else if c == '*' && z.r.Peek(1) == '/' {
			z.r.Move(2)
			return true
		}
		z.r.Move(1)
	}
	return true
}

func (z *Tokenizer) consumeNewline() bool {
	c := z.r.Peek(0)
	if c == '\n' || c == '\f' {
		z.line++
		z.r.Move(1)
		return true
	} else if c == '\r' {
		z.line++
		if z.r.Peek(1) == '\n' {
			z.r.Move(2)
		} else {
			z.r.Move(1)
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeWhitespace() bool {
	c := z.r.Peek(0)
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeDigit() bool {
	c := z.r.Peek(0)
	if c >= '0' && c <= '9' {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeHexDigit() bool {
	c := z.r.Peek(0)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		z.r.Move(1)
		return true
	}
	return false
}

// TODO: doesn't return replacement character when encountering EOF or when hexdigits are zero or ??? "surrogate code point".
func (z *Tokenizer) consumeEscape() bool {
	if z.r.Peek(0) != '\\' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(1)
	if z.consumeNewline() {
		z.r.MoveTo(nOld)
		return false
	} else if z.consumeHexDigit() {
		for k := 1; k < 6; k++ {
			if !z.consumeHexDigit() {
				break
			}
		}
		z.consumeWhitespace()
		return true
	} else if z.r.Peek(0) >= 0xC0 {
		return z.consumeRune()
	}
	z.r.Move(1)
	return true
}

func (z *Tokenizer) consumeIdentToken() bool {
	nOld := z.r.Pos()
	if z.r.Peek(0) == '-' {
		z.r.Move(1)
	}
	c := z.r.Peek(0)
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80) {
		if c != '\\' || !z.consumeEscape() {
			z.r.MoveTo(nOld)
			return false
		}
	} else {
		z.r.Move(1)
	}
	for {
		c := z.r.Peek(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			if c != '\\' || !z.consumeEscape() {
				break
			}
		} else {
			z.r.Move(1)
		}
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeAtKeywordToken() bool {
	// if z.r.Peek(0) != '@' {
	// 	return false
	// }
	z.r.Move(1)
	if !z.consumeIdentToken() {
		z.r.Move(-1)
		return false
	}
	return true
}

func (z *Tokenizer) consumeHashToken() bool {
	// if z.r.Peek(0) != '#' {
	// 	return false
	// }
	nOld := z.r.Pos()
	z.r.Move(1)
	c := z.r.Peek(0)
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
		if c != '\\' || !z.consumeEscape() {
			z.r.MoveTo(nOld)
			return false
		}
	} else {
		z.r.Move(1)
	}
	for {
		c := z.r.Peek(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			if c != '\\' || !z.consumeEscape() {
				break
			}
		} else {
			z.r.Move(1)
		}
	}
	return true
}

func (z *Tokenizer) consumeNumberToken() bool {
	nOld := z.r.Pos()
	c := z.r.Peek(0)
	if c == '+' || c == '-' {
		z.r.Move(1)
	}
	firstDigit := z.consumeDigit()
	if firstDigit {
		for z.consumeDigit() {
		}
	}
	if z.r.Peek(0) == '.' {
		z.r.Move(1)
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if firstDigit {
			// . could belong to the next token
			z.r.Move(-1)
			return true
		} else {
			z.r.MoveTo(nOld)
			return false
		}
	} else if !firstDigit {
		z.r.MoveTo(nOld)
		return false
	}
	nOld = z.r.Pos()
	c = z.r.Peek(0)
	if c == 'e' || c == 'E' {
		z.r.Move(1)
		c = z.r.Peek(0)
		if c == '+' || c == '-' {
			z.r.Move(1)
		}
		if !z.consumeDigit() {
			// e could belong to next token
			z.r.MoveTo(nOld)
			return true
		}
		for z.consumeDigit() {
		}
	}
	return true
}

func (z *Tokenizer) consumeUnicodeRangeToken() bool {
	c := z.r.Peek(0)
	if (c != 'u' && c != 'U') || z.r.Peek(1) != '+' {
		return false
	}
	nOld := z.r.Pos()
	z.r.Move(2)
	if z.consumeHexDigit() {
		// consume up to 6 hexDigits
		k := 1
		for ; k < 6; k++ {
			if !z.consumeHexDigit() {
				break
			}
		}

		// either a minus or a quenstion mark or the end is expected
		if z.consumeByte('-') {
			// consume another up to 6 hexDigits
			if z.consumeHexDigit() {
				for k := 1; k < 6; k++ {
					if !z.consumeHexDigit() {
						break
					}
				}
			} else {
				z.r.MoveTo(nOld)
				return false
			}
		} else {
			// could be filled up to 6 characters with question marks or else regular hexDigits
			if z.consumeByte('?') {
				k++
				for ; k < 6; k++ {
					if !z.consumeByte('?') {
						z.r.MoveTo(nOld)
						return false
					}
				}
			}
		}
	} else {
		// consume 6 question marks
		for k := 0; k < 6; k++ {
			if !z.consumeByte('?') {
				z.r.MoveTo(nOld)
				return false
			}
		}
	}
	return true
}

func (z *Tokenizer) consumeColumnToken() bool {
	if z.r.Peek(0) == '|' && z.r.Peek(1) == '|' {
		z.r.Move(2)
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDOToken() bool {
	if z.r.Peek(0) == '<' && z.r.Peek(1) == '!' && z.r.Peek(2) == '-' && z.r.Peek(3) == '-' {
		z.r.Move(4)
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDCToken() bool {
	if z.r.Peek(0) == '-' && z.r.Peek(1) == '-' && z.r.Peek(2) == '>' {
		z.r.Move(3)
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

// consumeMatch consumes any MatchToken.
func (z *Tokenizer) consumeMatch() TokenType {
	if z.r.Peek(1) == '=' {
		switch z.r.Peek(0) {
		case '~':
			z.r.Move(2)
			return IncludeMatchToken
		case '|':
			z.r.Move(2)
			return DashMatchToken
		case '^':
			z.r.Move(2)
			return PrefixMatchToken
		case '$':
			z.r.Move(2)
			return SuffixMatchToken
		case '*':
			z.r.Move(2)
			return SubstringMatchToken
		}
	}
	return ErrorToken
}

// consumeBracket consumes any bracket token.
func (z *Tokenizer) consumeBracket() TokenType {
	switch z.r.Peek(0) {
	case '(':
		z.r.Move(1)
		return LeftParenthesisToken
	case ')':
		z.r.Move(1)
		return RightParenthesisToken
	case '[':
		z.r.Move(1)
		return LeftBracketToken
	case ']':
		z.r.Move(1)
		return RightBracketToken
	case '{':
		z.r.Move(1)
		return LeftBraceToken
	case '}':
		z.r.Move(1)
		return RightBraceToken
	}
	return ErrorToken
}

// consumeNumeric consumes NumberToken, PercentageToken or DimensionToken.
func (z *Tokenizer) consumeNumeric() TokenType {
	if z.consumeNumberToken() {
		if z.consumeByte('%') {
			return PercentageToken
		} else if z.consumeIdentToken() {
			return DimensionToken
		}
		return NumberToken
	}
	return ErrorToken
}

// consumeString consumes a string and may return BadStringToken when a newline is encountered.
func (z *Tokenizer) consumeString() TokenType {
	delim := z.r.Peek(0)
	if delim != '"' && delim != '\'' {
		return ErrorToken
	}
	z.r.Move(1)
	for {
		c := z.r.Peek(0)
		if c == 0 {
			break
		} else if c == '\n' || c == '\r' || c == '\f' {
			return BadStringToken
		} else if c == delim {
			z.r.Move(1)
			break
		} else if c == '\\' {
			if !z.consumeEscape() {
				z.r.Move(1)
				z.consumeNewline()
			}
		} else {
			z.r.Move(1)
		}
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return ErrorToken
	}
	return StringToken
}

func (z *Tokenizer) consumeUnquotedURL() bool {
	for {
		if z.consumeWhitespace() {
			break
		} else if z.consumeByte(')') {
			z.r.Move(-1)
			break
		}
		c := z.r.Peek(0)
		if c == 0 {
			break
		} else if c == '"' || c == '\'' || c == '(' || (c >= 0 && c <= 8) || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F || c == '\\' {
			if c != '\\' || !z.consumeEscape() {
				return false
			}
		} else {
			z.r.Move(1)
		}
	}
	if err := z.Err(); err != nil && err != io.EOF {
		return false
	}
	return true
}

// consumeRemnantsBadUrl consumes bytes of a BadUrlToken so that normal tokenization may continue.
func (z *Tokenizer) consumeRemnantsBadURL() {
	for {
		if z.consumeByte(')') || z.Err() != nil {
			break
		} else if !z.consumeEscape() {
			z.r.Move(1)
		}
	}
}

// consumeIdentlike consumes IdentToken, FunctionToken or UrlToken.
func (z *Tokenizer) consumeIdentlike() TokenType {
	if z.consumeIdentToken() {
		if !z.consumeByte('(') {
			return IdentToken
		} else if !bytes.Equal(bytes.ToLower(bytes.Replace(z.r.Bytes(), []byte("\\"), []byte{}, -1)), []byte("url(")) {
			return FunctionToken
		}

		// consume url
		for z.consumeWhitespace() {
		}
		if t := z.consumeString(); t != ErrorToken {
			if t == BadStringToken {
				z.consumeRemnantsBadURL()
				return BadURLToken
			}
		} else if !z.consumeUnquotedURL() {
			z.consumeRemnantsBadURL()
			return BadURLToken
		}
		for z.consumeWhitespace() {
		}
		if !z.consumeByte(')') && z.Err() != io.EOF {
			z.consumeRemnantsBadURL()
			return BadURLToken
		}
		return URLToken
	}
	return ErrorToken
}

////////////////////////////////////////////////////////////////

// SplitNumberToken splits the data of a dimension token into the number and dimension parts.
func SplitNumberToken(b []byte) ([]byte, []byte) {
	i := len(b) - 1
	for ; i >= 0; i-- {
		if b[i] >= '0' && b[i] <= '9' {
			break
		}
	}
	return b[:i+1], b[i+1:]
}

// SplitDataURI splits the given URLToken and returns the mediatype, data and ok.
func SplitDataURI(uri []byte) ([]byte, []byte, bool) {
	if len(uri) > 10 && bytes.Equal(uri[:4], []byte("url(")) {
		uri = uri[4 : len(uri)-1]
		if (uri[0] == '\'' || uri[0] == '"') && uri[0] == uri[len(uri)-1] {
			uri = uri[1 : len(uri)-1]
		}
		if bytes.Equal(uri[:5], []byte("data:")) {
			uri = uri[5:]
			inBase64 := false
			mediatype := []byte{}
			i := 0
			for j, c := range uri {
				if c == '=' || c == ';' || c == ',' {
					if c != '=' && bytes.Equal(bytes.TrimSpace(uri[i:j]), []byte("base64")) {
						if len(mediatype) > 0 {
							mediatype = mediatype[:len(mediatype)-1]
						}
						inBase64 = true
						i = j
					} else if c != ',' {
						mediatype = append(append(mediatype, bytes.TrimSpace(uri[i:j])...), c)
						i = j + 1
					} else {
						mediatype = append(mediatype, bytes.TrimSpace(uri[i:j])...)
					}
					if c == ',' {
						if len(mediatype) == 0 || mediatype[0] == ';' {
							mediatype = []byte("text/plain")
						}
						data := uri[j+1:]
						if inBase64 {
							decoded := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
							n, err := base64.StdEncoding.Decode(decoded, data)
							if err != nil {
								return nil, nil, false
							}
							data = decoded[:n]
						} else {
							unescaped, err := url.QueryUnescape(string(data))
							if err != nil {
								return nil, nil, false
							}
							data = []byte(unescaped)
						}
						return mediatype, data, true
					}
				}
			}
		}
	}
	return nil, nil, false
}

// IsIdent returns true if the bytes are a valid identifier.
func IsIdent(b []byte) bool {
	z := NewTokenizer(bytes.NewBuffer(b))
	z.consumeIdentToken()
	return z.r.Pos() == len(b)
}

// IsUrlUnquoted returns true if the bytes are a valid unquoted URL.
func IsUrlUnquoted(b []byte) bool {
	z := NewTokenizer(bytes.NewBuffer(b))
	z.consumeUnquotedURL()
	return z.r.Pos() == len(b)
}
