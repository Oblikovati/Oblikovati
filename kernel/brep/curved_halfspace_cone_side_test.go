// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"slices"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A frustum cut by a plane PARALLEL to its axis, shallow enough to clip every cross-section
// (|D| < bottom radius), keeps an exact arc-band cone face — a flat milled the full side of a
// frustum (Oblikovati/Oblikovati#1372). The result must be a watertight solid whose curved face is
// still a geom.Cone (not faceted), with the hyperbola arms welded into the lid.
func TestConeSideHalfSpaceArcBand(t *testing.T) {
	t.Parallel()
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6) // tanα = 0.3, bottom r=3, top r=6
	plane, err := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0))      // x=2 ∥ z axis, |D|=2 < bottom r=3
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	cones, _, _ := faceTypeCounts(t, res)
	if cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the arc-band side stays analytic)", cones)
	}
	if slices.ContainsFunc(res.Faces(), hasHyperbolaEdge) {
		return
	}
	t.Error("no face carries a hyperbolic edge — the cut was not imprinted as a hyperbola")
}

// A plane parallel to the axis but clear of the whole frustum (|D| ≥ top radius) keeps the solid
// whole on the axis side and empty on the far side.
func TestConeSideHalfSpaceClears(t *testing.T) {
	t.Parallel()
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	keep, _ := geom.NewPlane(math.P3(7, 0, 0), math.V3(1, 0, 0)) // axis at -7 (negative side): whole kept
	res, err := HalfSpaceCut(frustum, keep)
	if err != nil {
		t.Fatalf("HalfSpaceCut(keep): %v", err)
	}
	if len(res.Faces()) != len(frustum.Faces()) {
		t.Errorf("clearing plane changed the face count: %d vs %d", len(res.Faces()), len(frustum.Faces()))
	}
	drop, _ := geom.NewPlane(math.P3(-7, 0, 0), math.V3(1, 0, 0)) // axis at +7 (positive side): emptied
	empty, err := HalfSpaceCut(frustum, drop)
	if err != nil {
		t.Fatalf("HalfSpaceCut(drop): %v", err)
	}
	if len(empty.Faces()) != 0 {
		t.Errorf("plane on the far side should empty the solid, got %d faces", len(empty.Faces()))
	}
}

// The vertex-inside-band arrangement (the flat fades before the bottom rim, bottom r ≤ |D| < top r),
// keeping the axis side, leaves an ANNULUS cone face: the intact small rim plus a notched top loop
// whose tongue is bitten out down to the hyperbola vertex (Oblikovati/Oblikovati#1374).
func TestConeSideHalfSpaceVertexInsideAnnulus(t *testing.T) {
	t.Parallel()
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(4, 0, 0), math.V3(1, 0, 0)) // |D|=4: cuts the top (r=6) but not the bottom (r=3)
	res, err := HalfSpaceCut(frustum, plane)                      // keeps x<4, the axis side: apex kept → annulus
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	cones, _, _ := faceTypeCounts(t, res)
	if cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the notched annulus stays analytic)", cones)
	}
	if !anyFaceHasHyperbola(res) {
		t.Error("no face carries a hyperbolic edge — the notch was not imprinted as a hyperbola")
	}
}

// The same arrangement keeping the FAR side drops the small rim entirely and leaves a single TONGUE
// cone face narrowing to the hyperbola vertex (Oblikovati/Oblikovati#1374).
func TestConeSideHalfSpaceVertexInsideTongue(t *testing.T) {
	t.Parallel()
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(4, 0, 0), math.V3(-1, 0, 0)) // keeps x>4, the far side: apex dropped → tongue
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	cones, _, planes := faceTypeCounts(t, res)
	if cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the tongue stays analytic)", cones)
	}
	if planes != 2 { // top-cap minor segment + lid only; the small cap is dropped
		t.Errorf("result has %d planar faces, want 2 (top-cap segment + lid, no small cap)", planes)
	}
	if !anyFaceHasHyperbola(res) {
		t.Error("no face carries a hyperbolic edge — the tongue was not imprinted as a hyperbola")
	}
}

// anyFaceHasHyperbola reports whether any face of the body carries a hyperbolic edge.
func anyFaceHasHyperbola(b *topo.Body) bool {
	return slices.ContainsFunc(b.Faces(), hasHyperbolaEdge)
}

// mustFrustum builds a frustum solid (bottom radius < top radius) or fails the test.
func mustFrustum(t *testing.T, bottom, top math.Point3, rBot, rTop float64) *topo.Body {
	t.Helper()
	b, err := SolidCylinderCone(bottom, top, rBot, rTop, "frustum")
	if err != nil {
		t.Fatalf("SolidCylinderCone: %v", err)
	}
	return b
}

// hasHyperbolaEdge reports whether any edge of a face stores a hyperbolic arc.
func hasHyperbolaEdge(f *topo.Face) bool {
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if _, ok := u.Edge().Geometry().(geom.HyperbolicArc); ok {
				return true
			}
		}
	}
	return false
}
