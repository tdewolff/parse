/*
CSS3 tokenizer, written mostly along the lines of http://code.google.com/p/go.net/html
Implemented using the specifications at http://www.w3.org/TR/css3-syntax/
*/
package css

import (
	"errors"
	"io"
	"fmt"
	"strconv"
	"unicode/utf8"
)

////////////////////////////////////////////////////////////////

type TokenType uint32

const (
	ErrorToken TokenType = iota
	IdentToken
	FunctionToken
	AtKeywordToken
	HashToken
	StringToken
	UrlToken
	DelimToken
	NumberToken
	PercentageToken
	DimensionToken
	UnicodeRangeToken
	IncludeMatchToken
	DashMatchToken
	PrefixMatchToken
	SuffixMatchToken
	SubstringMatchToken
	ColumnToken
	WhitespaceToken
	CDOToken
	CDCToken
	ColonToken
	SemicolonToken
	CommaToken
	BracketToken
	CommentToken
)

var ErrBufferExceeded = errors.New("max buffer exceeded")

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
	case UrlToken:
		return "Url"
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
	case BracketToken:
		return "Bracket"
	case CommentToken:
		return "Comment"
	}
	return "Invalid(" + strconv.Itoa(int(t)) + ")"
}

type Token struct {
	Type TokenType
	Data string
}

func (t Token) String() string {
	return t.Data
}

////////////////////////////////////////////////////////////////

type Tokenizer struct {
	r    io.Reader
	line int

	buf   []byte
	start int
	end   int
	err   error

	lastRuneSize int
}

func NewTokenizer(r io.Reader) *Tokenizer {
	fmt.Print("")
	return &Tokenizer{
		r:      r,
		line:	1,
		buf:    make([]byte, 0, 1024),
	}
}

func (z *Tokenizer) Line() int {
	return z.line
}

func (z *Tokenizer) Err() error {
	return z.err
}

func (z *Tokenizer) token(t TokenType) Token {
	tt := Token{t, string(z.buf[z.start:z.end])}
	z.start = z.end
	return tt
}

func (z *Tokenizer) byteToken(t TokenType) Token {
	z.end++
	return z.token(t)
}

func (z *Tokenizer) error(s string) Token {
	z.err = errors.New(s)
	return z.token(ErrorToken)
}

func (z *Tokenizer) unreadLastRune() {
	z.end -= z.lastRuneSize
}

func (z *Tokenizer) readByte() byte {
	if z.end >= len(z.buf) {
		if z.err != nil {
			return 0
		}

		c := cap(z.buf)
		d := z.end - z.start
		var buf1 []byte
		if 2*d > c {
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = z.buf[:d]
		}
		copy(buf1, z.buf[z.start:z.end])

		// Read in to fill the buffer till capacity
		var n int
		n, z.err = z.r.Read(buf1[d:cap(buf1)])
		if n == 0 {
			return 0
		}
		z.start, z.end, z.buf = 0, d, buf1[:d+n]
	}

	x := z.buf[z.end]
	z.end++
	if z.end-z.start >= 4*4096 {
		z.err = ErrBufferExceeded
		return 0
	}
	return x
}

func (z *Tokenizer) readRune() rune {
	r := rune(z.readByte())
	z.lastRuneSize = 1
	if r == 0 && z.err != nil {
		z.lastRuneSize = 0
	} else if r == 0 {
		r = 0xFFFD
	} else if r >= 0x80 {
		cs := []byte{byte(r), z.readByte(), z.readByte(), z.readByte()}
		var n int
		r, n = utf8.DecodeRune(cs)
		z.end -= 4 - n
		z.lastRuneSize = n
	}
	return r
}

func (z *Tokenizer) readRunes(n int) []rune {
	c := make([]rune, n)
	for i := 0; i < n; i++ {
		c[i] = z.readRune()
	}
	return c
}

func (z *Tokenizer) consume(f func() bool) bool {
	end := z.end
	if !f() {
		z.end = end
		return false
	}
	return true
}

func (z *Tokenizer) comment() bool {
	cs := z.readRunes(2)
	if cs[0] != '/' || cs[1] != '*' {
		return false
	}

	afterStar := false
	for {
		switch z.readRune() {
		case '*':
			afterStar = true
		case '/':
			if afterStar {
				return true
			}
		}
		if z.err != nil {
			return true
		}
	}
}

func (z *Tokenizer) newline() bool {
	switch z.readRune() {
	case '\n', '\f':
		return true
	case '\r':
		if z.readRune() != '\n' {
			z.unreadLastRune()
		}
		return true
	default:
		return false
	}
}

func (z *Tokenizer) whitespace() bool {
	switch z.readRune() {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	default:
		return false
	}
}

func (z *Tokenizer) hexDigit() bool {
	c := z.readRune()
	if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		return true
	}
	return false
}

