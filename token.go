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
	n   int

	err error
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
	return z.err
}

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, string) {
	z.n = 0
	switch z.read(0) {
	case ' ', '\t', '\n', '\r', '\f':
		if z.consumeWhitespaceToken() {
			return WhitespaceToken, z.shiftString()
		}
	case '"':
		if t := z.consumeString(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '#':
		if z.consumeHashToken() {
			return HashToken, z.shiftString()
		}
	case '$', '*', '^', '~':
		if t := z.consumeMatch(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '\'':
		if t := z.consumeString(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '(', ')', '[', ']', '{', '}':
		if t := z.consumeBracket(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '+':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.shiftString()
		}
	case ',':
		z.n++
		return CommaToken, z.shiftString()
	case '-':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.shiftString()
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.shiftString()
		}
		if z.consumeCDCToken() {
			return CDCToken, z.shiftString()
		}
	case '.':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '/':
		if z.consumeComment() {
			return CommentToken, z.shiftString()
		}
	case ':':
		z.n++
		return ColonToken, z.shiftString()
	case ';':
		z.n++
		return SemicolonToken, z.shiftString()
	case '<':
		if z.consumeCDOToken() {
			return CDOToken, z.shiftString()
		}
	case '@':
		if z.consumeAtKeywordToken() {
			return AtKeywordToken, z.shiftString()
		}
	case '\\':
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.shiftString()
		}
		if z.err == nil {
			z.err = ErrBadEscape
		}
	case 'u', 'U':
		if z.consumeUnicodeRangeToken() {
			return UnicodeRangeToken, z.shiftString()
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.shiftString()
		}
	case '|':
		if t := z.consumeMatch(); t != ErrorToken {
			return t, z.shiftString()
		}
		if z.consumeColumnToken() {
			return ColumnToken, z.shiftString()
		}
	default:
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.shiftString()
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.shiftString()
		}
	}
	if z.err != nil && (z.read(0) == 0 || z.err != io.EOF) {
		return ErrorToken, ""
	}
	z.n++
	return DelimToken, z.shiftString()
}

////////////////////////////////////////////////////////////////

