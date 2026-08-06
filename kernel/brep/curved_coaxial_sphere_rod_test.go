// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial sphere ∩ cylinder — the ball-stud family (Oblikovati#2036), ported from OCCT's
// IntAna_QuadQuadGeo::Perform(gp_Cylinder, gp_Sphere). These tests pin the recognizer: the closed-form
// circle offset, the three configurations OCCT itself declines, and the rod extents that fall outside
// the one-circle family this handler builds.

// ballAndRod is the reference pair: a ball of radius R at the origin and a coaxial rod of radius rc
// running along +Y from y=y0 to y=y0+length.
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

// TestCoaxialSphereCircleOffsetMatchesTheClosedForm pins the ported formula: the seam circles sit at
// ±√(R_s²−R_c²) along the axis. A Ø10 ball on a Ø6 shank puts them 4 mm from the centre — the 3-4-5
// triangle the NopSCADlib ball stud in #2036 is built on.
func TestCoaxialSphereCircleOffsetMatchesTheClosedForm(t *testing.T) {
	for _, c := range []struct{ ballR, rodR, want float64 }{
		{5, 3, 4}, {0.5, 0.3, 0.4}, {13, 5, 12},
	} {
		sph, _ := geom.NewSphere(math.P3(0, 0, 0), c.ballR)
		cyl, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), c.rodR)
		got, ok := coaxialSphereCircleOffset(sph, cyl)
		if !ok {
			t.Fatalf("ball R=%g, rod r=%g: declined; want an offset of %g", c.ballR, c.rodR, c.want)
		}
		if stdmath.Abs(got-c.want) > 1e-12*c.ballR {
			t.Errorf("ball R=%g, rod r=%g: offset %g, want %g", c.ballR, c.rodR, got, c.want)
		}
	}
}

// TestCoaxialSphereCircleOffsetDeclinesWhatOCCTDeclines: the recognizer must refuse exactly the cases
// OCCT reports as something other than a pair of circles — an axis missing the centre
// (IntAna_NoGeometricSolution: a quartic space curve), a ball no bigger than the rod (IntAna_Empty),
// and equal radii, which OCCT reports as ONE circle because the contact is an internal tangency.
func TestCoaxialSphereCircleOffsetDeclinesWhatOCCTDeclines(t *testing.T) {
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), 5)
	for _, c := range []struct {
		name   string
		origin math.Point3
		rodR   float64
	}{
		{"axis offset from the centre", math.P3(1, 0, 0), 3},
		{"ball smaller than the rod", math.P3(0, 0, 0), 6},
		{"equal radii: an internal tangency", math.P3(0, 0, 0), 5},
	} {
		cyl, _ := geom.NewCylinder(c.origin, math.V3(0, 1, 0), c.rodR)
		if d, ok := coaxialSphereCircleOffset(sph, cyl); ok {
			t.Errorf("%s: accepted with offset %g, want a decline", c.name, d)
		}
	}
}

// TestCoaxialRodOfDeclinesDegenerateExtents guards the gate. Every extent is buildable — the rod may end
// inside the ball, clear it entirely, or stop part way through its shoulder — EXCEPT the ones whose
// result would carry a face of zero size: a cap sitting on a seam station (a zero-width band) or on a
// pole (a zero-radius disc), and a rod that never reaches the ball at all.
func TestCoaxialRodOfDeclinesDegenerateExtents(t *testing.T) {
	const R, rc = 0.5, 0.3
	d := stdmath.Sqrt(R*R - rc*rc) // 0.4
	for _, c := range []struct {
		name        string
		y0, length  float64
		explanation string
	}{
		{"cap on the seam station", d, 1.0, "the wall band from the seam to that cap is zero-width"},
		{"cap on the pole", 0, R, "the end disc shrinks to the pole"},
		{"rod short of the ball", 2.0, 1.0, "the pair does not meet"},
	} {
		ball, rod := ballAndRod(t, R, rc, c.y0, c.length)
		if _, ok := coaxialRodOf(ball, rod); ok {
			t.Errorf("%s accepted; want a decline (%s)", c.name, c.explanation)
		}
	}
}

