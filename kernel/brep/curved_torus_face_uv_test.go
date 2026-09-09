// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestTorusChartSplitsAnImprintAtTheTubeSeam is the regression for ADR-0062's last gap: the loop-framed
// sampler unwrapped and split a polyline only on the AZIMUTH, so an imprint that wraps the TUBE angle —
// a spiric oval, which every plane parallel to the torus axis produces — came out of the arrangement as
// one unbroken chain carrying a segment that jumped the whole v period. That segment sliced the
// parameter rectangle, the kept region fell into slivers, and the boolean's torus face emitted
// zero-length edges instead of the two closed ovals.
//
// The invariant is the sampler's, not the torus's: no imprint segment may span more than half a period
// in a direction the chart calls periodic.
func TestTorusChartSplitsAnImprintAtTheTubeSeam(t *testing.T) {
	t.Parallel()
	body, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "t")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	face := bodyTorusFace(t, body)
	tor, ok := torusFaceOf(face)
	if !ok {
		t.Fatal("the torus face is not chartable")
	}
	plane, err := geom.NewPlane(math.P3(1, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	imprint, sok := geom.TorusPlaneSection(tor, plane)
	if !sok {
		t.Fatal("the spiric section declined")
	}
	if len(imprint) != 2 {
		t.Fatalf("plane x=1 through a 5/2 torus makes two spiric ovals, got %d", len(imprint))
	}
	c := newTorusFaceUV(face, tor, Difference, false, func(math.Point3) bool { return false })
	c.placeSeams(imprint)
	for _, s := range c.assembleSegments(imprint) {
		if s.kind != segImprint {
			continue
		}
		if dv := stdmath.Abs(float64(s.a.Y - s.b.Y)); dv > stdmath.Pi {
			t.Fatalf("an imprint segment jumps %.4f in v (half a period is %.4f): the tube seam was not split", dv, stdmath.Pi)
		}
	}
}

// bodyTorusFace returns the body's single torus face as a curvedFace.
func bodyTorusFace(t *testing.T, body *topo.Body) curvedFace {
	t.Helper()
	for _, f := range body.Faces() {
		if _, ok := geom.TorusOf(f.Geometry()); ok {
			return curvedFaceOf(f)
		}
	}
	t.Fatal("the body has no torus face")
	return curvedFace{}
}
