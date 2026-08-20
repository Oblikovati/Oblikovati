// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// lContourProfile is an open L cross-section (out 1 cm, up 1 cm) — one right-angle corner, the
// simplest contour flange that carries a bend.
func lContourProfile() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	p0 := s.Points().Add(gmath.P2(0, 0))
	p1 := s.Points().Add(gmath.P2(1, 0))
	p2 := s.Points().Add(gmath.P2(1, 1))
	s.Lines().Add(p0, p1)
	s.Lines().Add(p1, p2)
	return s
}

// TestBendsReportsContourFlangeCorners is the #2076 integration regression: a contour flange's
// rounded corners must develop as real bends in the flat pattern. Before the feature implemented
// BendLineage they contributed nothing, so the developed blank was short by every corner's bend
// allowance and any DXF cut from it was undersized. The corner defers to the rule's radius, so the
// developed allowance is the rule's own bend allowance for a 90° corner.
func TestBendsReportsContourFlangeCorners(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addSquareFace(d, 4)

	edge := topXEdge(t, d.Features().Result()[0])
	pf := feature.NewSheetMetalContourFlangeFeatures(d.Features()).Add(&feature.SheetMetalContourFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Profile: lContourProfile(),
	})
	d.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("contour flange unhealthy: %s", pf.Health().Reason)
	}

	bends := d.Bends()
	if len(bends) != 1 {
		t.Fatalf("contour flange Bends len = %d, want 1 (its single corner)", len(bends))
	}
	b := bends[0]
	if b.Feature != pf.Name() {
		t.Errorf("bend feature = %q, want %q", b.Feature, pf.Name())
	}
	if math.Abs(b.Angle-math.Pi/2) > 1e-9 {
		t.Errorf("corner bend angle = %g, want π/2", b.Angle)
	}
	rule := d.SheetMetal()
	if b.Radius != rule.BendRadius() {
		t.Errorf("corner bend radius = %g, want the rule default %g (deferred)", b.Radius, rule.BendRadius())
	}
	if want := rule.Unfold().BendAllowance(b.Angle, b.Radius, b.Thickness); b.Allowance != want || want <= 0 {
		t.Errorf("corner allowance = %g, want the rule's %g (>0) — the flat was short by it", b.Allowance, want)
	}
	if got := d.TotalBendAllowance(); math.Abs(got-b.Allowance) > 1e-12 {
		t.Errorf("TotalBendAllowance = %g, want the corner's allowance %g", got, b.Allowance)
	}
}
