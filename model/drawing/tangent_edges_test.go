// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
)

// filletedBoxBox returns a plain (un-filleted) box — every edge a 90° crease, no tangent edges.
func filletedBoxBox(t *testing.T) *topo.Body {
	t.Helper()
	return subd.ToBody(subd.Box(4, 4, 4), "box")
}

// isoBase adds an isometric base view named "ISO" (so a filleted edge's bend lines project distinctly)
// and returns it.
func isoBase(t *testing.T, views *DrawingViews) *DrawingView {
	t.Helper()
	v, err := views.AddBase(BaseViewSpec{Name: "ISO", Orientation: types.BaseViewIso, Scale: 1, CenterX: 120, CenterY: 120})
	if err != nil {
		t.Fatalf("AddBase ISO: %v", err)
	}
	return v
}

// TestTangentEdgeKeysFindsFilletRunouts: filleting a box edge introduces two smooth (tangent) bend
// lines where the fillet cylinder meets each flat; the sharp box edges stay non-tangent.
func TestTangentEdgeKeysFindsFilletRunouts(t *testing.T) {
	body, _ := filletedBoxBend(t, 0.5)
	tangent := tangentEdgeKeys(body)
	if len(tangent) < 2 {
		t.Fatalf("tangent edges = %d, want ≥2 (the fillet's two bend lines)", len(tangent))
	}
	// A plain box has no smooth edges at all — every edge is a 90° crease.
	plain := tangentEdgeKeys(filletedBoxBox(t))
	if len(plain) != 0 {
		t.Errorf("plain box tangent edges = %d, want 0", len(plain))
	}
}

// TestTangentDisplayTogglesCurves: a base view of a filleted body carries tangent-tagged curves by
// default, and hiding tangent edges drops exactly those, leaving strictly fewer curves and none
// tagged tangent — while the default (shown) count is unchanged (#1984).
func TestTangentDisplayTogglesCurves(t *testing.T) {
	body, _ := filletedBoxBend(t, 0.5)
	c := NewContent()
	c.SetBodyResolver(fakeBodyResolver{body: body})
	c.SetModelReference("fillet.opd")
	v := isoBase(t, c.Sheets().Active().Views())

	onCount := v.CurveCount()
	tangentOn := countTangentCurves(v)
	if tangentOn == 0 {
		t.Fatal("no tangent curves in the default view — the fillet's bend lines should project")
	}

	if err := c.Sheets().Active().Views().SetDisplayTangentEdges(v.Name(), false); err != nil {
		t.Fatalf("SetDisplayTangentEdges: %v", err)
	}
	offCount := v.CurveCount()
	if offCount >= onCount {
		t.Errorf("tangent-off curves = %d, want strictly fewer than %d (on)", offCount, onCount)
	}
	if got := countTangentCurves(v); got != 0 {
		t.Errorf("tangent-off view still has %d tangent curves, want 0", got)
	}
	if offCount != onCount-tangentOn {
		t.Errorf("tangent-off dropped %d curves, want exactly the %d tangent curves", onCount-offCount, tangentOn)
	}

	// Turning tangent display back on restores the original count (associative re-projection).
	if err := c.Sheets().Active().Views().SetDisplayTangentEdges(v.Name(), true); err != nil {
		t.Fatalf("SetDisplayTangentEdges(true): %v", err)
	}
	if v.CurveCount() != onCount {
		t.Errorf("tangent restored count = %d, want %d", v.CurveCount(), onCount)
	}
}

// countTangentCurves counts a view's curves tagged as tangent edges.
func countTangentCurves(v *DrawingView) int {
	n := 0
	for _, c := range v.Curves() {
		if c.EdgeType() == types.TangentDrawingEdge {
			n++
		}
	}
	return n
}
