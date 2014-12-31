package css

import (
	"bytes"
	"errors"
	"io"
	"strconv"
)

// minBuf and maxBuf are the initial and maximal internal buffer size.
var minBuf = 1024
var maxBuf = 4096

// ErrBufferExceeded is returned when the internal buffer exceeds 4096 bytes, a string or comment must thus be smaller than 4kB!
var ErrBufferExceeded = errors.New("max buffer exceeded")

// ErrBadEscape is returned when an escaped sequence contains a newline.
var ErrBadEscape = errors.New("bad escape")

////////////////////////////////////////////////////////////////

// TokenType determines the type of token, eg. a number or a semicolon.
type TokenType uint32

// TokenType values
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
func (t TokenType) String() string {
	switch t {
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
	return "Invalid(" + strconv.Itoa(int(t)) + ")"
}

////////////////////////////////////////////////////////////////

type state struct {
	end int
	err error
}

// Tokenizer is the state for the tokenizer.
type Tokenizer struct {
	r    io.Reader
	line int

	buf []byte
	pos int

	//err     error // not-nil for immediate errors
	readErr error
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		r:    r,
		line: 1,
		buf:  make([]byte, 0, minBuf),
	}
}

// Line returns the current line that is being tokenized (1 + number of \n, \r or \r\n encountered).
func (z *Tokenizer) Line() int {
	return z.line
}

// Err returns the error encountered during tokenization, this is often io.EOF but also other errors can be returned.
func (z *Tokenizer) Err() error {
	return z.readErr
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, string) {
	i := 0
	switch z.read(0) {
	case ' ', '\t', '\n', '\r', '\f':
		if z.consumeWhitespaceToken(&i) {
			return WhitespaceToken, z.popString(i)
		}
	case '"':
		if y, t := z.consumeString(&i); y {
			return t, z.popString(i)
		}
	case '#':
		if z.consumeHashToken(&i) {
			return HashToken, z.popString(i)
		}
	case '$', '*', '^', '~':
		if y, t := z.consumeMatch(&i); y {
			return t, z.popString(i)
		}
	case '\'':
		if y, t := z.consumeString(&i); y {
			return t, z.popString(i)
		}
	case '(', ')', '[', ']', '{', '}':
		if y, t := z.consumeBracket(&i); y {
			return t, z.popString(i)
		}
	case '+':
		if y, t := z.consumeNumeric(&i); y {
			return t, z.popString(i)
		}
	case ',':
		i++
		return CommaToken, z.popString(i)
	case '-':
		if y, t := z.consumeNumeric(&i); y {
			return t, z.popString(i)
		}
		if y, t := z.consumeIdentlike(&i); y {
			return t, z.popString(i)
		}
		if z.consumeCDCToken(&i) {
			return CDCToken, z.popString(i)
		}
	case '.':
		if y, t := z.consumeNumeric(&i); y {
			return t, z.popString(i)
		}
	case '/':
		if z.consumeComment(&i) {
			return CommentToken, z.popString(i)
		}
	case ':':
		i++
		return ColonToken, z.popString(i)
	case ';':
		i++
		return SemicolonToken, z.popString(i)
	case '<':
		if z.consumeCDOToken(&i) {
			return CDOToken, z.popString(i)
		}
	case '@':
		if z.consumeAtKeywordToken(&i) {
			return AtKeywordToken, z.popString(i)
		}
	case '\\':
		if y, t := z.consumeIdentlike(&i); y {
			return t, z.popString(i)
		}
		if z.readErr == nil {
			z.readErr = ErrBadEscape
		}
	case 'u', 'U':
		if z.consumeUnicodeRangeToken(&i) {
			return UnicodeRangeToken, z.popString(i)
		}
		if y, t := z.consumeIdentlike(&i); y {
			return t, z.popString(i)
		}
	case '|':
		if y, t := z.consumeMatch(&i); y {
			return t, z.popString(i)
		}
		if z.consumeColumnToken(&i) {
			return ColumnToken, z.popString(i)
		}
	default:
		if y, t := z.consumeNumeric(&i); y {
			return t, z.popString(i)
		}
		if y, t := z.consumeIdentlike(&i); y {
			return t, z.popString(i)
		}
	}
	if z.readErr != nil && (z.read(0) == 0 || z.readErr != io.EOF) {
		return ErrorToken, ""
	}
	i++
	return DelimToken, z.popString(i)
}

