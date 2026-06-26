// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/diag"
)

// #1410: a non-recovered boundary constraint must never be recorded as recovered (a phantom con entry the
// flood cannot toggle at leaks the domain), and a genuine leak must fall back deterministically and be
// surfaced — while a benign non-recovery keeps the higher-quality refined mesh.

// edgeUseCount tallies how many of the given triangles use each undirected edge — the watertightness probe
// the recovery tests share (boundary edges are used once, interior edges twice, >2 is non-manifold).
func edgeUseCount(tris [][3]int) map[[2]int]int {
	use := map[[2]int]int{}
	for _, t := range tris {
		for i := 0; i < 3; i++ {
			use[conKey(t[i], t[(i+1)%3])]++
		}
	}
	return use
}

// TestConstrainRecordsOnlyRecoveredEdges is #1410 criterion 1: every edge recorded in con must actually
// exist in the mesh. The old code set con unconditionally, registering phantom boundaries with no edge;
// the invariant con ⟹ hasEdge guarantees floodInside has a real edge to toggle at for each constraint.
func TestConstrainRecordsOnlyRecoveredEdges(t *testing.T) {
	pts := [][2]float64{{0, 0}, {4, 0}, {4, 3}, {0, 3}} // a simple convex quad, every loop edge recoverable
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain([][]int{{0, 1, 2, 3}})
	if len(m.unrecovered) != 0 {
		t.Fatalf("a convex quad loop should fully recover, got %d unrecovered", len(m.unrecovered))
	}
	for e := range m.con {
		if !m.hasEdge(e[0], e[1]) {
			t.Errorf("con holds edge %v with no surviving mesh edge — a phantom constraint (#1410)", e)
		}
	}
}

// TestDomainLeakedFlagsFilledLoopNotFold checks the detector that gates the fallback: a loop dissolved
// into the domain interior (a hole the flood filled) is a leak, while an isolated interior edge (a local
// fold repairFolds later removes) and a cleanly cut loop are not — so a single fold never discards the
// refined mesh (#1410).
func TestDomainLeakedFlagsFilledLoopNotFold(t *testing.T) {
	// Distinct coordinates → representatives() is the identity, so loop indices address pts directly.
	pts := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {3, 3}, {6, 3}, {6, 6}}
	m := newCDT(pts)
	loops := [][]int{{0, 1, 2, 3}, {4, 5, 6}} // outer quad + triangular hole (edges 4-5, 5-6, 6-4)

	cut := [][3]int{{0, 1, 4}, {1, 5, 4}, {1, 2, 5}, {2, 6, 5}} // each hole edge appears once → boundary
	if m.domainLeaked(cut, loops) {
		t.Error("a cleanly cut hole (every loop edge on the boundary) must not be flagged as leaked")
	}

	// One hole edge (4-5) shared by two kept triangles, the other two still boundary: a local fold, where
	// dissolved edges (1) do NOT outnumber boundary edges (2), so it must NOT be flagged as a leak.
	fold := [][3]int{{4, 5, 6}, {4, 5, 0}}
	if m.domainLeaked(fold, loops) {
		t.Error("a single dissolved hole edge is a local fold, not a leak — must keep the refined mesh")
	}

	// Every hole edge shared by two kept triangles → the hole boundary fully dissolved (filled).
	filled := [][3]int{{4, 5, 6}, {4, 5, 0}, {5, 6, 1}, {6, 4, 2}}
	if !m.domainLeaked(filled, loops) {
		t.Error("a filled hole (every loop edge interior) must be flagged as a leak (#1410)")
	}
}

// TestEarcutFromLoopsProducesWatertightIndexedMesh checks the deterministic fallback: ear clipping the
// loops yields a manifold triangulation, indexed back into the original pts, whose area is the outer minus
// the hole — i.e. the hole is genuinely cut, never filled (#1410).
func TestEarcutFromLoopsProducesWatertightIndexedMesh(t *testing.T) {
	pts := [][2]float64{{0, 0}, {10, 0}, {10, 10}, {0, 10}, {3, 3}, {6, 3}, {6, 6}, {3, 6}}
	loops := [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}} // a 10×10 square with a 3×3 square hole
	tris := earcutFromLoops(pts, loops)
	if len(tris) == 0 {
		t.Fatal("earcut fallback produced no triangles")
	}
	for _, tri := range tris {
		for _, v := range tri {
			if v < 0 || v >= len(pts) {
				t.Fatalf("triangle index %d out of range for %d pts (remap broke)", v, len(pts))
			}
		}
	}
	for e, c := range edgeUseCount(tris) {
		if c > 2 {
			t.Errorf("edge %v used by %d triangles — non-manifold fallback", e, c)
		}
	}
	if area := loopTriArea(pts, tris); stdmath.Abs(area-(100-9)) > 1e-9 {
		t.Errorf("fallback area %.4f, want 91 (square 100 − hole 9); the hole was not cut", area)
	}
}

