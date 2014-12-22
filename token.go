package css

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"unicode/utf8"
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

	buf   []byte
	start int
	end   int

	err     error // not-nil for immediate errors
	readErr error

	stack []state
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
func (z *Tokenizer) Next() (TokenType, []byte) {
	z.start = z.end

	z.pushState()
	r := z.readRune()
	z.end = z.stack[len(z.stack)-1].end // don't revert err
	z.popState()

	switch r {
	case ' ', '\t', '\n', '\r', '\f':
		if z.consumeWhitespaceToken() {
			return WhitespaceToken, z.buffered()
		}
	case '"':
		if y, t := z.consumeString(); y {
			return t, z.buffered()
		}
	case '#':
		if z.consumeHashToken() {
			return HashToken, z.buffered()
		}
	case '$', '*', '^', '~':
		if y, t := z.consumeMatch(); y {
			return t, z.buffered()
		}
	case '\'':
		if y, t := z.consumeString(); y {
			return t, z.buffered()
		}
	case '(', ')', '[', ']', '{', '}':
		if y, t := z.consumeBracket(); y {
			return t, z.buffered()
		}
	case '+':
		if y, t := z.consumeNumeric(); y {
			return t, z.buffered()
		}
	case ',':
		z.end++
		return CommaToken, z.buffered()
	case '-':
		if y, t := z.consumeNumeric(); y {
			return t, z.buffered()
		}
		if y, t := z.consumeIdentlike(); y {
			return t, z.buffered()
		}
		if z.consumeCDCToken() {
			return CDCToken, z.buffered()
		}
	case '.':
		if y, t := z.consumeNumeric(); y {
			return t, z.buffered()
		}
	case '/':
		if z.consumeComment() {
			return CommentToken, z.buffered()
		}
	case ':':
		z.end++
		return ColonToken, z.buffered()
	case ';':
		z.end++
		return SemicolonToken, z.buffered()
	case '<':
		if z.consumeCDOToken() {
			return CDOToken, z.buffered()
		}
	case '@':
		if z.consumeAtKeywordToken() {
			return AtKeywordToken, z.buffered()
		}
	case '\\':
		if y, t := z.consumeIdentlike(); y {
			return t, z.buffered()
		}
		if z.err == nil {
			z.err = ErrBadEscape
		}
	case 'u', 'U':
		if z.consumeUnicodeRangeToken() {
			return UnicodeRangeToken, z.buffered()
		}
		if y, t := z.consumeIdentlike(); y {
			return t, z.buffered()
		}
	case '|':
		if y, t := z.consumeMatch(); y {
			return t, z.buffered()
		}
		if z.consumeColumnToken() {
			return ColumnToken, z.buffered()
		}
	default:
		if y, t := z.consumeNumeric(); y {
			return t, z.buffered()
		}
		if y, t := z.consumeIdentlike(); y {
			return t, z.buffered()
		}
	}
	if z.err != nil {
		return ErrorToken, z.buffered()
	}
	z.end++
	return DelimToken, z.buffered()
}

////////////////////////////////////////////////////////////////