// TestCoaxialRodOfAcceptsEitherArgumentOrder: a caller must not have to know which operand is the ball.
func TestCoaxialRodOfAcceptsEitherArgumentOrder(t *testing.T) {
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	forward, okF := coaxialRodOf(ball, rod)
	reverse, okR := coaxialRodOf(rod, ball)
	if !okF || !okR {
		t.Fatalf("ball-first ok=%v, rod-first ok=%v; want both", okF, okR)
	}
	if forward.sLo != reverse.sLo || forward.sHi != reverse.sHi {
		t.Errorf("argument order changed the rod's stations: [%g,%g] vs [%g,%g]",
			forward.sLo, forward.sHi, reverse.sLo, reverse.sHi)
	}
}

// TestCoaxialRodOfRecordsTheSeamOffset: the seam offset is OCCT's √(R²−r_c²), and the rod's stations are
// measured from the ball centre along `out`, which always runs lo→hi.
func TestCoaxialRodOfRecordsTheSeamOffset(t *testing.T) {
	for _, c := range []struct {
		name           string
		y0, length     float64
		wantLo, wantHi float64
	}{
		{"rod built along +Y", 0, 1.5, 0, 1.5},
		{"rod built along -Y", -1.5, 1.5, -1.5, 0},
		{"through rod", -1.0, 2.5, -1.0, 1.5},
	} {
		ball, rod := ballAndRod(t, 0.5, 0.3, c.y0, c.length)
		r, ok := coaxialRodOf(ball, rod)
		if !ok {
			t.Fatalf("%s: declined", c.name)
		}
		if stdmath.Abs(r.seamOffset-0.4) > 1e-12 {
			t.Errorf("%s: seam offset %g, want 0.4", c.name, r.seamOffset)
		}
		if stdmath.Abs(r.sLo-c.wantLo) > 1e-12 || stdmath.Abs(r.sHi-c.wantHi) > 1e-12 {
			t.Errorf("%s: stations [%g,%g], want [%g,%g]", c.name, r.sLo, r.sHi, c.wantLo, c.wantHi)
		}
		if r.sLo >= r.sHi {
			t.Errorf("%s: stations are not ordered lo→hi: [%g,%g]", c.name, r.sLo, r.sHi)
		}
	}
}

// TestBallSpansSplitAtEverySeamAndCap pins the one-dimensional classification the whole family rests on:
// the ball's surface, walked pole to pole, changes membership in the rod only at the seam stations and
// at a rod cap that lands between the poles — and adjacent runs of the same verdict merge, so a cut that
// changes nothing leaves no spurious face boundary.
func TestBallSpansSplitAtEverySeamAndCap(t *testing.T) {
	for _, c := range []struct {
		name       string
		y0, length float64
		want       []coaxialSpan
	}{
		{"rod ends inside the ball", 0, 1.5, []coaxialSpan{
			{-0.5, 0.4, false}, {0.4, 0.5, true}}},
		{"rod passes through", -1.0, 2.5, []coaxialSpan{
			{-0.5, -0.4, true}, {-0.4, 0.4, false}, {0.4, 0.5, true}}},
		{"rod stops in the shoulder", 0, 0.45, []coaxialSpan{
			{-0.5, 0.4, false}, {0.4, 0.45, true}, {0.45, 0.5, false}}},
	} {
		ball, rod := ballAndRod(t, 0.5, 0.3, c.y0, c.length)
		r, ok := coaxialRodOf(ball, rod)
		if !ok {
			t.Fatalf("%s: declined", c.name)
		}
		assertSpans(t, c.name, r.ballSpans(), c.want)
	}
}

