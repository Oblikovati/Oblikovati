// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"math"
	"testing"
)

// TestMatrixConcatApply checks that concat composes transforms in row-vector order: a
// translate-then-scale maps a point through the translate first.
func TestMatrixConcatApply(t *testing.T) {
	translate := translateMatrix(1, 2)
	scale := matrix{2, 0, 0, 2, 0, 0}
	m := concat(translate, scale) // apply translate, then scale
	x, y := m.apply(3, 4)
	if x != (3+1)*2 || y != (4+2)*2 {
		t.Errorf("concat(translate,scale).apply(3,4) = (%g,%g), want (8,12)", x, y)
	}
}

// TestParseNumberArrayNotRef pins the regression behind the pushback fix: an array of
// numbers must not be misread as indirect references, which truncated page dictionaries.
func TestParseNumberArrayNotRef(t *testing.T) {
	p := newParser(newLexer([]byte("[0 0 2384 1684]")))
	arr, ok := p.parseValue().(arrayObj)
	if !ok || len(arr) != 4 {
		t.Fatalf("parsed %v (ok=%v), want a 4-element array", arr, ok)
	}
	for i, want := range []float64{0, 0, 2384, 1684} {
		if n, ok := arr[i].(numberObj); !ok || float64(n) != want {
			t.Errorf("element %d = %v, want %g", i, arr[i], want)
		}
	}
}

// TestParseIndirectReference checks the three-token "n g R" form still parses as a ref.
func TestParseIndirectReference(t *testing.T) {
	p := newParser(newLexer([]byte("8 0 R")))
	r, ok := p.parseValue().(refObj)
	if !ok || r.num != 8 || r.gen != 0 {
		t.Fatalf("parsed %v (ok=%v), want refObj{8,0}", r, ok)
	}
}

// TestLexNameHexEscape checks a /Name with a #xx escape decodes (PDF 1.2+ name syntax).
func TestLexNameHexEscape(t *testing.T) {
	tok := newLexer([]byte("/Pa#20ge")).next()
	if tok.kind != tokName || tok.text != "Pa ge" {
		t.Errorf("lexed name = %q (kind %d), want \"Pa ge\"", tok.text, tok.kind)
	}
}

// TestToMM checks the point→millimetre conversion (1 inch = 72 pt = 25.4 mm).
func TestToMM(t *testing.T) {
	if got := toMM(72); math.Abs(got-25.4) > 1e-9 {
		t.Errorf("toMM(72) = %g, want 25.4", got)
	}
}
