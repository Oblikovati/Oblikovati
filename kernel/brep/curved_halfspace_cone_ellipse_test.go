// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
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
// the clips-rim arrangement, not yet built: it must defer cleanly so the CSG fallback covers it.
func TestConeSideHalfSpaceEllipseClipsRimDefers(t *testing.T) {
	frustum := mustFrustum(t, math.P3(0, 0, 0), math.P3(0, 0, 10), 3, 6)
	plane, _ := geom.NewPlane(math.P3(0, 0, 9.5), math.V3(0.5, 0, 0.866)) // ellipse crosses the top rim
	if _, err := HalfSpaceCut(frustum, plane); !errors.Is(err, ErrUnsupportedHalfSpace) {
		t.Errorf("clips-rim cut should defer with ErrUnsupportedHalfSpace, got %v", err)
	}
}

// wrapToPi folds an angle into (−π, π] from either direction.
func TestWrapToPi(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0}, {3, 3}, {4, 4 - 2*3.141592653589793}, {-4, -4 + 2*3.141592653589793},
	}
	for _, c := range cases {
		if got := wrapToPi(c.in); got-c.want > 1e-9 || c.want-got > 1e-9 {
			t.Errorf("wrapToPi(%g) = %g, want %g", c.in, got, c.want)
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
