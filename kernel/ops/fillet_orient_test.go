// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// sampleFilletLoop is a mixed arc/line triangle loop with DISTINCT provenance ids per point and per
// segment, so a reversal that shifted, dropped, or mis-paired srcV/srcE would be caught.
func sampleFilletLoop() (filletLoop, map[int]int) {
	p0, p1, p2 := m.P3(1, 0, 0), m.P3(0, 1, 0), m.P3(-1, 0, 0)
	arc0, _ := geom.Arc3dByThreePoints(p0, m.P3(0.7071, 0.7071, 0), p1) // P0→P1 upper arc
	arc2, _ := geom.Arc3dByThreePoints(p2, m.P3(0, -1, 0), p0)          // P2→P0 lower arc
	loop := filletLoop{
		pts:    []m.Point3{p0, p1, p2},
		curves: []geom.Curve3{arc0, nil, arc2}, // segment 1 (P1→P2) is straight
		srcV:   []uint64{201, 202, 203},
		srcE:   []uint64{101, 102, 103},
	}
	ptID := map[int]int{0: 0, 1: 1, 2: 2} // point index → stable id (identity here)
	return loop, ptID
}

// edgeSrcMap keys each segment's srcE by its UNDIRECTED point-pair (looked up by position), so two
// loops of the same physical edges compare equal regardless of traversal direction or anchor.
func edgeSrcMap(loop filletLoop, idOf func(m.Point3) int) map[[2]int]uint64 {
	out := map[[2]int]uint64{}
	n := len(loop.pts)
	for i := 0; i < n; i++ {
		a, b := idOf(loop.pts[i]), idOf(loop.pts[(i+1)%n])
		out[canon2(a, b)] = loop.srcE[i]
	}
	return out
}

// TestReverseFilletLoopPreservesEdgeIdentity is the load-bearing B2 assertion: after reversal, every
// segment's srcE still labels the SAME physical edge (same undirected endpoint pair). A naive reverse
// that kept srcE aligned to the point index (instead of shifting to the segment leaving the point)
// would move edge 101 onto the wrong pair and reintroduce the #1600 tangent-seam collapse.
func TestReverseFilletLoopPreservesEdgeIdentity(t *testing.T) {
	loop, _ := sampleFilletLoop()
	idOf := func(p m.Point3) int {
		switch {
		case p.DistanceTo(m.P3(1, 0, 0)) < 1e-9:
			return 0
		case p.DistanceTo(m.P3(0, 1, 0)) < 1e-9:
			return 1
		default:
			return 2
		}
	}
	fwd := edgeSrcMap(loop, idOf)
	rev := edgeSrcMap(reverseFilletLoop(loop), idOf)
	if len(rev) != len(fwd) {
		t.Fatalf("edge count changed: forward %d, reversed %d", len(fwd), len(rev))
	}
	for pair, src := range fwd {
		if rev[pair] != src {
			t.Errorf("edge %v: srcE %d forward, %d reversed — identity not preserved", pair, src, rev[pair])
		}
	}
}

// TestReverseFilletLoopSrcVFollowsPoints checks each reversed point still carries its own source
// vertex id (srcV rides the point index, unlike srcE which rides the leaving segment).
func TestReverseFilletLoopSrcVFollowsPoints(t *testing.T) {
	loop, _ := sampleFilletLoop()
	rev := reverseFilletLoop(loop)
	want := map[[3]float64]uint64{
		{1, 0, 0}:  201,
		{0, 1, 0}:  202,
		{-1, 0, 0}: 203,
	}
	for i, p := range rev.pts {
		key := [3]float64{p.X, p.Y, p.Z}
		if rev.srcV[i] != want[key] {
			t.Errorf("point %v: srcV %d, want %d", p, rev.srcV[i], want[key])
		}
	}
}

// TestReverseFilletLoopInvolution pins that reversing twice restores the loop exactly (pts, srcV,
// srcE), so the flip is a clean orientation toggle with no cumulative drift.
func TestReverseFilletLoopInvolution(t *testing.T) {
	loop, _ := sampleFilletLoop()
	back := reverseFilletLoop(reverseFilletLoop(loop))
	for i := range loop.pts {
		if back.pts[i].DistanceTo(loop.pts[i]) > 1e-9 {
			t.Errorf("pts[%d] = %v, want %v", i, back.pts[i], loop.pts[i])
		}
		if back.srcV[i] != loop.srcV[i] || back.srcE[i] != loop.srcE[i] {
			t.Errorf("meta[%d] = (v%d,e%d), want (v%d,e%d)", i, back.srcV[i], back.srcE[i], loop.srcV[i], loop.srcE[i])
		}
	}
}

// TestReverseIntRingMatchesLoopAnchor pins that reverseIntRing uses the same anchor convention as
// reverseFilletLoop, so the welded ring stays index-aligned with the reversed loop's points.
func TestReverseIntRingMatchesLoopAnchor(t *testing.T) {
	ring := []int{5, 6, 7, 8}
	got := reverseIntRing(ring)
	want := []int{5, 8, 7, 6} // index 0 fixed, rest reversed — mirrors pts[(n-i)%n]
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reverseIntRing[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}
