// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The coaxial ball-and-rod corpus (#2036/#2061, ADR-0045, ADR-0061 stage 4). A sphere and a cylinder
// sharing an axis meet in one or two PLANAR circles, so the family covers every way a closed surface can
// be divided: a rod ending inside the ball (the ball stud), a rod driven right through it (the bead and
// its axle), a rod ending in the ball's shoulder — past the seam circle but short of the pole — and a rod
// buried at both ends, which divides nothing at all.
//
// These rows used to drive CoaxialSphereRod{Join,Cut,Intersect}, three bespoke constructors that split the
// operands by construction because a sphere is not a rim-bounded band. They now drive Boolean, the one
// general pipeline, and every face census below is the one the constructors produced — the closed-surface
// pairing carries the family without them.

// ballAndRod returns a ball centred at the origin and a rod on the +Y axis from y0 of the given length.
func ballAndRod(t *testing.T, ballR, rodR, y0, length float64) (*topo.Body, *topo.Body) {
	t.Helper()
	ball, err := SolidSphere(math.P3(0, 0, 0), ballR, "ball")
	if err != nil {
		t.Fatalf("SolidSphere(R=%g): %v", ballR, err)
	}
	rod, err := SolidCylinder(math.P3(0, math.Scalar(y0), 0), math.V3(0, 1, 0), rodR, length)
	if err != nil {
		t.Fatalf("SolidCylinder(r=%g, y0=%g, L=%g): %v", rodR, y0, length, err)
	}
	return ball, rod
}

// surfaceKind names a face's analytic surface, so a result face can be matched to the operand face it
// descends from.
func surfaceKind(f *topo.Face) string {
	switch f.Geometry().(type) {
	case geom.Sphere:
		return "sphere"
	case geom.Cylinder:
		return "cylinder"
	default:
		return "plane"
	}
}

// sameCensus compares two face-kind tallies.
func sameCensus(got, want map[string]int) bool {
	if len(got) != len(want) {
		return false
	}
	for k, n := range want {
		if got[k] != n {
			return false
		}
	}
	return true
}

// assertCensus runs one boolean and checks the result is watertight with exactly the wanted face kinds.
func assertCensus(t *testing.T, name string, op Op, a, b *topo.Body, want map[string]int) {
	t.Helper()
	res, err := Boolean(op, a, b)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	assertWatertight(t, res)
	got := map[string]int{}
	for _, f := range res.Faces() {
		got[surfaceKind(f)]++
	}
	if !sameCensus(got, want) {
		t.Errorf("%s built faces %v, want %v", name, got, want)
	}
}

// TestCoaxialSphereRodBooleansAreAnalyticSolids drives the four operations over both the blind rod (one
// seam circle) and the through axle (two), and pins the analytic face census of each.
func TestCoaxialSphereRodBooleansAreAnalyticSolids(t *testing.T) {
	t.Parallel()
	blindBall, blindRod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	thruBall, thruRod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)

	assertCensus(t, "ball ∪ rod", Union, blindBall, blindRod, map[string]int{"sphere": 1, "cylinder": 1, "plane": 1})
	assertCensus(t, "ball − rod", Difference, blindBall, blindRod, map[string]int{"sphere": 1, "cylinder": 1, "plane": 1})
	assertCensus(t, "rod − ball", Difference, blindRod, blindBall, map[string]int{"sphere": 1, "cylinder": 1, "plane": 1})
	assertCensus(t, "ball ∩ rod", Intersection, blindBall, blindRod, map[string]int{"sphere": 1, "cylinder": 1, "plane": 1})
	assertCensus(t, "ball ∪ axle", Union, thruBall, thruRod, map[string]int{"sphere": 1, "cylinder": 2, "plane": 2})
	assertCensus(t, "ball − axle (bead)", Difference, thruBall, thruRod, map[string]int{"sphere": 1, "cylinder": 1})
	assertCensus(t, "axle − ball (two stubs)", Difference, thruRod, thruBall, map[string]int{"sphere": 2, "cylinder": 2, "plane": 2})
	assertCensus(t, "ball ∩ axle (core)", Intersection, thruBall, thruRod, map[string]int{"sphere": 2, "cylinder": 1})
}