func (z *Tokenizer) escape() bool {
	if z.readRune() != '\\' {
		return false
	}

	if z.consume(z.hexDigit) {
		for i := 1; i < 6; i++ {
			if !z.consume(z.hexDigit) {
				break
			}
		}
		z.consume(z.whitespace)
		return true
	} else if z.consume(z.newline) {
		return false
	}
	z.readRune()
	return true
}

func (z *Tokenizer) digit() bool {
	c := z.readRune()
	if c >= '0' && c <= '9' {
		return true
	}
	return false
}

func (z *Tokenizer) whitespaceToken() bool {
	if z.consume(z.whitespace) {
		for z.consume(z.whitespace) {}
		return true
	}
	return false
}

func (z *Tokenizer) identToken() bool {
	if z.readRune() != '-' {
		z.unreadLastRune()
	}

	if !z.consume(z.escape) {
		c := z.readRune()
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c >= 0x80) {
			return false
		}
	}

	for {
		if !z.consume(z.escape) {
			c := z.readRune()
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				z.unreadLastRune()
				break
			}
		}
	}
	return true
}

func (z *Tokenizer) atKeywordToken() bool {
	c := z.readRune()
	if c != '@' {
		return false
	}

	if !z.consume(z.identToken) {
		return false
	}
	return true
}

func (z *Tokenizer) hashToken() bool {
	c := z.readRune()
	if c != '#' {
		return false
	}

	if !z.consume(z.escape) {
		c := z.readRune()
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
			return false
		}
	}

	for {
		if !z.consume(z.escape) {
			c := z.readRune()
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c >= 0x80) {
				z.unreadLastRune()
				break
			}
		}
	}
	return true
}

func (z *Tokenizer) stringToken() bool {
	delim := z.readRune()
	if delim != '"' && delim != '\'' {
		return false
	}

	for {
		if !z.consume(z.escape) {
			c := z.readRune()
			if z.err != nil {
				return true
			}
			if c == '\\' {
				if !z.consume(z.newline) {
					return false
				}
			} else if c == delim {
				break
			} else if z.consume(z.newline) {
				return false
			}
		}
	}
	return true
}

func (z *Tokenizer) numberToken() bool {
	c := z.readRune()
	if c != '+' && c != '-' {
		z.unreadLastRune()
	}

	firstDigid := z.consume(z.digit)
	if firstDigid {
		for z.consume(z.digit) {}
	}

	end := z.end
	if z.readRune() == '.' {
		if z.consume(z.digit) {
			for z.consume(z.digit) {}
		} else if firstDigid {
			// . could belong to next token
			z.end = end
			return true
		} else {
			return false
		}
	} else if !firstDigid {
		return false
	} else {
		z.unreadLastRune()
	}

	end = z.end
	c = z.readRune()
	if c == 'e' || c == 'E' {
		c = z.readRune()
		if c != '+' && c != '-' {
			z.unreadLastRune()
		}
		if !z.consume(z.digit) {
			// e could belong to dimensiontoken (em)
			z.end = end
			return true
		}
	} else {
		z.unreadLastRune()
	}
	return true
}

func (z *Tokenizer) unicodeRangeToken() bool {
	c := z.readRune()
	if c != 'u' && c != 'U' {
		return false
	}
	if z.readRune() != '+' {
		return false
	}

	if z.consume(z.hexDigit) {
		// consume up to 6 hexDigits
		i := 1
		for ; i < 6; i++ {
			if !z.consume(z.hexDigit) {
				break
			}
		}

		// either a minus or a quenstion mark or the end is expected
		if z.readRune() == '-' {
			// consume another up to 6 hexDigits
			if z.consume(z.hexDigit) {
				for i := 1; i < 6; i++ {
					if !z.consume(z.hexDigit) {
						break
					}
				}
			} else {
				return false
			}
		} else {
			// could be filled up to 6 characters with question marks
			z.unreadLastRune()
			if z.readRune() == '?' {
				for ; i < 6; i++ {
					if z.readRune() != '?' {
						return false
					}
				}
			} else {
				// or just a simple hexDigit series
				z.unreadLastRune()
			}
		}
	} else {
		// consume 6 question marks
		for i := 0; i < 6; i++ {
			if z.readRune() != '?' {
				return false
			}
		}
	}
	return true
}

func (z *Tokenizer) includeMatchToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '~' && cs[1] == '=' {
		return true
	}
	return false
}

func (z *Tokenizer) dashMatchToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '|' && cs[1] == '=' {
		return true
	}
	return false
}

func (z *Tokenizer) prefixMatchToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '^' && cs[1] == '=' {
		return true
	}
	return false
}

func (z *Tokenizer) suffixMatchToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '$' && cs[1] == '=' {
		return true
	}
	return false
}

