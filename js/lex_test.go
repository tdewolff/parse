package js

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/tdewolff/test"
)

type TTs []TokenType

func TestTokens(t *testing.T) {
	var tokenTests = []struct {
		js       string
		expected []TokenType
	}{
		{" \t\v\f\u00A0\uFEFF\u2000", TTs{}}, // WhitespaceToken
		{"\n\r\r\n\u2028\u2029", TTs{LineTerminatorToken}},
		{"5.2 .04 0x0F 5e99", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"a = 'string'", TTs{IdentifierToken, EqToken, StringToken}},
		{"/*comment*/ //comment", TTs{SingleLineCommentToken, SingleLineCommentToken}},
		{"{ } ( ) [ ]", TTs{OpenBraceToken, CloseBraceToken, OpenParenToken, CloseParenToken, OpenBracketToken, CloseBracketToken}},
		{". ; , < > <= ...", TTs{DotToken, SemicolonToken, CommaToken, LtToken, GtToken, LtEqToken, EllipsisToken}},
		{">= == != === !==", TTs{GtEqToken, EqEqToken, NotEqToken, EqEqEqToken, NotEqEqToken}},
		{"+ - * / % ** ++ --", TTs{AddToken, SubToken, MulToken, DivToken, ModToken, ExpToken, IncrToken, DecrToken}},
		{"<< >> >>> & | ^", TTs{LtLtToken, GtGtToken, GtGtGtToken, BitAndToken, BitOrToken, BitXorToken}},
		{"! ~ && || ? : ?? ?.", TTs{NotToken, BitNotToken, AndToken, OrToken, QuestionToken, ColonToken, NullishToken, OptChainToken}},
		{"= += -= *= **= /= %= <<=", TTs{EqToken, AddEqToken, SubEqToken, MulEqToken, ExpEqToken, DivEqToken, ModEqToken, LtLtEqToken}},
		{">>= >>>= &= |= ^= =>", TTs{GtGtEqToken, GtGtGtEqToken, BitAndEqToken, BitOrEqToken, BitXorEqToken, ArrowToken}},
		{"?.5", TTs{QuestionToken, NumericToken}},
		{"?.a", TTs{OptChainToken, IdentifierToken}},
		{"async await break case catch class const continue", TTs{AsyncToken, AwaitToken, BreakToken, CaseToken, CatchToken, ClassToken, ConstToken, ContinueToken}},
		{"debugger default delete do else enum export extends", TTs{DebuggerToken, DefaultToken, DeleteToken, DoToken, ElseToken, EnumToken, ExportToken, ExtendsToken}},
		{"false finally for function if implements import in", TTs{FalseToken, FinallyToken, ForToken, FunctionToken, IfToken, ImplementsToken, ImportToken, InToken}},
		{"instanceof interface let new null package private protected", TTs{InstanceofToken, InterfaceToken, LetToken, NewToken, NullToken, PackageToken, PrivateToken, ProtectedToken}},
		{"public return static super switch this throw true", TTs{PublicToken, ReturnToken, StaticToken, SuperToken, SwitchToken, ThisToken, ThrowToken, TrueToken}},
		{"try typeof var void while with yield", TTs{TryToken, TypeofToken, VarToken, VoidToken, WhileToken, WithToken, YieldToken}},

		{"/*co\nm\u2028m/*ent*/ //co//mment\u2029//comment", TTs{MultiLineCommentToken, SingleLineCommentToken, LineTerminatorToken, SingleLineCommentToken}},
		{"<!-", TTs{LtToken, NotToken, SubToken}},
		{"1<!--2\n", TTs{NumericToken, SingleLineCommentToken, LineTerminatorToken}},
		{"x=y-->10\n", TTs{IdentifierToken, EqToken, IdentifierToken, DecrToken, GtToken, NumericToken, LineTerminatorToken}},
		{"  /*comment*/ -->nothing\n", TTs{SingleLineCommentToken, DecrToken, GtToken, IdentifierToken, LineTerminatorToken}},
		{"1 /*comment\nmultiline*/ -->nothing\n", TTs{NumericToken, MultiLineCommentToken, SingleLineCommentToken, LineTerminatorToken}},
		{"$ _\u200C \\u2000 \u200C", TTs{IdentifierToken, IdentifierToken, IdentifierToken, ErrorToken}},
		{">>>=>>>>=", TTs{GtGtGtEqToken, GtGtGtToken, GtEqToken}},
		{"1/", TTs{NumericToken, DivToken}},
		{"1/=", TTs{NumericToken, DivEqToken}},
		{"010xF", TTs{NumericToken, NumericToken, IdentifierToken}},
		{"50e+-0", TTs{NumericToken, IdentifierToken, AddToken, SubToken, NumericToken}},
		{"'str\\i\\'ng'", TTs{StringToken}},
		{"'str\\\\'abc", TTs{StringToken, IdentifierToken}},
		{"'str\\\ni\\\\u00A0ng'", TTs{StringToken}},

		{"0b0101 0o0707 0b17", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"`template`", TTs{TemplateToken}},
		{"`a${x+y}b`", TTs{TemplateStartToken, IdentifierToken, AddToken, IdentifierToken, TemplateEndToken}},
		{"`temp\nlate`", TTs{TemplateToken}},
		{"`outer${{x: 10}}bar${ raw`nested${2}endnest` }end`", TTs{TemplateStartToken, OpenBraceToken, IdentifierToken, ColonToken, NumericToken, CloseBraceToken, TemplateMiddleToken, IdentifierToken, TemplateStartToken, NumericToken, TemplateEndToken, TemplateEndToken}},
		{"`tmpl ${ a ? '' : `tmpl2 ${b ? 'b' : 'c'}` }`", TTs{TemplateStartToken, IdentifierToken, QuestionToken, StringToken, ColonToken, TemplateStartToken, IdentifierToken, QuestionToken, StringToken, ColonToken, StringToken, TemplateEndToken, TemplateEndToken}},

		// early endings
		{"'string", TTs{ErrorToken}},
		{"'\n", TTs{ErrorToken}},
		{"'\u2028", TTs{ErrorToken}},
		{"'str\\\U00100000ing\\0'", TTs{StringToken}},
		{"'strin\\00g'", TTs{StringToken}},
		{"/*comment", TTs{SingleLineCommentToken}},
		{"a=/regexp", TTs{IdentifierToken, EqToken, DivToken, IdentifierToken}},
		{"\\u002", TTs{ErrorToken}},
		{"`template", TTs{TemplateToken}},
		{"`template${x}template", TTs{TemplateStartToken, IdentifierToken, TemplateEndToken}},

		// null characters
		{"'string\x00'return", TTs{StringToken, ReturnToken}},
		{"//comment\x00comment\nreturn", TTs{SingleLineCommentToken, LineTerminatorToken, ReturnToken}},
		{"/*comment\x00*/return", TTs{SingleLineCommentToken, ReturnToken}},
		{"`template\x00`return", TTs{TemplateToken, ReturnToken}},
		{"`template\\\x00`return", TTs{TemplateToken, ReturnToken}},

		// coverage
		{"Ø a〉", TTs{IdentifierToken, IdentifierToken, ErrorToken}},
		{"0xg 0.f", TTs{NumericToken, IdentifierToken, NumericToken, DotToken, IdentifierToken}},
		{"0bg 0og", TTs{NumericToken, IdentifierToken, NumericToken, IdentifierToken}},
		{"\u00A0\uFEFF\u2000", TTs{}},
		{"\u2028\u2029", TTs{LineTerminatorToken}},
		{"\\u0029ident", TTs{IdentifierToken}},
		{"\\u{0029FEF}ident", TTs{IdentifierToken}},
		{"\\u{}", TTs{ErrorToken}},
		{"\\ugident", TTs{ErrorToken}},
		{"'str\u2028ing'", TTs{ErrorToken}},
		{"a=/\\\n", TTs{IdentifierToken, EqToken, DivToken, ErrorToken}},
		{"a=/x\n", TTs{IdentifierToken, EqToken, DivToken, IdentifierToken, LineTerminatorToken}},
		{"`\\``", TTs{TemplateToken}},
		{"`\\${ 1 }`", TTs{TemplateToken}},
		{"`\\\r\n`", TTs{TemplateToken}},

		// go fuzz
		{"`", TTs{TemplateToken}},
	}

	for _, tt := range tokenTests {
		t.Run(tt.js, func(t *testing.T) {
			l := NewLexer(bytes.NewBufferString(tt.js))
			i := 0
			tokens := []TokenType{}
			for {
				token, _ := l.Next()
				if token == ErrorToken {
					if l.Err() != io.EOF {
						tokens = append(tokens, token)
					}
					break
				} else if token == WhitespaceToken {
					continue
				}
				tokens = append(tokens, token)
				i++
			}
			test.T(t, tokens, tt.expected, "token types must match")
		})
	}

	test.That(t, IsPunctuator(CommaToken))
	test.That(t, IsPunctuator(GtGtEqToken))
	test.That(t, !IsPunctuator(WhileToken))
	test.That(t, !IsOperator(CommaToken))
	test.That(t, IsOperator(GtGtEqToken))
	test.That(t, !IsOperator(WhileToken))
	test.That(t, !IsIdentifier(CommaToken))
	test.That(t, !IsIdentifier(GtGtEqToken))
	test.That(t, IsIdentifier(WhileToken))

	// coverage
	for _, start := range []int{0, 0x1000, 0x3000, 0x4000, 0x8000} {
		for i := start; ; i++ {
			if TokenType(i).String() == fmt.Sprintf("Invalid(%d)", i) {
				break
			}
		}
	}
}

