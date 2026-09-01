// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The coaxial result's sphere BELT and the region its winding names (Oblikovati/Oblikovati#3447).
//
// A belt is bounded by two coaxial circles, and walked the other way round those same two circles name
// the sphere's COMPLEMENT of it — the two disjoint caps. So its loop directions are as load-bearing as
// a cap's, and the builder treated them as free: an intersection's ball faces are ALL belts, so its
// winding chain had no fixed piece to seed from, settleWindings started it at the arbitrary seed
// direction, and the belt came out naming the caps. The body still measured closed and manifold; only
// the readers of the trim disagreed with it (a Ø10 ball ∩ Ø6 shoulder rod read 298.45 mm² of caps where
// 15.71 mm² of belt was meant, and its analytic volume declined to a faceted 123.02 against a true
// 123.96).
//
// A forward walk — counter-clockwise about the circle's own normal — encloses the +normal side, so the
// belt is named by walking its LOW rim forward and its HIGH one backward.

// beltRimWalks maps each loop of the body's sole spherical face to whether it walks its rim backward,
// keyed by the rim's axial station. Every rim of this family is one circle, so one loop is one entry.
func beltRimWalks(t *testing.T, b *topo.Body) map[float64]bool {
	t.Helper()
	out := map[float64]bool{}
	for _, l := range soleSphereFace(t, b).Loops() {
		uses := l.EdgeUses()
		if len(uses) != 1 {
			t.Fatalf("a belt loop has %d edge uses, want 1 (a rim is one circle)", len(uses))
		}
		c, ok := uses[0].Edge().Geometry().(geom.Circle)
		if !ok {
			t.Fatalf("a belt rim is %T, want geom.Circle", uses[0].Edge().Geometry())
		}
		out[float64(c.Center.Y)] = uses[0].Reversed()
	}
	return out
}

// soleSphereFace returns the body's only spherical face.
func soleSphereFace(t *testing.T, b *topo.Body) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if surfaceKind(f) == "sphere" {
			return f
		}
	}
	t.Fatal("body has no spherical face")
	return nil
}

// assertBeltWinding pins the belt's two rims: the one at the LOW station walked forward, the one at the
// HIGH station backward. Inverted, the same two circles name the two caps instead.
func assertBeltWinding(t *testing.T, name string, b *topo.Body, lo, hi float64) {
	t.Helper()
	walks := beltRimWalks(t, b)
	if len(walks) != 2 {
		t.Fatalf("%s: the spherical face has %d rims, want 2 (a belt is bounded by both)", name, len(walks))
	}
	if rev, ok := walks[lo]; !ok || rev {
		t.Errorf("%s: the rim at y=%g walks reversed=%v (present=%v), want forward — the belt is above it",
			name, lo, rev, ok)
	}
	if rev, ok := walks[hi]; !ok || !rev {
		t.Errorf("%s: the rim at y=%g walks reversed=%v (present=%v), want backward — it is the belt's hole",
			name, hi, rev, ok)
	}
}

// TestBeadBeltIsWoundToNameTheBelt: a rod driven right through the ball leaves the ball's belt between
// the two seam circles at y=±0.4, and that face must walk the lower one forward and the upper one back.
func TestBeadBeltIsWoundToNameTheBelt(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	bead, ok := CoaxialSphereRodCut(ball, rod)
	if !ok {
		t.Fatal("ball − axle declined")
	}
	assertBeltWinding(t, "bead", bead, -0.4, 0.4)
}

// TestShoulderPlugBeltIsWoundToNameTheBand: the intersection of the ball with a rod stopping at y=0.45 —
// past the seam at y=0.4, short of the pole — keeps the ball's surface only over that 0.05 band. It is
// the case that regressed, because it has no spherical CAP anywhere to pin the winding chain from.
func TestShoulderPlugBeltIsWoundToNameTheBand(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	plug, ok := CoaxialSphereRodIntersect(ball, rod)
	if !ok {
		t.Fatal("ball ∩ shoulder rod declined")
	}
	assertBeltWinding(t, "shoulder plug", plug, 0.4, 0.45)
}

// TestShoulderPlugBeltTrimClaimsTheBandOnly is the same defect read end to end, through the trim
// classifier every consumer goes through: the shoulder band's face claims the stations on the band and
// none beyond either rim. Only the NARROW band is asserted this way — brep's geodesic winding projects
// the loops orthographically onto the tangent plane at the query point, which is 2-to-1 and so cannot
// classify a spherical region whose rims lie more than a quarter turn away (the bead's belt).
func TestShoulderPlugBeltTrimClaimsTheBandOnly(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	plug, ok := CoaxialSphereRodIntersect(ball, rod)
	if !ok {
		t.Fatal("ball ∩ shoulder rod declined")
	}
	f := soleSphereFace(t, plug)
	for _, c := range []struct {
		y    float64
		want bool
	}{{0.42, true}, {0.44, true}, {0.3, false}, {0.48, false}} {
		p := math.P3(math.Scalar(stdmath.Sqrt(0.25-c.y*c.y)), math.Scalar(c.y), 0)
		if got := PointInFaceTrim(f, p); got != c.want {
			t.Errorf("the shoulder band claims %v (y=%g) = %v, want %v", p, c.y, got, c.want)
		}
	}
}
