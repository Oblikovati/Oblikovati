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
// a cap's, and the bespoke builder treated them as free: an intersection's ball faces are ALL belts, so
// its winding chain had no fixed piece to seed from, the seeding started at an arbitrary direction, and
// the belt came out naming the caps. The body still measured closed and manifold; only the readers of
// the trim disagreed with it (a Ø10 ball ∩ Ø6 shoulder rod read 298.45 mm² of caps where 15.71 mm² of
// belt was meant, and its analytic volume declined to a faceted 123.02 against a true 123.96).
//
// A forward walk — counter-clockwise about the circle's own normal — encloses the +normal side, so the
// belt is named when each rim's walk encloses the side the belt lies on: the low rim toward +Y, the high
// rim toward −Y. Which of those is a forward walk depends on how the rim circle happens to be stored, so
// the assertion below reads the enclosed SIDE rather than the reversed flag. The builder is gone
// (ADR-0061 stage 4) and these rows now drive Boolean, so what they pin is the general pipeline's own
// winding rather than a retired assembler's.

// beltRimEnclosures maps each loop of the body's sole spherical face to the AXIAL DIRECTION its walk
// encloses, keyed by the rim's axial station. A forward walk — counter-clockwise about the circle's own
// normal — encloses the +normal side, and a reversed walk the −normal side, so this reads the region the
// loop names without assuming which way the rim circle happens to be stored. Every rim of this family is
// one circle, so one loop is one entry.
func beltRimEnclosures(t *testing.T, b *topo.Body) map[float64]float64 {
	t.Helper()
	out := map[float64]float64{}
	for _, l := range soleSphereFace(t, b).Loops() {
		uses := l.EdgeUses()
		if len(uses) != 1 {
			t.Fatalf("a belt loop has %d edge uses, want 1 (a rim is one circle)", len(uses))
		}
		c, ok := uses[0].Edge().Geometry().(geom.Circle)
		if !ok {
			t.Fatalf("a belt rim is %T, want geom.Circle", uses[0].Edge().Geometry())
		}
		side := float64(c.Normal.AsVector().Y)
		if uses[0].Reversed() {
			side = -side
		}
		out[float64(c.Center.Y)] = side
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

// assertBeltWinding pins that the spherical face names the BELT and not the two caps: each of its two
// rims must be walked so that it encloses the side the belt lies on — the low rim toward +Y, the high rim
// toward −Y. Inverted, the same two circles name the sphere's complement. Stations are matched within a
// weld because they are computed (√(R²−r²)), not exact literals.
func assertBeltWinding(t *testing.T, name string, b *topo.Body, lo, hi float64) {
	t.Helper()
	enc := beltRimEnclosures(t, b)
	if len(enc) != 2 {
		t.Fatalf("%s: the spherical face has %d rims, want 2 (a belt is bounded by both)", name, len(enc))
	}
	for _, c := range []struct {
		station, toward float64
		which           string
	}{{lo, +1, "low"}, {hi, -1, "high"}} {
		side, ok := stationEnclosure(enc, c.station)
		if !ok {
			t.Errorf("%s: no rim at y=%g among %v", name, c.station, enc)
			continue
		}
		if side*c.toward <= 0 {
			t.Errorf("%s: the %s rim at y=%g encloses the y%+.0f side, want y%+.0f — the belt is on that side",
				name, c.which, c.station, side, c.toward)
		}
	}
}

// stationEnclosure looks a rim up by its axial station within the family's weld (the stations are solved,
// so ±0.4 arrives as ±0.4000000000000002).
func stationEnclosure(enc map[float64]float64, station float64) (float64, bool) {
	for y, side := range enc {
		if stdmath.Abs(y-station) < 1e-9 { // tol:weld — a solved station against its nominal
			return side, true
		}
	}
	return 0, false
}

// TestBeadBeltIsWoundToNameTheBelt: a rod driven right through the ball leaves the ball's belt between
// the two seam circles at y=±0.4, and that face must walk the lower one forward and the upper one back.
func TestBeadBeltIsWoundToNameTheBelt(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, -1.0, 2.5)
	bead, err := Boolean(Difference, ball, rod)
	if err != nil {
		t.Fatalf("ball − axle: %v", err)
	}
	assertBeltWinding(t, "bead", bead, -0.4, 0.4)
}

// TestShoulderPlugBeltIsWoundToNameTheBand: the intersection of the ball with a rod stopping at y=0.45 —
// past the seam at y=0.4, short of the pole — keeps the ball's surface only over that 0.05 band. It is
// the case that regressed, because it has no spherical CAP anywhere to pin the winding chain from.
func TestShoulderPlugBeltIsWoundToNameTheBand(t *testing.T) {
	t.Parallel()
	ball, rod := ballAndRod(t, 0.5, 0.3, 0, 0.45)
	plug, err := Boolean(Intersection, ball, rod)
	if err != nil {
		t.Fatalf("ball ∩ shoulder rod: %v", err)
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
	plug, err := Boolean(Intersection, ball, rod)
	if err != nil {
		t.Fatalf("ball ∩ shoulder rod: %v", err)
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