// TestShoulderExtentsAreAnalyticSolids: a rod stopping PART WAY through the ball's shoulder is the extent
// where a plane∩sphere circle joins the seam circle, so the ball survives in two pieces and the rod's end
// cap survives as an ANNULUS. All three shoulder configurations are driven — one shoulder cap, two, and
// one paired with a free cap.
func TestShoulderExtentsAreAnalyticSolids(t *testing.T) {
	t.Parallel()
	shoulder := func(y0, length float64) (*topo.Body, *topo.Body) { return ballAndRod(t, 0.5, 0.3, y0, length) }
	oneBall, oneRod := shoulder(0, 0.45)    // buried cap → shoulder cap
	biBall, biRod := shoulder(-0.45, 0.9)   // a shoulder cap at each end
	studBall, studRod := shoulder(-0.45, 2) // shoulder cap → free cap

	assertCensus(t, "ball ∪ shoulder rod", Union, oneBall, oneRod, map[string]int{"sphere": 2, "cylinder": 1, "plane": 1})
	assertCensus(t, "ball − shoulder rod", Difference, oneBall, oneRod, map[string]int{"sphere": 2, "cylinder": 1, "plane": 2})
	assertCensus(t, "shoulder rod − ball", Difference, oneRod, oneBall, map[string]int{"sphere": 1, "cylinder": 1, "plane": 1})
	assertCensus(t, "ball ∩ shoulder rod", Intersection, oneBall, oneRod, map[string]int{"sphere": 1, "cylinder": 1, "plane": 2})
	// three ball pieces: a tip beyond each shoulder cap, and the belt between the two seams
	assertCensus(t, "ball ∪ bi-shoulder rod", Union, biBall, biRod, map[string]int{"sphere": 3, "cylinder": 2, "plane": 2})
	assertCensus(t, "bi-shoulder rod − ball", Difference, biRod, biBall, map[string]int{"sphere": 2, "cylinder": 2, "plane": 2})
	assertCensus(t, "ball ∪ shoulder stud", Union, studBall, studRod, map[string]int{"sphere": 2, "cylinder": 2, "plane": 2})
	assertCensus(t, "ball ∩ shoulder stud", Intersection, studBall, studRod, map[string]int{"sphere": 2, "cylinder": 1, "plane": 1})
}

// TestFullyBuriedRodIsAWholeBallOrAVoid: a rod that ends inside the ball at BOTH ends removes nothing of
// the ball's surface, so the union is the untouched sphere — one boundary-less face — and the cut is that
// same sphere with the rod as an interior VOID, a second shell facing into the cavity.
func TestFullyBuriedRodIsAWholeBallOrAVoid(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, -0.2, 0.4) // both caps at |s| = 0.2, well inside the seam at 0.4
	union, err := Boolean(Union, ball, rod)
	if err != nil {
		t.Fatalf("ball ∪ buried rod: %v", err)
	}
	if n := len(union.Faces()); n != 1 || surfaceKind(union.Faces()[0]) != "sphere" {
		t.Errorf("ball ∪ buried rod has %d faces, want the untouched sphere alone", n)
	}
	if n := len(union.Faces()[0].Loops()); n != 0 {
		t.Errorf("the untouched sphere carries %d loops, want none", n)
	}
	bored, err := Boolean(Difference, ball, rod)
	if err != nil {
		t.Fatalf("ball − buried rod: %v", err)
	}
	assertWatertight(t, bored)
	if n := len(bored.Shells()); n != 2 {
		t.Errorf("ball − buried rod is %d shell(s), want 2 (the ball plus the void the rod leaves)", n)
	}
}

// TestShoulderCapLeavesAnAnnulus is the shoulder extent's signature: the rod's end disc is crossed by the
// ball's own surface, so the face that survives on it carries a HOLE — an annulus, not a disc. Nothing in
// the other two extents produces one.
func TestShoulderCapLeavesAnAnnulus(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	res, err := Boolean(Union, ball, rod)
	if err != nil {
		t.Fatalf("ball ∪ shoulder rod: %v", err)
	}
	for _, f := range res.Faces() {
		if surfaceKind(f) != "plane" {
			continue
		}
		if n := len(f.Loops()); n != 2 {
			t.Errorf("the shoulder cap's face has %d loops, want 2 (the rod's rim and the ball's circle)", n)
		}
		return
	}
	t.Fatal("the shoulder union has no planar face")
}