// loopTriArea sums the unsigned area of pts-indexed triangles (the fallback's domain area).
func loopTriArea(pts [][2]float64, tris [][3]int) float64 {
	total := 0.0
	for _, t := range tris {
		a, b, c := pts[t[0]], pts[t[1]], pts[t[2]]
		total += stdmath.Abs((b[0]-a[0])*(c[1]-a[1])-(c[0]-a[0])*(b[1]-a[1])) / 2
	}
	return total
}

// TestRecordConstraintLeakSeverity checks the two-tier diagnostic: a genuine leak (fallback taken) is a
// counted Defect, a benign non-recovery is an Info, and a fully recovered triangulation records nothing —
// so non-recovery is never silent yet the common harmless case raises no false alarm (#1410).
func TestRecordConstraintLeakSeverity(t *testing.T) {
	none := &Mesh{}
	recordConstraintLeak(none, nil, false)
	if len(none.Diagnostics) != 0 {
		t.Errorf("full recovery should record nothing, got %d diagnostics", len(none.Diagnostics))
	}

	benign := &Mesh{}
	recordConstraintLeak(benign, [][2]int{{1, 2}}, false)
	if len(benign.Diagnostics) != 1 || benign.Diagnostics[0].Severity != diag.Info ||
		benign.Diagnostics[0].Code != CodeCDTConstraintLeak {
		t.Errorf("benign non-recovery should record one Info %s, got %+v", CodeCDTConstraintLeak, benign.Diagnostics)
	}

	leaked := &Mesh{}
	recordConstraintLeak(leaked, [][2]int{{1, 2}}, true)
	if len(leaked.Diagnostics) != 1 || leaked.Diagnostics[0].Severity != diag.Defect {
		t.Errorf("a genuine leak should record one Defect, got %+v", leaked.Diagnostics)
	}
}

// TestHoledCylinderWallNonRecoverySurfacedAndWatertight is the #1410 watertightness test for the real
// cap-hit / non-recovery path: the holed cylinder wall's long seam constraint does not recover, but the
// cut stays watertight (the seam borders the excluded super region), so the mesher keeps the refined mesh,
// reports a non-manifold-free result, AND surfaces the non-recovery as an Info diagnostic — never silent.
func TestHoledCylinderWallNonRecoverySurfacedAndWatertight(t *testing.T) {
	const uLo, uHi, vLo, vHi = stdmath.Pi / 4, stdmath.Pi / 2, 4.0, 8.0
	face := windowedCylinderWall(t, uLo, uHi, vLo, vHi)
	mesh := tessellateCurvedFace(face, DefaultQuality())
	if mesh == nil || mesh.TriangleCount() == 0 {
		t.Fatal("holed cylinder wall produced no mesh")
	}

	tris := make([][3]int, 0, mesh.TriangleCount())
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		tris = append(tris, [3]int{mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]})
	}
	for e, c := range edgeUseCount(tris) {
		if c > 2 {
			t.Errorf("mesh edge %v used by %d triangles — the non-recovery tore the mesh non-manifold", e, c)
		}
	}

	var info, defect int
	for _, d := range mesh.Diagnostics {
		if d.Code != CodeCDTConstraintLeak {
			continue
		}
		switch d.Severity {
		case diag.Info:
			info++
		case diag.Defect:
			defect++
		}
	}
	if info != 1 {
		t.Errorf("seam non-recovery must surface exactly one Info %s, got %d", CodeCDTConstraintLeak, info)
	}
	if defect != 0 {
		t.Errorf("the watertight holed wall must not raise a leak Defect, got %d", defect)
	}
}
