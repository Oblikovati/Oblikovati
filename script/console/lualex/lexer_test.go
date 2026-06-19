// SPDX-License-Identifier: GPL-2.0-only

package lualex

import "testing"

// span is the readable form of a token for table assertions: the kind and the source text it
// covers, so a test reads as "this text is a keyword" rather than column arithmetic.
type span struct {
	kind Kind
	text string
}

// spans tokenizes one line from a fresh state and projects each token to its (kind, text).
func spans(t *testing.T, line string) ([]span, State) {
	t.Helper()
	return spansFrom(line, State{})
}

// spansFrom tokenizes line carrying the incoming state and projects each token to (kind, text).
func spansFrom(line string, in State) ([]span, State) {
	toks, st := TokenizeLine(line, in)
	r := []rune(line)
	out := make([]span, len(toks))
	for i, tk := range toks {
		out[i] = span{tk.Kind, string(r[tk.Start:tk.End])}
	}
	return out, st
}

func TestKeywordBuiltinIdentClassification(t *testing.T) {
	got, _ := spans(t, "local x = print(foo)")
	want := []span{
		{KindKeyword, "local"}, {KindIdent, "x"}, {KindOperator, "="},
		{KindBuiltin, "print"}, {KindOperator, "("}, {KindIdent, "foo"},
		{KindOperator, ")"},
	}
	assertSpans(t, got, want)
}

func TestStringsAndEscapes(t *testing.T) {
	got, _ := spans(t, `s = "a\"b" .. 'c'`)
	want := []span{
		{KindIdent, "s"}, {KindOperator, "="}, {KindString, `"a\"b"`},
		{KindOperator, "."}, {KindOperator, "."}, {KindString, "'c'"},
	}
	assertSpans(t, got, want)
}

func TestNumbersHexAndFloatExp(t *testing.T) {
	got, _ := spans(t, "n = 0xFF + 3.14e-2")
	want := []span{
		{KindIdent, "n"}, {KindOperator, "="}, {KindNumber, "0xFF"},
		{KindOperator, "+"}, {KindNumber, "3.14e-2"},
	}
	assertSpans(t, got, want)
}

func TestLineComment(t *testing.T) {
	got, _ := spans(t, "x = 1 -- trailing note")
	if len(got) == 0 || got[len(got)-1] != (span{KindComment, "-- trailing note"}) {
		t.Fatalf("last token = %+v, want trailing line comment", got)
	}
}

func TestLongCommentSpansLines(t *testing.T) {
	// Open on line 1, body on line 2, close on line 3 — the State must thread `inLong`.
	l1, s1 := spans(t, "--[==[ open")
	if !s1.InLong || s1.Level != 2 || !s1.IsComment {
		t.Fatalf("after line 1 state = %+v, want long comment level 2 open", s1)
	}
	if l1[0].kind != KindComment {
		t.Fatalf("line 1 token kind = %v, want comment", l1[0].kind)
	}
	_, s2 := TokenizeLine("still inside", s1)
	if !s2.InLong {
		t.Fatalf("after line 2 state = %+v, want still open", s2)
	}
	l3, s3 := closeLine(t, "done ]==] after", s2)
	if s3.InLong {
		t.Errorf("after closer state = %+v, want closed", s3)
	}
	// Text after the closer is tokenized normally.
	if last := l3[len(l3)-1]; last.kind != KindIdent || last.text != "after" {
		t.Errorf("post-closer token = %+v, want ident 'after'", last)
	}
}

func TestLongStringLiteral(t *testing.T) {
	got, st := spans(t, "s = [[raw \"text\"]]")
	if st.InLong {
		t.Errorf("single-line long string should close: state %+v", st)
	}
	if last := got[len(got)-1]; last.kind != KindString || last.text != `[[raw "text"]]` {
		t.Errorf("long string token = %+v", last)
	}
}

// closeLine tokenizes a continuation line carrying state in, returning its spans and out-state.
func closeLine(t *testing.T, line string, in State) ([]span, State) {
	t.Helper()
	return spansFrom(line, in)
}

func assertSpans(t *testing.T, got, want []span) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("token count = %d, want %d\n got=%+v\nwant=%+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
