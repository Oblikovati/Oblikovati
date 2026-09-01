// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// planarVolumeCheck runs one boolean and asserts the result validates and matches the analytic
// volume two-sided (#1599, #1601) — a misclassified fragment shifts the volume by that fragment's
// prism, so the volume bracket is the cheap detector for a classification flip.
func planarVolumeCheck(t *testing.T, name string, op ops.PartFeatureOperation, a, b *topo.Body, want float64) {
	t.Helper()
	res, err := ops.Boolean(op, a, b)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if r := ops.Validate(res); !r.Valid || !r.Closed || !r.Manifold {
		t.Errorf("%s: result invalid: %+v", name, r)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want) > 1e-6+1e-9*want {
		t.Errorf("%s: volume = %.9f, want %.9f (analytic)", name, got, want)
	}
}

// TestPlanarBooleanVolumesAnalytic pins the planar boolean's fragment classification to analytic
// volumes on overlapping boxes and an L-bracket cut (#1599): every op, both operand orders.
func TestPlanarBooleanVolumesAnalytic(t *testing.T) {
	t.Parallel()
	mk := func() (*topo.Body, *topo.Body) {
		a, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "a") // V=8
		b, _ := brep.SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 3), "b") // V=8, overlap 1
		return a, b
	}
	a, b := mk()
	planarVolumeCheck(t, "union", ops.Join, a, b, 15)
	a, b = mk()
	planarVolumeCheck(t, "cut", ops.Cut, a, b, 7)
	a, b = mk()
	planarVolumeCheck(t, "cut reversed", ops.Cut, b, a, 7)
	a, b = mk()
	planarVolumeCheck(t, "intersect", ops.Intersect, a, b, 1)

	// L-bracket: cut a corner block out of a slab, leaving the L; then cut the L with a bar
	// crossing both legs — fragments whose sample points hug shared edges on every side.
	slab, _ := brep.SolidBlock(math.P3(0, 0, 0), math.P3(4, 4, 1), "slab") // V=16
	corner, _ := brep.SolidBlock(math.P3(2, 2, -1), math.P3(5, 5, 2), "corner")
	lbr, err := ops.Boolean(ops.Cut, slab, corner)
	if err != nil {
		t.Fatalf("L-bracket: %v", err)
	}
	bar, _ := brep.SolidBlock(math.P3(-1, 1, 0.25), math.P3(5, 1.5, 0.75), "bar") // 0.5×0.5 section across x
	// L = 16 − (2×2×1 corner) = 12; the bar crosses the L's full 4-wide leg at y∈[1,1.5]:
	// removed = 4 (x-span) × 0.5 × 0.5 = 1.
	planarVolumeCheck(t, "L-bracket bar cut", ops.Cut, lbr, bar, 11)
}