func (z *Tokenizer) read(i int) byte {
	if z.pos + z.n + i >= len(z.buf) {
		if z.err != nil {
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := z.n + i
		var buf1 []byte
		if 2*d > c {
			if 2*c > maxBuf {
				z.err = ErrBufferExceeded
				return 0
			}
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = z.buf[:d]
		}
		copy(buf1, z.buf[z.pos:z.pos+z.n+i])

		// Read in to fill the buffer till capacity
		var n int
		n, z.err = z.r.Read(buf1[d:cap(buf1)])
		z.pos, z.buf = 0, buf1[:d+n]

		if z.n+i >= d+n {
			return 0
		}
	}
	return z.buf[z.pos+z.n+i]
}

func (z *Tokenizer) buffered() []byte {
	return z.buf[z.pos:z.pos+z.n]
}

// buffered returns the text of the current token.
func (z *Tokenizer) shiftString() string {
	s := string(z.buffered()) // copy is required to ensure consequent Next() calls don't reallocate z.buf
	z.pos += z.n
	return s
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the railroad diagrams in http://www.w3.org/TR/css3-syntax/
*/

func (z *Tokenizer) consumeByte(c byte) bool {
	if z.read(0) == c {
		z.n++
		return true
	}
	return false
}

func (z *Tokenizer) consumeRune() bool {
	b := z.read(0)
	if b < 0xC0 {
		z.n += 1
	} else if b < 0xE0 {
		z.n += 2
	} else if b < 0xF0 {
		z.n += 3
	} else {
		z.n += 4
	}
	return true
}

func (z *Tokenizer) consumeComment() bool {
	if z.read(0) != '/' || z.read(1) != '*' {
		return false
	}
	nOld := z.n
	z.n += 2
	for {
		if z.read(0) == '*' && z.read(1) == '/' {
			z.n += 2
			return true
		}
		if z.err == io.EOF {
			return true
		}
		if z.err != nil {
			z.n = nOld
			return false
		}
		z.consumeRune()
	}
}

func (z *Tokenizer) consumeNewline() bool {
	switch z.read(0) {
	case '\n', '\f':
		z.n++
		return true
	case '\r':
		z.n++
		if z.read(0) == '\n' {
			z.n++
		}
		return true
	default:
		return false
	}
}

func (z *Tokenizer) consumeWhitespace() bool {
	switch z.read(0) {
	case ' ', '\t', '\n', '\r', '\f':
		z.n++
		return true
	default:
		return false
	}
}

func (z *Tokenizer) consumeHexDigit() bool {
	c := z.read(0)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		z.n++
		return true
	}
	return false
}

// TODO: doesn't return replacement character when encountering EOF or when hexdigits are zero or ??? "surrogate code point".
func (z *Tokenizer) consumeEscape() bool {
	if z.read(0) != '\\' {
		return false
	}
	nOld := z.n
	z.n++
	if z.consumeNewline() {
		z.n = nOld
		return false
	}
	if z.consumeHexDigit() {
		for k := 1; k < 6; k++ {
			if !z.consumeHexDigit() {
				break
			}
		}
		z.consumeWhitespace()
		return true
	}
	z.consumeRune()
	return true
}

func (z *Tokenizer) consumeDigit() bool {
	c := z.read(0)
	if c >= '0' && c <= '9' {
		z.n++
		return true
	}
	return false
}

func (z *Tokenizer) consumeWhitespaceToken() bool {
	if z.consumeWhitespace() {
		for z.consumeWhitespace() {
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeIdentToken() bool {
	nOld := z.n
	if z.read(0) == '-' {
		z.n++
	}
	if !z.consumeEscape() {
		c := z.read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80) {
			z.n = nOld
			return false
		}
		z.consumeRune()
	}
	for {
		if !z.consumeEscape() {
			c := z.read(0)
			if z.err != nil && z.err != io.EOF {
				return false
			}
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				break
			}
			z.consumeRune()
		}
	}
	return true
}

func (z *Tokenizer) consumeAtKeywordToken() bool {
	if z.read(0) != '@' {
		return false
	}
	z.n++
	if !z.consumeIdentToken() {
		z.n--
		return false
	}
	return true
}

func (z *Tokenizer) consumeHashToken() bool {
	if z.read(0) != '#' {
		return false
	}
	nOld := z.n
	z.n++
	if !z.consumeEscape() {
		c := z.read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			z.n = nOld
			return false
		}
		z.consumeRune()
	}
	for {
		if !z.consumeEscape() {
			c := z.read(0)
			if z.err != nil && z.err != io.EOF {
				return false
			}
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				break
			}
			z.consumeRune()
		}
	}
	return true
}

func (z *Tokenizer) consumeNumberToken() bool {
	nOld := z.n
	if z.read(0) == '+' || z.read(0) == '-' {
		z.n++
	}
	firstDigid := z.consumeDigit()
	if firstDigid {
		for z.consumeDigit() {
		}
	}
	if z.read(0) == '.' {
		z.n++
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if firstDigid {
			// . could belong to next token
			z.n--
			return true
		} else {
			z.n = nOld
			return false
		}
	} else if !firstDigid {
		z.n = nOld
		return false
	}
	nOld = z.n
	if z.read(0) == 'e' || z.read(0) == 'E' {
		z.n++
		if z.read(0) == '+' || z.read(0) == '-' {
			z.n++
		}
		if !z.consumeDigit() {
			// e could belong to dimensiontoken (em)
			z.n = nOld
			return true
		}
	}
	return true
}

func (z *Tokenizer) consumeUnicodeRangeToken() bool {
	if (z.read(0) != 'u' && z.read(0) != 'U') || z.read(1) != '+' {
		return false
	}
	nOld := z.n
	z.n += 2
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
				z.n = nOld
				return false
			}
		} else {
			// could be filled up to 6 characters with question marks or else regular hexDigits
			if z.consumeByte('?') {
				k++
				for ; k < 6; k++ {
					if !z.consumeByte('?') {
						z.n = nOld
						return false
					}
				}
			}
		}
	} else {
		// consume 6 question marks
		for k := 0; k < 6; k++ {
			if !z.consumeByte('?') {
				z.n = nOld
				return false
			}
		}
	}
	return true
}