////////////////////////////////////////////////////////////////

func (z *Tokenizer) read(i int) byte {
	if z.pos + i >= len(z.buf) {
		if z.readErr != nil {
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := i
		var buf1 []byte
		if 2*d > c {
			if 2*c > maxBuf {
				z.readErr = ErrBufferExceeded
				return 0
			}
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = z.buf[:d]
		}
		copy(buf1, z.buf[z.pos:z.pos+i])

		// Read in to fill the buffer till capacity
		var n int
		n, z.readErr = z.r.Read(buf1[d:cap(buf1)])
		z.pos, z.buf = 0, buf1[:d+n]

		if i >= d+n {
			return 0
		}
	}
	return z.buf[z.pos+i]
}

// buffered returns the text of the current token.
func (z *Tokenizer) popString(i int) string {
	s := string(z.buffered(i)) // copy is required to ensure consequent Next() calls don't reallocate z.buf
	z.pos += i
	return s
}

func (z *Tokenizer) buffered(i int) []byte {
	return z.buf[z.pos:z.pos+i]
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the railroad diagrams in http://www.w3.org/TR/css3-syntax/
*/

func (z *Tokenizer) consumeByte(c byte, i *int) bool {
	if z.read(*i) == c {
		*i++
		return true
	}
	return false
}

func (z *Tokenizer) consumeRune(i *int) bool {
	b := z.read(*i)
	if b < 0xC0 {
		*i += 1
	} else if b < 0xE0 {
		*i += 2
	} else if b < 0xF0 {
		*i += 3
	} else {
		*i += 4
	}
	return true
}

func (z *Tokenizer) consumeComment(i *int) bool {
	if z.read(*i) != '/' || z.read(*i+1) != '*' {
		return false
	}
	ii := *i
	*i += 2
	for {
		if z.read(*i) == '*' && z.read(*i+1) == '/' {
			*i += 2
			return true
		}
		if z.readErr == io.EOF {
			return true
		}
		if z.readErr != nil {
			*i = ii
			return false
		}
		z.consumeRune(i)
	}
}

func (z *Tokenizer) consumeNewline(i *int) bool {
	switch z.read(*i) {
	case '\n', '\f':
		*i++
		return true
	case '\r':
		*i++
		if z.read(*i) == '\n' {
			*i++
		}
		return true
	default:
		return false
	}
}

func (z *Tokenizer) consumeWhitespace(i *int) bool {
	switch z.read(*i) {
	case ' ', '\t', '\n', '\r', '\f':
		*i++
		return true
	default:
		return false
	}
}

func (z *Tokenizer) consumeHexDigit(i *int) bool {
	c := z.read(*i)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		*i++
		return true
	}
	return false
}

// TODO: doesn't return replacement character when encountering EOF or when hexdigits are zero or ??? "surrogate code point".
func (z *Tokenizer) consumeEscape(i *int) bool {
	if z.read(*i) != '\\' {
		return false
	}
	ii := *i
	*i++
	if z.consumeNewline(i) {
		*i = ii
		return false
	}
	if z.consumeHexDigit(i) {
		for k := 1; k < 6; k++ {
			if !z.consumeHexDigit(i) {
				break
			}
		}
		z.consumeWhitespace(i)
		return true
	}
	z.consumeRune(i)
	return true
}

func (z *Tokenizer) consumeDigit(i *int) bool {
	c := z.read(*i)
	if c >= '0' && c <= '9' {
		*i++
		return true
	}
	return false
}

func (z *Tokenizer) consumeWhitespaceToken(i *int) bool {
	if z.consumeWhitespace(i) {
		for z.consumeWhitespace(i) {
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeIdentToken(i *int) bool {
	ii := *i
	if z.read(*i) == '-' {
		*i++
	}
	if !z.consumeEscape(i) {
		c := z.read(*i)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80) {
			*i = ii
			return false
		}
		z.consumeRune(i)
	}
	for {
		if !z.consumeEscape(i) {
			c := z.read(*i)
			if z.readErr != nil && z.readErr != io.EOF {
				return false
			}
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				break
			}
			z.consumeRune(i)
		}
	}
	return true
}

func (z *Tokenizer) consumeAtKeywordToken(i *int) bool {
	if z.read(*i) != '@' {
		return false
	}
	*i++
	if !z.consumeIdentToken(i) {
		*i--
		return false
	}
	return true
}

func (z *Tokenizer) consumeHashToken(i *int) bool {
	if z.read(*i) != '#' {
		return false
	}
	ii := *i
	*i++
	if !z.consumeEscape(i) {
		c := z.read(*i)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			*i = ii
			return false
		}
		z.consumeRune(i)
	}
	for {
		if !z.consumeEscape(i) {
			c := z.read(*i)
			if z.readErr != nil && z.readErr != io.EOF {
				return false
			}
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				break
			}
			z.consumeRune(i)
		}
	}
	return true
}

