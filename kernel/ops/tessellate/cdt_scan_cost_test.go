// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "testing"

// The cost of a constraint-recovery EDGE QUERY (Oblikovati/Oblikovati#3548).
//
// cdt.go's #1409 note says the freeze it fixed was "each flipOneCrossing is an O(T) scan … at O(nsup)
// total scans instead of O(nsup²)". recoverByFlips then asked hasEdge — a scan of the whole ALLOCATED
// triangle array — once per iteration of a loop whose own bound was also O(len(tris)), which is the
// same shape again, in the guard rather than the body. hasEdgeAround, written for this call site and
// documented as "the local replacement for the O(T) hasEdge scan in the recovery hot path", was never
// used at it.
//
// This row measures the number of whole-mesh scans (cdt.fullScans) rather than wall time, so it is
// deterministic and cannot pass by running on a fast machine. The bound it holds is the STRUCTURAL
// one: after recovery, the scans a face costs must be at most one per FLIP — flipOneCrossing looks for
// any crossing edge and reordering that search would change which edge is flipped, so it keeps its
// scan — plus a small allowance for the incidence hint's own rescans. A presence test inside the loop
// would blow that bound instantly, because the loop iterates whether or not it flips.
//
// Measured on this fixture (a 264-vertex self-crossing band, the one that drives the recovery budget):
// 1186 whole-mesh scans before, 592 after — and 592 is EXACTLY the flip count, so what is left is
// flipOneCrossing's own scan and nothing else. The slack below went unused.
func TestRecoveryEdgeQueriesDoNotScanTheWholeMesh(t *testing.T) {
	t.Parallel()
	m := nonSimpleRecoveryCDT(t)
	if m.recoverFlipWork == 0 {
		t.Fatal("the fixture never reached recoverByFlips — the bound below covers nothing")
	}
	if m.fullScans > m.recoverFlipWork+recoveryScanSlack {
		t.Errorf("constraint recovery cost %d whole-mesh scans for %d flips (allowance %d) — an edge "+
			"query is scanning the triangle array again; hasEdgeAround circles the star in O(deg)",
			m.fullScans, m.recoverFlipWork, recoveryScanSlack)
	}
}

// recoveryScanSlack is how many whole-mesh scans beyond one-per-flip the recovery may cost: the
// incidence hint's own rescans (findIncidentScan) when a flip moves a vertex off its hinted triangle.
// Measured 0 on this fixture; the allowance is for a hint that goes stale on a different input.
const recoveryScanSlack = 64

// TestRecoveryBoundIsTheLiveTriangleCount pins the other half. recoverByFlips' iteration cap read
// len(m.tris), the ALLOCATED array, which only grows: every insertion appends its fan and marks the
// cavity dead. On a covering-sized point set that array runs far ahead of the mesh — 23 606 516
// allocated against 1 574 591 live on the J3 host chart at PropertyQuality — so the cap was not a
// bound on anything. The live count is maintained by addTri and fanCavity, and this asserts it agrees
// with a scan of the dead array, so the two can never drift.
func TestRecoveryBoundIsTheLiveTriangleCount(t *testing.T) {
	t.Parallel()
	m := nonSimpleRecoveryCDT(t)
	counted := 0
	for i := range m.tris {
		if !m.dead[i] {
			counted++
		}
	}
	if m.live != counted {
		t.Errorf("cdt.live is %d, a scan of dead[] counts %d — the incremental count drifted", m.live, counted)
	}
	if m.live >= len(m.tris) {
		t.Errorf("live %d is not below allocated %d on a fixture that deletes cavities; the fixture "+
			"no longer separates the two counts", m.live, len(m.tris))
	}
}

// nonSimpleRecoveryCDT is the 264-vertex self-crossing band of TestConstrainedDelaunayNonSimpleBudget,
// constrained — the one fixture in this package that drives recoverByFlips rather than the corridor
// march.
func nonSimpleRecoveryCDT(t *testing.T) *cdt {
	t.Helper()
	poly := selfCrossingBand(132)
	pts := make([][2]float64, len(poly))
	for i, p := range poly {
		pts[i] = [2]float64{float64(p.X), float64(p.Y)}
	}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain([][]int{rangeIndices(0, len(poly))})
	return m
}
