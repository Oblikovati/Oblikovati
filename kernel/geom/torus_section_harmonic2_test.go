// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// The second-harmonic reduction and its branch pairing (ADR-0061 stage 5, third slice). The derivation
// is in torus_section_harmonic2.go; these rows verify it against the quadric itself rather than against
// a restatement of the algebra.

// skewQuadricCase is one member of the family the reduction has to cover: the SURFACE, its quadric
// form, and whether its tensor is invariant about the torus axis. The surface is carried rather than
// rebuilt by name at the rows that need it — a second copy of the fixture would pair a newly added row
// with whatever its lookup happened to fall through to.
type skewQuadricCase struct {
	name      string
	surface   Surface
	quad      Quadric
	invariant bool
}

// skewTestQuadrics is the family the reduction has to cover: two axis-invariant members, whose second
// harmonic must VANISH, and three that are not, one of each kind the kernel can build.
func skewTestQuadrics(t *testing.T) []skewQuadricCase {
	t.Helper()
	ball, errBall := NewSphere(math.P3(3, 2, 1), 2.5)
	axial, errAxial := NewCylinder(math.P3(5, 0, 0), math.V3(0, 0, 1), 0.8)
	rod, errRod := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1)
	tilted, errTilted := NewCylinder(math.P3(5, 0, 0), math.V3(0.3, 0, 1), 0.8)
	skewCone, errCone := NewCone(math.P3(4, 1, -3), math.V3(0.4, 0.2, 1), 0.5)
	for _, err := range []error{errBall, errAxial, errRod, errTilted, errCone} {
		if err != nil {
			t.Fatalf("skew fixture: %v", err)
		}
	}
	return []skewQuadricCase{
		{"ball off centre", ball, ball.QuadricForm(), true},
		{"axial drill", axial, axial.QuadricForm(), true},
		{"rod across the ring", rod, rod.QuadricForm(), false},
		{"tilted drill", tilted, tilted.QuadricForm(), false},
		{"tilted cone", skewCone, skewCone.QuadricForm(), false},
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
// coefficients are EXACTLY zero and the rest is the one-harmonic form's own level, reach and phase —
// so the arccos path is not an approximation of the general one, it is the general one written out.
//
// The level is asserted against the CLOSED FORM's own arithmetic, `constant + m11·ρ²`, and not against
// torusHarmonic.level: since the arm64 fix (CI run 34280554924 macos-latest) harmonic() READS
// secondHarmonic().Level, so comparing the two structs would be comparing one expression with itself.
// The two spellings are not identical either — they differ by ρ²(m11−m22)/2, and axisInvariantEntries
// admits |m11−m22| up to axisInvarianceTol·scale before it calls the tensor invariant at all — so the
// bound is derived from that admission rather than guessed (levelFormsBound).
func TestTheSecondHarmonicVanishesOnTheAxisInvariantFamily(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	for _, c := range skewTestQuadrics(t) {
		h, invariant := torusHarmonicAt(ring, c.quad, 0.7)
		if invariant != c.invariant {
			t.Fatalf("%s: classified invariant=%v, want %v", c.name, invariant, c.invariant)
		}
		st := c.quad.stationOn(ring, 0.7)
		g := st.secondHarmonic()
		if !c.invariant {
			if g.Cos2 == 0 && g.Sin2 == 0 {
				t.Errorf("%s: the second harmonic is zero, but its tensor is not axis-invariant", c.name)
			}
			continue
		}
		if g.Cos2 != 0 || g.Sin2 != 0 {
			t.Errorf("%s: second harmonic (%g, %g), want exactly zero", c.name, g.Cos2, g.Sin2)
		}
		assertLevelFormsAgree(t, c.name, c.quad, st, g.Level, 0.7)
		assertNearly(t, c.name+" reach", stdmath.Hypot(g.Cos1, g.Sin1), h.reach)
		assertNearly(t, c.name+" phase", stdmath.Atan2(g.Sin1, g.Cos1), h.phase)
	}
}

// assertLevelFormsAgree checks the general form's Level against the one-harmonic closed form's own
// arithmetic for the same station, within what the invariance classification already permits. v is the
// tube angle st was read at: the closed form's two scalars are re-derived here, and re-deriving them at
// some OTHER station would compare two different numbers and call the difference rounding.
func assertLevelFormsAgree(t *testing.T, name string, q Quadric, st torusStation, level, v float64) {
	t.Helper()
	constant, m11 := quadricStationScalars(testRing(t), q, v)
	closed := constant + m11*st.rho*st.rho
	if bound := levelFormsBound(q, m11, st.rho, level); stdmath.Abs(level-closed) > bound {
		t.Errorf("%s: general level %.17g vs the closed form's %.17g differ by %.3e, over the %.3e the "+
			"invariance classification admits", name, level, closed, stdmath.Abs(level-closed), bound)
	}
}

// scaledBallForm is a sphere's own quadric written at a SCALE: F and c·F have the same zero set, and
// c is exactly what the in-plane entries read, m₁₁ = m₂₂ = c.
//
// It exists because the kernel cannot build an axis-invariant tensor whose entries are not 1. Every
// quadric of revolution it has is M = I − α·d̂d̂ᵀ (axialTensor3), and when d̂ IS the torus axis both
// in-plane directions are perpendicular to d̂, so m₁₁ = m₂₂ = 1 EXACTLY — for the sphere, the coaxial
// cylinder and the coaxial cone alike. A Quadric is a FORM and not a normalised one, and the reduction's
// algebra is written for arbitrary entries, so this is the member of the family that tells the algebra
// apart from that coincidence.
func scaledBallForm(centre math.Point3, radius, c float64) Quadric {
	return Quadric{Anchor: centre, M: SymmetricTensor3{XX: c, YY: c, ZZ: c}, K: -c * radius * radius}
}

// TestTheLevelSpellingsAgreeWhenTheTensorIsNotUnit is the other half of the reproduction proof above.
//
// TestTheSecondHarmonicVanishesOnTheAxisInvariantFamily holds the general Level against the closed
// form's `constant + m₁₁·ρ²` — but on every axis-invariant fixture the kernel can build m₁₁ = m₂₂ = 1
// (scaledBallForm says why), so the two spellings, `constant + m₁₁·ρ²` and `constant + ρ²(m₁₁+m₂₂)/2`,
// agree whether or not either of them reads the entry at all. This row is the same proof at
// m₁₁ = m₂₂ = 3, where a spelling that dropped the entry would be out by 2ρ², about 25 to 85 on this
// ring rather than by the ulps the bound admits.
func TestTheLevelSpellingsAgreeWhenTheTensorIsNotUnit(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	const scale = 3.0
	q := scaledBallForm(math.P3(3, 2, 1), 2.5, scale)
	for _, v := range []float64{0, 0.7, 2.0, 4.1} {
		if _, m11 := quadricStationScalars(ring, q, v); m11 == 1 {
			t.Fatalf("v=%g: the fixture's m11 is 1, so this row is back to proving the unit case", v)
		}
		h, invariant := torusHarmonicAt(ring, q, v)
		if !invariant {
			t.Fatalf("v=%g: a scaled ball's form is not classified axis-invariant — the classification "+
				"reads the departure relative to the tensor's own norm, so a scale cannot change it", v)
		}
		st := q.stationOn(ring, v)
		g := st.secondHarmonic()
		if g.Cos2 != 0 || g.Sin2 != 0 {
			t.Errorf("v=%g: second harmonic (%g, %g), want exactly zero", v, g.Cos2, g.Sin2)
		}
		assertLevelFormsAgree(t, "scaled ball", q, st, g.Level, v)
		assertNearly(t, "scaled ball reach", stdmath.Hypot(g.Cos1, g.Sin1), h.reach)
		assertNearly(t, "scaled ball phase", stdmath.Atan2(g.Sin1, g.Cos1), h.phase)
	}
}

// levelFormsBound is how far the station's two level spellings may legitimately differ.
//
// The closed form reads constant + m11·ρ² and the general one constant + ρ²(m11+m22)/2, so the two
// differ by ρ²(m11−m22)/2 — and axisInvariantEntries (the gate that decided this station IS invariant)
// admits |m11−m22| up to axisInvarianceTol·scale, with the same scale it uses. That term is the bound;
// levelFormsUlps adds the two spellings' own rounding on top, which is what a platform that fuses the
// closed form's product into its add costs.
func levelFormsBound(q Quadric, m11, rho, level float64) float64 {
	scale := stdmath.Max(q.M.Norm(), stdmath.Abs(m11))
	admitted := axisInvarianceTol * scale * rho * rho / 2
	return admitted + levelFormsUlps*ulpOf(level)
}

// quadricStationScalars re-derives the two scalars the CLOSED level form is written from, here in the
// test rather than read off the station — which is what makes the comparison a comparison of two
// spellings and not of one expression with itself.
func quadricStationScalars(t Torus, q Quadric, v float64) (constant, m11 float64) {
	axis, e1, _ := torusAxisFrame(t)
	_, sv := cosSin(v)
	w0 := q.Anchor.VectorTo(t.Center).Add(axis.Scale(math.Scalar(t.MinorRadius * sv)))
	mw0 := q.M.Apply(w0)
	return float64(w0.Dot(mw0)) + 2*float64(q.G.Dot(w0)) + q.K, float64(e1.Dot(q.M.Apply(e1)))
}

// ulpOf is the spacing of float64 at x — the unit the two spellings' own roundings are counted in.
func ulpOf(x float64) float64 {
	a := stdmath.Abs(x)
	return stdmath.Nextafter(a, stdmath.Inf(1)) - a
}

// levelFormsUlps is how many roundings apart the two spellings of one level may land: each forms its
// own product and sum, and one of them may fuse the product into the sum on a platform that contracts
// x*y+z. It counts ULPS of the value itself, so it carries no model scale.
const levelFormsUlps = 4 // tol:numeric — roundings between two spellings of one quantity

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
			l, ok := torusLaneAt(h, a)
			if !ok {
				t.Fatalf("v=%g anchor %g: the lane is unreadable, but torusLaneAnchors accepted the tracks", v, a)
			}
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
			assertLaneStraddles(t, h, v, a, l)
		}
	}
	if live == 0 || dead == 0 {
		t.Fatalf("the sweep saw %d live and %d dead lanes; it must exercise both", live, dead)
	}
}

