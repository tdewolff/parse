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
var EntitiesMap = map[string][]byte{
	"DoubleContourIntegral": []byte("Conint"),
	"DoubleDot":             []byte("Dot"),
	"DoubleLeftTee":         []byte("Dashv"),
	"HumpDownHump":          []byte("Bumpeq"),
	"NegativeVeryThinSpace": []byte("NegativeMediumSpace"),
	"NestedGreaterGreater":  []byte("Gt"),
	"NestedLessLess":        []byte("Lt"),
	"Poincareplane":         []byte("Hfr"),
	"Proportion":            []byte("Colon"),
	"Rfr":                   []byte("Re"),
	"ShortDownArrow":        []byte("DownArrow"),
	"ShortLeftArrow":        []byte("LeftArrow"),
	"ShortRightArrow":       []byte("RightArrow"),
	"Subset":                []byte("Sub"),
	"Supset":                []byte("Sup"),
	"angle":                 []byte("ang"),
	"approxeq":              []byte("ape"),
	"asympeq":               []byte("CupCap"),
	"barwedge":              []byte("barwed"),
	"bigcup":                []byte("Union"),
	"bigvee":                []byte("Vee"),
	"bigwedge":              []byte("Wedge"),
	"bottom":                []byte("UpTee"),
	"bullet":                []byte("bull"),
	"checkmark":             []byte("check"),
	"circledR":              []byte("REG"),
	"clubsuit":              []byte("clubs"),
	"coloneq":               []byte("Assign"),
	"complement":            []byte("comp"),
	"complexes":             []byte("Copf"),
	"curlyeqprec":           []byte("cuepr"),
	"curlyeqsucc":           []byte("cuesc"),
	"curvearrowleft":        []byte("cularr"),
	"curvearrowright":       []byte("curarr"),
	"ddagger":               []byte("Dagger"),
	"divide":                []byte("div"),
	"doublebarwedge":        []byte("Barwed"),
	"downdownarrows":        []byte("ddarr"),
	"downharpoonleft":       []byte("LeftDownVector"),
	"downharpoonright":      []byte("RightDownVector"),
	"drbkarow":              []byte("RBarr"),
	"emptyset":              []byte("empty"),
	"emptyv":                []byte("empty"),
	"epsilon":               []byte("epsi"),
	"eqcirc":                []byte("ecir"),
	"eqcolon":               []byte("ecolon"),
	"eqslantgtr":            []byte("egs"),
	"eqslantless":           []byte("els"),
	"expectation":           []byte("Escr"),
	"fallingdotseq":         []byte("efDot"),
	"ggg":                   []byte("Gg"),
	"gnapprox":              []byte("gnap"),
	"gneq":                  []byte("gne"),
	"gneqq":                 []byte("gnE"),
	"gtrapprox":             []byte("gap"),
	"gtrdot":                []byte("gtdot"),
	"gtreqqless":            []byte("gEl"),
	"heartsuit":             []byte("hearts"),
	"hslash":                []byte("hbar"),
	"hyphen":                []byte("dash"),
	"image":                 []byte("Ifr"),
	"imagline":              []byte("Iscr"),
	"imagpart":              []byte("Ifr"),
	"inodot":                []byte("imath"),
	"integers":              []byte("Zopf"),
	"intercal":              []byte("intcal"),
	"ldquor":                []byte("bdquo"),
	"leftarrowtail":         []byte("larrtl"),
	"leftharpoondown":       []byte("DownLeftVector"),
	"leftharpoonup":         []byte("LeftVector"),
	"leftrightsquigarrow":   []byte("harrw"),
	"leq":                   []byte("le"),
	"lessapprox":            []byte("lap"),
	"lesseqqgtr":            []byte("lEg"),
	"llcorner":              []byte("dlcorn"),
	"lmoustache":            []byte("lmoust"),
	"lnapprox":              []byte("lnap"),
	"lneq":                  []byte("lne"),
	"lneqq":                 []byte("lnE"),
	"looparrowleft":         []byte("larrlp"),
	"lozenge":               []byte("loz"),
	"lrcorner":              []byte("drcorn"),
	"maltese":               []byte("malt"),
	"measuredangle":         []byte("angmsd"),
	"midast":                []byte("ast"),
	"mstpos":                []byte("ac"),
	"nabla":                 []byte("Del"),
	"natural":               []byte("natur"),
	"naturals":              []byte("Nopf"),
	"nleftarrow":            []byte("nlarr"),
	"nleftrightarrow":       []byte("nharr"),
	"nleqq":                 []byte("nlE"),
	"nrightarrow":           []byte("nrarr"),
	"nsubseteqq":            []byte("nsubE"),
	"nsupseteqq":            []byte("nsupE"),
	"orderof":               []byte("order"),
	"pitchfork":             []byte("fork"),
	"planck":                []byte("hbar"),
	"plankv":                []byte("hbar"),
	"precapprox":            []byte("prap"),
	"primes":                []byte("Popf"),
	"quaternions":           []byte("Hopf"),
	"questeq":               []byte("equest"),
	"radic":                 []byte("Sqrt"),
	"rationals":             []byte("Qopf"),
	"real":                  []byte("Re"),
	"realine":               []byte("Rscr"),
	"realpart":              []byte("Re"),
	"reals":                 []byte("Ropf"),
	"rightarrowtail":        []byte("rarrtl"),
	"rightharpoondown":      []byte("DownRightVector"),
	"rightharpoonup":        []byte("RightVector"),
	"rightleftharpoons":     []byte("Equilibrium"),
	"rightsquigarrow":       []byte("rarrw"),
	"risingdotseq":          []byte("erDot"),
	"rmoustache":            []byte("rmoust"),
	"sfrown":                []byte("frown"),
	"smallsetminus":         []byte("Backslash"),
	"spadesuit":             []byte("spades"),
	"ssmile":                []byte("smile"),
	"sstarf":                []byte("Star"),
	"straightepsilon":       []byte("epsiv"),
	"straightphi":           []byte("phiv"),
	"strns":                 []byte("macr"),
	"subset":                []byte("sub"),
	"subseteqq":             []byte("subE"),
	"subsetneq":             []byte("subne"),
	"subsetneqq":            []byte("subnE"),
	"succapprox":            []byte("scap"),
	"succnapprox":           []byte("scnap"),
	"succneqq":              []byte("scnE"),
	"succnsim":              []byte("scnsim"),
	"supseteqq":             []byte("supE"),
	"supsetneq":             []byte("supne"),
	"supsetneqq":            []byte("supnE"),
	"thickapprox":           []byte("TildeTilde"),
	"thicksim":              []byte("Tilde"),
	"thksim":                []byte("Tilde"),
	"triangledown":          []byte("dtri"),
	"triangleleft":          []byte("ltri"),
	"triangleright":         []byte("rtri"),
	"twoheadleftarrow":      []byte("Larr"),
	"twoheadrightarrow":     []byte("Rarr"),
	"ulcorner":              []byte("ulcorn"),
	"upharpoonleft":         []byte("LeftUpVector"),
	"upharpoonright":        []byte("RightUpVector"),
	"upsih":                 []byte("Upsi"),
	"upsilon":               []byte("upsi"),
	"urcorner":              []byte("urcorn"),
	"varepsilon":            []byte("epsiv"),
	"varkappa":              []byte("kappav"),
	"varnothing":            []byte("empty"),
	"varphi":                []byte("phiv"),
	"varpi":                 []byte("piv"),
	"varrho":                []byte("rhov"),
	"varsigma":              []byte("sigmaf"),
	"vartriangleleft":       []byte("LeftTriangle"),
	"vartriangleright":      []byte("RightTriangle"),
	"vee":                   []byte("or"),
	"wedge":                 []byte("and"),
	"xvee":                  []byte("Vee"),
	"xwedge":                []byte("Wedge"),
	"zeetrf":                []byte("Zfr"),
}