func TestRegExp(t *testing.T) {
	var tokenTests = []struct {
		js       string
		expected []TokenType
	}{
		{"a = /[a-z/]/g", TTs{IdentifierToken, EqToken, RegExpToken}},
		{"a=/=/g1", TTs{IdentifierToken, EqToken, RegExpToken}},
		{"a = /'\\\\/\n", TTs{IdentifierToken, EqToken, RegExpToken, LineTerminatorToken}},
		{"a=/\\//g1", TTs{IdentifierToken, EqToken, RegExpToken}},
		{"new RegExp(a + /\\d{1,2}/.source)", TTs{NewToken, IdentifierToken, OpenParenToken, IdentifierToken, AddToken, RegExpToken, DotToken, IdentifierToken, CloseParenToken}},
		{"a=/regexp\x00/;return", TTs{IdentifierToken, EqToken, RegExpToken, SemicolonToken, ReturnToken}},
		{"a=/regexp\\\x00/;return", TTs{IdentifierToken, EqToken, RegExpToken, SemicolonToken, ReturnToken}},
		{"a=/x/\u200C\u3009", TTs{IdentifierToken, EqToken, RegExpToken, ErrorToken}},
		{"a=/end", TTs{IdentifierToken, EqToken, DivToken, IdentifierToken}},
		{"a=/\\\nend", TTs{IdentifierToken, EqToken, DivToken, ErrorToken}},
	}

	for _, tt := range tokenTests {
		t.Run(tt.js, func(t *testing.T) {
			l := NewLexer(bytes.NewBufferString(tt.js))
			i := 0
			tokens := []TokenType{}
			for {
				token, _ := l.Next()
				if token == DivToken || token == DivEqToken {
					token, _ = l.RegExp()
				}
				if token == ErrorToken {
					if l.Err() != io.EOF {
						tokens = append(tokens, token)
					}
					break
				} else if token == WhitespaceToken {
					continue
				}
				tokens = append(tokens, token)
				i++
			}
			test.T(t, tokens, tt.expected, "token types must match")
		})
	}

	token, _ := NewLexer(bytes.NewBufferString("")).RegExp()
	test.T(t, token, ErrorToken)
}

func TestOffset(t *testing.T) {
	l := NewLexer(bytes.NewBufferString(`var i=5;`))
	test.T(t, l.Offset(), 0)
	_, _ = l.Next()
	test.T(t, l.Offset(), 3) // var
	_, _ = l.Next()
	test.T(t, l.Offset(), 4) // ws
	_, _ = l.Next()
	test.T(t, l.Offset(), 5) // i
	_, _ = l.Next()
	test.T(t, l.Offset(), 6) // =
	_, _ = l.Next()
	test.T(t, l.Offset(), 7) // 5
	_, _ = l.Next()
	test.T(t, l.Offset(), 8) // ;
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("var x = 'lorem ipsum';"))
	out := ""
	for {
		tt, data := l.Next()
		if tt == ErrorToken {
			break
		}
		out += string(data)
	}
	fmt.Println(out)
	// Output: var x = 'lorem ipsum';
}
