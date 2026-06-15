// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestRimFilletTorusBand checks the closed-rim ("no run-out") curved fillet: a cylinder whose top
// rim is rounded into a toroidal band is a valid solid with one torus face, material removed, and a
// WATERTIGHT mesh at practical tolerances — exercising the doubly-periodic torus-band tessellation
// path (doublyPeriodicBandGrid). It is the foundation for the run-out cases: a torus band that closes
// on itself. The fine-tolerance seam between the torus and the smaller-radius cap circle still needs a
// loft-between-rings mesher (next increment), so watertightness is asserted at the practical
// tolerances the head and mass-properties use.
func TestRimFilletTorusBand(t *testing.T) {
	b, err := brep.SolidCylinderFilletedTop(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 2.0, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(b); !r.Valid || !b.IsSolid() {
		t.Fatalf("rim-filleted cylinder not a valid solid: %+v", r)
	}
	tor, cyl, pln := 0, 0, 0
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tor++
		case geom.Cylinder:
			cyl++
		case geom.Plane:
			pln++
		}
	}
	if tor != 1 || cyl != 1 || pln != 2 {
		t.Errorf("faces: torus=%d cyl=%d plane=%d, want 1/1/2", tor, cyl, pln)
	}
	for _, tol := range []float64{0.05, 1e-2} {
		m, _ := ops.TessellateBody(b, ops.Quality{ChordTolerance: tol})
		if open := meshOpenEdges(m); open != 0 {
			t.Errorf("rim-fillet mesh at tol %g has %d open edges, want watertight", tol, open)
		}
	}
	full := stdmath.Pi * 1.0 * 1.0 * 2.0 // π·R²·h
	v := ops.BodyGeometryProperties(b, ops.Quality{ChordTolerance: 1e-3}).Volume
	if v >= full || v < full-0.5 {
		t.Errorf("rim-fillet volume = %g, want a little under the full cylinder %g (rim notch removed)", v, full)
	}
}
