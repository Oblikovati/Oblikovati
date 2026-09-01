// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestFlushFaceUnionShipsExactCoordinates is audit A4's flush-contact regression (#1600): the union
// of two boxes sharing an EXACT face plane must come back watertight, at the analytic volume, and
// with every output vertex coinciding EXACTLY with an operand vertex — no coordinate displaced. A
// flush union resolves through the coplanar ON/ON path (not the tangency nudge), so exact coordinates
// are already the contract; this pins it, and would catch any 1e-5 nudge leaking into a mating face.
func TestFlushFaceUnionShipsExactCoordinates(t *testing.T) {
	t.Parallel()
	a := box(0, 0, 0, 2, 2, 2) // [0,2]³, V=8
	b := box(0, 0, 2, 2, 2, 2) // [0,2]²×[2,4], V=8; shares the whole z=2 face with a
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatalf("flush union: %v", err)
	}
	checkSolid(t, "flush-face-union", res, 16) // two abutting 8-vol boxes → one 2×2×4 box
	assertVerticesAreOperandExact(t, "flush-face-union", res, a, b)
}

// TestBossFlushUnionShipsExactCoordinates is the issue's headline case: a boss seated FLUSH on a wall
// (a smaller box whose bottom face lies exactly in the target's top face). The boss rim imprints the
// wall's top face; every result vertex — the wall corners, the boss corners, and the imprinted rim
// points (which ARE the boss's bottom corners) — must stay at its exact operand coordinate.
func TestBossFlushUnionShipsExactCoordinates(t *testing.T) {
	t.Parallel()
	wall := box(0, 0, 0, 4, 4, 2) // top face at z=2, V=32
	boss := box(1, 1, 2, 2, 2, 2) // bottom face flush on the wall's top, V=8
	res, err := brep.Boolean(brep.Union, wall, boss)
	if err != nil {
		t.Fatalf("boss-flush union: %v", err)
	}
	checkSolid(t, "boss-flush-union", res, 40) // 32 + 8, zero overlap volume
	assertVerticesAreOperandExact(t, "boss-flush-union", res, wall, boss)
}

// assertVerticesAreOperandExact fails if any result vertex is farther than a float-noise tolerance
// (1e-9 cm) from the NEAREST operand vertex. For a flush union no new interior intersection vertices
// arise (the mating faces are coplanar and drop out), so every survivor must be an exact operand
// corner — a 0.1 µm tangency nudge would move an operand's corners off this set and be caught.
func assertVerticesAreOperandExact(t *testing.T, label string, res *topo.Body, operands ...*topo.Body) {
	t.Helper()
	var src []math.Point3
	for _, o := range operands {
		for _, v := range o.Vertices() {
			src = append(src, v.Point())
		}
	}
	for _, v := range res.Vertices() {
		if d := nearestPointDistance(v.Point(), src); d > 1e-9 {
			t.Errorf("%s: result vertex %v sits %g cm off every operand vertex (>1e-9) — the boolean "+
				"displaced flush geometry; coordinates must be exact, no nudge", label, v.Point(), d)
		}
	}
}

// nearestPointDistance returns the distance from p to the closest point in pts (+Inf if pts empty).
func nearestPointDistance(p math.Point3, pts []math.Point3) float64 {
	best := stdmath.Inf(1)
	for _, q := range pts {
		if d := float64(p.DistanceTo(q)); d < best {
			best = d
		}
	}
	return best
}
