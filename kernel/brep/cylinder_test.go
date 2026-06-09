// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// SolidCylinder builds a valid analytic solid: one true cylinder face + two planar caps,
// with a volume of π·r²·h (the first curved-face B-rep in the kernel, K1b slice 2).
func TestSolidCylinderIsValidAnalyticSolid(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(cyl); !r.Valid || !cyl.IsSolid() {
		t.Fatalf("solid cylinder is not a valid solid: %+v", r)
	}
	if len(cyl.Faces()) != 3 {
		t.Fatalf("faces = %d, want 3 (two caps + one side)", len(cyl.Faces()))
	}

	nCyl, nPlane := 0, 0
	for _, f := range cyl.Faces() {
		switch f.Geometry().(type) {
		case geom.Cylinder:
			nCyl++
		case geom.Plane:
			nPlane++
		}
	}
	if nCyl != 1 || nPlane != 2 {
		t.Errorf("face surfaces = %d cylinder / %d plane, want 1 / 2 (a real curved face)", nCyl, nPlane)
	}
	// The full 2π-periodic side now tessellates over its true trim (periodicBandGrid), so the
	// volume is the inscribed polyhedron's ≈ π·r²·h — always a hair UNDER exact (chords inside
	// the arc), here ~2.5% at default circle resolution.
	want := stdmath.Pi * 4 * 5 // π·r²·h
	if v := vol(cyl); v > want+1e-9 || (want-v)/want > 0.03 {
		t.Errorf("volume = %g, want a hair under %g (π·2²·5, inscribed)", v, want)
	}
}

// The side is a periodic face: its seam edge carries two opposite uses (watertight).
func TestSolidCylinderWatertight(t *testing.T) {
	cyl, _ := brep.SolidCylinder(math.P3(1, 2, 0), math.V3(0, 0, 1), 1.5, 3)
	if open := ops.BoundaryEdges(cyl); len(open) != 0 {
		t.Fatalf("solid cylinder has %d boundary edges, want 0 (watertight)", len(open))
	}
}
