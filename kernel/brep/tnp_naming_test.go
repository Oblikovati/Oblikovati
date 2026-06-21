// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"bytes"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Topological-naming regression harness (M31-F01, Oblikovati/Oblikovati#1151). These tests
// specify the BEHAVIOUR the topological-naming fixes must deliver: a reference key minted for
// an entity GENERATED inside a boolean (an intersection edge, a split-face fragment) must
// re-bind to the geometrically-same entity after an UNRELATED upstream edit.
//
// Today generated edges are named `Tok("brep","edge",i)` where i is the entity's rank in the
// stitch's coordinate-canonical ordering (kernel/brep/boolean_stitch.go), and split fragments
// `split#k` by ordinal piece index (boolean.go). That ordering is stable for edits that don't
// disturb the rank — but adding a feature at a LOWER coordinate shifts the rank of everything
// after it, so a downstream reference (a fillet on the slot rim) silently jumps to another edge
// or is lost. That is the classic topological naming problem.
//
// The perturbation here is therefore "add a lower-X prong to the comb tool" — it changes
// neither the tracked rim edge (x=3) nor the tracked middle top fragment (x∈[7,10])
// geometrically, yet because both the stitched-edge ordering and the split-piece ordering are
// coordinate-canonical, inserting geometry at a lower coordinate renumbers them today (verified:
// the rim edge moves edge#10→edge#17, the middle fragment split#1→split#2). The cross-edit
// invariance assertions are guarded with t.Skip referencing the fix issue (#1153 parent-named
// edges, #1154 parent-named fragments); everything else runs today and guards the harness
// itself (the geometry builds, the boolean succeeds, the tracked entity is found before AND
// after the edit). Un-skip the relevant assertion when its fix lands.

// rimMidpoint is the midpoint of the first prong's −X rim edge on the top face — born from
// (tool −X wall ∩ base top), geometrically fixed at x=3, z=4, y∈[0,10].
var rimMidpoint = math.P3(3, 5, 4)

// middleFragmentPoint sits on the top face's middle fragment (x∈[7,10], between the two prongs),
// which the added lower prong does not touch — so it is the SAME face region before and after.
var middleFragmentPoint = math.P3(8.5, 5, 4)

// combSlottedTop cuts a 16×10×4 base, in ONE difference, with a comb tool: two through-slots (in
// Y) across x∈[3,7] and x∈[10,12], each a blind slot from the top down to z=2. With withLowerProng
// the comb also carries a third prong across x∈[0.5,1.5] — a lower-X feature that renumbers the
// auto-named topology in the same boolean without moving the tracked rim edge or middle fragment.
// A single boolean (not sequential cuts) is essential: it is what forces all generated edges and
// all split pieces to share one coordinate-canonical numbering, the renumbering the fixes remove.
func combSlottedTop(t *testing.T, withLowerProng bool) *topo.Body {
	t.Helper()
	comb := union(t, boxNamed("p1", 3, -1, 2, 4, 12, 4), boxNamed("p2", 10, -1, 2, 2, 12, 4))
	wantVol := 16.0*10*4 - 4*10*2 - 2*10*2
	if withLowerProng {
		comb = union(t, comb, boxNamed("p0", 0.5, -1, 2, 1, 12, 4))
		wantVol -= 1 * 10 * 2
	}
	res := cutDifference(t, boxNamed("base", 0, 0, 0, 16, 10, 4), comb)
	if v := vol(res); stdmath.Abs(v-wantVol) > 1e-6 {
		t.Fatalf("comb-slotted body volume = %g, want %g (withLowerProng=%v)", v, wantVol, withLowerProng)
	}
	return res
}

// cutDifference subtracts tool from b, failing the test on error.
func cutDifference(t *testing.T, b, tool *topo.Body) *topo.Body {
	t.Helper()
	res, err := brep.Boolean(brep.Difference, b, tool)
	if err != nil {
		t.Fatalf("difference failed: %v", err)
	}
	return res
}

// union fuses two bodies into one (used to build a multi-prong comb tool), failing on error.
func union(t *testing.T, a, b *topo.Body) *topo.Body {
	t.Helper()
	res, err := brep.Boolean(brep.Union, a, b)
	if err != nil {
		t.Fatalf("union failed: %v", err)
	}
	return res
}

