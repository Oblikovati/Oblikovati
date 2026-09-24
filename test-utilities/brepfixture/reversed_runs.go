// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Finding a boundary that walks a stretch of space and immediately walks it back (Oblikovati#3521).
//
// A boolean's result must keep none. This lives here rather than in a package's own _test.go files
// for the reason the package doc gives: kernel/brep's internal and external test packages both need
// it, kernel/ops will want it for the merge bodies that live there, and Go shares no test helper
// across packages. It is a QUERY — it reports what it found and asserts nothing — so it cannot make
// a test pass for the wrong reason.
//
// It must stay a TEST-only import for kernel packages: kernel/ops imports kernel/brep and this
// package imports kernel/subd, so a production edge from kernel/brep to here closes a cycle that
// breaks kernel/subd's own test build. From a _test.go file there is no cycle.

// ReversedRunPair names one cyclically adjacent pair of one loop that retraces its predecessor.
// OneEdge separates the two kinds, which mean different things: ONE edge used both ways is a SLIT
// the boolean should have dropped, while TWO different edges walking one stretch back is the pair a
// curve-VALUE comparison would have merged and an edge-IDENTITY comparison correctly keeps.
type ReversedRunPair struct {
	Loop, At int
	OneEdge  bool
}

// FirstReversedRun returns the first such pair on a face, in loop and position order. It reports the
// first rather than all of them because a result must have NONE: the first names the defect, and a
// list would only repeat one failure per loop.
//
// Example:
//
//	if p, ok := brepfixture.FirstReversedRun(wall, res.Weld()); ok {
//		t.Errorf("the merged wall keeps a boundary that bounds nothing: %+v", p)
//	}
func FirstReversedRun(f *topo.Face, tol float64) (ReversedRunPair, bool) {
	for li, l := range f.Loops() {
		if p, ok := firstLoopReversedRun(li, l.EdgeUses(), tol); ok {
			return p, true
		}
	}
	return ReversedRunPair{}, false
}

// firstLoopReversedRun scans one loop's cyclically adjacent pairs. A loop of a single use has no
// pair — comparing that use with itself asks whether a curve retraces itself, a different question.
func firstLoopReversedRun(li int, uses []*topo.EdgeUse, tol float64) (ReversedRunPair, bool) {
	for i, u := range uses {
		next := uses[(i+1)%len(uses)]
		if len(uses) < 2 || !UsesWalkOneStretchBack(u, next, tol) {
			continue
		}
		return ReversedRunPair{Loop: li, At: i, OneEdge: u.Edge() == next.Edge()}, true
	}
	return ReversedRunPair{}, false
}

// UsesWalkOneStretchBack reports whether b retraces the stretch a walked, backwards.
//
// Example: brepfixture.UsesWalkOneStretchBack(up, down, res.Weld()) // true for a seam walked twice
func UsesWalkOneStretchBack(a, b *topo.EdgeUse, tol float64) bool {
	a0, a1 := useTraversal(a)
	b0, b1 := useTraversal(b)
	return StretchWalkedBack(a.Edge().Geometry(), a0, a1, b.Edge().Geometry(), b0, b1, tol)
}

// StretchWalkedBack samples the second traversal against the first reversed and reports whether they
// coincide everywhere within tol. Nine stations, because two distinct curves that share both
// endpoints — a polyline seam against the straight one between the same vertices — are separated
// only by an INTERIOR sample, and a single interior sample can land on a crossing.
//
// A curveless edge — which a synthesized boundary can be — retraces nothing.
//
// Example: brepfixture.StretchWalkedBack(c, 0, 1, c, 1, 0, res.Weld()) // true: one curve, both ways
func StretchWalkedBack(ca geom.Curve3, a0, a1 float64, cb geom.Curve3, b0, b1, tol float64) bool {
	if ca == nil || cb == nil {
		return false
	}
	const stations = 8
	for i := 0; i <= stations; i++ {
		s := float64(i) / stations
		pa := ca.PointAt(a0 + (a1-a0)*s)
		pb := cb.PointAt(b1 + (b0-b1)*s)
		if float64(pa.DistanceTo(pb)) > tol {
			return false
		}
	}
	return true
}

// useTraversal is the parameter span one edge use walks on its edge's curve, oriented to its loop:
// from the edge's start vertex to its end vertex, swapped when the use is reversed, and the curve's
// whole domain for a closed edge (one whose two vertices coincide — a full seam circle).
func useTraversal(u *topo.EdgeUse) (float64, float64) {
	e := u.Edge()
	c := e.Geometry()
	t0, t1 := c.Domain()
	if e.StartVertex() != e.EndVertex() {
		t0, _ = geom.CurveParamAtPoint3(c, e.StartVertex().Point())
		t1, _ = geom.CurveParamAtPoint3(c, e.EndVertex().Point())
	}
	if u.Reversed() {
		return t1, t0
	}
	return t0, t1
}
