// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// The torus × torus corpus (ADR-0066, Oblikovati#3514).
//
// ADR-0061 stage 5 refused this pair by name on the premise that the second-harmonic reduction needs
// an implicit QUADRIC on the other side. These rows are the evidence that the premise was wrong: the
// reduction needs an implicit form whose restriction to a CIRCLE is degree two, and a torus's quartic
// is one, because four of the eight intersections Bézout counts are spent at the circular points at
// infinity.
//
// Each row certifies a different layer, and the layers are ordered so a failure names its own cause:
// the coefficients against the quartic they claim to be; the certified roots against both surfaces;
// the built section against both surfaces and against its own closure; and the chart assignment
// against the caller's argument order.

// torusQuarticOracle is the torus's implicit quartic, written here from its DEFINITION rather than
// read off the reduction — so a row comparing the two compares two derivations and not one expression
// with itself.
//
//	F(X) = (|W|² + R² − r²)² − 4R²(|W|² − (W·â)²),  W = X − Center
func torusQuarticOracle(t Torus, p math.Point3) float64 {
	w := t.Center.VectorTo(p)
	s := float64(w.Dot(w))
	a := float64(w.Dot(t.AxisDir.AsVector()))
	g := s + t.MajorRadius*t.MajorRadius - t.MinorRadius*t.MinorRadius
	return g*g - 4*t.MajorRadius*t.MajorRadius*(s-a*a)
}

// distanceToTorusSurface is the distance from p to the SURFACE OF REVOLUTION of the torus's meridian
// circle, taken independently of anything in the kernel. It reads the meridian circle on BOTH sides of
// the axis, because a torus whose tube reaches past its own axis sweeps the far half of that circle
// into the near half-plane — so this is the right oracle for a ring torus and a spindle alike.
func distanceToTorusSurface(t Torus, p math.Point3) float64 {
	w := t.Center.VectorTo(p)
	z := float64(w.Dot(t.AxisDir.AsVector()))
	d := float64(w.Sub(t.AxisDir.AsVector().Scale(math.Scalar(z))).Length())
	near := stdmath.Abs(stdmath.Hypot(d-t.MajorRadius, z) - t.MinorRadius)
	far := stdmath.Abs(stdmath.Hypot(d+t.MajorRadius, z) - t.MinorRadius)
	return stdmath.Min(near, far)
}

// randomRingTorus draws a ring torus (tube inside the hole radius, the family a solid is built from)
// with its centre and axis anywhere.
func randomRingTorus(t *testing.T, rng *rand.Rand, spread float64) Torus {
	t.Helper()
	major := 1 + rng.Float64()*8
	minor := 0.1 + rng.Float64()*(major-0.15)
	centre := math.P3(rng.NormFloat64()*spread, rng.NormFloat64()*spread, rng.NormFloat64()*spread)
	axis := math.V3(rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64())
	tor, err := NewTorus(centre, axis, major, minor)
	if err != nil {
		t.Fatalf("random torus (major %g, minor %g, axis %v): %v", major, minor, axis, err)
	}
	return tor
}

// TestTheTorusReductionIsTheOtherTorusQuartic is the first layer: the five coefficients the reduction
// produces at a station must BE the other torus's quartic restricted to that tube circle, at every
// azimuth. Everything downstream — the roots, the lanes, the folds, the certificate — reads only these
// five numbers, so an error here would be invisible to every later row and fatal to all of them.
//
// The residual is judged against the polynomial's OWN coefficient scale, which is what
// torusSecondHarmonic.azimuths certifies a candidate root against (torusRootResidualTol, 1e-9). The
// margin is therefore the one that matters, not an absolute distance.
func TestTheTorusReductionIsTheOtherTorusQuartic(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(3))
	worst := 0.0
	for range torusReductionSamples {
		chart, other := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
		v, u := rng.Float64()*twoPi, rng.Float64()*twoPi
		poly := other.stationOn(chart, v).secondHarmonic()
		got, want := poly.valueAt(u), torusQuarticOracle(other, chart.PointAt(u, v))
		worst = stdmath.Max(worst, stdmath.Abs(got-want)/poly.scale())
	}
	t.Logf("worst relative residual over %d random (chart, other, u, v): %.3e", torusReductionSamples, worst)
	if worst > torusReductionResidualBound {
		t.Errorf("the reduction departs from the quartic by %.3e of the polynomial's own scale, over the %.3e "+
			"a certified root is allowed", worst, torusReductionResidualBound)
	}
}