// assertLaneStraddles is the property the lane's whole design rests on: its two azimuths lie on
// OPPOSITE sides of the extremum that names it, each strictly between that extremum and the flanking
// one on its side. Nothing weaker distinguishes a correct pairing from the complementary one — a lane
// that took both roots from the same side would still have two certified roots and a positive
// discriminant, and would trace a curve through the wrong arc.
func assertLaneStraddles(t *testing.T, h torusSecondHarmonic, v, anchor float64, l torusLane) {
	t.Helper()
	ex := h.extrema()
	i, n := nearestAngleIndex(ex, l.center), len(ex)
	forward := turnBetween(l.center, l.upper, true)
	backward := turnBetween(l.center, l.lower, false)
	toNext := turnBetween(l.center, ex[(i+1)%n], true)
	toPrev := turnBetween(l.center, ex[(i+n-1)%n], false)
	if forward <= 0 || forward >= toNext {
		t.Fatalf("v=%g anchor %g: the upper azimuth is %g past the extremum, outside (0, %g) to the next one",
			v, anchor, forward, toNext)
	}
	if backward <= 0 || backward >= toPrev {
		t.Fatalf("v=%g anchor %g: the lower azimuth is %g back from the extremum, outside (0, %g) to the previous one",
			v, anchor, backward, toPrev)
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
		l, ok := torusLaneAt(torusSecondHarmonicAt(ring, q, v), h.phase)
		if !ok {
			t.Fatalf("v=%g: the one-harmonic station has no readable lane", v)
		}
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

// TestADroppedSectionLoopIsCaughtByTheAzimuthCount is the live proof of the loop-set certificate. The
// section builder SKIPS a window whose branch pair merges at a flanking extremum, on the premise that
// the neighbouring lane carries that pair itself. Nothing about the skip verifies the premise, and a
// mis-fire deletes a whole section loop — a hole in a solid that simply is not there, with no error.
// So the finished set is counted against the stations: every azimuth a station carries must belong to
// a loop whose window covers it, two per loop. This row removes each loop in turn, and doubles one, and
// requires the count to catch every case.
func TestADroppedSectionLoopIsCaughtByTheAzimuthCount(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	rod, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 1)
	q := rod.QuadricForm()
	loops, why, ok := torusSkewSection(ring, q, ResolutionForSize(12))
	if !ok || why != DeclineNone || len(loops) != 4 {
		t.Fatalf("rod across the ring: ok=%v why=%v loops=%d, want four loops and no decline", ok, why, len(loops))
	}
	if got := torusCurvesAccountForEveryAzimuth(ring, q, loops); got != DeclineNone {
		t.Fatalf("the correct loop set is reported as %v", got)
	}
	for i := range loops {
		short := append(append([]Curve3{}, loops[:i]...), loops[i+1:]...)
		if got := torusCurvesAccountForEveryAzimuth(ring, q, short); got != DeclineTorusLaneUnaccounted {
			t.Errorf("dropping loop %d goes unnoticed: %v", i, got)
		}
	}
	if got := torusCurvesAccountForEveryAzimuth(ring, q, append(loops, loops[0])); got != DeclineTorusLaneUnaccounted {
		t.Errorf("a doubled loop goes unnoticed: %v", got)
	}
}

// TestAnOrdinaryRefusalIsNotAConditioningDemotion: a torus PAIR has no closed form in any bucket, and
// that is not a degradation — nothing was given up. Reporting it as one would put a defect on every
// marched boolean in the system, which is the noise that makes a diagnostic worthless.
func TestAnOrdinaryRefusalIsNotAConditioningDemotion(t *testing.T) {
	t.Parallel()
	first, err := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 4, 2)
	if err != nil {
		t.Fatalf("elliptical cylinder: %v", err)
	}
	second, err := NewEllipticalCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0), 4, 2)
	if err != nil {
		t.Fatalf("crossing elliptical cylinder: %v", err)
	}
	_, why, ok := IntersectSurfacesAnalyticDeclining(first, second, ResolutionForSize(12))
	if ok || why != DeclineNoClosedForm || why.IsConditioning() {
		t.Errorf("elliptical cylinder pair: ok=%v why=%v conditioning=%v, want the ordinary refusal", ok, why, why.IsConditioning())
	}
}