// readByte returns the next byte of data from the reader.
// It also manages the internal buffer size and expands it or reallocates it when needed.
// When an error occurs, it sets z.err and returns 0.
func (z *Tokenizer) readByte() byte {
	if z.end >= len(z.buf) {
		if z.readErr != nil {
			z.err = z.readErr
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(z.buf)
		d := z.end - z.start
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
		copy(buf1, z.buf[z.start:z.end])

		// Read in to fill the buffer till capacity
		var n int
		n, z.readErr = z.r.Read(buf1[d:cap(buf1)])
		if n == 0 {
			z.err = z.readErr
			return 0
		}

		// shift all end values on the stack
		for i := range z.stack {
			z.stack[i].end -= z.start
		}
		z.start, z.end, z.buf = 0, d, buf1[:d+n]
	}
	x := z.buf[z.end]
	z.end++
	if z.end >= maxBuf {
		z.err = ErrBufferExceeded
		return 0
	}
	return x
}

// readRune returns the next rune and may use readByte up to 4 times.
func (z *Tokenizer) readRune() rune {
	r := rune(z.readByte())
	if r == 0 {
		if z.err != nil {
			// error
			return 0
		}
		// replacement character
		return 0xFFFD
	}
	if r >= 0x80 {
		// rune of more than one byte
		cs := []byte{byte(r)}
		for i := 1; i < utf8.UTFMax; i++ {
			c := z.readByte()
			if z.err != nil {
				break
			}
			cs = append(cs, c)
		}

		var n int
		r, n = utf8.DecodeRune(cs)
		z.end -= utf8.UTFMax - n
	}
	return r
}

// backup moves the end of the buffer back and reverses EOFs.
// Nilling EOFs is allowed because we move back and will encounter the EOF at a later time.
func (z *Tokenizer) revertState() {
	z.end = z.stack[len(z.stack)-1].end
	z.err = z.stack[len(z.stack)-1].err
}

func (z *Tokenizer) pushState() {
	z.stack = append(z.stack, state{z.end, z.err})
}

func (z *Tokenizer) popState() {
	z.stack = z.stack[:len(z.stack)-1]
}

// tryReadRune reads a rune and returns true if it matches, else it backs-up.
func (z *Tokenizer) tryReadRune(r rune) bool {
	z.pushState()
	if z.readRune() == r {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
	return false
}

// buffered returns the unescaped text of the current token.
func (z *Tokenizer) buffered() []byte {
	return bytes.Replace(z.buf[z.start:z.end], []byte("\\"), []byte(""), -1)
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the railroad diagrams in http://www.w3.org/TR/css3-syntax/
*/

func (z *Tokenizer) consumeComment() bool {
	z.pushState()
	if z.readRune() != '/' || z.readRune() != '*' {
		z.revertState()
		z.popState()
		return false
	}
	z.popState()

	afterStar := false
	for {
		switch z.readRune() {
		case '*':
			afterStar = true
		case '/':
			if afterStar {
				return true
			}
		default:
			afterStar = false
		}
		if z.err != nil {
			return true
		}
	}
}

func (z *Tokenizer) consumeNewline() bool {
	z.pushState()
	switch z.readRune() {
	case '\n', '\f':
		z.popState()
		return true
	case '\r':
		z.tryReadRune('\n')
		z.popState()
		return true
	default:
		z.revertState()
		z.popState()
		return false
	}
}

func (z *Tokenizer) consumeWhitespace() bool {
	z.pushState()
	switch z.readRune() {
	case ' ', '\t', '\n', '\r', '\f':
		z.popState()
		return true
	default:
		z.revertState()
		z.popState()
		return false
	}
}

func (z *Tokenizer) consumeHexDigit() bool {
	z.pushState()
	r := z.readRune()
	if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
	return false
}

// TODO: doesn't return replacement character when encountering EOF or when hexdigits are zero or ??? "surrogate code point".
func (z *Tokenizer) consumeEscape() bool {
	z.pushState()
	if !z.tryReadRune('\\') {
		z.popState()
		return false
	}
	if z.consumeNewline() {
		z.revertState()
		z.popState()
		return false
	}
	if z.consumeHexDigit() {
		for i := 1; i < 6; i++ {
			if !z.consumeHexDigit() {
				break
			}
		}
		z.consumeWhitespace()
		z.popState()
		return true
	}
	z.readRune()
	z.popState()
	return true
}

func (z *Tokenizer) consumeDigit() bool {
	z.pushState()
	r := z.readRune()
	if r >= '0' && r <= '9' {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
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
	z.pushState()
	z.tryReadRune('-')
	if !z.consumeEscape() {
		r := z.readRune()
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r >= 0x80) {
			z.revertState()
			z.popState()
			return false
		}
	}

	for {
		if !z.consumeEscape() {
			z.pushState()
			r := z.readRune()
			if z.err != nil && z.err != io.EOF {
				z.popState()
				z.popState()
				return false
			}
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r >= 0x80) {
				z.revertState()
				z.popState()
				break
			}
			z.popState()
		}
	}
	z.popState()
	return true
}

func (z *Tokenizer) consumeAtKeywordToken() bool {
	z.pushState()
	if !z.tryReadRune('@') {
		z.popState()
		return false
	}

	if !z.consumeIdentToken() {
		z.revertState()
		z.popState()
		return false
	}
	z.popState()
	return true
}

func (z *Tokenizer) consumeHashToken() bool {
	z.pushState()
	if !z.tryReadRune('#') {
		z.popState()
		return false
	}

	if !z.consumeEscape() {
		r := z.readRune()
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r >= 0x80) {
			z.revertState()
			z.popState()
			return false
		}
	}

	for {
		if !z.consumeEscape() {
			z.pushState()
			r := z.readRune()
			if z.err != nil && z.err != io.EOF {
				z.popState()
				z.popState()
				return false
			}
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r >= 0x80) {
				z.revertState()
				z.popState()
				break
			}
			z.popState()
		}
	}
	z.popState()
	return true
}

