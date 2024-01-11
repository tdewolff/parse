package js

import (
	"io"
)

func isLHSExpr(i IExpr) bool {
	switch i.(type) {
	case *CommaExpr, *CondExpr, *YieldExpr, *ArrowFunc, *BinaryExpr, *UnaryExpr:
		return false
	}
	return true
}

// AsIdentifierName returns true if a valid identifier name is given.
func AsIdentifierName(b []byte) bool {
	if len(b) == 0 || !identifierStartTable[b[0]] {
		return false
	}

	i := 1
	for i < len(b) {
		if identifierTable[b[i]] {
			i++
		} else {
			return false
		}
	}
	return true
}

// AsDecimalLiteral returns true if a valid decimal literal is given.
func AsDecimalLiteral(b []byte) bool {
	if len(b) == 0 || (b[0] < '0' || '9' < b[0]) && (b[0] != '.' || len(b) == 1) {
		return false
	} else if b[0] == '0' {
		return len(b) == 1
	}
	i := 1
	for i < len(b) && '0' <= b[i] && b[i] <= '9' {
		i++
	}
	if i < len(b) && b[i] == '.' && b[0] != '.' {
		i++
		for i < len(b) && '0' <= b[i] && b[i] <= '9' {
			i++
		}
	}
	return i == len(b)
}

type Indenter struct {
	w io.Writer
	b []byte
}

func NewIndenter(w io.Writer, n int) Indenter {
	if wi, ok := w.(Indenter); ok {
		w = wi.w
		n += len(wi.b)
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return Indenter{
		w: w,
		b: b,
	}
}

func (in Indenter) Write(b []byte) (int, error) {
	n, j := 0, 0
	for i, c := range b {
		if c == '\n' {
			m, _ := in.w.Write(b[j : i+1])
			n += m
			m, _ = in.w.Write(in.b)
			n += m
			j = i + 1
		}
	}
	m, err := in.w.Write(b[j:])
	return n + m, err
}
