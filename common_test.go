package parse

import (
	"encoding/base64"
	"mime"
	"net/url"
	"regexp"
	"testing"

	"github.com/tdewolff/test"
)

var entitySlices [][]byte
var encodedURLSlices [][]byte
var urlSlices [][]byte

func init() {
	entitySlices = helperRandStrings(100, 5, []string{"&quot;", "&#39;", "&#x027;", "    ", " ", "test"})
	encodedURLSlices = helperRandStrings(100, 5, []string{"%20", "%3D", "test"})
	urlSlices = helperRandStrings(100, 5, []string{"~", "\"", "<", "test"})
}

func TestParseNumber(t *testing.T) {
	var numberTests = []struct {
		number   string
		expected int
	}{
		{"5", 1},
		{"0.51", 4},
		{"0.5e-99", 7},
		{"0.5e-", 3},
		{"+50.0", 5},
		{".0", 2},
		{"0.", 1},
		{"", 0},
		{"+", 0},
		{".", 0},
		{"a", 0},
	}
	for _, tt := range numberTests {
		t.Run(tt.number, func(t *testing.T) {
			n := Number([]byte(tt.number))
			test.T(t, n, tt.expected)
		})
	}
}

func TestParseDimension(t *testing.T) {
	var dimensionTests = []struct {
		dimension    string
		expectedNum  int
		expectedUnit int
	}{
		{"5px", 1, 2},
		{"5px ", 1, 2},
		{"5%", 1, 1},
		{"5em", 1, 2},
		{"px", 0, 0},
		{"1", 1, 0},
		{"1~", 1, 0},
	}
	for _, tt := range dimensionTests {
		t.Run(tt.dimension, func(t *testing.T) {
			num, unit := Dimension([]byte(tt.dimension))
			test.T(t, num, tt.expectedNum, "number")
			test.T(t, unit, tt.expectedUnit, "unit")
		})
	}
}

func TestMediatype(t *testing.T) {
	var mediatypeTests = []struct {
		mediatype        string
		expectedMimetype string
		expectedParams   map[string]string
	}{
		{"text/plain", "text/plain", nil},
		{"text/plain;charset=US-ASCII", "text/plain", map[string]string{"charset": "US-ASCII"}},
		{" text/plain  ; charset = US-ASCII ", "text/plain", map[string]string{"charset": "US-ASCII"}},
		{" text/plain  a", "text/plain", nil},
		{"text/plain;base64", "text/plain", map[string]string{"base64": ""}},
		{"text/plain;inline=;base64", "text/plain", map[string]string{"inline": "", "base64": ""}},
		{"ÿ   ", "ÿ ", nil}, // OSS-Fuzz; ÿ is two bytes in UTF8
		{"ÿ  ;", "ÿ ", map[string]string{"": ""}},
	}
	for _, tt := range mediatypeTests {
		t.Run(tt.mediatype, func(t *testing.T) {
			mimetype, params := Mediatype([]byte(tt.mediatype))
			test.String(t, string(mimetype), tt.expectedMimetype, "mimetype")
			test.T(t, params, tt.expectedParams, "parameters") // TODO
		})
	}
}

func TestParseDataURI(t *testing.T) {
	var dataURITests = []struct {
		dataURI          string
		expectedMimetype string
		expectedData     string
		expectedErr      error
	}{
		{"www.domain.com", "", "", ErrBadDataURI},
		{"data:,", "text/plain", "", nil},
		{"data:text/xml,", "text/xml", "", nil},
		{"data:,text", "text/plain", "text", nil},
		{"data:;base64,dGV4dA==", "text/plain", "text", nil},
		{"data:image/svg+xml,", "image/svg+xml", "", nil},
		{"data:;base64,()", "", "", base64.CorruptInputError(0)},
		{"data:image/svg+xml,%3Cpath%20stroke-width='9.38%'/%3E", "image/svg+xml", "<path stroke-width='9.38%'/>", nil},
		{"data:,%ii", "text/plain", "%ii", nil},
	}
	for _, tt := range dataURITests {
		t.Run(tt.dataURI, func(t *testing.T) {
			mimetype, data, err := DataURI([]byte(tt.dataURI))
			test.T(t, err, tt.expectedErr)
			test.String(t, string(mimetype), tt.expectedMimetype, "mimetype")
			test.String(t, string(data), tt.expectedData, "data")
		})
	}
}

func TestParseQuoteEntity(t *testing.T) {
	var quoteEntityTests = []struct {
		quoteEntity   string
		expectedQuote byte
		expectedN     int
	}{
		{"&#34;", '"', 5},
		{"&#039;", '\'', 6},
		{"&#x0022;", '"', 8},
		{"&#x27;", '\'', 6},
		{"&quot;", '"', 6},
		{"&apos;", '\'', 6},
		{"&gt;", 0x00, 0},
		{"&amp;", 0x00, 0},
	}
	for _, tt := range quoteEntityTests {
		t.Run(tt.quoteEntity, func(t *testing.T) {
			quote, n := QuoteEntity([]byte(tt.quoteEntity))
			test.T(t, quote, tt.expectedQuote, "quote")
			test.T(t, n, tt.expectedN, "quote length")
		})
	}
}

