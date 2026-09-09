// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Keeping the identity narrowing of isReverseTwin honest (Oblikovati#3521).
//
// isReverseTwin was narrowed from a curve VALUE comparison to edge IDENTITY, and the licence for that
// narrowing was a measurement: over the corpus, no pair it refuses is a pair a value comparison would
// have caught. A measurement in a commit message is a claim; these rows make it a gate, so the day
// emitImprintRun starts producing a real value-only twin the suite says so instead of a doc comment
// still saying zero.
//
// The gate is on the RESULT, which is where the harm would be. A removal that SHOULD have fired and
// did not leaves the slit in the shipped body's loops — that is exactly what neutering isReverseTwin
// produces (the merged wall's outer loop grows from the bare rim to the rim plus a ruling walked both
// ways), so an adjacent pair that walks one stretch straight back is observable without instrumenting
// the merge.

// valueOnlyReverseRun reports the pair the retired comparison would have paired and the identity test
// refuses: two edges that walk the same stretch of space in opposite directions WITHOUT being one edge.
// It exists only here — production must never pair on value, because the two edges are two edges.
func valueOnlyReverseRun(a, b loopEdge, res geom.Resolution) bool {
	if isReverseTwin(a, b) {
		return false
	}
	return walksTheSameStretchBack(a, b, res)
}

// walksTheSameStretchBack samples b's traversal against a's reversed and asks whether they coincide.
// The comparison is a WELD: if the two are one stretch walked twice they are one curve evaluated
// twice, which is the same-computation class (ADR-0042). Nine stations, because two distinct curves
// that share both endpoints (the polyline seam against a straight one) must be separated by an
// INTERIOR sample, and one interior sample can land on a crossing.
func walksTheSameStretchBack(a, b loopEdge, res geom.Resolution) bool {
	if a.curve == nil || b.curve == nil {
		return false
	}
	const stations = 8
	for i := 0; i <= stations; i++ {
		s := float64(i) / stations
		pa := a.curve.PointAt(a.t0 + (a.t1-a.t0)*s)
		pb := b.curve.PointAt(b.t1 + (b.t0-b.t1)*s)
		if float64(pa.DistanceTo(pb)) > res.Weld() {
			return false
		}
	}
	return true
}

// TestNoCorpusBodyKeepsAPairThatWalksAStretchBack is the gate the narrowing's licence rests on. Over
// the merge corpus — the bodies whose walls the cocylindrical merge actually joins — no face keeps a
// cyclically adjacent pair that walks one stretch straight back, whether the two are ONE edge (a slit
// dropSeamSlits should have removed) or TWO (the pair only a value comparison would have caught).
//
// The two counts are complementary: a surviving one-edge slit is what neutering isReverseTwin
// produces, and a surviving two-edge pair is the case the narrowing gave up. Both are zero, and the
// second is the whole reason the narrowing is safe.
func TestNoCorpusBodyKeepsAPairThatWalksAStretchBack(t *testing.T) {
	t.Parallel()
	for _, tc := range chartCorpus(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNoStretchIsWalkedBack(t, tc.body)
		})
	}
}

// assertNoStretchIsWalkedBack checks every loop of every face of one body.
func assertNoStretchIsWalkedBack(t *testing.T, b *topo.Body) {
	t.Helper()
	for _, f := range b.Faces() {
		res := geom.ResolutionForBox(f.RangeBox())
		for _, l := range f.Loops() {
			assertLoopKeepsNoReverseRun(t, f, loopEdgesOf(l), res)
		}
	}
}

// assertLoopKeepsNoReverseRun names which of the two kinds it found, because they mean different
// things: one edge is a slit that survived, two edges is a boundary the value comparison would have
// deleted and the identity test correctly kept.
func assertLoopKeepsNoReverseRun(t *testing.T, f *topo.Face, edges []loopEdge, res geom.Resolution) {
	t.Helper()
	for i, e := range edges {
		next := edges[(i+1)%len(edges)]
		if isReverseTwin(e, next) {
			t.Errorf("face %q keeps ONE edge walked both ways at position %d: a slit dropSeamSlits did "+
				"not remove", string(f.ReferenceKey()), i)
		}
		if valueOnlyReverseRun(e, next, res) {
			t.Errorf("face %q keeps TWO edges walking one stretch back at position %d: the identity "+
				"narrowing gave this pair up, so it must not be a slit — re-open Oblikovati#3521",
				string(f.ReferenceKey()), i)
		}
	}
}

// TestTheOnlyValueOnlyPairIsTheOneValuePairingWouldGetWRONG is the other half of the licence, and it
// is the stronger half. Search the whole corpus for a pair the value comparison would catch and the
// identity test refuses and you find exactly two, both here: the two DIFFERENT seam edges of
// seamWalkedWall's twoSeams fixture, carrying identical polylines. Value-pairing would delete that
// boundary. So the narrowing does not merely miss nothing — the one place where value and identity
// disagree is the place where value is wrong.
func TestTheOnlyValueOnlyPairIsTheOneValuePairingWouldGetWRONG(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForSize(10)
	two := seamWalkedWall(t, polylineSeam, true)
	up, down := two[1], two[3]
	if isReverseTwin(up, down) {
		t.Fatal("two different edges of equal shape were paired by identity; they are two edges")
	}
	if !valueOnlyReverseRun(up, down, res) {
		t.Fatal("the two equal polyline seams do not read as one stretch walked back; the row cannot " +
			"show what value-pairing would have done, and is vacuous")
	}
	if got := withoutSlitPairs([]loopEdge{up, down}); len(got) != 2 {
		t.Errorf("the pair value-pairing would delete was dropped after all (%d edges left, want 2)", len(got))
	}
	one := seamWalkedWall(t, polylineSeam, false)
	if !isReverseTwin(one[1], one[3]) {
		t.Error("ONE polyline seam walked both ways is not paired by identity; the narrowing would be a no-op")
	}
}
