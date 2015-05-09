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
		if c == '&' {
			entities = true
			if quote, _, ok := parse.QuoteEntity(b[i:]); ok {
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
		} else if unquoted && (c == '`' || c == '<' || c == '=' || c == '>' || parse.IsWhitespace(c)) {
			unquoted = false
		}
	}
	if unquoted {
		return b
	} else if !entities && len(orig) == len(b)+2 && (singles == 0 && orig[0] == '\'' || doubles == 0 && orig[0] == '"') {
		return orig
	}

	var quote byte
	var escapedQuote []byte
	if doubles > singles {
		quote = '\''
		escapedQuote = singleQuoteEntityBytes
	} else {
		quote = '"'
		escapedQuote = doubleQuoteEntityBytes
	}
	if len(b)+2 > cap(*buf) {
		*buf = make([]byte, 0, len(b)+2) // maximum size, not actual size
	}
	t := (*buf)[:len(b)+2] // maximum size, not actual size
	t[0] = quote
	j := 1
	start := 0
	for i, c := range b {
		if c == '&' {
			if entityQuote, n, ok := parse.QuoteEntity(b[i:]); ok {
				j += copy(t[j:], b[start:i])
				if entityQuote != quote {
					j += copy(t[j:], []byte{entityQuote})
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
