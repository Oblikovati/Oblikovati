// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The end-to-end row for the seam a cocylindrical merge orphans (Oblikovati#3521).
//
// Every other statement about dropSeamSlits drives it through a hand-built fixture, and a unit fixture
// is not the corpus: "a bug fix adds the failing input to the operation's corpus and makes the general
// pipeline pass it". This row drives a REAL boolean and asserts the RESULT — the merged wall's loops,
// both parents' keys on it, and the body's volume — never the internal call sequence, so a restructured
// merge (Oblikovati#3523) is measured against the same body.

// The boss standing on a host band of ONE radius, with a flat planed onto the boss.
const (
	seamSlitRadius  = 3.0  // host and boss share this radius, so their walls are one surface
	seamSlitHostTop = 6.0  // the host band is z ∈ [0, 6]
	seamSlitBossTop = 10.0 // the boss stands on it, z ∈ [6, 10]
	seamSlitPlaneX  = 2.4  // planed off at x = 2.4, so the boss shares only an ARC of the z = 6 rim
)

// TestBossOnACocylindricalHostDropsTheOrphanedSeam is the corpus body whose merge orphans a seam, and
// the one that shows the identity narrowing of isReverseTwin is LIVE rather than a no-op.
//
// The host wraps its cylinder, so its wall walks its seam twice; the boss shares only an arc of the
// z = 6 rim, so splitAtSharedRunEnds cuts the host's rim and dissolving the shared arc leaves the
// host's two seam traversals adjacent. Nothing else in the corpus does that — two bands that share a
// WHOLE rim (rod on rod, a bore continuing a bore) keep the other band's edges between the two
// traversals, and their scans find no twin at all.
//
// With the slit left in, the merged wall's outer loop is the bottom rim PLUS that ruling walked up and
// straight back down: a slit dangling into the face's interior, which a user can select and a fillet
// would try to round.
func TestBossOnACocylindricalHostDropsTheOrphanedSeam(t *testing.T) {
	t.Parallel()
	rec := &diag.Recorder{}
	host, boss := cocylindricalHost(t), planedCocylindricalBoss(t)
	hostKey, bossKey := cylinderWallKey(t, host), cylinderWallKey(t, boss)
	body, err := brep.BooleanDiag(brep.Union, host, boss, rec)
	if err != nil {
		t.Fatalf("union of the host and the boss standing on it: %v", err)
	}
	checkSolid(t, "boss on a cocylindrical host", body, bossOnHostVolume())
	wall := theMergedCylinderWall(t, body)
	assertOuterLoopIsTheBareRim(t, wall)
	assertNoLoopWalksAnEdgeStraightBack(t, wall)
	assertMergedWallResolves(t, body, hostKey, bossKey)
	assertSeamSlitDropWasRecorded(t, rec)
}

// cocylindricalHost is the wrapping band the seam belongs to: z ∈ [0, 6] at the shared radius.
func cocylindricalHost(t *testing.T) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), seamSlitRadius, seamSlitHostTop)
	if err != nil {
		t.Fatalf("host cylinder: %v", err)
	}
	return b
}

// planedCocylindricalBoss is the boss standing on the host, planed off at x = seamSlitPlaneX so it
// shares only an ARC of the rim it stands on — which is what orphans the host's seam.
func planedCocylindricalBoss(t *testing.T) *topo.Body {
	t.Helper()
	stock, err := brep.SolidCylinder(math.P3(0, 0, seamSlitHostTop), math.V3(0, 0, 1),
		seamSlitRadius, seamSlitBossTop-seamSlitHostTop)
	if err != nil {
		t.Fatalf("boss cylinder: %v", err)
	}
	planer, err := brep.SolidBlock(math.P3(seamSlitPlaneX, -4, seamSlitHostTop-1),
		math.P3(5, 4, seamSlitBossTop+1), "planer")
	if err != nil {
		t.Fatalf("planer block: %v", err)
	}
	boss, err := brep.Boolean(brep.Difference, stock, planer)
	if err != nil {
		t.Fatalf("planing the boss: %v", err)
	}
	return boss
}

// theMergedCylinderWall is the body's single cylinder face. That there is exactly ONE is the merge's
// own post-condition, and the seam-slit assertions rest on it.
func theMergedCylinderWall(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	var walls []*topo.Face
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			walls = append(walls, f)
		}
	}
	if len(walls) != 1 {
		t.Fatalf("the body has %d cylinder walls, want 1: the host band and the boss standing on it lie "+
			"on ONE cylinder and share the arc of rim between them", len(walls))
	}
	return walls[0]
}

