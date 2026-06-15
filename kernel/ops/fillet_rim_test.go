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
// WATERTIGHT mesh across tolerances — exercising the doubly-periodic torus-band loft
// (closedBandLoftMesh), which lofts the tube between each circle edge's OWN tessellation so the
// differing-radius cap and cyl circles both seam to their neighbours watertight at any resolution.
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
	for _, tol := range []float64{0.05, 1e-2, 1e-3, 1e-4} {
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

// TestRimFilletWatertightAcrossSizes checks the torus-band loft stays a watertight valid solid over
// a range of radius/height/fillet ratios (the cap and cyl circles tessellate at different counts, so
// the loft's differing-ring stitch is exercised differently each time).
func TestRimFilletWatertightAcrossSizes(t *testing.T) {
	cases := []struct{ R, h, r float64 }{{1, 2, 0.3}, {2, 3, 0.5}, {0.5, 1, 0.1}, {1.5, 4, 0.7}, {3, 2, 0.9}}
	for _, c := range cases {
		b, err := brep.SolidCylinderFilletedTop(math.P3(0, 0, 0), math.V3(0, 0, 1), c.R, c.h, c.r)
		if err != nil {
			t.Errorf("R=%g h=%g r=%g: %v", c.R, c.h, c.r, err)
			continue
		}
		if rep := ops.Validate(b); !rep.Valid || !b.IsSolid() {
			t.Errorf("R=%g r=%g: not a valid solid: %+v", c.R, c.r, rep.Issues)
		}
		for _, tol := range []float64{0.05, 1e-2, 1e-3, 1e-4} {
			m, _ := ops.TessellateBody(b, ops.Quality{ChordTolerance: tol})
			if open := meshOpenEdges(m); open != 0 {
				t.Errorf("R=%g h=%g r=%g at tol %g: %d open edges", c.R, c.h, c.r, tol, open)
			}
		}
	}
}
