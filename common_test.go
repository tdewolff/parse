package parse // import "github.com/tdewolff/parse"

import (
	"encoding/base64"
	"mime"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		number := Number([]byte(tt.number))
		assert.Equal(t, tt.expected, number, "Number must give expected result in "+tt.number)
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
		num, unit := Dimension([]byte(tt.dimension))
		assert.Equal(t, tt.expectedNum, num, "Dimension must give expected result in "+tt.dimension)
		assert.Equal(t, tt.expectedUnit, unit, "Dimension must give expected result in "+tt.dimension)
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
	}
	for _, tt := range mediatypeTests {
		mimetype, params := Mediatype([]byte(tt.mediatype))
		assert.Equal(t, tt.expectedMimetype, string(mimetype), "Mediatype must give expected result in "+tt.mediatype)
		assert.Equal(t, tt.expectedParams, params, "Mediatype must give expected result in "+tt.mediatype)
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
	}
	for _, tt := range dataURITests {
		mimetype, data, err := DataURI([]byte(tt.dataURI))
		assert.Equal(t, tt.expectedMimetype, string(mimetype), "DataURI must give expected result in "+tt.dataURI)
		assert.Equal(t, tt.expectedData, string(data), "DataURI must give expected result in "+tt.dataURI)
		assert.Equal(t, tt.expectedErr, err, "DataURI must give expected result in "+tt.dataURI)
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
		quote, n := QuoteEntity([]byte(tt.quoteEntity))
		assert.Equal(t, tt.expectedQuote, quote, "QuoteEntity must give expected result in "+tt.quoteEntity)
		assert.Equal(t, tt.expectedN, n, "QuoteEntity must give expected result in "+tt.quoteEntity)
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