func (z *Tokenizer) consumeColumnToken() bool {
	if z.read(0) == '|' && z.read(1) == '|' {
		z.n += 2
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDOToken() bool {
	if z.read(0) == '<' && z.read(1) == '!' && z.read(2) == '-' && z.read(3) == '-' {
		z.n += 4
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDCToken() bool {
	if z.read(0) == '-' && z.read(1) == '-' && z.read(2) == '>' {
		z.n += 3
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

// consumeMatch consumes any MatchToken.
func (z *Tokenizer) consumeMatch() TokenType {
	if z.read(1) == '=' {
		switch z.read(0) {
		case '~':
			z.n += 2
			return IncludeMatchToken
		case '|':
			z.n += 2
			return DashMatchToken
		case '^':
			z.n += 2
			return PrefixMatchToken
		case '$':
			z.n += 2
			return SuffixMatchToken
		case '*':
			z.n += 2
			return SubstringMatchToken
		}
	}
	return ErrorToken
}

// consumeBracket consumes any bracket token.
func (z *Tokenizer) consumeBracket() TokenType {
	switch z.read(0) {
	case '(':
		z.n++
		return LeftParenthesisToken
	case ')':
		z.n++
		return RightParenthesisToken
	case '[':
		z.n++
		return LeftBracketToken
	case ']':
		z.n++
		return RightBracketToken
	case '{':
		z.n++
		return LeftBraceToken
	case '}':
		z.n++
		return RightBraceToken
	}
	return ErrorToken
}

// consumeNumeric consumes NumberToken, PercentageToken or DimensionToken.
func (z *Tokenizer) consumeNumeric() TokenType {
	if z.consumeNumberToken() {
		if z.consumeByte('%') {
			return PercentageToken
		}
		if z.consumeIdentToken() {
			return DimensionToken
		}
		return NumberToken
	}
	return ErrorToken
}

// consumeString consumes a string and may return BadStringToken when a newline is encountered.
func (z *Tokenizer) consumeString() TokenType {
	delim := z.read(0)
	if delim != '"' && delim != '\'' {
		return ErrorToken
	}
	z.n++
	for {
		if !z.consumeEscape() {
			if z.consumeNewline() {
				return BadStringToken
			}

			c := z.read(0)
			if z.err == io.EOF {
				break
			}
			if z.err != nil {
				return ErrorToken
			}
			if c == delim {
				z.n++
				break
			}
			if c == '\\' {
				if !z.consumeEscape() {
					z.n++
					z.consumeNewline()
				}
			} else {
				z.consumeRune()
			}
		}
	}
	return StringToken
}

// consumeRemnantsBadUrl consumes bytes of a BadUrlToken so that normal tokenization may continue.
func (z *Tokenizer) consumeRemnantsBadURL() {
	for {
		if !z.consumeEscape() {
			if z.consumeByte(')') || z.err != nil {
				break
			}
			z.consumeRune()
		}
	}
}

// consumeIdentlike consumes IdentToken, FunctionToken or UrlToken.
func (z *Tokenizer) consumeIdentlike() TokenType {
	if z.consumeIdentToken() {
		if !z.consumeByte('(') {
			return IdentToken
		}
		if !bytes.Equal(bytes.Replace(z.buffered(), []byte("\\"), []byte{}, -1), []byte("url(")) {
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
		} else {
			for {
				if !z.consumeEscape() {
					if z.consumeWhitespace() {
						break
					}
					if z.consumeByte(')') {
						z.n--
						break
					}
					c := z.read(0)
					if z.err == io.EOF {
						break
					}
					if z.err != nil || c == '"' || c == '\'' || c == '(' || c == '\\' || (c >= 0 && c <= 8) || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F {
						z.consumeRemnantsBadURL()
						return BadURLToken
					}
					z.consumeRune()
				}
			}
		}
		for z.consumeWhitespace() {
		}
		if !z.consumeByte(')') && z.err != io.EOF {
			z.consumeRemnantsBadURL()
			return BadURLToken
		}
		return URLToken
	}
	return ErrorToken
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