// torusReductionSamples is how many random (chart, other, u, v) the reduction is checked at. The
// identity is algebraic, so this is a search for a conditioning regime rather than a statistical claim.
const torusReductionSamples = 20000

// torusReductionResidualBound is measured, not chosen: the worst residual over the sample above is
// 4.56e-14 relative, and this allows a little over an order of magnitude on top. It is far under the 1e-9 a root has
// to certify within, which is the margin the layers above rest on.
const torusReductionResidualBound = 1e-12

// TestEveryTorusPairStationRootLiesOnBothTori is the second layer, and it is the one the ground rule
// asks for by name: "certify a root or branch choice at runtime against the geometry (position,
// second-order test)". A root that certifies against the POLYNOMIAL could still be a point the
// polynomial should not have had — so every certified azimuth is measured against both surfaces, with
// an oracle that knows nothing about the reduction.
func TestEveryTorusPairStationRootLiesOnBothTori(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(5))
	worst, roots := 0.0, 0
	for range torusStationRootPairs {
		chart, other := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
		v := rng.Float64() * twoPi
		for _, u := range other.stationOn(chart, v).secondHarmonic().azimuths() {
			p := chart.PointAt(u, v)
			worst = stdmath.Max(worst, stdmath.Max(distanceToTorusSurface(chart, p), distanceToTorusSurface(other, p)))
			roots++
		}
	}
	t.Logf("%d certified station roots over %d random pairs, worst distance to both surfaces %.3e",
		roots, torusStationRootPairs, worst)
	if roots == 0 {
		t.Fatal("no station carried a root: the corpus proves nothing")
	}
	if worst > torusStationRootDistanceBound {
		t.Errorf("a certified root sits %.3e off a surface it claims to be on, over the %.3e bound", worst, torusStationRootDistanceBound)
	}
}

// torusStationRootPairs is how many random torus pairs contribute one station each.
const torusStationRootPairs = 400

// torusStationRootDistanceBound is the measured worst (9.23e-14 over the 284 roots the sample above
// carries) with two orders of headroom. It is an absolute LENGTH because the corpus's tori are of order 10 units, so
// it is a relative 1e-13 at that size.
const torusStationRootDistanceBound = 1e-11

// TestATorusPairSectionLiesOnBothTori is the third layer: the built curves, sampled along their own
// parameter, on both surfaces — and each closed curve closing on itself, which is what an imprint needs
// of a section before it can bound a face.
func TestATorusPairSectionLiesOnBothTori(t *testing.T) {
	t.Parallel()
	for _, row := range torusPairCorpus(t) {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			curves, why, ok := IntersectSurfacesAnalyticDeclining(row.a, row.b, ResolutionForSize(20))
			if !ok {
				t.Fatalf("declined %v, want an exact section", why)
			}
			chart, co, _ := torusSectionRoles(row.a, row.b)
			if want := torusSectionComponents(chart, co); len(curves) != want {
				t.Errorf("%d section curves, but the chart's zero set has %d connected components", len(curves), want)
			}
			if len(curves) != row.curves {
				t.Fatalf("%d section curves, want %d", len(curves), row.curves)
			}
			assertSectionOnBothTori(t, row.a, row.b, curves)
		})
	}
}

// torusPairRow is one corpus pair with the section it must produce.
type torusPairRow struct {
	name   string
	a, b   Torus
	curves int
}

