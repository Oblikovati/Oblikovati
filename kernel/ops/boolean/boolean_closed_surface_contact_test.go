// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// A section that CROSSES the receiving face's own boundary is contact, and the pairing CARRIES it: the
// section is clipped to that face's trim, once, and the bounded arcs go to both sides.
//
// It used to ask whether the section sat WHOLLY inside the receiver's trim, so a crossing read as "no
// contact at all": nothing was imprinted, and a sphere intersected with a box passed through WHOLE — a
// valid solid of entirely the wrong shape. That became a named decline, and then this: the ordinary
// answer (ADR-0061 stage 4).
func TestASectionCrossingItsReceiverIsCarried(t *testing.T) {
	t.Parallel()
	sphere, err := brep.SolidSphere(math.P3(0, 0, 0), 5, "s")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	// The box keeps x ≤ 2 and z ≤ 0; both section circles leave through their own face's edge.
	box, err := brep.SolidBlock(math.P3(-10, -10, -10), math.P3(2, 10, 0), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	res, err := brep.Boolean(brep.Intersection, sphere, box)
	if err != nil {
		t.Fatalf("brep.Boolean(Intersection): %v", err)
	}
	spheres := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			spheres++
		}
	}
	if spheres != 1 {
		t.Errorf("result has %d sphere faces, want 1 — the curved cap must survive as exact geometry", spheres)
	}
	if got := len(res.Faces()); got != 3 {
		t.Errorf("result has %d faces, want 3 (two box planes + the sphere cap)", got)
	}
}

// A sphere poking out of a box through THREE of its faces leaves an ANNULUS on the sphere: an outer
// three-arc loop that wraps the azimuth, and an inner three-arc loop around the box's corner that does
// not. Traced inner-first the face read as "the little corner patch minus everything else" — a VALID
// solid whose volume came back as the box exactly, the worst kind of wrong answer (ADR-0061 stage 4).
func TestSpherePokingThroughThreeBoxFacesUnionsToTheRightVolume(t *testing.T) {
	t.Parallel()
	const boxVol = 4.0 * 4.0 * 4.0
	sphereVol := 4.0 / 3.0 * stdmath.Pi * 1.5 * 1.5 * 1.5
	// Three equal caps of height 0.5 poke out of x=2, y=2, z=2; they overlap only in a set of measure
	// far below the tessellated budget this is checked against.
	capVol := stdmath.Pi * 0.5 * 0.5 * (3*1.5 - 0.5) / 3
	for _, tc := range []struct {
		op   ops.PartFeatureOperation
		want float64
	}{
		{ops.Join, boxVol + 3*capVol},
		{ops.Intersect, sphereVol - 3*capVol},
	} {
		t.Run(tc.op.String(), func(t *testing.T) {
			t.Parallel()
			box, err := brep.SolidBlock(math.P3(-2, -2, -2), math.P3(2, 2, 2), "box")
			if err != nil {
				t.Fatalf("box: %v", err)
			}
			sphere, err := brep.SolidSphere(math.P3(1, 1, 1), 1.5, "s")
			if err != nil {
				t.Fatalf("sphere: %v", err)
			}
			res, err := ops.Boolean(tc.op, box, sphere)
			if err != nil {
				t.Fatalf("Boolean(%v): %v", tc.op, err)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-tc.want) / tc.want; rel > 0.02 {
				t.Errorf("%v volume %.4f, want %.4f (rel %.4f > 2%%)", tc.op, got, tc.want, rel)
			}
		})
	}
}
