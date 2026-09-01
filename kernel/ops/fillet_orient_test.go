// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
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
	for i := range n {
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	ring := []int{5, 6, 7, 8}
	got := reverseIntRing(ring)
	want := []int{5, 8, 7, 6} // index 0 fixed, rest reversed — mirrors pts[(n-i)%n]
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("reverseIntRing[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// onSurfaceReversal is one loop segment lying exactly on a surface, plus what its reversal must be.
type onSurfaceReversal struct {
	name    string
	curve   geom.Curve3
	surface geom.Surface
	refit   bool // true when the segment's locus is a CIRCLE, so the three-point re-fit is exact and kept
}

// spiricOnTorus is the oblique torus section D4/D5/D9/E3/E4 and N7 carry as a run-out rail: a quartic
// spiric of Perseus, exactly on the torus and exactly on the cut plane.
func spiricOnTorus() (geom.SpiricArc, geom.Torus) {
	tor, _ := geom.NewTorus(m.P3(0, 0, 0), m.V3(0, 0, 1), 20, 5)
	pl, _ := geom.NewPlane(m.P3(6, 0, 1), m.V3(0.3, 0, 1))
	phi, mm, k, c := geom.TorusSectionCoeffs(tor, pl)
	return geom.SpiricArc{Torus: tor, Phi: phi, M: mm, K: k, C: c, Branch: +1, V0: 0.3, V1: 1.9}, tor
}

// ellipseOnCylinder is F4's rail class: the ellipse an oblique plane cuts from a cylinder — here the
// z = 0.4x section of the r=10 cylinder about +Z, whose semi-axes are 10·√1.16 along (1,0,0.4) and 10.
func ellipseOnCylinder() (geom.EllipticalArc, geom.Cylinder) {
	cyl, _ := geom.NewCylinder(m.P3(0, 0, 0), m.V3(0, 0, 1), 10)
	e, _ := geom.NewEllipticalArc(m.P3(0, 0, 0), m.V3(-0.4, 0, 1), m.V3(1, 0, 0.4), 10*stdmath.Sqrt(1.16), 10, 0.2, 1.1)
	return e, cyl
}

// onSurfaceReversalCases covers every locus class the shell-orientation pass reverses: the two analytic
// non-circular rails (spiric, ellipse), a canal-style SUB-SPAN of one (the geom.TrimmedCurve3 chain N7/O1
// and the run-out plate cases register), a NEARLY STRAIGHT sub-span (the collinear input that made
// Arc3dByThreePoints fail, whose ignored error installed a degenerate zero Arc3d at the ORIGIN), and the
// circular controls whose historical three-point re-fit must be kept.
func onSurfaceReversalCases() []onSurfaceReversal {
	sp, tor := spiricOnTorus()
	el, cyl := ellipseOnCylinder()
	arc, _ := geom.Arc3dByThreePoints(m.P3(10, 0, 3), m.P3(0, 10, 3), m.P3(-10, 0, 3)) // on cyl, z=3
	return []onSurfaceReversal{
		{"spiric-on-torus", sp, tor, false},
		{"elliptical-arc-on-cylinder", el, cyl, false},
		{"trimmed-spiric-sub-span", geom.TrimmedCurve3{Base: sp, Lo: 0.25, Hi: 0.6}, tor, false},
		{"nearly-straight-trimmed-spiric", geom.TrimmedCurve3{Base: sp, Lo: 0.5, Hi: 0.5001}, tor, false},
		{"arc-on-cylinder", arc, cyl, true},
		{"trimmed-arc-on-cylinder", geom.TrimmedCurve3{Base: arc, Lo: 0.2, Hi: 0.8}, cyl, true},
	}
}

// TestReverseSegmentCurveStaysOnItsSurface is the guard on the reversal invariant: reversing a loop
// segment may re-parameterise it but must never re-derive its SHAPE, so the reversed segment still lies
// on the surface it bounds. The three-point re-fit this replaced silently turned every non-circular rail
// into a CIRCLE THROUGH ITS ENDPOINTS — measured on the corpus, 1.245 off its own torus on D4/D5/D9/E3/E4,
// 0.0562 (1.1% of r) on N4's corner patch — and turned a nearly straight sub-rail into a degenerate zero
// arc at the origin (up to 25 off the face). A circular locus keeps the re-fit, which is exact for it.
func TestReverseSegmentCurveStaysOnItsSurface(t *testing.T) {
	t.Parallel()
	for _, tc := range onSurfaceReversalCases() {
		t.Run(tc.name, func(t *testing.T) {
			lo, hi := tc.curve.Domain()
			start, end := tc.curve.PointAt(lo), tc.curve.PointAt(hi)
			rev := reverseSegmentCurve(tc.curve, end, tc.curve.PointAt(0.5), start)
			assertReversedEnds(t, rev, end, start)
			if d := maxSegmentOffSurface(tc.surface, rev); d > 1e-9 {
				t.Fatalf("reversed %s lies %.6g off its own surface, want ≤ 1e-9 (the reversal re-derived its shape)", tc.name, d)
			}
			if _, isArc := rev.(geom.Arc3d); isArc != tc.refit {
				t.Fatalf("reversed %s is geom.Arc3d = %v, want %v (a circular locus keeps the three-point re-fit; nothing else may)", tc.name, isArc, tc.refit)
			}
		})
	}
}

// assertReversedEnds pins that the reversal actually reversed: the returned curve starts where the
// caller's traversal now starts and ends where it now ends (otherwise "on the surface" is vacuous).
func assertReversedEnds(t *testing.T, rev geom.Curve3, wantStart, wantEnd m.Point3) {
	t.Helper()
	lo, hi := rev.Domain()
	if d := rev.PointAt(lo).DistanceTo(wantStart); d > 1e-9 {
		t.Fatalf("reversed curve starts %.6g from the reversed traversal's start point", d)
	}
	if d := rev.PointAt(hi).DistanceTo(wantEnd); d > 1e-9 {
		t.Fatalf("reversed curve ends %.6g from the reversed traversal's end point", d)
	}
}

// maxSegmentOffSurface is the largest distance from 9 samples of c to the closest point on s.
func maxSegmentOffSurface(s geom.Surface, c geom.Curve3) float64 {
	lo, hi := c.Domain()
	worst := 0.0
	for k := 0; k <= 8; k++ {
		p := c.PointAt(lo + (hi-lo)*float64(k)/8)
		_, _, foot := geom.ClosestPointOnSurface(s, p)
		if d := p.DistanceTo(foot); d > worst {
			worst = d
		}
	}
	return worst
}