func TestReplaceMultipleWhitespace(t *testing.T) {
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a")), []byte(" a"))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("a  ")), []byte("a "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("a  b  ")), []byte("a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a  b  ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte(" a b  ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("  a b ")), []byte(" a b "))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("   a")), []byte(" a"))
	test.Bytes(t, ReplaceMultipleWhitespace([]byte("a  b")), []byte("a b"))
}

func TestReplaceMultipleWhitespaceRandom(t *testing.T) {
	wsRegexp := regexp.MustCompile("[ \t\f]+")
	wsNewlinesRegexp := regexp.MustCompile("[ ]*[\r\n][ \r\n]*")
	for _, e := range wsSlices {
		reference := wsRegexp.ReplaceAll(e, []byte(" "))
		reference = wsNewlinesRegexp.ReplaceAll(reference, []byte("\n"))
		test.Bytes(t, ReplaceMultipleWhitespace(Copy(e)), reference, "in '"+string(e)+"'")
	}
}

func TestReplaceEntities(t *testing.T) {
	entitiesMap := map[string][]byte{
		"varphi": []byte("&phiv;"),
		"varpi":  []byte("&piv;"),
		"quot":   []byte("\""),
		"apos":   []byte("'"),
		"amp":    []byte("&"),
	}
	revEntitiesMap := map[byte][]byte{
		'\'': []byte("&#39;"),
	}
	var entityTests = []struct {
		entity   string
		expected string
	}{
		{"&#34;", `"`},
		{"&#039;", `&#39;`},
		{"&#x0022;", `"`},
		{"&#x27;", `&#39;`},
		{"&#160;", `&#160;`},
		{"&quot;", `"`},
		{"&apos;", `&#39;`},
		{"&#9191;", `&#9191;`},
		{"&#x23e7;", `&#9191;`},
		{"&#x23E7;", `&#9191;`},
		{"&#x23E7;", `&#9191;`},
		{"&#x270F;", `&#9999;`},
		{"&#x2710;", `&#x2710;`},
		{"&apos;&quot;", `&#39;"`},
		{"&#34", `&#34`},
		{"&#x22", `&#x22`},
		{"&apos", `&apos`},
		{"&amp;", `&`},
		{"&#39;", `&#39;`},
		{"&amp;amp;", `&amp;amp;`},
		{"&amp;#34;", `&amp;#34;`},
		//{"&amp;a mp;", `&a mp;`},
		{"&amp;DiacriticalAcute;", `&amp;DiacriticalAcute;`},
		{"&amp;CounterClockwiseContourIntegral;", `&amp;CounterClockwiseContourIntegral;`},
		//{"&amp;CounterClockwiseContourIntegralL;", `&CounterClockwiseContourIntegralL;`},
		{"&amp;parameterize", `&amp;parameterize`},
		{"&varphi;", "&phiv;"},
		{"&varpi;", "&piv;"},
		{"&varnone;", "&varnone;"},
	}
	for _, tt := range entityTests {
		t.Run(tt.entity, func(t *testing.T) {
			b := ReplaceEntities([]byte(tt.entity), entitiesMap, revEntitiesMap)
			test.T(t, string(b), tt.expected, "in '"+tt.entity+"'")
		})
	}
}

func TestReplaceEntitiesRandom(t *testing.T) {
	entitiesMap := map[string][]byte{
		"quot": []byte("\""),
		"apos": []byte("'"),
	}
	revEntitiesMap := map[byte][]byte{
		'\'': []byte("&#39;"),
	}

	quotRegexp := regexp.MustCompile("&quot;")
	aposRegexp := regexp.MustCompile("(&#39;|&#x027;)")
	for _, e := range entitySlices {
		reference := quotRegexp.ReplaceAll(e, []byte("\""))
		reference = aposRegexp.ReplaceAll(reference, []byte("&#39;"))
		test.Bytes(t, ReplaceEntities(Copy(e), entitiesMap, revEntitiesMap), reference, "in '"+string(e)+"'")
	}
}

func TestReplaceMultipleWhitespaceAndEntities(t *testing.T) {
	entitiesMap := map[string][]byte{
		"varphi": []byte("&phiv;"),
	}
	var entityTests = []struct {
		entity   string
		expected string
	}{
		{"  &varphi;  &#34; \n ", " &phiv; \"\n"},
	}
	for _, tt := range entityTests {
		t.Run(tt.entity, func(t *testing.T) {
			b := ReplaceMultipleWhitespaceAndEntities([]byte(tt.entity), entitiesMap, nil)
			test.T(t, string(b), tt.expected, "in '"+tt.entity+"'")
		})
	}
}

func TestReplaceMultipleWhitespaceAndEntitiesRandom(t *testing.T) {
	entitiesMap := map[string][]byte{
		"quot": []byte("\""),
		"apos": []byte("'"),
	}
	revEntitiesMap := map[byte][]byte{
		'\'': []byte("&#39;"),
	}

	wsRegexp := regexp.MustCompile("[ ]+")
	quotRegexp := regexp.MustCompile("&quot;")
	aposRegexp := regexp.MustCompile("(&#39;|&#x027;)")
	for _, e := range entitySlices {
		reference := wsRegexp.ReplaceAll(e, []byte(" "))
		reference = quotRegexp.ReplaceAll(reference, []byte("\""))
		reference = aposRegexp.ReplaceAll(reference, []byte("&#39;"))
		test.Bytes(t, ReplaceMultipleWhitespaceAndEntities(Copy(e), entitiesMap, revEntitiesMap), reference, "in '"+string(e)+"'")
	}
}

func TestDecodeURL(t *testing.T) {
	var urlTests = []struct {
		url      string
		expected string
	}{
		{"%20%3F%7E", " ?~"},
		{"%80", "%80"},
		{"%2B%2b", "++"},
		{"%' ", "%' "},
		{"a+b", "a b"},
	}
	for _, tt := range urlTests {
		t.Run(tt.url, func(t *testing.T) {
			b := DecodeURL([]byte(tt.url))
			test.T(t, string(b), tt.expected, "in '"+tt.url+"'")
		})
	}
}

func TestDecodeURLRandom(t *testing.T) {
	for _, e := range encodedURLSlices {
		reference, _ := url.QueryUnescape(string(e))
		test.Bytes(t, DecodeURL(Copy(e)), []byte(reference), "in '"+string(e)+"'")
	}
}

func TestEncodeURL(t *testing.T) {
	var urlTests = []struct {
		url      string
		expected string
	}{
		{"AZaz09-_.!~*'()", "AZaz09-_.!~*'()"},
		{"<>", "%3C%3E"},
		{"\u2318", "%E2%8C%98"},
		{"a b", "a%20b"},
	}
	for _, tt := range urlTests {
		t.Run(tt.url, func(t *testing.T) {
			b := EncodeURL([]byte(tt.url), URLEncodingTable)
			test.T(t, string(b), tt.expected, "in '"+tt.url+"'")
		})
	}
}

func TestEncodeDataURI(t *testing.T) {
	var urlTests = []struct {
		url      string
		expected string
	}{
		{`<svg xmlns="http://www.w3.org/2000/svg"></svg>`, `%3Csvg%20xmlns=%22http://www.w3.org/2000/svg%22%3E%3C/svg%3E`},
	}
	for _, tt := range urlTests {
		t.Run(tt.url, func(t *testing.T) {
			b := EncodeURL([]byte(tt.url), DataURIEncodingTable)
			test.T(t, string(b), tt.expected, "in '"+tt.url+"'")
		})
	}
}

func TestEncodeURLRandom(t *testing.T) {
	for _, e := range urlSlices {
		reference := url.QueryEscape(string(e))
		test.Bytes(t, EncodeURL(Copy(e), URLEncodingTable), []byte(reference), "in '"+string(e)+"'")
	}
}

////////////////////////////////////////////////////////////////

func BenchmarkParseMediatypeStd(b *testing.B) {
	mediatype := "text/plain"
	for i := 0; i < b.N; i++ {
		mime.ParseMediaType(mediatype)
	}
}

func BenchmarkParseMediatypeParamStd(b *testing.B) {
	mediatype := "text/plain;inline=1"
	for i := 0; i < b.N; i++ {
		mime.ParseMediaType(mediatype)
	}
}

func BenchmarkParseMediatypeParamsStd(b *testing.B) {
	mediatype := "text/plain;charset=US-ASCII;language=US-EN;compression=gzip;base64"
	for i := 0; i < b.N; i++ {
		mime.ParseMediaType(mediatype)
	}
}

func BenchmarkParseMediatypeParse(b *testing.B) {
	mediatype := []byte("text/plain")
	for i := 0; i < b.N; i++ {
		Mediatype(mediatype)
	}
}

func BenchmarkParseMediatypeParamParse(b *testing.B) {
	mediatype := []byte("text/plain;inline=1")
	for i := 0; i < b.N; i++ {
		Mediatype(mediatype)
	}
}

func BenchmarkParseMediatypeParamsParse(b *testing.B) {
	mediatype := []byte("text/plain;charset=US-ASCII;language=US-EN;compression=gzip;base64")
	for i := 0; i < b.N; i++ {
		Mediatype(mediatype)
	}
}

func BenchmarkReplaceMultipleWhitespace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, e := range wsSlices {
			ReplaceMultipleWhitespace(e)
		}
	}
}
