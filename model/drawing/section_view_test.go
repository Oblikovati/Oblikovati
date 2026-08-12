// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
)

// curveKinds tallies a view's curves by kind.
func curveKinds(v *DrawingView) (edge, section, hatch int) {
	for _, c := range v.Curves() {
		switch c.Kind() {
		case types.DrawingSectionCurve:
			section++
		case types.DrawingHatchCurve:
			hatch++
		default:
			edge++
		}
	}
	return
}

// TestAddSectionViewCutsAndHatches sections a 2×3×4 box through a horizontal line across the
// middle of a FRONT view, and checks the result carries a bold cut outline and hatch fill (the
// exposed cross-section) plus retained-half edges.
func TestAddSectionViewCutsAndHatches(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	// A horizontal cut line across the front view at its centre (sheet mm).
	minX, _, maxX, _, _ := front.BoundsMM()
	sec, err := views.AddSection(SectionViewSpec{
		Name: "A-A", ParentView: "FRONT", X1: minX - 5, Y1: 100, X2: maxX + 5, Y2: 100, CenterX: 100, CenterY: 220,
	})
	if err != nil {
		t.Fatalf("AddSection: %v", err)
	}
	if sec.Type() != types.DrawingViewSection || sec.BaseViewName() != "FRONT" {
		t.Fatalf("section view = type %v parent %q, want section off FRONT", sec.Type(), sec.BaseViewName())
	}
	edge, section, hatch := curveKinds(sec)
	if section == 0 {
		t.Error("section view has no cut-outline curves")
	}
	if hatch == 0 {
		t.Error("section view has no hatch fill")
	}
	if edge == 0 {
		t.Error("section view has no retained-half edges")
	}
}

// TestSectionOptionsCarryThroughAddSection checks the reverse/depth/type options survive
// AddSection onto the view's accessors, that the millimetre depth converts to model centimetres
// and back, and that a limited-depth cut keeps fewer edges than a full one (#1982).
func TestSectionOptionsCarryThroughAddSection(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	front, _ := views.ByName("FRONT")
	minX, _, maxX, _, _ := front.BoundsMM()
	line := func(name string) SectionViewSpec {
		return SectionViewSpec{Name: name, ParentView: "FRONT", X1: minX - 5, Y1: 100, X2: maxX + 5, Y2: 100, CenterX: 100, CenterY: 220}
	}

	full, err := views.AddSection(line("FULL"))
	if err != nil {
		t.Fatalf("AddSection full: %v", err)
	}
	spec := line("A-A")
	spec.Reverse, spec.Depth, spec.Type = true, 5, types.HalfSectionView
	sec, err := views.AddSection(spec)
	if err != nil {
		t.Fatalf("AddSection options: %v", err)
	}
	if !sec.SectionReverse() || sec.SectionType() != types.HalfSectionView {
		t.Errorf("options lost: reverse=%v type=%v", sec.SectionReverse(), sec.SectionType())
	}
	if got := sec.SectionDepthMM(); got != 5 {
		t.Errorf("SectionDepthMM round-trip = %g, want 5", got)
	}
	// The box is 40 mm deep along the cut; a 5 mm slab retains strictly fewer edges than the
	// full through-cut (the far wall is clipped away).
	edgeFull, _, _ := curveKinds(full)
	edgeLimited, _, _ := curveKinds(sec)
	if edgeLimited >= edgeFull {
		t.Errorf("limited-depth edges = %d, want fewer than full %d", edgeLimited, edgeFull)
	}
}

// TestSectionUnknownTypeRejected checks a bad section type spelling is rejected at the router.
func TestSectionUnknownTypeRejected(t *testing.T) {
	if _, ok := types.ParseSectionViewType("octant"); ok {
		t.Error("ParseSectionViewType(octant) accepted, want rejected")
	}
}

// TestSectionRejectsNonBaseParent checks a section can only cut a base view.
func TestSectionRejectsNonBaseParent(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	if _, err := views.AddSection(SectionViewSpec{ParentView: "NOPE", X1: 0, Y1: 0, X2: 10, Y2: 0}); err == nil {
		t.Error("section off a missing parent = ok, want error")
	}
}

// TestSectionRecipeRoundTrip checks a section view's type, parent and cut line survive
// persistence (its curves re-project on open).
func TestSectionRecipeRoundTrip(t *testing.T) {
	c := drawingWithBox(t)
	views := c.Sheets().Active().Views()
	frontBase(t, views)
	if _, err := views.AddSection(SectionViewSpec{Name: "A-A", ParentView: "FRONT", X1: 80, Y1: 100, X2: 120, Y2: 100, CenterX: 100, CenterY: 220}); err != nil {
		t.Fatalf("AddSection: %v", err)
	}
	v, ok := reopen(t, c).Sheets().Active().Views().ByName("A-A")
	if !ok || v.Type() != types.DrawingViewSection {
		t.Fatalf("restored section view missing/mistyped: ok=%v", ok)
	}
	x1, _, _, _ := v.SectionLineMM()
	if x1 != 80 {
		t.Errorf("restored section line x1 = %g, want 80", x1)
	}
	if _, section, _ := curveKinds(v); section == 0 {
		t.Error("restored section view re-projected no cut outline")
	}
}
