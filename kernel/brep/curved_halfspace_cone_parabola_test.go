// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The boundary tilt — a plane PARALLEL to a generator — cuts the frustum side in a parabola (the limit
// between the elliptic and hyperbolic sections). The kept side is an exact cone arc-band bounded by the
// two parabola arms, watertight (Oblikovati/Oblikovati#1375). A frustum whose axis is tilted by its own
// half-angle has one generator vertical, so an axis-aligned x-plane is parallel to it.
func TestConeSideHalfSpaceParabola(t *testing.T) {
	t.Parallel()
	a := stdmath.Atan(0.3)
	top := math.P3(math.Scalar(stdmath.Sin(a)*10), 0, math.Scalar(stdmath.Cos(a)*10))
	frustum := mustFrustum(t, math.P3(0, 0, 0), top, 3, 6)
	plane, err := geom.NewPlane(math.P3(2, 0, 0), math.V3(1, 0, 0)) // x=2, parallel to the vertical generator
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	res, err := HalfSpaceCut(frustum, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	if cones, _, _ := faceTypeCounts(t, res); cones != 1 {
		t.Errorf("got %d cone faces, want 1", cones)
	}
	if !anyFaceHasParabola(res) {
		t.Error("no parabolic edge — the boundary-tilt cut was not imprinted as a parabola")
	}
}

func anyFaceHasParabola(b *topo.Body) bool {
	for _, f := range b.Faces() {
		for _, l := range f.Loops() {
			for _, u := range l.EdgeUses() {
				if _, ok := u.Edge().Geometry().(geom.ParabolicArc); ok {
					return true
				}
			}
		}
	}
	return false
}
