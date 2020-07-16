package js

import (
	"bytes"
	"fmt"
	"hash/maphash"
	"math/rand"
	"testing"

	"github.com/tdewolff/parse/v2/css"
)

var z = 0
var n = []int{4, 12, 20, 30, 40, 50, 150}
var randStrings [][]byte
var mapStrings []map[string]bool
var mapInts []map[int]bool
var arrayStrings [][]string
var arrayBytes [][][]byte
var arrayInts [][]int

func helperRandString() string {
	cs := []byte("abcdefghijklmnopqrstuvwxyz")
	b := make([]byte, rand.Intn(10))
	for i := range b {
		b[i] = cs[rand.Intn(len(cs))]
	}
	return string(b)
}

func init() {
	for j := 0; j < len(n); j++ {
		ms := map[string]bool{}
		mi := map[int]bool{}
		as := []string{}
		ab := [][]byte{}
		ai := []int{}
		for i := 0; i < n[j]; i++ {
			s := helperRandString()
			ms[s] = true
			mi[i] = true
			as = append(as, s)
			ab = append(ab, []byte(s))
			ai = append(ai, i)
		}
		mapStrings = append(mapStrings, ms)
		mapInts = append(mapInts, mi)
		arrayStrings = append(arrayStrings, as)
		arrayBytes = append(arrayBytes, ab)
		arrayInts = append(arrayInts, ai)
	}
	for j := 0; j < 1000; j++ {
		randStrings = append(randStrings, []byte(helperRandString()))
	}
}

func BenchmarkAddMapStrings(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				m := map[string]bool{}
				for i := 0; i < n[j]; i++ {
					m[arrayStrings[j][i]] = true
				}
			}
		})
	}
}

func BenchmarkAddMapInts(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				m := map[int]bool{}
				for i := 0; i < n[j]; i++ {
					m[arrayInts[j][i]] = true
				}
			}
		})
	}
}

func BenchmarkAddArrayStrings(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				a := []string{}
				for i := 0; i < n[j]; i++ {
					a = append(a, arrayStrings[j][i])
				}
			}
		})
	}
}

func BenchmarkAddArrayBytes(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				a := [][]byte{}
				for i := 0; i < n[j]; i++ {
					a = append(a, arrayBytes[j][i])
				}
			}
		})
	}
}

func BenchmarkAddArrayInts(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				a := []int{}
				for i := 0; i < n[j]; i++ {
					a = append(a, arrayInts[j][i])
				}
			}
		})
	}
}

func BenchmarkLookupMapStrings(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					if mapStrings[j][arrayStrings[j][i]] == true {
						z++
					}
				}
			}
		})
	}
}

func BenchmarkLookupMapBytes(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					if mapStrings[j][string(arrayBytes[j][i])] == true {
						z++
					}
				}
			}
		})
	}
}

func BenchmarkLookupMapInts(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					if mapInts[j][arrayInts[j][i]] == true {
						z++
					}
				}
			}
		})
	}
}

func BenchmarkLookupArrayStrings(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					s := arrayStrings[j][i]
					for _, ss := range arrayStrings[j] {
						if s == ss {
							z++
							break
						}
					}
				}
			}
		})
	}
}

func BenchmarkLookupArrayBytes(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					s := arrayBytes[j][i]
					for _, ss := range arrayBytes[j] {
						if bytes.Equal(s, ss) {
							z++
							break
						}
					}
				}
			}
		})
	}
}

func BenchmarkLookupArrayInts(b *testing.B) {
	for j := 0; j < len(n); j++ {
		b.Run(fmt.Sprintf("%v", n[j]), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				for i := 0; i < n[j]; i++ {
					q := arrayInts[j][i]
					for _, qq := range arrayInts[j] {
						if q == qq {
							z++
							break
						}
					}
				}
			}
		})
	}
}

func BenchmarkMapHash(b *testing.B) {
	h := &maphash.Hash{}
	s := []byte(helperRandString())
	for k := 0; k < b.N; k++ {
		h.Write(s)
		_ = h.Sum64()
		h.Reset()
	}
}

func BenchmarkHash(b *testing.B) {
	s := []byte(helperRandString())
	for k := 0; k < b.N; k++ {
		_ = css.ToHash(s)
	}
}

type benchRef uint

type benchPtr struct {
	data []byte
}

type benchVar struct {
	ptr  *benchVar
	data []byte
}

var listAST []interface{}
var listPtr []*benchPtr
var listVar []benchVar

func BenchmarkASTPtr(b *testing.B) {
	for j := 0; j < 3; j++ {
		b.Run(fmt.Sprintf("%v", j), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				listAST = listAST[:0:0]
				listPtr = listPtr[:0:0]
				for _, b := range randStrings {
					v := &benchPtr{b}
					listAST = append(listAST, &v)
					listPtr = append(listPtr, v)
				}
			}
		})
	}
}

func BenchmarkASTIdx(b *testing.B) {
	for j := 0; j < 3; j++ {
		b.Run(fmt.Sprintf("%v", j), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				listAST = listAST[:0:0]
				listPtr = listPtr[:0:0]
				for _, b := range randStrings {
					v := &benchPtr{b}
					ref := benchRef(len(listPtr))
					listAST = append(listAST, &ref)
					listPtr = append(listPtr, v)
				}
			}
		})
	}
}

func BenchmarkASTVar(b *testing.B) {
	for j := 0; j < 3; j++ {
		b.Run(fmt.Sprintf("%v", j), func(b *testing.B) {
			for k := 0; k < b.N; k++ {
				listAST = listAST[:0:0]
				listVar = listVar[:0:0]
				for _, b := range randStrings {
					v := benchVar{data: b}
					v.ptr = &v
					listAST = append(listAST, len(listPtr))
					listVar = append(listVar, v)
				}
			}
		})
	}
}

var listInterface []interface{}
var listVars []*Var

func BenchmarkInterfaceAddPtr(b *testing.B) {
	listInterface = listInterface[:0:0]
	listVars = listVars[:0:0]
	for k := 0; k < b.N; k++ {
		v := &Var{VarRef(len(listVars)), 0, 0, nil}
		listInterface = append(listInterface, &v.Ref)
	}
}

func BenchmarkInterfaceAddVal32(b *testing.B) {
	listInterface = listInterface[:0:0]
	listVars = listVars[:0:0]
	for k := 0; k < b.N; k++ {
		v := &Var{VarRef(len(listVars)), 0, 0, nil}
		listInterface = append(listInterface, v.Ref)
	}
}

func BenchmarkInterfaceAddVal64(b *testing.B) {
	listInterface = listInterface[:0:0]
	listVars = listVars[:0:0]
	for k := 0; k < b.N; k++ {
		v := &Var{VarRef(len(listVars)), 0, 0, nil}
		listInterface = append(listInterface, uint64(v.Ref))
	}
}

func BenchmarkInterfaceCheckPtr(b *testing.B) {
	ref := VarRef(0)
	i := interface{}(&ref)
	for k := 0; k < b.N; k++ {
		if r, ok := i.(*VarRef); ok {
			_ = r
			z++
		}
	}
}

func BenchmarkInterfaceCheckVal(b *testing.B) {
	ref := VarRef(0)
	i := interface{}(ref)
	for k := 0; k < b.N; k++ {
		if r, ok := i.(VarRef); ok {
			_ = r
			z++
		}
	}
}
