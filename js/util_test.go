package js

import (
	"testing"

	"github.com/tdewolff/test"
)

func TestAsIdentifierName(t *testing.T) {
	test.That(t, !AsIdentifierName([]byte("")))
	test.That(t, !AsIdentifierName([]byte("5")))
	test.That(t, AsIdentifierName([]byte("ab")))
	test.That(t, !AsIdentifierName([]byte("a=")))
}

func TestAsNumericPropertyKey(t *testing.T) {
	test.That(t, AsNumericPropertyKey([]byte("0")))
	test.That(t, AsNumericPropertyKey([]byte("12")))
	test.That(t, AsNumericPropertyKey([]byte("12.56")))
	test.That(t, AsNumericPropertyKey([]byte("100000000000000000000")))
	test.That(t, !AsNumericPropertyKey([]byte("")))
	test.That(t, !AsNumericPropertyKey([]byte("a")))
	test.That(t, !AsNumericPropertyKey([]byte("00")))
	test.That(t, !AsNumericPropertyKey([]byte("1.0")))                    // 1.0 is key "1"
	test.That(t, !AsNumericPropertyKey([]byte("1.50")))                   // 1.50 is key "1.5"
	test.That(t, !AsNumericPropertyKey([]byte(".5")))                     // .5 is key "0.5"
	test.That(t, !AsNumericPropertyKey([]byte("5.")))                     // 5. is key "5"
	test.That(t, !AsNumericPropertyKey([]byte("9007199254740993")))       // rounds to 9007199254740992
	test.That(t, !AsNumericPropertyKey([]byte("1000000000000000000000"))) // 1e21 is key "1e+21"
}

func TestAsDecimalLiteral(t *testing.T) {
	test.That(t, !AsDecimalLiteral([]byte("")))
	test.That(t, !AsDecimalLiteral([]byte("a")))
	test.That(t, AsDecimalLiteral([]byte("12")))
	test.That(t, AsDecimalLiteral([]byte("12.56")))
	test.That(t, AsDecimalLiteral([]byte(".56")))
	test.That(t, !AsDecimalLiteral([]byte(".56a")))
	test.That(t, !AsDecimalLiteral([]byte(".")))
	test.That(t, AsDecimalLiteral([]byte("0")))
	test.That(t, !AsDecimalLiteral([]byte("00")))
}