// TestBeadKeepsTheBallsBeltAndNothingElse: cutting a through rod out of the ball must leave the ball's
// belt (one sphere face bounded by BOTH seam circles) and the open bore — not a cap that quietly closed
// one end off. The two loops on that face are what say so.
func TestBeadKeepsTheBallsBeltAndNothingElse(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	bead, err := Boolean(Difference, ball, rod)
	if err != nil {
		t.Fatalf("ball − axle: %v", err)
	}
	for _, f := range bead.Faces() {
		if surfaceKind(f) != "sphere" {
			continue
		}
		if n := len(f.Loops()); n != 2 {
			t.Errorf("the bead's spherical face has %d loops, want 2 (a belt is bounded by both seams)", n)
		}
		return
	}
	t.Fatal("the bead has no spherical face")
}

// TestCoaxialSphereRodCutFacesTheRightWay: which operand is the target decides which cap survives and
// which face is inverted, so the two directions must NOT come out the same body. rod − ball keeps the
// small cap (as a dimple in the stub's base); ball − rod keeps the large one (the rest of the ball).
func TestCoaxialSphereRodCutFacesTheRightWay(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	bored, errB := Boolean(Difference, ball, rod)
	stub, errS := Boolean(Difference, rod, ball)
	if errB != nil || errS != nil {
		t.Fatalf("ball−rod err=%v, rod−ball err=%v; want both", errB, errS)
	}
	if boredRev, stubRev := sphereFaceReversed(t, bored), sphereFaceReversed(t, stub); boredRev || !stubRev {
		t.Errorf("sphere sense: ball−rod reversed=%v (want false, material inside the ball), "+
			"rod−ball reversed=%v (want true, a dimple)", boredRev, stubRev)
	}
}

// sphereFaceReversed reports whether the body's spherical face was added with its material on the far
// side of the surface normal — the difference between a ball's own surface and a dimple cut into a rod.
func sphereFaceReversed(t *testing.T, b *topo.Body) bool {
	t.Helper()
	for _, f := range b.Faces() {
		if surfaceKind(f) == "sphere" {
			return f.Reversed()
		}
	}
	t.Fatal("body has no spherical face")
	return false
}

// TestCoaxialSphereRodJoinKeepsOperandLineage: the ball stud's three faces all descend from an operand
// face, so each must keep that face's key or a reference bound to the shank's end face (a chamfer, a
// hole) would break on the join (ADR-0043 K1a).
func TestCoaxialSphereRodJoinKeepsOperandLineage(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	want := map[string]string{
		"sphere":   soleFaceKey(t, ball, "sphere"),
		"cylinder": soleFaceKey(t, rod, "cylinder"),
		"plane":    capFaceKey(t, rod, 1.5), // the FREE cap, the only one the join keeps
	}
	res, err := Boolean(Union, ball, rod)
	if err != nil {
		t.Fatalf("ball stud join: %v", err)
	}
	for _, f := range res.Faces() {
		kind := surfaceKind(f)
		if got := f.Lineage().KeyString(); got != want[kind] {
			t.Errorf("%s face lineage %q, want the operand's %q", kind, got, want[kind])
		}
	}
}

// soleFaceKey returns the lineage key of the body's only face of the given surface kind.
func soleFaceKey(t *testing.T, b *topo.Body, kind string) string {
	t.Helper()
	for _, f := range b.Faces() {
		if surfaceKind(f) == kind {
			return f.Lineage().KeyString()
		}
	}
	t.Fatalf("body has no %s face", kind)
	return ""
}

// capFaceKey returns the lineage key of the rod's planar cap at height y along the +Y axis.
func capFaceKey(t *testing.T, b *topo.Body, y float64) string {
	t.Helper()
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if ok && stdmath.Abs(float64(pl.Origin.Y)-y) < 1e-9 {
			return f.Lineage().KeyString()
		}
	}
	t.Fatalf("body has no planar cap at y=%g", y)
	return ""
}