func (z *Tokenizer) consumeNumberToken(i *int) bool {
	ii := *i
	if z.read(*i) == '+' || z.read(*i) == '-' {
		*i++
	}
	firstDigid := z.consumeDigit(i)
	if firstDigid {
		for z.consumeDigit(i) {
		}
	}
	if z.read(*i) == '.' {
		*i++
		if z.consumeDigit(i) {
			for z.consumeDigit(i) {
			}
		} else if firstDigid {
			// . could belong to next token
			*i--
			return true
		} else {
			*i = ii
			return false
		}
	} else if !firstDigid {
		*i = ii
		return false
	}
	ii = *i
	if z.read(*i) == 'e' || z.read(*i) == 'E' {
		if z.read(*i) == '+' || z.read(*i) == '-' {
			*i++
		}
		if !z.consumeDigit(i) {
			// e could belong to dimensiontoken (em)
			*i = ii
			return true
		}
	}
	return true
}

func (z *Tokenizer) consumeUnicodeRangeToken(i *int) bool {
	if (z.read(*i) != 'u' && z.read(*i) != 'U') || z.read(*i+1) != '+' {
		return false
	}
	ii := *i
	*i += 2
	if z.consumeHexDigit(i) {
		// consume up to 6 hexDigits
		k := 1
		for ; k < 6; k++ {
			if !z.consumeHexDigit(i) {
				break
			}
		}

		// either a minus or a quenstion mark or the end is expected
		if z.consumeByte('-', i) {
			// consume another up to 6 hexDigits
			if z.consumeHexDigit(i) {
				for k := 1; k < 6; k++ {
					if !z.consumeHexDigit(i) {
						break
					}
				}
			} else {
				*i = ii
				return false
			}
		} else {
			// could be filled up to 6 characters with question marks or else regular hexDigits
			if z.consumeByte('?', i) {
				k++
				for ; k < 6; k++ {
					if !z.consumeByte('?', i) {
						*i = ii
						return false
					}
				}
			}
		}
	} else {
		// consume 6 question marks
		for k := 0; k < 6; k++ {
			if !z.consumeByte('?', i) {
				*i = ii
				return false
			}
		}
	}
	return true
}

