// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/test-utilities/brepfixture"
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
// It exists only in tests — production must never pair on value, because two edges are two edges.
//
// The tolerance is the WELD, and NOT because the operands are the same computation: this function is
// only ever reached when isReverseTwin has already said no, so its two operands are always two
// different edges — independent sources, which the ADR-0042 classification would put at Sew(). The
// weld is right for a different reason. The comparison stands in for the retired `a.curve == b.curve`,
// which was EXACT, so ANY tolerance already over-approximates its reach; and a gate that asserts ZERO
// must not trip on two genuinely distinct boundaries that happen to pass within a generous sew gap.
// Every pair this must catch agrees to 0 exactly, so the tight class has five orders of headroom.
func valueOnlyReverseRun(a, b loopEdge, res geom.Resolution) bool {
	if isReverseTwin(a, b) {
		return false
	}
	return brepfixture.StretchWalkedBack(a.curve, a.t0, a.t1, b.curve, b.t0, b.t1, res.Weld())
}

// TestNoCorpusBodyKeepsAPairThatWalksAStretchBack is the gate the narrowing's licence rests on. Over
// chartCorpus's FIVE bodies — the package's cocylindrical-merge corpus, whose "cocylindrical boss on
// a wall" row is the same host-plus-planed-boss body the end-to-end row drives — no face keeps a
// cyclically adjacent pair that walks one stretch straight back, whether the two are ONE edge (a slit
// dropSeamSlits should have removed) or TWO (the pair only a value comparison would have caught).
//
// The two counts are complementary: a surviving one-edge slit is what neutering isReverseTwin
// produces, and a surviving two-edge pair is the case the narrowing gave up. Both are zero, and the
// second is the whole reason the narrowing is safe.
//
// It is a RATCHET, not an exhaustive sweep. The corpus-wide figures quoted at isReverseTwin were
// measured over ./kernel/... and ./model/..., and this row gates five of those bodies; one more body
// that drops a slit on every run — TestCocylindricalCapOnWallIsOneAnalyticFace — lives in
// kernel/ops/boolean and is not gated here. brepfixture.ReversedRunPairs takes a *topo.Face and no
// package-private state, so gating it there is an import and three lines whenever that is wanted.
func TestNoCorpusBodyKeepsAPairThatWalksAStretchBack(t *testing.T) {
	t.Parallel()
	for _, tc := range chartCorpus(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertNoStretchIsWalkedBack(t, tc.body)
		})
	}
}

// assertNoStretchIsWalkedBack checks every face of one body.
func assertNoStretchIsWalkedBack(t *testing.T, b *topo.Body) {
	t.Helper()
	for _, f := range b.Faces() {
		assertFaceKeepsNoReverseRun(t, f)
	}
}

// assertFaceKeepsNoReverseRun names which of the two kinds it found, because they mean different
// things: one edge is a slit that survived, two edges is a boundary the value comparison would have
// deleted and the identity test correctly kept.
func assertFaceKeepsNoReverseRun(t *testing.T, f *topo.Face) {
	t.Helper()
	p, found := brepfixture.FirstReversedRun(f, geom.ResolutionForBox(f.RangeBox()).Weld())
	if !found {
		return
	}
	if p.OneEdge {
		t.Errorf("face %q keeps ONE edge walked both ways at loop %d position %d: a slit dropSeamSlits "+
			"did not remove", string(f.ReferenceKey()), p.Loop, p.At)
		return
	}
	t.Errorf("face %q keeps TWO edges walking one stretch back at loop %d position %d: the identity "+
		"narrowing gave this pair up, so it must not be a slit — re-open Oblikovati#3521",
		string(f.ReferenceKey()), p.Loop, p.At)
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
