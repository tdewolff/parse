package html // import "github.com/tdewolff/parse/html"

import "github.com/tdewolff/parse"

var (
	singleQuoteEntityBytes = []byte("&#39;")
	doubleQuoteEntityBytes = []byte("&#34;")
)

// EscapeAttrVal returns the escaped attribute value bytes without quotes.
func EscapeAttrVal(buf *[]byte, orig, b []byte) []byte {
	singles := 0
	doubles := 0
	unquoted := true
	entities := false
	for i, c := range b {
		if unquoted && (parse.IsWhitespace(c) || c == '`' || c == '<' || c == '=' || c == '>') {
			unquoted = false
		} else if c == '&' {
			entities = true
			if quote, n := parse.QuoteEntity(b[i:]); n > 0 {
				if quote == '"' {
					doubles++
					unquoted = false
				} else {
					singles++
					unquoted = false
				}
			}
		} else if c == '"' {
			doubles++
			unquoted = false
		} else if c == '\'' {
			singles++
			unquoted = false
		}
	}
	if unquoted {
		return b
	} else if !entities && len(orig) == len(b)+2 && (singles == 0 && orig[0] == '\'' || doubles == 0 && orig[0] == '"') {
		return orig
	}

	n := len(b) + 2
	var quote byte
	var escapedQuote []byte
	if doubles > singles {
		n += singles * 4
		quote = '\''
		escapedQuote = singleQuoteEntityBytes
	} else {
		n += doubles * 4
		quote = '"'
		escapedQuote = doubleQuoteEntityBytes
	}
	if n > cap(*buf) {
		*buf = make([]byte, 0, n) // maximum size, not actual size
	}
	t := (*buf)[:n] // maximum size, not actual size
	t[0] = quote
	j := 1
	start := 0
	for i, c := range b {
		if c == '&' {
			if entityQuote, n := parse.QuoteEntity(b[i:]); n > 0 {
				j += copy(t[j:], b[start:i])
				if entityQuote != quote {
					t[j] = entityQuote
					j++
				} else {
					j += copy(t[j:], escapedQuote)
				}
				start = i + n
			}
		} else if c == quote {
			j += copy(t[j:], b[start:i])
			j += copy(t[j:], escapedQuote)
			start = i + 1
		}
	}
	j += copy(t[j:], b[start:])
	t[j] = quote
	return t[:j+1]
}
