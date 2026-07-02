// SPDX-License-Identifier: GPL-2.0-only

package part21

import (
	"strings"
	"testing"
)

// lexAll drains the lexer into a slice, failing the test on a lex error.
func lexAll(t *testing.T, src string) []Token {
	t.Helper()
	lx := newLexer([]byte(src))
	var toks []Token
	for {
		tok, err := lx.next()
		if err != nil {
			t.Fatalf("lex %q: %v", src, err)
		}
		toks = append(toks, tok)
		if tok.Kind == TokEOF {
			return toks
		}
	}
}

func TestLexKeywordAndRef(t *testing.T) {
	toks := lexAll(t, "CARTESIAN_POINT #42")
	if toks[0].Kind != TokKeyword || toks[0].Text != "CARTESIAN_POINT" {
		t.Errorf("first token = %v, want keyword CARTESIAN_POINT", toks[0])
	}
	if toks[1].Kind != TokRef || toks[1].Text != "#42" {
		t.Errorf("second token = %v, want ref #42", toks[1])
	}
}

func TestLexStringWithDoubledQuote(t *testing.T) {
	toks := lexAll(t, "'O''Brien'")
	if toks[0].Kind != TokString || toks[0].Text != "O'Brien" {
		t.Errorf("string token = %v, want decoded O'Brien", toks[0])
	}
}

func TestLexEnum(t *testing.T) {
	toks := lexAll(t, ".T. .STEEL.")
	if toks[0].Kind != TokEnum || toks[0].Text != ".T." {
		t.Errorf("enum token = %v, want .T.", toks[0])
	}
	if toks[1].Kind != TokEnum || toks[1].Text != ".STEEL." {
		t.Errorf("enum token = %v, want .STEEL.", toks[1])
	}
}

func TestLexRealVariants(t *testing.T) {
	cases := map[string]TokenKind{
		"3.14": TokReal, "-2.": TokReal, "1.5E-3": TokReal, "6.022e23": TokReal,
		"42": TokInt, "-7": TokInt,
	}
	for src, want := range cases {
		toks := lexAll(t, src)
		if toks[0].Kind != want {
			t.Errorf("lex %q kind = %s, want %s", src, toks[0].Kind, want)
		}
	}
}

func TestLexSkipsComment(t *testing.T) {
	toks := lexAll(t, "/* a comment */ FOO")
	if toks[0].Kind != TokKeyword || toks[0].Text != "FOO" {
		t.Errorf("after comment got %v, want keyword FOO", toks[0])
	}
}

func TestLexNestedListPunctuation(t *testing.T) {
	toks := lexAll(t, "(1,(2,3))")
	kinds := []TokenKind{TokLParen, TokInt, TokComma, TokLParen, TokInt, TokComma, TokInt, TokRParen, TokRParen, TokEOF}
	for i, want := range kinds {
		if toks[i].Kind != want {
			t.Errorf("token %d = %s, want %s", i, toks[i].Kind, want)
		}
	}
}

func TestLexUnterminatedStringErrors(t *testing.T) {
	lx := newLexer([]byte("'oops"))
	if _, err := lx.next(); err == nil {
		t.Error("unterminated string should error")
	}
}

func TestLexUnterminatedCommentErrors(t *testing.T) {
	lx := newLexer([]byte("/* never closed"))
	if _, err := lx.next(); err == nil {
		t.Error("unterminated comment should error")
	}
}

func TestLexErrorPositionTracked(t *testing.T) {
	lx := newLexer([]byte("FOO\n  'bad"))
	_, _ = lx.next()    // FOO
	_, err := lx.next() // unterminated string at line 2
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "2:3") {
		t.Errorf("error %q should cite position 2:3", got)
	}
}
