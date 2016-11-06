package json // import "github.com/tdewolff/parse/json"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/tdewolff/test"
)

func helperStringify(t *testing.T, input string) string {
	s := ""
	p := NewParser(bytes.NewBufferString(input))
	for i := 0; i < 10; i++ {
		gt, text := p.Next()
		if gt == ErrorGrammar {
			if p.Err() != nil {
				s += gt.String() + "('" + p.Err().Error() + "')"
			} else {
				s += gt.String() + "(nil)"
			}
			break
		} else if gt == WhitespaceGrammar {
			continue
		} else {
			s += gt.String() + "('" + string(text) + "') "
		}
	}
	return s
}

////////////////////////////////////////////////////////////////

type GTs []GrammarType

func TestGrammars(t *testing.T) {
	var grammarTests = []struct {
		json     string
		expected []GrammarType
	}{
		{" \t\n\r", GTs{}}, // WhitespaceGrammar
		{"null", GTs{LiteralGrammar}},
		{"[]", GTs{StartArrayGrammar, EndArrayGrammar}},
		{"[15.2, 0.4, 5e9, -4E-3]", GTs{StartArrayGrammar, NumberGrammar, NumberGrammar, NumberGrammar, NumberGrammar, EndArrayGrammar}},
		{"[true, false, null]", GTs{StartArrayGrammar, LiteralGrammar, LiteralGrammar, LiteralGrammar, EndArrayGrammar}},
		{`["", "abc", "\"", "\\"]`, GTs{StartArrayGrammar, StringGrammar, StringGrammar, StringGrammar, StringGrammar, EndArrayGrammar}},
		{"{}", GTs{StartObjectGrammar, EndObjectGrammar}},
		{`{"a": "b", "c": "d"}`, GTs{StartObjectGrammar, StringGrammar, StringGrammar, StringGrammar, StringGrammar, EndObjectGrammar}},
		{`{"a": [1, 2], "b": {"c": 3}}`, GTs{StartObjectGrammar, StringGrammar, StartArrayGrammar, NumberGrammar, NumberGrammar, EndArrayGrammar, StringGrammar, StartObjectGrammar, StringGrammar, NumberGrammar, EndObjectGrammar, EndObjectGrammar}},
		{"[null,]", GTs{StartArrayGrammar, LiteralGrammar, EndArrayGrammar}},
		{"[\"x\\\x00y\", 0]", GTs{StartArrayGrammar, StringGrammar, NumberGrammar, EndArrayGrammar}},
	}
	for _, tt := range grammarTests {
		stringify := helperStringify(t, tt.json)
		p := NewParser(bytes.NewBufferString(tt.json))
		i := 0
		for {
			grammar, _ := p.Next()
			if grammar == ErrorGrammar {
				test.That(t, i == len(tt.expected), "when error occurred we must be at the end in "+stringify)
				test.Error(t, p.Err(), io.EOF, "in "+stringify)
				break
			} else if grammar == WhitespaceGrammar {
				continue
			}
			test.That(t, i < len(tt.expected), "index", i, "must not exceed expected grammar types size", len(tt.expected), "in "+stringify)
			if i < len(tt.expected) {
				test.That(t, grammar == tt.expected[i], "grammar types must match at index "+strconv.Itoa(i)+" in "+stringify)
			}
			i++
		}
	}

	test.String(t, WhitespaceGrammar.String(), "Whitespace")
	test.String(t, GrammarType(100).String(), "Invalid(100)")
	test.String(t, ValueState.String(), "Value")
	test.String(t, ObjectKeyState.String(), "ObjectKey")
	test.String(t, ObjectValueState.String(), "ObjectValue")
	test.String(t, ArrayState.String(), "Array")
	test.String(t, State(100).String(), "Invalid(100)")
}

func TestGrammarsError(t *testing.T) {
	var grammarErrorTests = []struct {
		json     string
		expected error
	}{
		{"true, false", ErrBadComma},
		{"[true false]", ErrNoComma},
		{"]", ErrBadArrayEnding},
		{"}", ErrBadObjectEnding},
		{"{0: 1}", ErrBadObjectKey},
		{"{\"a\" 1}", ErrBadObjectDeclaration},
		{"1.", ErrNoComma},
		{"1e+", ErrNoComma},
		{`{"":"`, io.EOF},
		{"\"a\\", io.EOF},
	}
	for _, tt := range grammarErrorTests {
		stringify := helperStringify(t, tt.json)
		p := NewParser(bytes.NewBufferString(tt.json))
		for {
			grammar, _ := p.Next()
			if grammar == ErrorGrammar {
				test.Error(t, p.Err(), tt.expected, "in "+stringify)
				break
			}
		}
	}
}

func TestStates(t *testing.T) {
	var stateTests = []struct {
		json     string
		expected []State
	}{
		{"null", []State{ValueState}},
		{"[null]", []State{ArrayState, ArrayState, ValueState}},
		{"{\"\":null}", []State{ObjectKeyState, ObjectValueState, ObjectKeyState, ValueState}},
	}
	for _, tt := range stateTests {
		stringify := helperStringify(t, tt.json)
		p := NewParser(bytes.NewBufferString(tt.json))
		i := 0
		for {
			grammar, _ := p.Next()
			state := p.State()
			if grammar == ErrorGrammar {
				test.That(t, i == len(tt.expected), "when error occurred we must be at the end in "+stringify)
				test.Error(t, p.Err(), io.EOF, "in "+stringify)
				break
			} else if grammar == WhitespaceGrammar {
				continue
			}
			test.That(t, i < len(tt.expected), "index", i, "must not exceed expected states size", len(tt.expected), "in "+stringify)
			if i < len(tt.expected) {
				test.That(t, state == tt.expected[i], "states must match at index "+strconv.Itoa(i)+" in "+stringify)
			}
			i++
		}
	}
}

////////////////////////////////////////////////////////////////

func ExampleNewParser() {
	p := NewParser(bytes.NewBufferString(`{"key": 5}`))
	out := ""
	for {
		state := p.State()
		gt, data := p.Next()
		if gt == ErrorGrammar {
			break
		}
		out += string(data)
		if state == ObjectKeyState && gt != EndObjectGrammar {
			out += ":"
		}
		// not handling comma insertion
	}
	fmt.Println(out)
	// Output: {"key":5}
}