// TestWallSpansSplitAtTheSeams is the same for the rod's own side: it crosses the ball's surface only at
// the seam stations, and only where those fall inside its own extent.
func TestWallSpansSplitAtTheSeams(t *testing.T) {
	for _, c := range []struct {
		name       string
		y0, length float64
		want       []coaxialSpan
	}{
		{"rod ends inside the ball", 0, 1.5, []coaxialSpan{{0, 0.4, true}, {0.4, 1.5, false}}},
		{"rod passes through", -1.0, 2.5, []coaxialSpan{
			{-1.0, -0.4, false}, {-0.4, 0.4, true}, {0.4, 1.5, false}}},
		{"rod stops in the shoulder", 0, 0.45, []coaxialSpan{{0, 0.4, true}, {0.4, 0.45, false}}},
	} {
		ball, rod := ballAndRod(t, 0.5, 0.3, c.y0, c.length)
		r, ok := coaxialRodOf(ball, rod)
		if !ok {
			t.Fatalf("%s: declined", c.name)
		}
		assertSpans(t, c.name, r.wallSpans(), c.want)
	}
}

// TestCapSpansSplitWhereTheBallCrossesTheCap: a rod cap inside the ball is wholly inside, one outside is
// wholly outside, and one landing in the shoulder is split at the ball's own circle there.
func TestCapSpansSplitWhereTheBallCrossesTheCap(t *testing.T) {
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	r, ok := coaxialRodOf(ball, rod)
	if !ok {
		t.Fatal("shoulder-stop rod declined")
	}
	if got := r.capSpans(r.sLo); len(got) != 1 || !got[0].inside {
		t.Errorf("the buried cap splits into %v, want one wholly-inside run", got)
	}
	rho := stdmath.Sqrt(0.25 - 0.45*0.45) // the ball's radius at the shoulder cap
	got := r.capSpans(r.sHi)
	if len(got) != 2 || !got[0].inside || got[1].inside {
		t.Fatalf("the shoulder cap splits into %v, want inside then outside", got)
	}
	if stdmath.Abs(got[0].hi-rho) > 1e-12 {
		t.Errorf("the shoulder cap splits at radius %g, want the ball's own %g there", got[0].hi, rho)
	}
}

// assertSpans compares a run decomposition against the expected one.
func assertSpans(t *testing.T, name string, got, want []coaxialSpan) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: %d spans %v, want %d %v", name, len(got), got, len(want), want)
		return
	}
	for i := range got {
		if stdmath.Abs(got[i].lo-want[i].lo) > 1e-9 || stdmath.Abs(got[i].hi-want[i].hi) > 1e-9 ||
			got[i].inside != want[i].inside {
			t.Errorf("%s: span %d = %v, want %v", name, i, got[i], want[i])
		}
	}
}