func (z *Tokenizer) consumeColumnToken(i *int) bool {
	if z.read(*i) == '|' && z.read(*i+1) == '|' {
		*i += 2
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDOToken(i *int) bool {
	if z.read(*i) == '<' && z.read(*i+1) == '!' && z.read(*i+2) == '-' && z.read(*i+3) == '-' {
		*i += 4
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDCToken(i *int) bool {
	if z.read(*i) == '-' && z.read(*i+1) == '-' && z.read(*i+2) == '>' {
		*i += 3
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

// consumeMatch consumes any MatchToken.
func (z *Tokenizer) consumeMatch(i *int) (bool, TokenType) {
	if z.read(*i+1) == '=' {
		switch z.read(*i) {
		case '~':
			*i += 2
			return true, IncludeMatchToken
		case '|':
			*i += 2
			return true, DashMatchToken
		case '^':
			*i += 2
			return true, PrefixMatchToken
		case '$':
			*i += 2
			return true, SuffixMatchToken
		case '*':
			*i += 2
			return true, SubstringMatchToken
		}
	}
	return false, ErrorToken
}

// consumeBracket consumes any bracket token.
func (z *Tokenizer) consumeBracket(i *int) (bool, TokenType) {
	switch z.read(*i) {
	case '(':
		*i += 1
		return true, LeftParenthesisToken
	case ')':
		*i += 1
		return true, RightParenthesisToken
	case '[':
		*i += 1
		return true, LeftBracketToken
	case ']':
		*i += 1
		return true, RightBracketToken
	case '{':
		*i += 1
		return true, LeftBraceToken
	case '}':
		*i += 1
		return true, RightBraceToken
	}
	return false, ErrorToken
}

// consumeNumeric consumes NumberToken, PercentageToken or DimensionToken.
func (z *Tokenizer) consumeNumeric(i *int) (bool, TokenType) {
	if z.consumeNumberToken(i) {
		if z.consumeByte('%', i) {
			return true, PercentageToken
		}
		if z.consumeIdentToken(i) {
			return true, DimensionToken
		}
		return true, NumberToken
	}
	return false, ErrorToken
}

// consumeString consumes a string and may return BadStringToken when a newline is encountered.
func (z *Tokenizer) consumeString(i *int) (bool, TokenType) {
	delim := z.read(*i)
	if delim != '"' && delim != '\'' {
		return false, ErrorToken
	}
	*i++
	for {
		if !z.consumeEscape(i) {
			if z.consumeNewline(i) {
				return true, BadStringToken
			}

			c := z.read(*i)
			if z.readErr == io.EOF {
				break
			}
			if z.readErr != nil {
				return false, ErrorToken
			}
			if c == delim {
				*i++
				break
			}
			if c == '\\' {
				if !z.consumeEscape(i) {
					*i++
					z.consumeNewline(i)
				}
			} else {
				z.consumeRune(i)
			}
		}
	}
	return true, StringToken
}

// consumeRemnantsBadUrl consumes bytes of a BadUrlToken so that normal tokenization may continue.
func (z *Tokenizer) consumeRemnantsBadURL(i *int) {
	for {
		if !z.consumeEscape(i) {
			if z.consumeByte(')', i) || z.readErr != nil {
				break
			}
			z.consumeRune(i)
		}
	}
}

// consumeIdentlike consumes IdentToken, FunctionToken or UrlToken.
func (z *Tokenizer) consumeIdentlike(i *int) (bool, TokenType) {
	if z.consumeIdentToken(i) {
		if !z.consumeByte('(', i) {
			return true, IdentToken
		}
		if !bytes.Equal(bytes.Replace(z.buffered(*i), []byte("\\"), []byte{}, -1), []byte("url(")) {
			return true, FunctionToken
		}

		// consume url
		for z.consumeWhitespace(i) {
		}
		if y, t := z.consumeString(i); y {
			if t == BadStringToken {
				z.consumeRemnantsBadURL(i)
				return true, BadURLToken
			}
		} else {
			for {
				if !z.consumeEscape(i) {
					if z.consumeWhitespace(i) {
						break
					}
					if z.consumeByte(')', i) {
						*i--
						break
					}
					c := z.read(*i)
					if z.readErr == io.EOF {
						break
					}
					if z.readErr != nil || c == '"' || c == '\'' || c == '(' || c == '\\' || (c >= 0 && c <= 8) || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F {
						z.consumeRemnantsBadURL(i)
						return true, BadURLToken
					}
					z.consumeRune(i)
				}
			}
		}
		for z.consumeWhitespace(i) {
		}
		if !z.consumeByte(')', i) && z.readErr != io.EOF {
			z.consumeRemnantsBadURL(i)
			return true, BadURLToken
		}
		return true, URLToken
	}
	return false, ErrorToken
}

////////////////////////////////////////////////////////////////

// SplitDimensionToken splits teh data of a dimension token into the number and dimension parts
// func SplitDimensionToken(s string) (string, string) {
// 	i := 0
// 	if i < len(s) && (s[i] == '+' || s[i] == '-') {
// 		i++
// 	}
// 	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
// 		i++
// 	}
// 	if i+1 < len(s) && s[i] == '.' && (s[i+1] >= '0' && s[i+1] <= '9') {
// 		i += 2
// 		for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
// 			i++
// 		}
// 	}
// 	j := i
// 	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
// 		i++
// 		if i < len(s) && (s[i] == '+' || s[i] == '-') {
// 			i++
// 		}
// 		if i < len(s) && (s[i] >= '0' && s[i] <= '9') {
// 			i++
// 			for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
// 				i++
// 			}
// 			return s[:i], s[i:]
// 		}
// 	}
// 	return s[:j], s[j:]
// }

// // lenEscape returns the length of an escape sequence
// func lenEscape(s string) int {
// 	i := 0
// 	if i < len(s) && s[i] == '\\' {
// 		i++
// 		if i < len(s) && ((s[i] >= 'a' && s[i] <= 'f') || (s[i] >= 'A' && s[i] <= 'F') || (s[i] >= '0' && s[i] <= '9')) {
// 			i++
// 			j := 1
// 			for i < len(s) && j < 6 && ((s[i] >= 'a' && s[i] <= 'f') || (s[i] >= 'A' && s[i] <= 'F') || (s[i] >= '0' && s[i] <= '9')) {
// 				i++
// 				j++
// 			}
// 			if i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\f') {
// 				i++
// 			}
// 		} else if i < len(s) && !(s[i] == '\n' || s[i] == '\r' || s[i] == '\f') {
// 			i++
// 		}
// 	}
// 	return i
// }

// // IsIdent returns true if string is a valid sequence for an identifier
// func IsIdent(s string) bool {
// 	i := 0
// 	if i < len(s) && s[i] == '-' {
// 		i++
// 	}
// 	if i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || s[i] == '_' || s[i] >= 0x80 || s[i] == '\\') {
// 		if s[i] == '\\' {
// 			if n := lenEscape(s[i:]); n > 0 {
// 				i += n
// 			} else {
// 				return false
// 			}
// 		} else {
// 			i++
// 		}
// 	}
// 	for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_' || s[i] == '-' || s[i] >= 0x80 || s[i] == '\\') {
// 		if s[i] == '\\' {
// 			if n := lenEscape(s[i:]); n > 0 {
// 				i += n
// 			} else {
// 				return false
// 			}
// 		} else {
// 			i++
// 		}
// 	}
// 	return i == len(s)
// }
// func IsUrlUnquoted(s string) bool {
// 	i := 0
// 	for i < len(s) && s[i] != '"' && s[i] != '\'' && s[i] != '(' && s[i] != ')' && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\r' && s[i] != '\f' && s[i] != 0x00 && s[i] != 0x08 && s[i] != 0x0B && (s[i] < 0x0E || s[i] > 0x1F) && s[i] != 0x7F {
// 		if s[i] == '\\' {
// 			if n := lenEscape(s[i:]); n > 0 {
// 				i += n
// 			} else {
// 				return false
// 			}
// 		} else {
// 			i++
// 		}
// 	}
// 	return i == len(s)
// }