func (z *Tokenizer) substringMatchToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '*' && cs[1] == '=' {
		return true
	}
	return false
}

func (z *Tokenizer) columnToken() bool {
	cs := z.readRunes(2)
	if cs[0] == '|' && cs[1] == '|' {
		return true
	}
	return false
}

func (z *Tokenizer) cdoToken() bool {
	cs := z.readRunes(4)
	if cs[0] == '<' && cs[1] == '!' && cs[2] == '-' && cs[3] == '-' {
		return true
	}
	return false
}

func (z *Tokenizer) cdcToken() bool {
	cs := z.readRunes(3)
	if cs[0] == '-' && cs[1] == '-' && cs[2] == '>' {
		return true
	}
	return false
}

func (z *Tokenizer) numeric() (bool, TokenType) {
	if z.consume(z.numberToken) {
		if z.readRune() == '%' {
			return true, PercentageToken
		}
		z.unreadLastRune()

		if z.consume(z.identToken) {
			return true, DimensionToken
		}
		return true, NumberToken
	}
	return false, ErrorToken
}

func (z *Tokenizer) identlike() (bool, TokenType) {
	end := z.end
	if z.consume(z.identToken) {
		c := z.readRune()
		if c != '(' {
			z.unreadLastRune()
			return true, IdentToken
		}

		if string(z.buf[z.start:z.end]) != "url(" {
			return true, FunctionToken
		}

		for z.consume(z.whitespace) {}
		if !z.consume(z.stringToken) {
			for {
				if !z.consume(z.escape) {
					c := z.readRune()
					if z.err != nil || c == '"' || c == '\'' || c == '(' || c == ')' || c == '\\' || (c >= 0 && c <= 8) || c == 0x0B || (c >= 0x0E && c <= 0x1F) || c == 0x7F {
						z.end = end
						return false, ErrorToken
					}
				}
			}
		}
		for z.consume(z.whitespace) {}
		if z.readRune() != ')' {
			z.end = end
			return false, ErrorToken
		}
		return true, UrlToken
	}
	return false, ErrorToken
}

func (z *Tokenizer) Next() Token {
	c := z.readRune()
	z.unreadLastRune()
	switch c {
	case ' ', '\t', '\n', '\r', '\f':
		z.consume(z.whitespaceToken)
		return z.token(WhitespaceToken)
	case '"':
		z.consume(z.stringToken)
		return z.token(StringToken)
	case '#':
		if z.consume(z.hashToken) {
			return z.token(HashToken)
		}
	case '$':
		if z.consume(z.suffixMatchToken) {
			return z.token(SuffixMatchToken)
		}
	case '\'':
		z.consume(z.stringToken)
		return z.token(StringToken)
	case '(', ')', '[', ']', '{', '}':
		return z.byteToken(BracketToken)
	case '*':
		if z.consume(z.substringMatchToken) {
			return z.token(SubstringMatchToken)
		}
	case '+':
		if y, t := z.numeric(); y {
			return z.token(t)
		}
	case ',':
		return z.byteToken(CommaToken)
	case '-':
		if y, t := z.numeric(); y {
			return z.token(t)
		}
		if y, t := z.identlike(); y {
			return z.token(t)
		}
		if z.consume(z.cdcToken) {
			return z.token(CDCToken)
		}
	case '.':
		if y, t := z.numeric(); y {
			return z.token(t)
		}
	case '/':
		if z.consume(z.comment) {
			return z.token(CommentToken)
		}
	case ':':
		return z.byteToken(ColonToken)
	case ';':
		return z.byteToken(SemicolonToken)
	case '<':
		if z.consume(z.cdoToken) {
			return z.token(CDOToken)
		}
	case '@':
		if z.consume(z.atKeywordToken) {
			return z.token(AtKeywordToken)
		}
	case '\\':
		if y, t := z.identlike(); y {
			return z.token(t)
		}
		z.error("bad escape")
	case '^':
		if z.consume(z.prefixMatchToken) {
			return z.token(PrefixMatchToken)
		}
	case 'u', 'U':
		if z.consume(z.unicodeRangeToken) {
			return z.token(UnicodeRangeToken)
		}
		if y, t := z.identlike(); y {
			return z.token(t)
		}
	case '|':
		if z.consume(z.dashMatchToken) {
			return z.token(DashMatchToken)
		} else if z.consume(z.columnToken) {
			return z.token(ColumnToken)
		}
	case '~':
		if z.consume(z.includeMatchToken) {
			return z.token(IncludeMatchToken)
		}
	default:
		if y, t := z.numeric(); y {
			return z.token(t)
		}
		if y, t := z.identlike(); y {
			return z.token(t)
		}
	}
	if c == 0 {
		return z.token(ErrorToken)
	}
	return z.byteToken(DelimToken)
}
