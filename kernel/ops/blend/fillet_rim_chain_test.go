// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// rimChainTestRing builds a closed footprint-rim ring on the unit circle: n points and n leaving
// curves, curve k running ring[k] → ring[(k+1)%n] as an exact sub-span of the full circle — the
// same shape bossRimRing emits for a subdivided intact-boss footprint.
func rimChainTestRing(t *testing.T, n int) ([]math.Point3, []geom.Curve3) {
	t.Helper()
	circle, err := geom.NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 1, 0, 2*stdPi())
	if err != nil {
		t.Fatalf("test circle: %v", err)
	}
	return sampleCurveNTrimmed(circle, n, false)
}

func stdPi() float64 { return 3.141592653589793 }

// TestRingLeavingCurveMatchesTraversal proves the ring leaving-curve indexing in BOTH traversal
// directions — the invariant transformLoop/addEdgeInserts rely on when they hang the footprint
// conic's sub-spans onto the wall rim's sub-edges (t3-plane-sliver-report.md): the k-th VISITED
// ring point's leaving curve must START at that point and END at the next visited point. Forward,
// visited order is the ring order; reversed (orientedInserts), it is seam then the tail reversed,
// and the leaving curve is the REVERSE of the forward curve that ARRIVED at the visited point. A
// mutation that drops the reversal, or the (n-1-k) shift, moves an endpoint a whole chord away.
func TestRingLeavingCurveMatchesTraversal(t *testing.T) {
	t.Parallel()
	const n = 7
	ring, chain := rimChainTestRing(t, n)
	for _, rev := range []bool{false, true} {
		visited := ringVisitOrder(ring, rev)
		for k := range n {
			c := ringLeavingCurve(chain, k, rev)
			start, end := c.PointAt(0), c.PointAt(1)
			next := visited[(k+1)%n]
			if float64(start.DistanceTo(visited[k])) > 1e-12 {
				t.Fatalf("rev=%v k=%d: leaving curve starts %v, want the visited point %v", rev, k, start, visited[k])
			}
			if float64(end.DistanceTo(next)) > 1e-12 {
				t.Fatalf("rev=%v k=%d: leaving curve ends %v, want the NEXT visited point %v", rev, k, end, next)
			}
		}
	}
}

// ringVisitOrder is the order transformLoop visits a subdivided rim's points in: the seam first,
// then the inserts — forward as stored, reversed per orientedInserts (tail reversed, seam kept first).
func ringVisitOrder(ring []math.Point3, rev bool) []math.Point3 {
	if !rev {
		return ring
	}
	out := append([]math.Point3{ring[0]}, orientedInserts(ring[1:], true)...)
	return out
}

// TestReverseLeavingChainKeepsSegmentsOnTheirPoints proves reverseLeavingChain re-indexes a whole
// chain to the REVERSED ring (orientRingChainToEdge's transform): after reversing both, curve k must
// still run newRing[k] → newRing[(k+1)%n]. This is the store-time half of the same one-slot-shift
// rule TestRingLeavingCurveMatchesTraversal proves at consume time; a mutation that reverses the
// points without re-indexing the chain leaves every segment hanging between the WRONG pair.
func TestReverseLeavingChainKeepsSegmentsOnTheirPoints(t *testing.T) {
	t.Parallel()
	const n = 6
	ring, chain := rimChainTestRing(t, n)
	newRing := append([]math.Point3{ring[0]}, reversePts(ring[1:])...)
	newChain := reverseLeavingChain(chain)
	for k := range n {
		start, end := newChain[k].PointAt(0), newChain[k].PointAt(1)
		if float64(start.DistanceTo(newRing[k])) > 1e-12 {
			t.Fatalf("k=%d: reversed-chain curve starts %v, want reversed-ring point %v", k, start, newRing[k])
		}
		if float64(end.DistanceTo(newRing[(k+1)%n])) > 1e-12 {
			t.Fatalf("k=%d: reversed-chain curve ends %v, want next reversed-ring point %v", k, end, newRing[(k+1)%n])
		}
	}
}