// edgeMidpoint returns the point at the middle of an edge's curve domain.
func edgeMidpoint(e *topo.Edge) math.Point3 {
	lo, hi := e.Geometry().Domain()
	return e.Geometry().PointAt((lo + hi) / 2)
}

// edgeAtMidpoint finds the edge whose curve midpoint coincides with p (within tol) — the
// geometry-anchored way to locate "the same" edge across two independent rebuilds.
func edgeAtMidpoint(b *topo.Body, p math.Point3, tol float64) (*topo.Edge, bool) {
	for _, e := range b.Edges() {
		if float64(edgeMidpoint(e).DistanceTo(p)) <= tol {
			return e, true
		}
	}
	return nil, false
}

// topFragmentAt finds the +Z-facing face whose range box contains p — the top-face fragment
// covering that point.
func topFragmentAt(b *topo.Body, p math.Point3) (*topo.Face, bool) {
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Dot(math.V3(0, 0, 1)) > 0.9 && f.RangeBox().Contains(p) {
			return f, true
		}
	}
	return nil, false
}

// TestGeneratedEdgeKeySurvivesUpstreamEdit (G1, #1153): the rim edge at (3,5,4) is born from the
// boolean. Adding a lower-X prong to the comb tool — which touches neither that edge nor its
// faces — must not change the edge's reference key.
func TestGeneratedEdgeKeySurvivesUpstreamEdit(t *testing.T) {
	const tol = 1e-9

	before := combSlottedTop(t, false)
	e1, ok := edgeAtMidpoint(before, rimMidpoint, tol)
	if !ok {
		t.Fatalf("rim edge not found at %v before the edit (harness invalid)", rimMidpoint)
	}
	key := e1.ReferenceKey()

	after := combSlottedTop(t, true) // lower-X prong added; the rim edge stays at (3,5,4)
	e2, ok := edgeAtMidpoint(after, rimMidpoint, tol)
	if !ok {
		t.Fatalf("rim edge not found at %v after the edit (harness invalid)", rimMidpoint)
	}

	// Fixed in F03 (#1153): the rim edge is named by its parent faces (base top × tool wall),
	// so the lower-X prong no longer renumbers it.
	if !bytes.Equal(key, e2.ReferenceKey()) {
		t.Errorf("generated edge key changed across an unrelated edit:\n before = %q\n after  = %q",
			key, e2.ReferenceKey())
	}
}

// TestSplitFaceFragmentKeySurvivesUpstreamEdit (G2, #1154): the top face is split by the comb
// into three fragments; the middle one (x∈[7,10]) is untouched by the added lower prong, so its
// reference key must not change.
func TestSplitFaceFragmentKeySurvivesUpstreamEdit(t *testing.T) {
	before := combSlottedTop(t, false)
	f1, ok := topFragmentAt(before, middleFragmentPoint)
	if !ok {
		t.Fatalf("middle top fragment not found over %v before the edit (harness invalid)", middleFragmentPoint)
	}
	key := f1.ReferenceKey()

	after := combSlottedTop(t, true)
	f2, ok := topFragmentAt(after, middleFragmentPoint)
	if !ok {
		t.Fatalf("middle top fragment not found over %v after the edit (harness invalid)", middleFragmentPoint)
	}

	// Fixed in F04 (#1154): the middle fragment is named by the cutting faces bordering it
	// (the two inner prong walls), a set the lower prong does not change, so its key is stable.
	if !bytes.Equal(key, f2.ReferenceKey()) {
		t.Errorf("split-face fragment key changed across an unrelated edit:\n before = %q\n after  = %q",
			key, f2.ReferenceKey())
	}
}

// TestSurvivingFaceKeyIsTheWorkingBaseline is the positive control that proves the harness can
// tell a robust key from a fragile one: the base's −X side face is untouched by either slot, so
// (K1a) its key already survives the very same edit today. If this regressed, the cross-edit
// comparisons above would be meaningless.
func TestSurvivingFaceKeyIsTheWorkingBaseline(t *testing.T) {
	leftSide := math.V3(-1, 0, 0)

	before := combSlottedTop(t, false)
	f1 := faceWithNormal(before, leftSide)
	if f1 == nil {
		t.Fatal("no −X side face before the edit (harness invalid)")
	}
	key := f1.ReferenceKey()

	after := combSlottedTop(t, true)
	if _, ok := after.FindFaceByKey(key); !ok {
		t.Fatal("an untouched side face's key did not survive the edit — the K1a baseline is broken")
	}
}
