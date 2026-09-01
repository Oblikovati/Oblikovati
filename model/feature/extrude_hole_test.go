// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/model/param"
)

// TestExtrudeHonorsHole is a regression for the prism builder ignoring a profile's
// inner loops: an annular profile (square with a square hole) extruded into a solid
// block instead of a frame. The extruded volume must be (outer-inner)·height.
func TestExtrudeHonorsHole(t *testing.T) {
	t.Parallel()
	ps := param.NewParameters()
	fs := NewPartFeatures(ps)
	const side, hole, height = 4.0, 2.0, 3.0
	sk := squareWithHoleSketch(side, hole)

	// pick the annular profile (the one with an inner loop)
	profIdx := -1
	for i := 0; i < sk.Profiles().Count(); i++ {
		if len(sk.Profiles().Item(i).InnerLoops()) > 0 {
			profIdx = i
		}
	}
	if profIdx < 0 {
		t.Fatal("no annular profile detected")
	}

	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, profIdx, ops.NewBody, func() float64 { return height })
	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("extrude produced %d bodies (want 1 solid)", len(bodies))
	}
	want := (side*side - hole*hole) * height // 16-4 = 12 * 3 = 36
	got := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("hollow extrude volume = %.4f, want %.4f (hole not honored?)", got, want)
	}
}

// TestExtrudeHonorsMultipleHoles is a regression for the multi-hole cap: a plate with FOUR
// square holes extruded as a solid. Previously the exact B-rep path produced a valid but
// wrongly-measured body — the planar cap face carrying several holes tessellated to roughly
// half its true area — so the volume came out far too low. The fix is the earcut planar
// triangulator (kernel/ops/earcut.go); the volume must now be (plate − 4·hole)·height.
func TestExtrudeHonorsMultipleHoles(t *testing.T) {
	t.Parallel()
	ps := param.NewParameters()
	fs := NewPartFeatures(ps)
	const w, h, hole, height = 8.0, 6.0, 1.0, 2.0
	sk := plateWithHolesSketch(w, h, hole, [][2]float64{{2, 1.5}, {6, 1.5}, {2, 4.5}, {6, 4.5}})

	profIdx := -1
	for i := 0; i < sk.Profiles().Count(); i++ {
		if len(sk.Profiles().Item(i).InnerLoops()) == 4 {
			profIdx = i
		}
	}
	if profIdx < 0 {
		t.Fatal("no 4-hole profile detected")
	}

	NewExtrudeFeatures(fs).AddByDistanceExtent(sk, profIdx, ops.NewBody, func() float64 { return height })
	fs.Recompute()
	bodies := fs.Result()
	if len(bodies) != 1 || !bodies[0].IsSolid() {
		t.Fatalf("extrude produced %d bodies (want 1 solid)", len(bodies))
	}
	if rep := ops.Validate(bodies[0]); !rep.Valid {
		t.Fatalf("4-hole extrude is not a valid solid: %+v", rep)
	}
	want := (w*h - 4*hole*hole) * height
	got := query.BodyGeometryProperties(bodies[0], ops.DefaultQuality()).Volume
	if stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("4-hole extrude volume = %.4f, want %.4f (multi-hole cap mis-tessellated?)", got, want)
	}
}