// torusPairCorpus is the shape corpus. The loop count is not asserted from the kernel's own answer: it
// is the number of connected components the section has on the chart, counted independently by
// torusSectionComponents, so a row that lost a loop fails here rather than passing quietly.
func torusPairCorpus(t *testing.T) []torusPairRow {
	t.Helper()
	ring := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	return []torusPairRow{
		// The pair ADR-0061 recorded as the standing refusal, and the fixture
		// ops/boolean's TestATorusPairIsRefusedByName drove.
		{"linked rings", ring, mustTorus(t, math.P3(5, 0, 0), math.V3(1, 0, 0), 5, 1.5), 2},
		// kernel/brep's TestCurvedImprintTorusPairDefers fixture, at its own radii.
		{"brep guard rings", mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1),
			mustTorus(t, math.P3(4, 0, 0), math.V3(1, 0, 0), 4, 1), 2},
		// A small ring threaded through the big one's hole and out through its tube — the shape a
		// chain link makes against the link it hangs from.
		{"small ring through the hole", ring, mustTorus(t, math.P3(3.5, 0, 0), math.V3(1, 0, 0), 2, 0.7), 2},
		// A ring TILTED out of the first ring's plane about a shared centre — ADR-0066 recorded this as
		// a Tracks refusal; #3515's full-period branches carry it (worst 3.91e-15 off the co-form over
		// 20 001 samples per curve).
		{"a ring tilted out of the ring's plane", ring,
			mustTorus(t, math.P3(0, 0, 0), math.V3(0.4, 0, 1), 5, 1.2), 2},
		// A torus boss sunk into the ring's tube — ADR-0066 recorded this as an unreadable station;
		// its branches wrap and #3515 carries them (worst 5.76e-15).
		{"a torus boss sunk into the tube", ring,
			mustTorus(t, math.P3(5, 0, 0), math.V3(0, 0, 1), 2, 0.6), 2},
		// COAXIAL: two rings on one axis whose meridian circles cross, so the section is whole tube
		// circles about that axis and no azimuth is resolved at all — the family the classification
		// routes away from the lanes entirely.
		{"coaxial rings", ring, mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 6, 1.5), 2},
		// The same family with the second ring displaced ALONG the shared axis, so the level term's
		// roots sit at tube angles neither ring's own symmetry supplies.
		{"coaxial rings, offset along the axis", ring,
			mustTorus(t, math.P3(0, 0, 2), math.V3(0, 0, 1), 5, 1.5), 2},
	}
}

// TestTheCoCentredPerpendicularRingsSectionIsExact is the pair ADR-0066 recorded as
// [DeclineTorusSectionOffItsForm] — the row that made torusSectionSatisfiesItsForm necessary, because
// the folded-window reading put its points 1.353e-5 off the surface they claimed to be on.
//
// It now BUILDS, and the cause is #3515's chart shift rather than anything about folds. The pair's
// branches sit at azimuth 0 and π at the tangency, and the station quartic in tan(u/2) carries its
// leading coefficient at f(π): a root there collapses the solve and costs the other roots their
// residual certificate, which is exactly the excursion ADR-0066 measured. With the station read on a
// chart turned off its pole the four loops are exact.
//
// It is a row of its own rather than a corpus row because the component ORACLE disagrees with the curve
// count here, for a reason that is understood and measured rather than unexplained. The four loops meet
// PAIRWISE at the two tangency folds — loops 0 and 1 both pass through (0, 6.5, 0) and (0, 3.5, 0), and
// loops 2 and 3 through their antipodes — so the section's point SET has two connected components while
// the reduction's answer is four closed curves that cross at four points. Both are right about their own
// question; torusSectionComponents counts the set, and a pinch is invisible to it.
func TestTheCoCentredPerpendicularRingsSectionIsExact(t *testing.T) {
	t.Parallel()
	ring := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	perp := mustTorus(t, math.P3(0, 0, 0), math.V3(1, 0, 0), 5, 1.5)
	curves, why, ok := IntersectSurfacesAnalyticDeclining(ring, perp, ResolutionForSize(20))
	if !ok {
		t.Fatalf("declined %v, want an exact section", why)
	}
	if len(curves) != 4 {
		t.Fatalf("%d section curves, want the four loops the two lanes' two windows carry", len(curves))
	}
	assertSectionOnBothTori(t, ring, perp, curves)
	assertEveryCurveIsClosed(t, curves)
}