// TestCoaxialSphereRodBuildersAssembleAnalyticSolids walks the whole family at the builder level, in
// both extents: every result is a watertight solid of exactly the expected analytic faces. The volumes
// are asserted where the boolean runs (kernel/ops, against the OCC oracle); what belongs here is the
// SHAPE, because an assembly that keeps the wrong halves still stitches into a plausible-looking body.
func TestCoaxialSphereRodBuildersAssembleAnalyticSolids(t *testing.T) {
	blindBall, blindRod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	thruBall, thruRod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	for _, c := range []struct {
		name  string
		build func() (*topo.Body, bool)
		want  map[string]int
	}{
		{"ball ∪ rod", func() (*topo.Body, bool) { return CoaxialSphereRodJoin(blindBall, blindRod) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"ball − rod", func() (*topo.Body, bool) { return CoaxialSphereRodCut(blindBall, blindRod) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"rod − ball", func() (*topo.Body, bool) { return CoaxialSphereRodCut(blindRod, blindBall) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"ball ∩ rod", func() (*topo.Body, bool) { return CoaxialSphereRodIntersect(blindBall, blindRod) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"ball ∪ axle", func() (*topo.Body, bool) { return CoaxialSphereRodJoin(thruBall, thruRod) },
			map[string]int{"sphere": 1, "cylinder": 2, "plane": 2}},
		{"ball − axle (bead)", func() (*topo.Body, bool) { return CoaxialSphereRodCut(thruBall, thruRod) },
			map[string]int{"sphere": 1, "cylinder": 1}},
		{"axle − ball (two stubs)", func() (*topo.Body, bool) { return CoaxialSphereRodCut(thruRod, thruBall) },
			map[string]int{"sphere": 2, "cylinder": 2, "plane": 2}},
		{"ball ∩ axle (core)", func() (*topo.Body, bool) { return CoaxialSphereRodIntersect(thruBall, thruRod) },
			map[string]int{"sphere": 2, "cylinder": 1}},
	} {
		res, ok := c.build()
		if !ok {
			t.Errorf("%s declined", c.name)
			continue
		}
		assertWatertight(t, res)
		got := map[string]int{}
		for _, f := range res.Faces() {
			got[surfaceKind(f)]++
		}
		if !sameCensus(got, c.want) {
			t.Errorf("%s built faces %v, want %v", c.name, got, c.want)
		}
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

// TestShoulderExtentsAssembleAnalyticSolids: a rod stopping PART WAY through the ball's shoulder is the
// extent where a plane∩sphere circle joins the seam circle, so the ball survives in two pieces and the
// rod's end cap survives as an ANNULUS. All three shoulder configurations are driven — one shoulder cap,
// two, and one paired with a free cap.
func TestShoulderExtentsAssembleAnalyticSolids(t *testing.T) {
	shoulder := func(y0, length float64) (*topo.Body, *topo.Body) { return ballAndRod(t, 0.5, 0.3, y0, length) }
	oneBall, oneRod := shoulder(0, 0.45)    // buried cap → shoulder cap
	biBall, biRod := shoulder(-0.45, 0.9)   // a shoulder cap at each end
	studBall, studRod := shoulder(-0.45, 2) // shoulder cap → free cap
	for _, c := range []struct {
		name  string
		build func() (*topo.Body, bool)
		want  map[string]int
	}{
		{"ball ∪ shoulder rod", func() (*topo.Body, bool) { return CoaxialSphereRodJoin(oneBall, oneRod) },
			map[string]int{"sphere": 2, "cylinder": 1, "plane": 1}},
		{"ball − shoulder rod", func() (*topo.Body, bool) { return CoaxialSphereRodCut(oneBall, oneRod) },
			map[string]int{"sphere": 2, "cylinder": 1, "plane": 2}},
		{"shoulder rod − ball", func() (*topo.Body, bool) { return CoaxialSphereRodCut(oneRod, oneBall) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 1}},
		{"ball ∩ shoulder rod", func() (*topo.Body, bool) { return CoaxialSphereRodIntersect(oneBall, oneRod) },
			map[string]int{"sphere": 1, "cylinder": 1, "plane": 2}},
		// three ball pieces: a tip beyond each shoulder cap, and the belt between the two seams
		{"ball ∪ bi-shoulder rod", func() (*topo.Body, bool) { return CoaxialSphereRodJoin(biBall, biRod) },
			map[string]int{"sphere": 3, "cylinder": 2, "plane": 2}},
		{"bi-shoulder rod − ball", func() (*topo.Body, bool) { return CoaxialSphereRodCut(biRod, biBall) },
			map[string]int{"sphere": 2, "cylinder": 2, "plane": 2}},
		{"ball ∪ shoulder stud", func() (*topo.Body, bool) { return CoaxialSphereRodJoin(studBall, studRod) },
			map[string]int{"sphere": 2, "cylinder": 2, "plane": 2}},
		{"ball ∩ shoulder stud", func() (*topo.Body, bool) { return CoaxialSphereRodIntersect(studBall, studRod) },
			map[string]int{"sphere": 2, "cylinder": 1, "plane": 1}},
	} {
		res, ok := c.build()
		if !ok {
			t.Errorf("%s declined", c.name)
			continue
		}
		assertWatertight(t, res)
		got := map[string]int{}
		for _, f := range res.Faces() {
			got[surfaceKind(f)]++
		}
		if !sameCensus(got, c.want) {
			t.Errorf("%s built faces %v, want %v", c.name, got, c.want)
		}
	}
}

// TestSettleWindingsDeclinesAnUnsatisfiableChain guards the escape hatch. The propagation cannot bend a
// spherical cap — its direction names which cap survives — so a chain whose two caps demand opposite
// parities has no solution, and the assembly must DECLINE rather than emit a body that measures right
// and fails ops.Validate's orientation check. The chain here is hand-built with the parity broken,
// because the geometry never produces one: an orientable surface always admits a consistent walk.
func TestSettleWindingsDeclinesAnUnsatisfiableChain(t *testing.T) {
	cap := func(forward bool) coaxialPiece {
		return coaxialPiece{fixed: true, rims: []coaxialRim{{0, 1, forward}},
			build: func(bool) (curvedFace, bool) { return curvedFace{}, true }}
	}
	if _, ok := settleWindings([]coaxialPiece{cap(true), cap(false)}); !ok {
		t.Error("two caps walking one rim opposite ways were rejected; that pair is consistent")
	}
	if _, ok := settleWindings([]coaxialPiece{cap(true), cap(true)}); ok {
		t.Error("two caps walking one rim the SAME way were accepted; neither can flip, so decline")
	}
	if _, ok := settleWindings([]coaxialPiece{cap(true)}); ok {
		t.Error("a lone rim, walked by ONE piece, was accepted; that is an open boundary, not a solid")
	}
}

// TestFullyBuriedRodIsAWholeBallOrAVoid: a rod that ends inside the ball at BOTH ends removes nothing of
// the ball's surface, so the union is the untouched sphere — one boundary-less face — and the cut is that
// same sphere with the rod as an interior VOID, a second shell facing into the cavity.
func TestFullyBuriedRodIsAWholeBallOrAVoid(t *testing.T) {
	ball, rod := ballAndRod(t, 0.5, 0.3, -0.2, 0.4) // both caps at |s| = 0.2, well inside the seam at 0.4
	union, ok := CoaxialSphereRodJoin(ball, rod)
	if !ok {
		t.Fatal("ball ∪ buried rod declined")
	}
	if n := len(union.Faces()); n != 1 || surfaceKind(union.Faces()[0]) != "sphere" {
		t.Errorf("ball ∪ buried rod has %d faces, want the untouched sphere alone", n)
	}
	if n := len(union.Faces()[0].Loops()); n != 0 {
		t.Errorf("the untouched sphere carries %d loops, want none", n)
	}
	bored, ok := CoaxialSphereRodCut(ball, rod)
	if !ok {
		t.Fatal("ball − buried rod declined")
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
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	res, ok := CoaxialSphereRodJoin(ball, rod)
	if !ok {
		t.Fatal("ball ∪ shoulder rod declined")
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
	ball, rod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	bead, ok := CoaxialSphereRodCut(ball, rod)
	if !ok {
		t.Fatal("ball − axle declined")
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
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	bored, okB := CoaxialSphereRodCut(ball, rod)
	stub, okS := CoaxialSphereRodCut(rod, ball)
	if !okB || !okS {
		t.Fatalf("ball−rod ok=%v, rod−ball ok=%v; want both", okB, okS)
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
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 1.5)
	want := map[string]string{
		"sphere":   soleFaceKey(t, ball, "sphere"),
		"cylinder": soleFaceKey(t, rod, "cylinder"),
		"plane":    capFaceKey(t, rod, 1.5), // the FREE cap, the only one the join keeps
	}
	res, ok := CoaxialSphereRodJoin(ball, rod)
	if !ok {
		t.Fatal("ball stud join declined")
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