func (z *Tokenizer) consumeNumberToken() bool {
	z.pushState()
	if !z.tryReadRune('+') {
		z.tryReadRune('-')
	}
	firstDigid := z.consumeDigit()
	if firstDigid {
		for z.consumeDigit() {
		}
	}

	z.pushState()
	if z.tryReadRune('.') {
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if firstDigid {
			// . could belong to next token
			z.revertState()
			z.popState()
			z.popState()
			return true
		} else {
			z.popState()
			z.revertState()
			z.popState()
			return false
		}
	} else if !firstDigid {
		z.popState()
		z.revertState()
		z.popState()
		return false
	}
	z.popState()

	z.pushState()
	if z.tryReadRune('e') || z.tryReadRune('E') {
		if !z.tryReadRune('+') {
			z.tryReadRune('-')
		}
		if !z.consumeDigit() {
			// e could belong to dimensiontoken (em)
			z.revertState()
			z.popState()
			z.popState()
			return true
		}
	}
	z.popState()
	z.popState()
	return true
}

func (z *Tokenizer) consumeUnicodeRangeToken() bool {
	z.pushState()
	if !z.tryReadRune('u') && !z.tryReadRune('U') {
		z.popState()
		return false
	}
	if !z.tryReadRune('+') {
		z.popState()
		return false
	}

	if z.consumeHexDigit() {
		// consume up to 6 hexDigits
		i := 1
		for ; i < 6; i++ {
			if !z.consumeHexDigit() {
				break
			}
		}

		// either a minus or a quenstion mark or the end is expected
		if z.tryReadRune('-') {
			// consume another up to 6 hexDigits
			if z.consumeHexDigit() {
				for i := 1; i < 6; i++ {
					if !z.consumeHexDigit() {
						break
					}
				}
			} else {
				z.revertState()
				z.popState()
				return false
			}
		} else {
			// could be filled up to 6 characters with question marks or else regular hexDigits
			if z.tryReadRune('?') {
				i++
				for ; i < 6; i++ {
					if z.readRune() != '?' {
						z.revertState()
						z.popState()
						return false
					}
				}
			}
		}
	} else {
		// consume 6 question marks
		for i := 0; i < 6; i++ {
			if z.readRune() != '?' {
				z.revertState()
				z.popState()
				return false
			}
		}
	}
	z.popState()
	return true
}

