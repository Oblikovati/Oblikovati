// SPDX-License-Identifier: GPL-2.0-only

package editor

import (
	"strings"
	"testing"
)

// caretInside builds a model whose caret sits at the '|' marker in src (single line), for
// readable signature-help cases.
func caretInside(src string) *Model {
	col := strings.IndexRune(src, '|')
	m := New(strings.Replace(src, "|", "", 1))
	m.SetCaret(pos(0, col), false)
	return m
}

func TestEnclosingCallInsideTableCall(t *testing.T) {
	m := caretInside("oblikovati.sketch.rectangle{ width = |40 }")
	chain, ok := m.EnclosingCall()
	if !ok {
		t.Fatal("expected to be inside a call")
	}
	if got := strings.Join(chain, "."); got != "oblikovati.sketch.rectangle" {
		t.Errorf("chain = %q, want oblikovati.sketch.rectangle", got)
	}
}

func TestEnclosingCallParensAndNesting(t *testing.T) {
	// The caret is inside the inner print(...) call, not the outer pcall.
	m := caretInside("pcall(function() print(|x) end)")
	chain, ok := m.EnclosingCall()
	if !ok || strings.Join(chain, ".") != "print" {
		t.Fatalf("chain = %v (ok=%v), want [print]", chain, ok)
	}
}

func TestEnclosingCallNotInCall(t *testing.T) {
	m := caretInside("local x = 1|0")
	if _, ok := m.EnclosingCall(); ok {
		t.Error("caret not inside any call should report false")
	}
}

func TestEnclosingCallClosedCallIgnored(t *testing.T) {
	// The call before the caret is already closed, so the caret is not inside it.
	m := caretInside("foo() |bar")
	if _, ok := m.EnclosingCall(); ok {
		t.Error("a closed call should not be reported as enclosing")
	}
}

func TestEnclosingCallAnonymousBracket(t *testing.T) {
	m := caretInside("y = (|1 + 2)")
	if _, ok := m.EnclosingCall(); ok {
		t.Error("a grouping '(' with no name before it is not a call")
	}
}

func TestMethodChainAtCoversHoveredSegment(t *testing.T) {
	m := New("x = oblikovati.documents.create{}")
	// Hover within "create" (the last segment).
	chain, ok := m.MethodChainAt(pos(0, 27))
	if !ok || strings.Join(chain, ".") != "oblikovati.documents.create" {
		t.Fatalf("chain over 'create' = %v (ok=%v)", chain, ok)
	}
	// Hover within "documents" yields the chain up to that segment only.
	chain2, _ := m.MethodChainAt(pos(0, 18))
	if strings.Join(chain2, ".") != "oblikovati.documents" {
		t.Errorf("chain over 'documents' = %v, want oblikovati.documents", chain2)
	}
}

func TestMethodChainAtOffWord(t *testing.T) {
	m := New("a = 1")
	if _, ok := m.MethodChainAt(pos(0, 2)); ok { // on the '='/space, not a word
		t.Error("hover off a word should yield ok=false")
	}
}
