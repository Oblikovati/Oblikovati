// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// The second-harmonic reduction and its branch pairing (ADR-0061 stage 5, third slice). The derivation
// is in torus_quadric_harmonic2.go; these rows verify it against the quadric itself rather than against
// a restatement of the algebra.

// skewTestQuadrics is the family the reduction has to cover: two axis-invariant members, whose second
// harmonic must VANISH, and three that are not, one of each kind the kernel can build.
func skewTestQuadrics(t *testing.T) []struct {
	name      string
	quad      Quadric
	invariant bool
} {
	t.Helper()
	ball, _ := NewSphere(math.P3(3, 2, 1), 2.5)
	axial, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0, 0, 1), 0.8)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1)
	tilted, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0.3, 0, 1), 0.8)
	skewCone, _ := NewCone(math.P3(4, 1, -3), math.V3(0.4, 0.2, 1), 0.5)
	return []struct {
		name      string
		quad      Quadric
		invariant bool
	}{
		{"ball off centre", ball.QuadricForm(), true},
		{"axial drill", axial.QuadricForm(), true},
		{"rod across the ring", rod.QuadricForm(), false},
		{"tilted drill", tilted.QuadricForm(), false},
		{"tilted cone", skewCone.QuadricForm(), false},
	}
}

// TestTheSecondHarmonicIsTheQuadricOnTheTorus verifies the derivation itself: the five coefficients,
// evaluated as Level + Cos1·cos u + Sin1·sin u + Cos2·cos 2u + Sin2·sin 2u, must reproduce the
// quadric's own value at the torus point (u, v) — for every quadric, at random stations and azimuths.
func TestTheSecondHarmonicIsTheQuadricOnTheTorus(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	rng := rand.New(rand.NewSource(5))
	for _, c := range skewTestQuadrics(t) {
		worst := 0.0
		for range 4000 {
			u, v := twoPi*rng.Float64(), twoPi*rng.Float64()
			want := c.quad.ValueAt(ring.PointAt(u, v))
			got := torusSecondHarmonicAt(ring, c.quad, v).valueAt(u)
			worst = stdmath.Max(worst, stdmath.Abs(got-want)/stdmath.Max(1, stdmath.Abs(want)))
		}
		if worst > 1e-12 { // tol:numeric — the reduction's own rounding against the quadric it restates
			t.Errorf("%s: the reduction departs from the quadric by %.3e relative", c.name, worst)
		}
	}
}

// TestTheSecondHarmonicVanishesOnTheAxisInvariantFamily is the reproduction proof for the closed form
// this reduction generalises. Where M is invariant about the torus axis the two second-harmonic
// coefficients are EXACTLY zero and the rest is torusHarmonicAt's own level, reach and phase — so the
// arccos path is not an approximation of the general one, it is the general one written out.
func TestTheSecondHarmonicVanishesOnTheAxisInvariantFamily(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	for _, c := range skewTestQuadrics(t) {
		h, invariant := torusHarmonicAt(ring, c.quad, 0.7)
		if invariant != c.invariant {
			t.Fatalf("%s: classified invariant=%v, want %v", c.name, invariant, c.invariant)
		}
		g := torusSecondHarmonicAt(ring, c.quad, 0.7)
		if !c.invariant {
			if g.Cos2 == 0 && g.Sin2 == 0 {
				t.Errorf("%s: the second harmonic is zero, but its tensor is not axis-invariant", c.name)
			}
			continue
		}
		if g.Cos2 != 0 || g.Sin2 != 0 {
			t.Errorf("%s: second harmonic (%g, %g), want exactly zero", c.name, g.Cos2, g.Sin2)
		}
		if g.Level != h.level {
			t.Errorf("%s: level %.17g, want the one-harmonic form's %.17g bit for bit", c.name, g.Level, h.level)
		}
		assertNearly(t, c.name+" reach", stdmath.Hypot(g.Cos1, g.Sin1), h.reach)
		assertNearly(t, c.name+" phase", stdmath.Atan2(g.Sin1, g.Cos1), h.phase)
	}
}

// assertNearly compares two readings of the same quantity at the level their own arithmetic differs by.
func assertNearly(t *testing.T, what string, got, want float64) {
	t.Helper()
	if stdmath.Abs(got-want) > 1e-14*stdmath.Max(1, stdmath.Abs(want)) { // tol:numeric — two orderings of one product
		t.Errorf("%s = %.17g, want %.17g", what, got, want)
	}
}

// TestEveryStationAzimuthIsCertifiedAndComplete: the station solver must return every azimuth where the
// quadric meets the tube circle and nothing else. Completeness is measured against a dense sign-change
// scan of the same function — an independent count, since it uses no polynomial algebra at all.
func TestEveryStationAzimuthIsCertifiedAndComplete(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	for _, c := range skewTestQuadrics(t) {
		for i := range 97 {
			v := twoPi * float64(i) / 97
			h := torusSecondHarmonicAt(ring, c.quad, v)
			roots := h.azimuths()
			for _, u := range roots {
				if off := stdmath.Abs(c.quad.ValueAt(ring.PointAt(u, v))); off > 1e-9*h.scale() {
					t.Fatalf("%s at v=%g: azimuth %g has residual %.3e — an uncertified root", c.name, v, u, off)
				}
			}
			if want := signChangeCount(h); len(roots) != want {
				t.Errorf("%s at v=%g: %d azimuths, a dense scan finds %d sign changes", c.name, v, len(roots), want)
			}
		}
	}
}