var Entities = map[string]byte{
	"AM":               '&',
	"AMP":              '&',
	"DiacriticalGrave": '`',
	"G":                '>',
	"GT":               '>',
	"Hat":              '^',
	"L":                '<',
	"LT":               '<',
	"NewLine":          '\n',
	"QUO":              '"',
	"QUOT":             '"',
	"Tab":              '\t',
	"UnderBar":         '_',
	"VerticalLine":     '|',
	"am":               '&',
	"amp":              '&',
	"apos":             '\'',
	"ast":              '*',
	"bsol":             '\\',
	"colon":            ':',
	"comma":            ',',
	"commat":           '@',
	"dollar":           '$',
	"equals":           '=',
	"excl":             '!',
	"grave":            '`',
	"g":                '>',
	"gt":               '>',
	"lbrace":           '{',
	"lbrack":           '[',
	"lcub":             '{',
	"lowbar":           '_',
	"lpar":             '(',
	"lsqb":             '[',
	"l":                '<',
	"lt":               '<',
	"midast":           '*',
	"num":              '#',
	"percnt":           '%',
	"period":           '.',
	"plus":             '+',
	"quest":            '?',
	"quo":              '"',
	"quot":             '"',
	"rbrace":           '}',
	"rbrack":           ']',
	"rcub":             '}',
	"rpar":             ')',
	"rsqb":             ']',
	"semi":             ';',
	"sol":              '/',
	"verbar":           '|',
	"vert":             '|',
}