// assertEveryCurveIsClosed requires each section curve to come back to where it started — what an
// imprint needs of a section before it can bound a face.
func assertEveryCurveIsClosed(t *testing.T, curves []Curve3) {
	t.Helper()
	for i, c := range curves {
		if gap := float64(c.PointAt(0).VectorTo(c.PointAt(1)).Length()); gap > torusStationRootDistanceBound {
			t.Errorf("curve %d does not close: its ends are %.3e apart", i, gap)
		}
	}
}

// TestATorusPairOutsideTheEnvelopeIsRefusedByName is the other half of the corpus, and it is a row
// rather than a deletion because a refusal is a result. Each of these pairs MEETS and each is refused,
// by a name that says which conditioning gate stopped it — never silently, and never with a section.
//
// Both names belong to the lane machinery this reduction reuses, not to the reduction itself:
//
//   - Tracks: torusLaneAnchors seeds the lane labels from the station at v = 0 and requires every one
//     of the 720 stations to carry the same number of extrema, tracking those seeds. A torus co-form's
//     station changes between two and four extrema over the turn far more often than a quadric's does,
//     and the gate then cannot say which branch pair is which. It is the capability's dominant
//     remaining gap, and the follow-up ADR-0066 names — seeding from a root-carrying station — is
//     scoped against it rather than claimed to close it.
//
// Re-measured at #3515's head over 2343 meeting random ring pairs (seed 3), which is the same
// experiment ADR-0066 reported as 821 built of 2410:
//
//	BUILT                                     1258
//	extremum tracks are not separable          1009
//	curves are not the azimuths certified        76
//
// ADR-0066's FullTurn column (25) is gone with the decline itself, its post-condition column (1) is
// empty, and the built share goes 34% → 54%. The Tracks gate is still the gap.
//   - Separation: the two branches never part by more than the stitch resolution, so the loop would be
//     a sliver two faces could not be told apart across.
func TestATorusPairOutsideTheEnvelopeIsRefusedByName(t *testing.T) {
	t.Parallel()
	ring := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	for _, row := range []struct {
		name string
		a, b Torus
		want SectionDecline
	}{
		{"torus boss on the ring's flank", ring,
			mustTorus(t, math.P3(6, 0, 0), math.V3(0, 0, 1), 1.2, 0.5), DeclineTorusLaneTracks},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			curves, why, ok := IntersectSurfacesAnalyticDeclining(row.a, row.b, ResolutionForSize(20))
			if ok || len(curves) != 0 {
				t.Fatalf("ok=%v with %d curves, want the named refusal %v", ok, len(curves), row.want)
			}
			if why != row.want {
				t.Errorf("refused %q, want %q", why, row.want)
			}
			if !why.IsConditioning() {
				t.Error("a conditioning gate's refusal must report as one, so the boolean records a defect")
			}
		})
	}
}

// mustTorus builds a fixture torus or fails the test.
func mustTorus(t *testing.T, centre math.Point3, axis math.Vector3, major, minor float64) Torus {
	t.Helper()
	tor, err := NewTorus(centre, axis, major, minor)
	if err != nil {
		t.Fatalf("torus at %v: %v", centre, err)
	}
	return tor
}

// assertSectionOnBothTori samples every curve and measures each point against both surfaces, then
// checks the curve closes.
func assertSectionOnBothTori(t *testing.T, a, b Torus, curves []Curve3) {
	t.Helper()
	worst := 0.0
	for i, c := range curves {
		for s := 0; s <= torusSectionSamples; s++ {
			p := c.PointAt(float64(s) / torusSectionSamples)
			worst = stdmath.Max(worst, stdmath.Max(distanceToTorusSurface(a, p), distanceToTorusSurface(b, p)))
		}
		if gap := float64(c.PointAt(0).VectorTo(c.PointAt(1)).Length()); gap > torusStationRootDistanceBound {
			t.Errorf("curve %d (%T) does not close: its ends are %.3e apart", i, c, gap)
		}
	}
	t.Logf("%d curves sampled at %d points each, worst distance to both surfaces %.3e", len(curves), torusSectionSamples+1, worst)
	if worst > torusStationRootDistanceBound {
		t.Errorf("a section point sits %.3e off a surface it claims to be on, over the %.3e bound", worst, torusStationRootDistanceBound)
	}
}