// assertOuterLoopIsTheBareRim: the merged wall's outer loop is the bottom rim circle and NOTHING else.
// This is the assertion the orphaned seam breaks — left in, the loop is that circle plus the host's
// seam ruling walked up and straight back down, three edges instead of one.
func assertOuterLoopIsTheBareRim(t *testing.T, f *topo.Face) {
	t.Helper()
	if n := len(f.Loops()); n != 2 {
		t.Fatalf("the merged wall has %d loops, want 2 — a band that wraps its cylinder has two boundary "+
			"components: the bottom rim, and the planed boss's top boundary", n)
	}
	uses := f.Loops()[0].EdgeUses()
	if len(uses) != 1 {
		t.Fatalf("the merged wall's outer loop walks %d edges, want 1 (the whole bottom rim circle); a "+
			"second and third are the orphaned seam walked both ways", len(uses))
	}
	if _, ok := uses[0].Edge().Geometry().(geom.Circle); !ok {
		t.Errorf("the merged wall's outer loop walks a %T, want the bottom rim circle",
			uses[0].Edge().Geometry())
	}
}

// assertNoLoopWalksAnEdgeStraightBack states the slit invariant on the RESULT: no loop may use one edge
// and then immediately use it the other way, because a curve walked straight back bounds nothing.
func assertNoLoopWalksAnEdgeStraightBack(t *testing.T, f *topo.Face) {
	t.Helper()
	for _, l := range f.Loops() {
		assertLoopHasNoSlit(t, l)
	}
}

// assertLoopHasNoSlit checks one loop for a cyclically adjacent pair that is one edge walked both ways.
func assertLoopHasNoSlit(t *testing.T, l *topo.Loop) {
	t.Helper()
	uses := l.EdgeUses()
	for i, u := range uses {
		next := uses[(i+1)%len(uses)]
		if u.Edge() == next.Edge() && u.Reversed() != next.Reversed() {
			t.Errorf("the loop walks edge %v up at position %d and straight back down: a slit dangling "+
				"into the merged face's interior", u.Edge().Lineage(), i)
		}
	}
}

// assertMergedWallResolves: dropping the seam must not drop an identity. Both parents' wall keys still
// resolve to the merged wall (ADR-0043 — a pick must survive the operation that consumed it).
func assertMergedWallResolves(t *testing.T, b *topo.Body, keys ...[]byte) {
	t.Helper()
	for _, k := range keys {
		f, ok := b.FindFaceByKey(k)
		if !ok {
			t.Fatalf("the merged wall does not resolve parent key %q", string(k))
		}
		if _, isCyl := f.Geometry().(geom.Cylinder); !isCyl {
			t.Errorf("parent key %q resolved to a %T, not the merged cylinder wall", string(k), f.Geometry())
		}
	}
}

// cylinderWallKey is the reference key of a body's single cylinder wall.
func cylinderWallKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			return f.ReferenceKey()
		}
	}
	t.Fatal("the body has no cylinder wall")
	return nil
}

// assertSeamSlitDropWasRecorded: the census says the merge ran and how much of what reached it the slit
// test could pair at all, and the drop note says the removal FIRED. They are the pair of positive
// markers that separate "no slit was there" from "the slit test could not have seen one" — the gap
// Oblikovati#3521 is about, since a source-less edge is nobody's twin.
func assertSeamSlitDropWasRecorded(t *testing.T, rec *diag.Recorder) {
	t.Helper()
	if !recordedDetail(rec, brep.CodeMergeEdgeSources, "reaching the cocylindrical merge") {
		t.Error("no merge-edge-source census on the recorder: the merge did not run, or stopped counting")
	}
	if !recordedDetail(rec, brep.CodeSeamSlitDropped, "dropped 2 orphaned seam edge") {
		t.Errorf("the merge did not report dropping the host's two seam traversals; got %v", rec.Records())
	}
}

// recordedDetail reports whether the recorder carries an Info of the given code naming the fragment.
func recordedDetail(rec *diag.Recorder, code diag.Code, fragment string) bool {
	for _, d := range rec.Records() {
		if d.Code == code && d.Severity == diag.Info && strings.Contains(d.Detail, fragment) {
			return true
		}
	}
	return false
}

// bossOnHostVolume is the analytic volume: the host band, plus the boss band less the circular segment
// the planer cut off it.
func bossOnHostVolume() float64 {
	r, d := seamSlitRadius, seamSlitPlaneX
	segment := r*r*stdmath.Acos(d/r) - d*stdmath.Sqrt(r*r-d*d)
	return stdmath.Pi*r*r*seamSlitHostTop + (stdmath.Pi*r*r-segment)*(seamSlitBossTop-seamSlitHostTop)
}
