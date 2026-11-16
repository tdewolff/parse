package js

import "strconv"

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

// AsNumericPropertyKey returns true if the string property key b can be written as a numeric
// literal without changing the key. A numeric key is converted with ToString(ToNumber(b)), so this
// only holds when that gives back b exactly: {"1.0":x} is not {1.0:x}, which is {"1":x}.
func AsNumericPropertyKey(b []byte) bool {
	if !AsDecimalLiteral(b) || b[0] == '.' || b[len(b)-1] == '.' {
		return false
	}
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil || 1e21 <= f {
		// ToString uses exponent notation from 1e21 onwards
		return false
	}
	return strconv.FormatFloat(f, 'f', -1, 64) == string(b)
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