// torusSectionSamples is how many points each section curve is measured at.
const torusSectionSamples = 400

// TestTheChartAssignmentDoesNotDependOnTheCallerOrder is the byte-identity row. A torus PAIR is the one
// case where both role assignments apply, so it is the one case where a first-match dispatch would let
// the arrangement's face order decide which surface the section is parametrised on — and two different
// parametrisations of one curve are two different sets of bytes downstream.
func TestTheChartAssignmentDoesNotDependOnTheCallerOrder(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewSource(9))
	for range torusChartOrderPairs {
		a, b := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
		forward, _, okF := torusSectionRoles(a, b)
		backward, _, okB := torusSectionRoles(b, a)
		if !okF || !okB {
			t.Fatal("a torus pair must always assign roles")
		}
		if forward != backward {
			t.Fatalf("chart depends on the argument order: %v forward, %v backward", forward.Center, backward.Center)
		}
	}
}

// torusChartOrderPairs is how many random pairs the order-independence row draws.
const torusChartOrderPairs = 2000

// TestTheNarrowerTubeIsTheBetterChart is the evidence behind torusChartPrecedes, and it is here because
// a claim about conditioning is worth what it is measured at. Every station is built from the chart's
// tube circle at tube angle v — radius R + r·cos v, centre offset r·sin v — so the CHART's own tube sets
// how far the station's geometry swings over the turn: a narrow chart's extremum tracks keep their
// identity, a wide one's change in number. The chart should therefore be the NARROWER torus.
//
// This row asserts NARROW against WIDE and nothing finer. Which narrow KEY — the minor radius or the
// aspect r/R — is not decided here and cannot be: the two are inside each other's noise on every seed
// measured, including one where the aspect wins. That choice is made by the `small ring through the
// hole` row of torusPairCorpus, and torusChartPrecedes says so.
//
// ADR-0066 measured this the other way round and wrote the wider-tube rule. Its experiment was run
// while a branch pair that never folds was still a REFUSAL — which is exactly the shape a narrow chart
// produces, so its wins were being declined rather than counted. #3515 carries those branches, and the
// same experiment reverses on every seed (Oblikovati/Oblikovati#3515).
//
// The row asserts the DIRECTION, not a rate: the chart order's own assignment must build at least as
// many sections as the reverse one. A pair both assignments decline is a genuine refusal of the
// reduction and counts for neither. "Builds" here means the section came back non-empty AND lies on the
// co-form: a build that is wrong is worse than a decline, and an objective that counts only builds
// would prefer the assignment that is wrong more often.
func TestTheNarrowerTubeIsTheBetterChart(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier: `make test-corpus`")
	}
	t.Parallel()
	rng := rand.New(rand.NewSource(13))
	chosen, reversed, both := 0, 0, 0
	for range torusChartCorpusPairs {
		a, b := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
		picked, other := a, b
		if torusChartPrecedes(b, a) {
			picked, other = b, a
		}
		builtPicked, builtOther := torusSectionBuilds(t, picked, other), torusSectionBuilds(t, other, picked)
		switch {
		case builtPicked && builtOther:
			both++
		case builtPicked:
			chosen++
		case builtOther:
			reversed++
		}
	}
	t.Logf("over %d random pairs: both assignments build %d, only the CHOSEN (thin) chart builds %d, "+
		"only the REVERSED (fat) chart builds %d", torusChartCorpusPairs, both, chosen, reversed)
	if chosen < reversed {
		t.Errorf("the chosen chart built %d sections the reverse declined and lost %d the other way; "+
			"torusChartPrecedes picks the worse chart", chosen, reversed)
	}
}

// torusSectionBuilds reports the reduction solving this role assignment to a non-empty section that
// LIES on the co-form, walked at 2001 samples per curve — an order of magnitude finer than the section's
// own 257-sample post-condition, so the chart comparison is not decided by the gate it is comparing.
func torusSectionBuilds(t *testing.T, chart, other Torus) bool {
	t.Helper()
	curves, _, ok := TorusSection(chart, other, ResolutionForSize(20))
	if !ok || len(curves) == 0 {
		return false
	}
	for _, c := range curves {
		for i := range 2001 {
			if d := other.distanceTo(c.PointAt(float64(i) / 2000)); !(d <= torusChartWeld(chart)) {
				return false
			}
		}
	}
	return true
}

