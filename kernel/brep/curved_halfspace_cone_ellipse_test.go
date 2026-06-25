// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// An oblique plane tilted steeper than the cone's generators cuts the frustum side in a closed
// ellipse wholly within the band; the kept side is an exact cone band (circle rim + elliptical rim)
// capped by a planar elliptical lid, watertight (Oblikovati/Oblikovati#1375).
func TestConeSideHalfSpaceEllipse(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, err := geom.NewPlane(math.P3(0, 0, 5), math.V3(0.5, 0, 0.866)) // tilt 60° > α≈16.7° → ellipse
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
		t.Errorf("result has %d cone faces, want exactly 1 (the ellipse band stays analytic)", cones)
	}
	if !anyFaceHasEllipse(res) {
		t.Error("no face carries an elliptical edge — the cut was not imprinted as an ellipse")
	}
}

// An oblique plane positioned so its ellipse straddles the top rim (some of the section above z=10) is
// the clips-rim arrangement: the kept band's UPPER boundary is the cut ellipse arc PLUS the surviving
// top-rim arc, and the planar lid is bounded by that ellipse arc and the top cap's chord. The (u,v)
// arrangement split builds it exactly — one analytic cone band, watertight, no CSG fallback
// (Oblikovati/Oblikovati#1375).
func TestConeSideHalfSpaceEllipseClipsRim(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(0, 0, 9.5), math.V3(0.5, 0, 0.866)) // ellipse crosses the top rim
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("clips-rim cut should now be handled exactly, got %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the clipped band stays analytic)", cones)
	}
	if !anyFaceHasEllipse(res) {
		t.Error("no face carries an elliptical edge — the rim-clipping ellipse was not imprinted")
	}
}

// The same rim-straddling ellipse kept from the OTHER side is the non-wrapping TONGUE: the kept region
// survives only over the single azimuth span where the section sits inside the band, pinching to a point
// at each end (where the section meets the top rim). The (u,v) split's coneSideUVTongue builds it as one
// loop — the surviving top-rim arc plus the section arc — watertight, one analytic cone face, no CSG
// (Oblikovati/Oblikovati#1375).
func TestConeSideHalfSpaceEllipseTongue(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(0, 0, 9.5), math.V3(-0.5, 0, -0.866)) // keep the wedge the ellipse cuts off
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("tongue cut should be handled exactly, got %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("result has %d cone faces, want exactly 1 (the tongue stays analytic)", cones)
	}
	if !anyFaceHasEllipse(res) {
		t.Error("no face carries an elliptical edge — the rim-clipping ellipse was not imprinted")
	}
}

// unwrapParamNear shifts a wrapped [0,1) parameter by whole turns to within ±0.5 of the reference, so a
// section sub-arc straddling the ellipse param seam sweeps through the tongue interior, not the major arc.
func TestUnwrapParamNear(t *testing.T) {
	cases := []struct{ ref, x, want float64 }{
		{0.9, 0.1, 1.1},  // forward across the seam: 0.1 lifts a turn so 0.9→1.1 is the short arc
		{0.1, 0.9, -0.1}, // backward across the seam: 0.9 drops a turn
		{0.4, 0.6, 0.6},  // within half a turn: unchanged
		{0.0, 0.5, 0.5},  // exactly half a turn: unchanged (boundary)
	}
	for _, c := range cases {
		if got := unwrapParamNear(c.ref, c.x); got-c.want > 1e-9 || c.want-got > 1e-9 {
			t.Errorf("unwrapParamNear(%g, %g) = %g, want %g", c.ref, c.x, got, c.want)
		}
	}
}

func anyFaceHasEllipse(b *topo.Body) bool {
	for _, f := range b.Faces() {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				switch u.Edge().Geometry().(type) {
				case geom.EllipseFull, geom.EllipticalArc:
					return true
				}
			}
		}
	}
	return false
}

// A SKEW oblique plane (normal not in an axis plane) cuts an ellipse whose seam azimuth lands between
// scan samples, exercising the bisection that anchors the seam ruling; the band must still be watertight.
func TestConeSideHalfSpaceEllipseSkew(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(0, 0, 5), math.V3(0.37, 0.21, 0.9)) // steep + skew → ellipse
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("got %d cone faces, want 1", cones)
	}
	if !anyFaceHasEllipse(res) {
		t.Error("no elliptical edge")
	}
}

func TestConeSideHalfSpaceEllipseKeepTop(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 6, 8), 3, 6)
	plane, _ := geom.NewPlane(math.P3(0, 0, 4), math.V3(0, 0, -1)) // keeps z>4 (top), ellipse = lower rim
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("got %d cone faces, want 1", cones)
	}
	if !anyFaceHasEllipse(res) {
		t.Error("no elliptical edge")
	}
}
