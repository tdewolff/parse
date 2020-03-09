package js

import (
	"bytes"
	"testing"
)

var precedence = map[TokenType]int{
	AddToken: 0,
	SubToken: 0,
	MulToken: 1,
	DivToken: 1,
	ExpToken: 2,
}

var data []byte = []byte("2 + 3**2 - 3*5 + 10 - 10 * 5 / 4")

type PExpr struct {
	Data interface{}
}

func (e PExpr) String() string {
	return e.Data.(interface{ String() string }).String()
}

type BinExpr struct {
	Op          Tok
	Left, Right PExpr
}

func (e BinExpr) String() string {
	return "(" + e.Left.String() + " " + e.Op.String() + " " + e.Right.String() + ")"
}

type Tok struct {
	TokenType
	Value []byte
}

func (t Tok) String() string {
	return string(t.Value)
}

func BenchmarkParseTree(b *testing.B) {
	l := NewLexer(bytes.NewReader(data))
	for i := 0; i < b.N; i++ {
		l.Reset()

		tt, data := l.Next()
		left := PExpr{&Tok{tt, data}}
		for tt != ErrorToken {
			tt, data = l.Next()
			op := Tok{tt, data}

			tt, data = l.Next()
			right := PExpr{&Tok{tt, data}}

			left = PExpr{&BinExpr{op, left, right}}
		}
	}
}

func BenchmarkParseTree2(b *testing.B) {
	l := NewLexer(bytes.NewReader(data))
	for i := 0; i < b.N; i++ {
		l.Reset()

		tt, data := l.Next()
		left := PExpr{Tok{tt, data}}
		for tt != ErrorToken {
			tt, data = l.Next()
			op := Tok{tt, data}

			tt, data = l.Next()
			right := PExpr{Tok{tt, data}}

			left = PExpr{BinExpr{op, left, right}}
		}
	}
}

type N struct {
	TokenType
	Value []byte

	GrammarType
	Nodes []N
}

func (n N) String() string {
	return string(n.Value)
}

func BenchmarkParseSlice(b *testing.B) {
	l := NewLexer(bytes.NewReader(data))
	for i := 0; i < b.N; i++ {
		l.Reset()

		nodes := []N{}
		tt, data := l.Next()
		for tt != ErrorToken {
			nodes = append(nodes, N{tt, data, 0, nil})
			tt, data = l.Next()
		}
	}
}

func BenchmarkParseSlice2(b *testing.B) {
	l := NewLexer(bytes.NewReader(data))
	for i := 0; i < b.N; i++ {
		l.Reset()

		stack := [16]N{}
		nodes := stack[:0]
		tt, data := l.Next()
		for tt != ErrorToken {
			nodes = append(nodes, N{tt, data, 0, nil})
			tt, data = l.Next()
		}
	}
}