// signChangeCount counts the station polynomial's sign changes over one turn, by brute sampling — the
// independent oracle for "how many azimuths are there".
func signChangeCount(h torusSecondHarmonic) int {
	const probes = 20000
	n, prev := 0, h.valueAt(0)
	for i := 1; i <= probes; i++ {
		cur := h.valueAt(twoPi * float64(i) / probes)
		if (prev > 0) != (cur > 0) {
			n++
		}
		prev = cur
	}
	return n
}

// TestALaneStraddlesItsOwnExtremum: a lane's two azimuths must bracket the extremum that names it, one
// on each side, and both must be roots. Where the lane has no roots — beyond its fold — both must be
// the extremum ITSELF, which is what closes a window loop exactly on its ends.
func TestALaneStraddlesItsOwnExtremum(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1)
	q := rod.QuadricForm()
	anchors, ok := torusLaneAnchors(ring, q)
	if !ok || len(anchors) != 4 {
		t.Fatalf("lane anchors: ok=%v %v, want the four extremum tracks of a rod across the ring", ok, anchors)
	}
	live, dead := 0, 0
	for i := range 401 {
		v := twoPi * float64(i) / 401
		h := torusSecondHarmonicAt(ring, q, v)
		for _, a := range anchors {
			l := torusLaneAt(h, a)
			if l.discriminant() <= 0 {
				dead++
				if l.lower != l.center || l.upper != l.center {
					t.Fatalf("v=%g anchor %g: a dead lane reads %g/%g, want its extremum %g", v, a, l.lower, l.upper, l.center)
				}
				continue
			}
			live++
			assertLaneRoot(t, h, v, l.lower)
			assertLaneRoot(t, h, v, l.upper)
			if turnBetween(l.center, l.upper, true)+turnBetween(l.center, l.lower, false) != l.separation() {
				t.Fatalf("v=%g anchor %g: the separation does not span the pair", v, a)
			}
		}
	}
	if live == 0 || dead == 0 {
		t.Fatalf("the sweep saw %d live and %d dead lanes; it must exercise both", live, dead)
	}
}

// assertLaneRoot requires one of a lane's azimuths to be a root of the station polynomial.
func assertLaneRoot(t *testing.T, h torusSecondHarmonic, v, u float64) {
	t.Helper()
	if off := stdmath.Abs(h.valueAt(u)); off > 1e-9*h.scale() { // tol:numeric — a certified root's residual
		t.Fatalf("v=%g: lane azimuth %g has residual %.3e, so it is not a root", v, u, off)
	}
}

// TestALaneReproducesTheOneHarmonicRoots: on the axis-invariant family the lane reader and the arccos
// must name the SAME two azimuths, which is the certificate that the general path is a generalisation
// rather than a second algorithm. The two orderings are mirror images — the arccos measures from the
// harmonic's peak and the lane from the extremum it straddles — so the pair is compared as a set.
func TestALaneReproducesTheOneHarmonicRoots(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	drill, _ := NewCylinder(math.P3(5, 0, 0), math.V3(0, 0, 1), 0.8)
	q := drill.QuadricForm()
	checked := 0
	for i := range 257 {
		v := twoPi * float64(i) / 257
		h, ok := torusHarmonicAt(ring, q, v)
		if !ok || h.discriminant() <= 0 {
			continue
		}
		l := torusLaneAt(torusSecondHarmonicAt(ring, q, v), h.phase)
		for _, want := range []float64{h.root(true), h.root(false)} {
			near := stdmath.Min(stdmath.Abs(shortestTurnDelta(want, l.lower)), stdmath.Abs(shortestTurnDelta(want, l.upper)))
			if near > 1e-9 { // tol:angular — the arccos and the quartic on the same root
				t.Fatalf("v=%g: the arccos root %g is %g away from both lane azimuths (%g, %g)", v, want, near, l.lower, l.upper)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no station had two roots; the row proved nothing")
	}
}

// TestALaneAnchorTrackIsRefusedWhenItCannotBeFollowed: the anchors are seeded at one station and
// followed by nearest extremum, so the reduction declines outright when a station's extrema do not
// match the seeds one to one. A constant station — a quadric that vanishes on the whole tube circle —
// has no extremum to seed from, and that is the decline this row drives.
func TestALaneAnchorTrackIsRefusedWhenItCannotBeFollowed(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	if _, ok := torusLaneAnchors(ring, Quadric{}); ok {
		t.Error("a quadric with no form at all must not yield lane anchors")
	}
}
