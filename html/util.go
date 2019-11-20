package html

var (
	singleQuoteEntityBytes = []byte("&#39;")
	doubleQuoteEntityBytes = []byte("&#34;")
)

// EscapeAttrVal returns the escaped attribute value bytes without quotes.
func EscapeAttrVal(buf *[]byte, orig, b []byte, isXML bool) []byte {
	singles := 0
	doubles := 0
	unquoted := true
	entities := false
	for _, c := range b {
		if charTable[c] {
			unquoted = false
			if c == '"' {
				doubles++
			} else if c == '\'' {
				singles++
			}
		}
	}
	if unquoted && !isXML {
		return b
	} else if !entities && len(orig) == len(b)+2 && (singles == 0 && orig[0] == '\'' || doubles == 0 && orig[0] == '"') {
		return orig
	}

	n := len(b) + 2
	var quote byte
	var escapedQuote []byte
	if singles >= doubles || isXML {
		n += doubles * 4
		quote = '"'
		escapedQuote = doubleQuoteEntityBytes
	} else {
		n += singles * 4
		quote = '\''
		escapedQuote = singleQuoteEntityBytes
	}
	if n > cap(*buf) {
		*buf = make([]byte, 0, n) // maximum size, not actual size
	}
	t := (*buf)[:n] // maximum size, not actual size
	t[0] = quote
	j := 1
	start := 0
	for i, c := range b {
		if c == quote {
			j += copy(t[j:], b[start:i])
			j += copy(t[j:], escapedQuote)
			start = i + 1
		}
	}
	j += copy(t[j:], b[start:])
	t[j] = quote
	return t[:j+1]
}

var charTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, true, true, false, true, true, false, false, // tab, line feed, form feed, carriage return
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	true, false, true, false, false, false, false, true, // space, "), '
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, true, true, true, false, // <, =, >

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	true, false, false, false, false, false, false, false, // `
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}

// Entities are all named character entities.
var Entities = map[string]byte{
	"AElig":            '\u00C6',
	"AMP":              '\u0026',
	"Aacute":           '\u00C1',
	"Acirc":            '\u00C2',
	"Agrave":           '\u00C0',
	"Aring":            '\u00C5',
	"Atilde":           '\u00C3',
	"Auml":             '\u00C4',
	"COPY":             '\u00A9',
	"Ccedil":           '\u00C7',
	"Cedilla":          '\u00B8',
	"CenterDot":        '\u00B7',
	"DiacriticalAcute": '\u00B4',
	"DiacriticalGrave": '\u0060',
	"Dot":              '\u00A8',
	"DoubleDot":        '\u00A8',
	"ETH":              '\u00D0',
	"Eacute":           '\u00C9',
	"Ecirc":            '\u00CA',
	"Egrave":           '\u00C8',
	"Euml":             '\u00CB',
	"GT":               '\u003E',
	"Hat":              '\u005E',
	"Iacute":           '\u00CD',
	"Icirc":            '\u00CE',
	"Igrave":           '\u00CC',
	"Iuml":             '\u00CF',
	"LT":               '\u003C',
	"NewLine":          '\u000A',
	"NonBreakingSpace": '\u00A0',
	"Ntilde":           '\u00D1',
	"Oacute":           '\u00D3',
	"Ocirc":            '\u00D4',
	"Ograve":           '\u00D2',
	"Oslash":           '\u00D8',
	"Otilde":           '\u00D5',
	"Ouml":             '\u00D6',
	"PlusMinus":        '\u00B1',
	"QUOT":             '\u0022',
	"REG":              '\u00AE',
	"THORN":            '\u00DE',
	"Tab":              '\u0009',
	"Uacute":           '\u00DA',
	"Ucirc":            '\u00DB',
	"Ugrave":           '\u00D9',
	"UnderBar":         '\u005F',
	"Uuml":             '\u00DC',
	"VerticalLine":     '\u007C',
	"Yacute":           '\u00DD',
	"aacute":           '\u00E1',
	"acirc":            '\u00E2',
	"acute":            '\u00B4',
	"aelig":            '\u00E6',
	"agrave":           '\u00E0',
	"amp":              '\u0026',
	"angst":            '\u00C5',
	"apos":             '\u0027',
	"aring":            '\u00E5',
	"ast":              '\u002A',
	"atilde":           '\u00E3',
	"auml":             '\u00E4',
	"bsol":             '\u005C',
	"ccedil":           '\u00E7',
	"cedil":            '\u00B8',
	"cent":             '\u00A2',
	"centerdot":        '\u00B7',
	"circledR":         '\u00AE',
	"colon":            '\u003A',
	"comma":            '\u002C',
	"commat":           '\u0040',
	"copy":             '\u00A9',
	"curren":           '\u00A4',
	"deg":              '\u00B0',
	"die":              '\u00A8',
	"div":              '\u00F7',
	"divide":           '\u00F7',
	"eacute":           '\u00E9',
	"ecirc":            '\u00EA',
	"egrave":           '\u00E8',
	"equals":           '\u003D',
	"eth":              '\u00F0',
	"euml":             '\u00EB',
	"excl":             '\u0021',
	"frac12":           '\u00BD',
	"frac14":           '\u00BC',
	"frac34":           '\u00BE',
	"gt":               '\u003E',
	"half":             '\u00BD',
	"iacute":           '\u00ED',
	"icirc":            '\u00EE',
	"iexcl":            '\u00A1',
	"iuml":             '\u00EF',
	"laquo":            '\u00AB',
	"lbrace":           '\u007B',
	"lbrack":           '\u005B',
	"lcub":             '\u007B',
	"lowbar":           '\u005F',
	"lpar":             '\u0028',
	"lsqb":             '\u005B',
	"lt":               '\u003C',
	"macr":             '\u00AF',
	"micro":            '\u00B5',
	"midast":           '\u002A',
	"middot":           '\u00B7',
	"nbsp":             '\u00A0',
	"ntilde":           '\u00F1',
	"num":              '\u0023',
	"oacute":           '\u00F3',
	"ocirc":            '\u00F4',
	"ograve":           '\u00F2',
	"ordf":             '\u00AA',
	"ordm":             '\u00BA',
	"oslash":           '\u00F8',
	"otilde":           '\u00F5',
	"ouml":             '\u00F6',
	"para":             '\u00B6',
	"percnt":           '\u0025',
	"period":           '\u002E',
	"plus":             '\u002B',
	"plusmn":           '\u00B1',
	"pm":               '\u00B1',
	"quest":            '\u003F',
	"quot":             '\u0022',
	"raquo":            '\u00BB',
	"rbrace":           '\u007D',
	"rbrack":           '\u005D',
	"rcub":             '\u007D',
	"reg":              '\u00AE',
	"rpar":             '\u0029',
	"rsqb":             '\u005D',
	"sect":             '\u00A7',
	"semi":             '\u003B',
	"shy":              '\u00AD',
	"sol":              '\u002F',
	"strns":            '\u00AF',
	"sup1":             '\u00B9',
	"sup2":             '\u00B2',
	"sup3":             '\u00B3',
	"szlig":            '\u00DF',
	"thorn":            '\u00FE',
	"times":            '\u00D7',
	"uacute":           '\u00FA',
	"ucirc":            '\u00FB',
	"ugrave":           '\u00F9',
	"uml":              '\u00A8',
	"uuml":             '\u00FC',
	"verbar":           '\u007C',
	"vert":             '\u007C',
	"yacute":           '\u00FD',
	"yen":              '\u00A5',
	"yuml":             '\u00FF',
}
