// SPDX-License-Identifier: GPL-2.0-only

package complete

import (
	"slices"
	"testing"
)

// methodsFixture is a small but representative slice of the dotted wire-method names the host
// publishes, exercising multiple groups and a two-level namespace.
var methodsFixture = []string{
	"documents.activate",
	"documents.close",
	"sketch.rectangle",
	"sketch.circle",
	"sketch.line.add",
}

// texts projects candidates to their insert text for order-sensitive assertions.
func texts(cs []Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Text
	}
	return out
}

func TestSuggestApiNamespaceListsChildren(t *testing.T) {
	e := New(methodsFixture)
	// `oblikovati.` (caret after the dot) lists the top-level groups.
	got, ctx := e.Suggest("oblikovati.", 11)
	if want := []string{"documents", "sketch"}; !equal(texts(got), want) {
		t.Fatalf("top-level groups = %v, want %v", texts(got), want)
	}
	if ctx.ReplaceStart != 11 {
		t.Errorf("ReplaceStart = %d, want 11 (after the dot)", ctx.ReplaceStart)
	}
}

func TestSuggestApiPrefixFiltersAndDetails(t *testing.T) {
	e := New(methodsFixture)
	line := "oblikovati.sketch.rec"
	got, ctx := e.Suggest(line, len(line))
	if want := []string{"rectangle"}; !equal(texts(got), want) {
		t.Fatalf("filtered methods = %v, want %v", texts(got), want)
	}
	if got[0].Kind != KindMethod {
		t.Errorf("kind = %v, want KindMethod", got[0].Kind)
	}
	if got[0].Detail != "oblikovati.sketch.rectangle" {
		t.Errorf("detail = %q, want full dotted reference", got[0].Detail)
	}
	if ctx.ReplaceStart != len("oblikovati.sketch.") {
		t.Errorf("ReplaceStart = %d, want start of 'rec'", ctx.ReplaceStart)
	}
}

func TestNamespaceNodeTaggedAsModule(t *testing.T) {
	e := New(methodsFixture)
	line := "oblikovati.sketch.li"
	got, _ := e.Suggest(line, len(line))
	if want := []string{"line"}; !equal(texts(got), want) {
		t.Fatalf("got %v, want %v", texts(got), want)
	}
	if got[0].Kind != KindModule {
		t.Errorf("kind = %v, want KindModule (line has a child .add)", got[0].Kind)
	}
}

func TestBarePrefixCompletesKeywordsAndBuiltins(t *testing.T) {
	e := New(methodsFixture)
	got, ctx := e.Suggest("  pr", 4)
	names := texts(got)
	if !slices.Contains(names, "print") {
		t.Errorf("bare 'pr' = %v, want it to include builtin 'print'", names)
	}
	if ctx.ReplaceStart != 2 {
		t.Errorf("ReplaceStart = %d, want 2 (start of 'pr')", ctx.ReplaceStart)
	}
	// 'lo' should surface the keyword 'local'.
	got2, _ := e.Suggest("lo", 2)
	if !slices.Contains(texts(got2), "local") {
		t.Errorf("bare 'lo' = %v, want keyword 'local'", texts(got2))
	}
}

func TestUnknownNamespaceYieldsNothing(t *testing.T) {
	e := New(methodsFixture)
	if got, _ := e.Suggest("oblikovati.nope.x", len("oblikovati.nope.x")); got != nil {
		t.Errorf("unknown namespace returned %v, want nil", texts(got))
	}
	// A non-API root is not completed against the API tree.
	if got, _ := e.Suggest("other.thing", len("other.thing")); got != nil {
		t.Errorf("non-API chain returned %v, want nil", texts(got))
	}
}

func TestParseChainHandlesConcatOperator(t *testing.T) {
	// `a..b` is string concat, not a member access — the chain must not span the `..`.
	chain, _ := parseChain([]rune("a..b"), 4)
	if len(chain) != 1 || chain[0] != "b" {
		t.Errorf("chain across `..` = %v, want just {\"b\"}", chain)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
