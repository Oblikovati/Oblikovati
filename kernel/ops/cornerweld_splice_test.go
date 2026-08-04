// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// Gates for the chain-aware loop splice — the half of the C4 face-split variant that lives in the host
// retrim. A rim continuation makes a bite span SEVERAL loop edges with SEVERAL rail pieces; these pin that
// the run recogniser accepts exactly the contiguous case and declines every other, so a mis-grouped bite
// floors instead of silently re-clipping the wrong span.

// TestContiguousSegRunAcceptsOnlyAContiguousRing pins the run recogniser: one index, an adjacent pair, and a
// wrapping pair are runs; a scattered pair, a duplicate and an out-of-range index are not.
func TestContiguousSegRunAcceptsOnlyAContiguousRing(t *testing.T) {
	cases := []struct {
		name   string
		idx    []int
		n      int
		wantOK bool
		lo, hi int
	}{
		{"single", []int{2}, 5, true, 2, 2},
		{"adjacent", []int{1, 2}, 5, true, 1, 2},
		{"adjacent reversed input", []int{2, 1}, 5, true, 1, 2},
		{"wrapping", []int{4, 0}, 5, true, 4, 0},
		{"triple", []int{1, 2, 3}, 5, true, 1, 3},
		{"scattered", []int{0, 2}, 5, false, 0, 0},
		{"duplicate", []int{2, 2}, 5, false, 0, 0},
		{"out of range", []int{5}, 5, false, 0, 0},
		{"empty", nil, 5, false, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run, ok := contiguousSegRun(tc.idx, tc.n)
			if ok != tc.wantOK {
				t.Fatalf("contiguousSegRun(%v, %d) ok=%v, want %v", tc.idx, tc.n, ok, tc.wantOK)
			}
			if ok && (run.lo != tc.lo || run.hi != tc.hi) {
				t.Fatalf("run = %d..%d, want %d..%d", run.lo, run.hi, tc.lo, tc.hi)
			}
		})
	}
}

// TestSegRunMembershipWraps pins the wrap-aware membership the ring rebuild uses to drop a consumed run.
func TestSegRunMembershipWraps(t *testing.T) {
	wrap := segRun{lo: 3, hi: 1}
	for _, k := range []int{3, 4, 0, 1} {
		if !wrap.has(k) {
			t.Fatalf("wrapping run 3..1 must contain %d", k)
		}
	}
	if wrap.has(2) {
		t.Fatal("wrapping run 3..1 must not contain 2")
	}
	if (segRun{lo: -1, hi: -1}).has(0) {
		t.Fatal("the empty run must contain nothing")
	}
}

// zzSquareRing is a unit square loop, used to exercise the splice geometry without a topo body.
func zzSquareRing() []endSeg {
	p := []math.Point3{math.P3(0, 0, 0), math.P3(4, 0, 0), math.P3(4, 4, 0), math.P3(0, 4, 0)}
	out := make([]endSeg, 4)
	for i := range p {
		out[i] = endSeg{from: p[i], to: p[(i+1)%4]}
	}
	return out
}

// TestSpliceSingleRunDeclinesOnAFootOffTheFlank is the do-no-harm floor for the single-arm chain splice: a
// rail whose foot is not on the flanking edge's supporting line cannot be welded, so it declines rather than
// snapping the flank to it.
func TestSpliceSingleRunDeclinesOnAFootOffTheFlank(t *testing.T) {
	segs := zzSquareRing()
	rails := []endSeg{{from: math.P3(3, 1, 0), to: math.P3(1, 1, 0)}} // both feet off the ring's edges
	if _, ok := spliceSingleRun(segs, segRun{lo: 1, hi: 1}, rails, 1e-9); ok {
		t.Fatal("a rail chain whose feet leave the flanking edges must decline")
	}
}

// TestSpliceSingleRunDeclinesWithOnlyOneFlank pins the small-loop guard: when the consumed run leaves a single
// shared flank there is no independent edge to re-terminate on each side, so the single-arm splice declines
// (the two-arm corner splice has its own reterminateBothEnds path for that shape).
func TestSpliceSingleRunDeclinesWithOnlyOneFlank(t *testing.T) {
	segs := zzSquareRing()
	rails := []endSeg{{from: math.P3(3, 0, 0), to: math.P3(3, 4, 0)}}
	if _, ok := spliceSingleRun(segs, segRun{lo: 1, hi: 3}, rails, 1e-9); ok {
		t.Fatal("a run leaving one shared flank must decline in the single-arm splice")
	}
}

// TestSpliceSingleRunOrientsTheChainToTheLoop pins the orientation rule: the chain is traversed so its first
// foot sits on the PRECEDING flank and its last on the FOLLOWING one, whichever way the builder registered it.
// Getting this backwards would emit a self-crossing loop that meshes inside out.
func TestSpliceSingleRunOrientsTheChainToTheLoop(t *testing.T) {
	segs := zzSquareRing()
	forward := []endSeg{{from: math.P3(3, 0, 0), to: math.P3(3, 4, 0)}}
	loop, ok := spliceSingleRun(segs, segRun{lo: 1, hi: 1}, forward, 1e-9)
	if !ok {
		t.Fatal("the forward chain must splice")
	}
	reversed := []endSeg{{from: math.P3(3, 4, 0), to: math.P3(3, 0, 0)}}
	loopRev, okRev := spliceSingleRun(segs, segRun{lo: 1, hi: 1}, reversed, 1e-9)
	if !okRev {
		t.Fatal("the reversed chain must splice too — the layer orients it, the builder need not")
	}
	if len(loop.pts) != len(loopRev.pts) {
		t.Fatalf("orientation changed the loop size: %d vs %d", len(loop.pts), len(loopRev.pts))
	}
	for i := range loop.pts {
		if loop.pts[i] != loopRev.pts[i] {
			t.Fatalf("point %d differs (%v vs %v): the same chain must splice identically either way", i, loop.pts[i], loopRev.pts[i])
		}
	}
}
