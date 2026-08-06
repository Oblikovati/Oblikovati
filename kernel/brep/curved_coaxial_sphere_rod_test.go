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

// TestCoaxialRodOfDeclinesOutOfScopeExtents guards the documented scope. A rod's caps must each clear
// the ball cleanly — both strictly inside, both strictly outside, or one of each; a cap landing in the
// annular band between the seam plane and the pole is a different construction and must decline, so
// kernel/ops keeps its guarded fallback rather than shipping a solid this file cannot describe.
func TestCoaxialRodOfDeclinesOutOfScopeExtents(t *testing.T) {
	for _, c := range []struct {
		name        string
		y0, length  float64
		explanation string
	}{
		{"shoulder stop", 0, 0.45, "the far cap lands between the seam plane and the pole"},
		{"buried rod", -0.2, 0.4, "both caps inside: the union is the ball alone"},
	} {
		ball, rod := ballAndRod(t, 0.5, 0.3, c.y0, c.length)
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
	if forward.hiSeam.Center != reverse.hiSeam.Center || forward.hiSeam.Radius != reverse.hiSeam.Radius {
		t.Errorf("argument order changed the seam: %v vs %v", forward.hiSeam, reverse.hiSeam)
	}
}

// TestCoaxialRodOfRecognisesTheThroughExtent: a rod that clears the ball at BOTH ends has two real seam
// circles, and both must land on the sphere at ±√(R²−r²) with the rod's own radius.
func TestCoaxialRodOfRecognisesTheThroughExtent(t *testing.T) {
	ball, rod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	r, ok := coaxialRodOf(ball, rod)
	if !ok {
		t.Fatal("a rod passing right through the ball was declined")
	}
	if !r.through {
		t.Fatal("the through extent was recognised as a rod ending inside the ball")
	}
	for _, c := range []struct {
		name  string
		seam  geom.Circle
		wantY float64
	}{{"hi", r.hiSeam, 0.4}, {"lo", r.loSeam, -0.4}} {
		if stdmath.Abs(float64(c.seam.Center.Y)-c.wantY) > 1e-12 {
			t.Errorf("%s seam at y=%g, want %g", c.name, float64(c.seam.Center.Y), c.wantY)
		}
		if stdmath.Abs(c.seam.Radius-0.3) > 1e-12 {
			t.Errorf("%s seam radius %g, want the rod's 0.3", c.name, c.seam.Radius)
		}
	}
}

// TestCoaxialRodOfOrientsTheAxisFromTheBuriedEnd: `out` must run from the buried cap toward the free
// one whichever way the rod was modelled, since every face's side is named against it.
func TestCoaxialRodOfOrientsTheAxisFromTheBuriedEnd(t *testing.T) {
	for _, c := range []struct {
		name       string
		y0, length float64
		wantY      float64
	}{
		{"rod built along +Y", 0, 1.5, 1},
		{"rod built along -Y", -1.5, 1.5, -1},
	} {
		ball, rod := ballAndRod(t, 0.5, 0.3, c.y0, c.length)
		r, ok := coaxialRodOf(ball, rod)
		if !ok {
			t.Fatalf("%s: declined", c.name)
		}
		if got := float64(r.out.Y); stdmath.Abs(got-c.wantY) > 1e-12 {
			t.Errorf("%s: out.Y = %g, want %g (buried cap → free cap)", c.name, got, c.wantY)
		}
		if float64(r.hiSeam.Center.Y)*c.wantY <= 0 {
			t.Errorf("%s: seam at y=%g is on the buried side", c.name, float64(r.hiSeam.Center.Y))
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
