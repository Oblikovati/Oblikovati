// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestFreeEdgeCountCountsARealCrack is the detection direction: two triangles that do NOT share their
// common edge (one vertex displaced far beyond any weld tolerance) leave every edge 1-incident.
func TestFreeEdgeCountCountsARealCrack(t *testing.T) {
	t.Parallel()
	m := &Mesh{
		Positions: []math.Point3{
			math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0),
			math.P3(1, 0, 0), math.P3(1, 1, 0.5), math.P3(0.5, 1, 0), // 0.5 away in z: a genuine tear
		},
		Indices: []int{0, 1, 2, 3, 4, 5},
	}
	if got := FreeEdgeCount(m); got != 6 {
		t.Errorf("two disjoint triangles have %d free edges; want 6 (each of the 6 edges 1-incident)", got)
	}
}

// TestFreeEdgeCountDoesNotOverMergeSubMicronFeatures is the false-positive direction, and the reason the
// weld grid is model-relative rather than a fixed 1e-6 (ADR-0042).
//
// The fixture is TWO disjoint closed tetrahedra 3e-7 apart, so the honest answer is 0 free edges: every
// edge of each is used by exactly two triangles. A fixed 1e-6 weld grid rounds the two copies onto the
// same lattice cells, fuses each corresponding vertex pair, and turns all six edges into 4-incident ones
// that a `!= 2` test scores as free — a mesh reported cracked because the RULER is coarser than the
// model, not because the mesh is torn.
//
// That is not hypothetical: it is what the property-quality sweep of the near-pinch gates surfaced. On
// the #1818 crossing-cylinder Intersect (R=3, |Δr|=2e-5) at PropertyQuality the fixed grid welded 8 of
// 5428 vertices onto a DIFFERENT position — e.g. vertices 310 and 311, measured 5.985e-07 apart — and
// produced eight 4-incident edges. At the model's own weld resolution (1.039e-8) it merged 0 of 5428 and
// every one of those edges is 2-incident.
func TestFreeEdgeCountDoesNotOverMergeSubMicronFeatures(t *testing.T) {
	t.Parallel()
	const gap = 3e-7
	base := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 1)}
	m := &Mesh{}
	for _, dx := range []float64{0, gap} {
		off := len(m.Positions)
		for _, p := range base {
			m.Positions = append(m.Positions, math.P3(p.X+math.Scalar(dx), p.Y, p.Z))
		}
		for _, tri := range [][3]int{{0, 2, 1}, {0, 1, 3}, {0, 3, 2}, {1, 2, 3}} {
			m.Indices = append(m.Indices, off+tri[0], off+tri[1], off+tri[2])
		}
	}
	// Assert the premise rather than trusting it: a fixed 1e-6 grid fuses the two copies, the model's own
	// resolution keeps them apart.
	fixed := map[[3]int64]bool{}
	relative := map[[3]int64]bool{}
	grid := geom.ResolutionForPoints(m.Positions).Weld()
	for _, p := range m.Positions {
		fixed[[3]int64{int64(float64(p.X)*1e6 + 0.5), int64(float64(p.Y)*1e6 + 0.5), int64(float64(p.Z)*1e6 + 0.5)}] = true
		relative[WeldKey(p, grid)] = true
	}
	if len(fixed) != 4 || len(relative) != 8 {
		t.Fatalf("fixture premise broken: a fixed 1e-6 grid sees %d distinct vertices (want 4, fused) and the "+
			"model-relative grid %.3e sees %d (want 8, distinct)", len(fixed), grid, len(relative))
	}
	if got := FreeEdgeCount(m); got != 0 {
		t.Errorf("two disjoint closed tetrahedra %g apart report %d free edges; want 0 — the weld tolerance "+
			"must follow the model, not a fixed 1e-6 lattice", gap, got)
	}
}
