// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// topRimKey returns the reference key of a cylinder body's top rim circle (z near the top).
func topRimKey(t *testing.T, b *topo.Body, topZ float64) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		if _, ok := e.Geometry().(geom.Circle); ok && e.RangeBox().Center().Z > topZ-1e-3 {
			return e.ReferenceKey()
		}
	}
	t.Fatal("no top rim circle")
	return nil
}

// TestFilletEdgesRoutesRim drives the public blend.FilletEdges with a circular cylinder/cap rim: it routes
// to the toroidal-band rim fillet, producing a valid solid with one torus face, watertight across
// tolerances, with the rim material removed.
func TestFilletEdgesRoutesRim(t *testing.T) {
	t.Parallel()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 2.0)
	if err != nil {
		t.Fatal(err)
	}
	res, err := blend.FilletEdges(b, [][]byte{topRimKey(t, b, 2.0)}, 0.3)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("rim-filleted cylinder not a valid solid: %+v", r)
	}
	tor := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			tor++
		}
	}
	if tor != 1 {
		t.Errorf("torus faces = %d, want 1", tor)
	}
	for _, tol := range []float64{0.05, 1e-2, 1e-3, 1e-4} {
		m, _ := tessellate.TessellateBody(res, ops.Quality{ChordTolerance: tol})
		if open := meshOpenEdges(m); open != 0 {
			t.Errorf("rim fillet at tol %g: %d open edges", tol, open)
		}
	}
	full := stdmath.Pi * 2.0
	if v := query.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= full || v < full-0.5 {
		t.Errorf("rim-fillet volume = %g, want under %g (rim notch removed)", v, full)
	}
}

// TestFilletRimRadiusTooLarge rejects a rim radius at/over the cylinder radius.
func TestFilletRimRadiusTooLarge(t *testing.T) {
	t.Parallel()
	b, _ := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 2.0)
	if _, err := blend.FilletEdges(b, [][]byte{topRimKey(t, b, 2.0)}, 1.5); err == nil {
		t.Fatal("a rim radius larger than the cylinder radius should fail")
	}
}
