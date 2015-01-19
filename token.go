package css

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/url"
	"strconv"
)

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
	r    ShiftBuffer
	line int
}

// NewTokenizer returns a new Tokenizer for a given io.Reader.
func NewTokenizer(r io.Reader) *Tokenizer {
	return &Tokenizer{
		NewShiftBufferReader(r),
		1,
	}
}

func NewTokenizerBytes(b []byte) *Tokenizer {
	return &Tokenizer{
		NewShiftBufferBytes(b),
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

// Next returns the next Token. It returns ErrorToken when an error was encountered. Using Err() one can retrieve the error message.
func (z *Tokenizer) Next() (TokenType, []byte) {
	switch z.r.Read(0) {
	case ' ', '\t', '\n', '\r', '\f':
		if z.consumeWhitespaceToken() {
			return WhitespaceToken, z.r.Shift()
		}
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
	case '"':
		if t := z.consumeString(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '\'':
		if t := z.consumeString(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '.':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '+':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '-':
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
		if z.consumeCDCToken() {
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
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
	case '|':
		if t := z.consumeMatch(); t != ErrorToken {
			return t, z.r.Shift()
		}
		if z.consumeColumnToken() {
			return ColumnToken, z.r.Shift()
		}
	default:
		if t := z.consumeNumeric(); t != ErrorToken {
			return t, z.r.Shift()
		}
		if t := z.consumeIdentlike(); t != ErrorToken {
			return t, z.r.Shift()
		}
	}
	if z.Err() != nil {
		return ErrorToken, []byte{}
	}
	z.r.Move(1)
	return DelimToken, z.r.Shift()
}

////////////////////////////////////////////////////////////////

/*
The following functions follow the railroad diagrams in http://www.w3.org/TR/css3-syntax/
*/

func (z *Tokenizer) consumeByte(c byte) bool {
	if z.r.Read(0) == c {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeRune() bool {
	c := z.r.Read(0)
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
	if z.r.Read(0) != '/' || z.r.Read(1) != '*' {
		return false
	}
	nOld := z.r.Len()
	z.r.Move(2)
	for {
		if z.r.Read(0) == '*' && z.r.Read(1) == '/' {
			z.r.Move(2)
			return true
		}
		if z.r.Read(0) == 0 {
			break
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		z.r.MoveTo(nOld)
		return false
	}
	return true
}

func (z *Tokenizer) consumeNewline() bool {
	c := z.r.Read(0)
	if c == '\n' || c == '\f' {
		z.line++
		z.r.Move(1)
		return true
	}
	if c == '\r' {
		z.line++
		if z.r.Read(1) == '\n' {
			z.r.Move(2)
		} else {
			z.r.Move(1)
		}
		return true
	}
	return false
}

func (z *Tokenizer) consumeWhitespace() bool {
	c := z.r.Read(0)
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
		z.r.Move(1)
		return true
	}
	return false
}

func (z *Tokenizer) consumeHexDigit() bool {
	c := z.r.Read(0)
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		z.r.Move(1)
		return true
	}
	return false
}

// TODO: doesn't return replacement character when encountering EOF or when hexdigits are zero or ??? "surrogate code point".
func (z *Tokenizer) consumeEscape() bool {
	if z.r.Read(0) != '\\' {
		return false
	}
	nOld := z.r.Len()
	z.r.Move(1)
	if z.consumeNewline() {
		z.r.MoveTo(nOld)
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
	c := z.r.Read(0)
	if c >= '0' && c <= '9' {
		z.r.Move(1)
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
	nOld := z.r.Len()
	if z.r.Read(0) == '-' {
		z.r.Move(1)
	}

	if !z.consumeEscape() {
		c := z.r.Read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80) {
			z.r.MoveTo(nOld)
			return false
		}
		z.consumeRune()
	}

	for {
		c := z.r.Read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			if c == '\\' && z.consumeEscape() {
				continue
			}
			break
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeAtKeywordToken() bool {
	if z.r.Read(0) != '@' {
		return false
	}
	z.r.Move(1)
	if !z.consumeIdentToken() {
		z.r.Move(-1)
		return false
	}
	return true
}

func (z *Tokenizer) consumeHashToken() bool {
	if z.r.Read(0) != '#' {
		return false
	}
	nOld := z.r.Len()
	z.r.Move(1)
	if !z.consumeEscape() {
		c := z.r.Read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			z.r.MoveTo(nOld)
			return false
		}
		z.consumeRune()
	}
	for {
		c := z.r.Read(0)
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			if c == '\\' && z.consumeEscape() {
				continue
			}
			break
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		return false
	}
	return true
}

func (z *Tokenizer) consumeNumberToken() bool {
	nOld := z.r.Len()
	c := z.r.Read(0)
	if c == '+' || c == '-' {
		z.r.Move(1)
	}
	firstDigid := z.consumeDigit()
	if firstDigid {
		for z.consumeDigit() {
		}
	}
	if z.r.Read(0) == '.' {
		z.r.Move(1)
		if z.consumeDigit() {
			for z.consumeDigit() {
			}
		} else if firstDigid {
			// . could belong to next token
			z.r.Move(-1)
			return true
		} else {
			z.r.MoveTo(nOld)
			return false
		}
	} else if !firstDigid {
		z.r.MoveTo(nOld)
		return false
	}
	nOld = z.r.Len()
	c = z.r.Read(0)
	if c == 'e' || c == 'E' {
		z.r.Move(1)
		c = z.r.Read(0)
		if c == '+' || c == '-' {
			z.r.Move(1)
		}
		if !z.consumeDigit() {
			// e could belong to dimensiontoken (em)
			z.r.MoveTo(nOld)
			return true
		}
		for z.consumeDigit() {
		}
	}
	return true
}

func (z *Tokenizer) consumeUnicodeRangeToken() bool {
	c := z.r.Read(0)
	if (c != 'u' && c != 'U') || z.r.Read(1) != '+' {
		return false
	}
	nOld := z.r.Len()
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
	if z.r.Read(0) == '|' && z.r.Read(1) == '|' {
		z.r.Move(2)
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDOToken() bool {
	if z.r.Read(0) == '<' && z.r.Read(1) == '!' && z.r.Read(2) == '-' && z.r.Read(3) == '-' {
		z.r.Move(4)
		return true
	}
	return false
}

func (z *Tokenizer) consumeCDCToken() bool {
	if z.r.Read(0) == '-' && z.r.Read(1) == '-' && z.r.Read(2) == '>' {
		z.r.Move(3)
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////

// consumeMatch consumes any MatchToken.
func (z *Tokenizer) consumeMatch() TokenType {
	if z.r.Read(1) == '=' {
		switch z.r.Read(0) {
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
	switch z.r.Read(0) {
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
	delim := z.r.Read(0)
	if delim != '"' && delim != '\'' {
		return ErrorToken
	}
	z.r.Move(1)
	for {
		if z.consumeNewline() {
			return BadStringToken
		}
		c := z.r.Read(0)
		if c == 0 {
			break
		}
		if c == delim {
			z.r.Move(1)
			break
		}
		if c == '\\' {
			if !z.consumeEscape() {
				z.r.Move(1)
				z.consumeNewline()
			}
			continue
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		return ErrorToken
	}
	return StringToken
}

func (z *Tokenizer) consumeUnquotedURL() bool {
	for {
		if z.consumeWhitespace() {
			break
		}
		if z.consumeByte(')') {
			z.r.Move(-1)
			break
		}
		c := z.r.Read(0)
		if c == 0 {
			break
		}
		if c == '\\' && z.consumeEscape() {
			continue
		}
		if c == '"' || c == '\'' || c == '(' || c == '\\' || (c >= 0 && c <= 8) || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F {
			return false
		}
		z.consumeRune()
	}
	err := z.Err()
	if err != nil && err != io.EOF {
		return false
	}
	return true
}

// consumeRemnantsBadUrl consumes bytes of a BadUrlToken so that normal tokenization may continue.
func (z *Tokenizer) consumeRemnantsBadURL() {
	for {
		if z.consumeByte(')') || z.Err() != nil {
			break
		}
		if z.consumeEscape() {
			continue
		}
		z.consumeRune()
	}
}

// consumeIdentlike consumes IdentToken, FunctionToken or UrlToken.
func (z *Tokenizer) consumeIdentlike() TokenType {
	if z.consumeIdentToken() {
		if !z.consumeByte('(') {
			return IdentToken
		}
		if !bytes.Equal(bytes.ToLower(bytes.Replace(z.r.Bytes(), []byte("\\"), []byte{}, -1)), []byte("url(")) {
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
	z := NewTokenizerBytes(b)
	z.consumeNumberToken()
	return b[:z.r.Len()], b[z.r.Len():]
}

// SplitDataURI splits the given URLToken and returns the mediatype, data and ok.
func SplitDataURI(b []byte) ([]byte, []byte, bool) {
	if len(b) > 10 && bytes.Equal(b[:4], []byte("url(")) {
		b = b[4 : len(b)-1]
		if (b[0] == '\'' || b[0] == '"') && b[0] == b[len(b)-1] {
			b = b[1 : len(b)-1]
		}
		if bytes.Equal(b[:5], []byte("data:")) {
			b = b[5:]
			if i := bytes.IndexByte(b, ','); i != -1 {
				meta := bytes.Split(b[:i], []byte(";"))
				mime := []byte("text/plain")
				charset := []byte("charset=US-ASCII")
				data := b[i+1:]

				inBase64 := false
				if len(meta) > 0 {
					mime = meta[0]
					for _, m := range meta[1:] {
						if bytes.Equal(m, []byte("base64")) {
							inBase64 = true
						} else if len(m) > 8 && bytes.Equal(m[:8], []byte("charset=")) {
							charset = m
						}
					}
				}
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
				return append(append(append([]byte{}, mime...), ';'), charset...), data, true
			}
		}
	}
	return nil, nil, false
}

// IsIdent returns true if the bytes are a valid identifier
func IsIdent(b []byte) bool {
	z := NewTokenizerBytes(b)
	z.consumeIdentToken()
	return z.r.Len() == len(b)
}

// IsUrlUnquoted returns true if the bytes are a valid unquoted URL
func IsUrlUnquoted(b []byte) bool {
	z := NewTokenizerBytes(b)
	z.consumeUnquotedURL()
	return z.r.Len() == len(b)
}
