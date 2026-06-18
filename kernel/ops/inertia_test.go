// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	"oblikovati.org/kernel/brep"
	gmath "oblikovati.org/math"
)

// TestBodyInertiaBox checks the inertia tensor of a box against the analytic value: for a box of
// full dimensions (lx,ly,lz) and uniform unit density, I_xx = V(ly²+lz²)/12 about the centroid, and
// the products of inertia vanish (the box is axis-aligned and centroid-symmetric).
func TestBodyInertiaBox(t *testing.T) {
	const lx, ly, lz = 4.0, 6.0, 10.0
	block, err := brep.SolidBlock(gmath.P3(0, 0, 0), gmath.P3(lx, ly, lz), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	it := BodyInertia(block, DefaultQuality())

	v := lx * ly * lz
	wantXX := v * (ly*ly + lz*lz) / 12
	wantYY := v * (lx*lx + lz*lz) / 12
	wantZZ := v * (lx*lx + ly*ly) / 12
	approxI(t, "Ixx", it.Ixx, wantXX)
	approxI(t, "Iyy", it.Iyy, wantYY)
	approxI(t, "Izz", it.Izz, wantZZ)
	approxI(t, "Ixy", it.Ixy, 0)
	approxI(t, "Iyz", it.Iyz, 0)
	approxI(t, "Izx", it.Izx, 0)
}

func approxI(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6*math.Max(1, math.Abs(want)) {
		t.Errorf("%s = %g, want %g", name, got, want)
	}
}
