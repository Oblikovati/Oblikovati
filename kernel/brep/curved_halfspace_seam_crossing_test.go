// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A cylinder cut by a plane PARALLEL to the axis and passing through a cap's seam vertex must split
// watertight, not defer: the cap-boundary circle's seam lands on the cut plane, the closed-edge crossing
// degeneracy that closedEdgeCrossings fixes. The seam of a SolidCylinder cap circle is at angle 0 (the
// +RefDir point); a plane through that point parallel to the axis makes the seam a crossing.
func TestHalfSpaceThroughCapSeam(t *testing.T) {
	t.Parallel()
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	// The cap circle's seam point is at radius along +RefDir. A plane through the axis whose normal is
	// perpendicular to that ref makes the seam point lie on the plane (a chord through the seam).
	plane, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0)) // y=0 through the axis: seam (5,0,*) on it
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	res, err := HalfSpaceCut(cyl, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut through a cap seam should not defer: %v", err)
	}
	assertWatertight(t, res)
	// A genuine half-cylinder, not the whole solid kept: half side + two half-cap segments + the planar
	// lid — more faces than the original three. Before the fix the seam-coincident crossing was missed, so
	// the cap split deferred and the whole cut fell to CSG.
	if got := len(res.Faces()); got <= len(cyl.Faces()) {
		t.Errorf("result has %d faces, want > %d (the half-cut must add a lid, not keep the solid whole)", got, len(cyl.Faces()))
	}
	if !hasPlanarLid(res, plane) {
		t.Error("result has no planar lid on the cut plane — the body was not actually split")
	}
}

// hasPlanarLid reports whether the body has a planar face coincident with the cut plane (the lid the
// split adds).
func hasPlanarLid(b *topo.Body, plane geom.Plane) bool {
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && samePoint(pl.Origin, plane.Origin, geom.ResolutionForSize(1)) {
			return true
		}
	}
	return false
}