// TestAFoldedLoopSitsOnItsStationsTangencyAtEveryFold is the fold invariant TorusSectionLoop rests on.
// At a fold the tube circle TOUCHES the quadric: the two azimuths have merged onto the station's own
// extremum, so the quadric's rate of change along that circle vanishes there. The loop's fold point
// must sit at that tangency.
//
// A window end is a bisected root of the discriminant, so the discriminant at it is zero only to
// rounding, and where it rounds POSITIVE the branch pair still separates — by half a square root of
// that rounding, ~1e-8 in azimuth, which is a slope of ~1e-8 of the station's scale rather than zero.
// The loop's closure point then carried that noise, and whether it landed on a host wall's own seam
// ruling was decided by the last bit: on arm64, where the compiler fuses x*y+z, a tilted drill's
// section closed 1e-7 off the wall's seam and the wall's chart arranged into a different set of cells
// (CI run 34280554924 macos-latest, ADR-0061).
func TestAFoldedLoopSitsOnItsStationsTangencyAtEveryFold(t *testing.T) {
	t.Parallel()
	ring := testRing(t)
	folds := 0
	for _, c := range skewTestQuadrics(t) {
		curves, _, ok := IntersectSurfacesAnalyticDeclining(ring, c.surface, ResolutionForSize(20))
		if !ok {
			continue
		}
		for _, cv := range curves {
			l, folded := cv.(TorusSectionLoop)
			if !folded {
				continue
			}
			folds += 2
			assertFoldIsTangent(t, c.name, ring, c.quad, l, 0)
			assertFoldIsTangent(t, c.name, ring, c.quad, l, stdmath.Pi)
			if start, end := l.PointAt(0), l.PointAt(1); start != end {
				t.Errorf("%s: the loop's ends are %v and %v; a folded loop closes on ONE point", c.name, start, end)
			}
		}
	}
	if folds < 2 { // never pass vacuously: the family must yield folded loops for this row to prove anything
		t.Fatalf("the family produced %d folds; this row proves nothing without them", folds)
	}
}

// assertFoldIsTangent reads the station polynomial's slope at the azimuth the loop takes at one fold.
func assertFoldIsTangent(t *testing.T, name string, ring Torus, q Quadric, l TorusSectionLoop, s float64) {
	t.Helper()
	v := l.vAt(s)
	h := torusSecondHarmonicAt(ring, q, v)
	u := l.azimuthAt(s, v)
	if slope := stdmath.Abs(h.slopeAt(u)); slope > foldTangencyTol*h.scale() {
		t.Errorf("%s: at the fold v=%g the loop takes azimuth %.17g, where the station's slope is %.3e "+
			"(%.3e relative) — the branches have not merged onto the extremum", name, v, u, slope, slope/h.scale())
	}
}

// foldTangencyTol is how far off tangency a fold azimuth may read, relative to the station polynomial's
// own coefficient scale. It compares a slope with the coefficients it was formed from, so it carries no
// model scale; a branch root read at a fold misses it by ~1e-8, six orders above this.
const foldTangencyTol = 1e-13 // tol:numeric — relative slope at a station's tangency
