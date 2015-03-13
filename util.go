package parse // import "github.com/tdewolff/parse"

func Copy(src []byte) (dst []byte) {
	dst = make([]byte, len(src))
	copy(dst, src)
	return
}

func ToLower(src []byte) []byte {
	for i, c := range src {
		if c >= 'A' && c <= 'Z' {
			src[i] = c + ('a' - 'A')
		}
	}
	return src
}

func CopyToLower(src []byte) []byte {
	dst := Copy(src)
	for i, c := range dst {
		if c >= 'A' && c <= 'Z' {
			dst[i] = c + ('a' - 'A')
		}
	}
	return dst
}