// torusChartCorpusPairs is the chart-order corpus's size.
const torusChartCorpusPairs = 4000

// TestACoaxialSectionNarrowerThanTheOldGridIsStillFound is the row that makes the coaxial family's
// exactification load-bearing rather than cosmetic.
//
// How many circles a coaxial section has is a TOPOLOGICAL question, and it used to be answered by
// scanning the level at 720 tube angles and bisecting each sign change. A grid answers a topological
// question wrongly whenever the feature is narrower than the grid, and bisection afterwards cannot
// recover a crossing the scan stepped over — which is the shape ADR-0065's review found fatal one
// bucket over, where a wrap decided from 720 samples built bodies that were badly wrong while Validate
// called them valid.
//
// The fixture is DERIVED rather than pasted, from the algebra torus_coaxial_section.go sets out. With
// the chart (R, r) and a coaxial co-form of major radius R − δ offset d along the shared axis, the
// level's near factor is
//
//	F(v) = L + reach·cos(v − phase),  L = δ² + d² + r² − r_b²,  reach = 2r·hypot(δ, d),  phase = atan2(d, δ)
//
// so its two roots sit at phase ± arccos(−L/reach). Choosing where that window's CENTRE and HALF-WIDTH
// should fall therefore fixes the co-form:
//
//	d = δ·tan(centre),   r_b = √(δ² + d² + r² + reach·cos(halfWidth))
//
// The centre is put HALF a grid step off a probe and the half-width at an eighth of a step, so the whole
// window falls strictly between two of the old scan's samples. Geometrically it is a pair whose meridian
// circles are almost internally tangent, poking out of each other over a fifth of a milliradian.
func TestACoaxialSectionNarrowerThanTheOldGridIsStillFound(t *testing.T) {
	t.Parallel()
	const major, minor, delta = 5.0, 1.5, 0.5
	step := float64(twoPi / torusStationProbes)
	centre, halfWidth := float64(step/2), float64(step/8)
	d := delta * stdmath.Tan(centre)
	reach := 2 * minor * stdmath.Hypot(delta, d)
	coMinor := stdmath.Sqrt(delta*delta + d*d + minor*minor + reach*stdmath.Cos(halfWidth))
	chart := mustTorus(t, math.P3(0, 0, 0), math.V3(0, 0, 1), major, minor)
	co := mustTorus(t, math.P3(0, 0, float64(math.Scalar(-d))), math.V3(0, 0, 1), major-delta, coMinor)
	if fam := co.sectionFamily(chart); fam != torusFamilyCoaxial {
		t.Fatalf("the pair classified as family %d, want the coaxial family", fam)
	}
	curves, why, ok := TorusSection(chart, co, ResolutionForSize(20))
	if !ok {
		t.Fatalf("declined %v, want the two tube circles", why)
	}
	if len(curves) != 2 {
		t.Fatalf("%d section circles, want 2 — the near-tangent pair crosses twice", len(curves))
	}
	assertSectionOnBothTori(t, chart, co, curves)
	if found := signChangesOnTheOldGrid(chart, co); found > 0 {
		t.Errorf("a %d-probe scan of the level found %d sign changes; the row no longer proves the "+
			"closed form catches what a grid steps over", torusStationProbes, found)
	}
}

// signChangesOnTheOldGrid counts the level's sign changes the way the coaxial family used to find them:
// a fixed scan at torusStationProbes tube angles. It exists only so the row above can show that scan
// missing a section the closed form resolves.
func signChangesOnTheOldGrid(chart Torus, co TorusCoForm) int {
	level := func(v float64) float64 { return co.stationOn(chart, v).secondHarmonic().Level }
	n, prev := 0, level(0)
	for i := 1; i <= torusStationProbes; i++ {
		cur := level(float64(twoPi * float64(i) / torusStationProbes))
		if (prev > 0) != (cur > 0) {
			n++
		}
		prev = cur
	}
	return n
}