func (z *Tokenizer) consumeColumnToken() bool {
	z.pushState()
	if z.readRune() == '|' && z.readRune() == '|' {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
	return false
}

func (z *Tokenizer) consumeCDOToken() bool {
	z.pushState()
	if z.readRune() == '<' && z.readRune() == '!' && z.readRune() == '-' && z.readRune() == '-' {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
	return false
}

func (z *Tokenizer) consumeCDCToken() bool {
	z.pushState()
	if z.readRune() == '-' && z.readRune() == '-' && z.readRune() == '>' {
		z.popState()
		return true
	}
	z.revertState()
	z.popState()
	return false
}

////////////////////////////////////////////////////////////////

// consumeMatch consumes any MatchToken.
func (z *Tokenizer) consumeMatch() (bool, TokenType) {
	z.pushState()
	r0 := z.readRune()
	r1 := z.readRune()
	if r1 == '=' {
		switch r0 {
		case '~':
			z.popState()
			return true, IncludeMatchToken
		case '|':
			z.popState()
			return true, DashMatchToken
		case '^':
			z.popState()
			return true, PrefixMatchToken
		case '$':
			z.popState()
			return true, SuffixMatchToken
		case '*':
			z.popState()
			return true, SubstringMatchToken
		}
	}
	z.revertState()
	z.popState()
	return false, ErrorToken
}

// consumeBracket consumes any bracket token.
func (z *Tokenizer) consumeBracket() (bool, TokenType) {
	z.pushState()
	switch z.readRune() {
	case '(':
		z.popState()
		return true, LeftParenthesisToken
	case ')':
		z.popState()
		return true, RightParenthesisToken
	case '[':
		z.popState()
		return true, LeftBracketToken
	case ']':
		z.popState()
		return true, RightBracketToken
	case '{':
		z.popState()
		return true, LeftBraceToken
	case '}':
		z.popState()
		return true, RightBraceToken
	}
	z.revertState()
	z.popState()
	return false, ErrorToken
}

// consumeNumeric consumes NumberToken, PercentageToken or DimensionToken.
func (z *Tokenizer) consumeNumeric() (bool, TokenType) {
	if z.consumeNumberToken() {
		if z.tryReadRune('%') {
			return true, PercentageToken
		}
		if z.consumeIdentToken() {
			return true, DimensionToken
		}
		return true, NumberToken
	}
	return false, ErrorToken
}

// consumeString consumes a string and may return BadStringToken when a newline is encountered.
func (z *Tokenizer) consumeString() (bool, TokenType) {
	z.pushState()
	delim := z.readRune()
	if delim != '"' && delim != '\'' {
		z.revertState()
		z.popState()
		return false, ErrorToken
	}

	for {
		if !z.consumeEscape() {
			if z.consumeNewline() {
				z.popState()
				return true, BadStringToken
			}

			r := z.readRune()
			if r == delim || z.err == io.EOF {
				break
			}
			if r == '\\' {
				z.consumeNewline()
			}
			if z.err != nil {
				z.revertState()
				z.popState()
				return false, ErrorToken
			}
		}
	}
	z.popState()
	return true, StringToken
}

// consumeRemnantsBadUrl consumes bytes of a BadUrlToken so that normal tokenization may continue.
func (z *Tokenizer) consumeRemnantsBadURL() {
	for {
		if !z.consumeEscape() {
			if z.readRune() == ')' || z.err != nil {
				break
			}
		}
	}
}

// consumeIdentlike consumes IdentToken, FunctionToken or UrlToken.
func (z *Tokenizer) consumeIdentlike() (bool, TokenType) {
	if z.consumeIdentToken() {
		if !z.tryReadRune('(') {
			return true, IdentToken
		}
		if string(z.buffered()) != "url(" {
			return true, FunctionToken
		}

		// consume url
		for z.consumeWhitespace() {
		}
		if y, t := z.consumeString(); y {
			if t == BadStringToken {
				z.consumeRemnantsBadURL()
				return true, BadURLToken
			}
		} else {
			for {
				if !z.consumeEscape() {
					if z.consumeWhitespace() {
						break
					}
					z.pushState()
					r := z.readRune()
					if r == ')' || z.err == io.EOF {
						z.revertState()
						z.popState()
						break
					}
					if z.err != nil || r == '"' || r == '\'' || r == '(' || r == '\\' || (r >= 0 && r <= 8) || r == 0x0B || (r >= 0x0E && r <= 0x1F) || r == 0x7F {
						z.consumeRemnantsBadURL()
						z.popState()
						return true, BadURLToken
					}
					z.popState()
				}
			}
		}
		for z.consumeWhitespace() {
		}

		r := z.readRune()
		if r != ')' && z.err != io.EOF {
			z.consumeRemnantsBadURL()
			return true, BadURLToken
		}
		return true, URLToken
	}
	return false, ErrorToken
}

////////////////////////////////////////////////////////////////

// SplitDimensionToken splits teh data of a dimension token into the number and dimension parts
func SplitDimensionToken(s string) (string, string) {
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i+1 < len(s) && s[i] == '.' && s[i+1] >= '0' && s[i+1] <= '9' {
		i += 2
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	}
	j := i
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		if i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			return s[:i], s[i:]
		}
	}
	return s[:j], s[j:]
}
