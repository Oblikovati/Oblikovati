// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"math/rand"
	"testing"

	"oblikovati.org/math"
)

// The corpus behind the full-period arc (Oblikovati/Oblikovati#3515, review round 2).
//
// A section that BALANCES is not a section that is RIGHT. Review round 2 found the wrap building bodies
// whose curves ran up to 8 units off the tool — one of them a boolean carrying thirteen times the
// correct volume, valid, closed, manifold, with nothing recorded — on an input the wave base had refused
// by name. The azimuth census could not see it, because a census reads the same stations the
// construction did.
//
// So the gate here is not a count and not an identity. It is the only thing that cannot be satisfied by
// a wrong curve: every point of every arc the closed form ships must lie ON the quadric. It is sampled
// at 997 parameters offset by an irrational fraction of a step, so the check cannot land on the same
// tube angles the construction or the census read.
//
// FOLDED LOOPS ARE OUT OF SCOPE HERE, and deliberately: four of the rows this sweep generates build
// loops that run off the quadric or evaluate NaN, and all four do so IDENTICALLY at the wave base
// fff94140 (measured: rows at R=4.044/0.692, R=5.045/1.181, R=5.794/2.990, R=2.605/1.474). They are a
// pre-existing defect of the folded-window path, proven pre-existing by bisect and not touched by this
// branch, and they need their own issue. Gating them here would either fail on work this branch did not
// do or license a failure count, and a licensed failure count is not a gate.

// TestEveryFullPeriodArcLiesOnBothSurfaces sweeps random torus × cylinder pairs and requires every arc
// the closed form builds to be exactly on the quadric — it is on the torus by construction, being
// evaluated on the torus's own chart.
func TestEveryFullPeriodArcLiesOnBothSurfaces(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~35s): `make test-corpus`")
	}
	t.Parallel()
	rng, arcs := rand.New(rand.NewSource(7)), 0
	for range wrapCorpusRows {
		ring, rod, ok := wrapCorpusPair(rng)
		if !ok {
			continue
		}
		arcs += assertArcsLieOnTheQuadric(t, ring, rod)
	}
	if arcs < 100 { // never pass vacuously: the sweep has to produce the shape it gates
		t.Fatalf("the sweep built %d full-period arcs; this row proves nothing without them", arcs)
	}
}

// TestTheSectionThatBeatTheSampler pins the input that broke the sampled wrap. The wave base refused it
// by name ("the torus section's branch pair never folds"); rounds 1 and 2 built it with one arc 26.2
// units off the rod, because at v = 2.953097 the station's quartic loses two of its four roots — a root
// sits at the half-turn there, the quartic drops toward a cubic, and the ill-conditioning costs the
// others their residual certificate — and the arc then evaluated to its lane's own extremum.
//
// It is here as a row rather than as a fixed expectation about roots: the assertion is that the built
// curves are on the rod, which is the property that was violated.
func TestTheSectionThatBeatTheSampler(t *testing.T) {
	t.Parallel()
	ring, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 9.775632, 2.621941)
	if err != nil {
		t.Fatalf("NewTorus: %v", err)
	}
	rod, err := NewCylinder(
		math.P3(3.360585805574098, -1.7574864466724531, 0.3390622878733773),
		math.V3(-0.9065753377189599, 0.33242730045605945, -0.2600254736583529), 6.285269)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	if n := assertArcsLieOnTheQuadric(t, ring, rod); n != 4 {
		t.Fatalf("the row built %d full-period arcs, want the four its stations carry", n)
	}
}

// TestTheChartTurnsAwayFromItsPole is the mechanism, isolated. The tan(u/2) substitution's quartic has
// its leading coefficient at f(π), so a root sitting at the half-turn collapses the solve and costs the
// OTHER roots their residual certificate. At this tube angle the station's DERIVATIVE has a root there,
// and the unshifted solve reports one extremum where the polynomial has four — which is a station no lane
// can be read on, and which is why an arc built through it evaluated 26 units off the rod.
func TestTheChartTurnsAwayFromItsPole(t *testing.T) {
	t.Parallel()
	ring, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.7212809718869881, 0.30874449734939147)
	rod, _ := NewCylinder(
		math.P3(-0.38335245856404776, 1.0499003912577654, 0.027046400310449591),
		math.V3(0.98945558324734528, -0.012704152684197204, -0.1442783881433134), 0.72106715640905117)
	slope := torusSecondHarmonicAt(ring, rod.QuadricForm(), 0.8527883816143).derivative()
	if got := len(unshiftedCertifiedRoots(slope)); got != 2 {
		t.Fatalf("the unshifted chart certified %d roots; this row needs the station that beats it", got)
	}
	if slope.chartShift() == 0 {
		t.Fatal("the pole floor did not fire on a station whose root sits at the half-turn")
	}
	if got := slope.azimuths(); len(got) != 4 {
		t.Fatalf("the turned chart certified %d roots %v; the polynomial has four", len(got), got)
	}
}

// unshiftedCertifiedRoots solves the station in the plain chart only — [torusSecondHarmonic.azimuths]
// without the shift — so a row can show what the shift is for.
func unshiftedCertifiedRoots(h torusSecondHarmonic) []float64 {
	out := make([]float64, 0, 4)
	for _, u := range trigQuadraticRoots(2*h.Cos2, h.Sin2, h.Cos1, h.Sin1, h.Level-h.Cos2) {
		u = h.polish(u)
		if stdmath.Abs(h.valueAt(u)) <= torusRootResidualTol*h.scale() {
			out = append(out, u)
		}
	}
	return sortedDedupedAngles(out)
}

// assertArcsLieOnTheQuadric builds the section and requires every full-period arc in it to lie on the
// quadric at every sampled parameter. It returns how many arcs it checked.
func assertArcsLieOnTheQuadric(t *testing.T, ring Torus, rod Cylinder) int {
	t.Helper()
	q := rod.QuadricForm()
	res := ResolutionForSize(4 * ring.MajorRadius)
	curves, _, ok := torusSkewSection(ring, q, res)
	if !ok {
		return 0
	}
	n := 0
	for _, cv := range curves {
		arc, isArc := cv.(TorusSectionArc)
		if !isArc {
			continue
		}
		n++
		assertOneArcLiesOnTheQuadric(t, ring, rod, arc, res)
	}
	return n
}

// assertOneArcLiesOnTheQuadric samples one arc at parameters offset by an irrational fraction of a step,
// so the check cannot share a grid with the sweep that built the arc or the census that passed it.
func assertOneArcLiesOnTheQuadric(t *testing.T, ring Torus, rod Cylinder, arc TorusSectionArc, res Resolution) {
	t.Helper()
	q := rod.QuadricForm()
	for i := range wrapCorpusSamples {
		p := arc.PointAt(float64((float64(i) + wrapCorpusOffset) / wrapCorpusSamples))
		if off := quadricSurfaceGap(q, p); stdmath.IsNaN(off) || off > res.Weld() { // NaN fails this, and must
			t.Fatalf("ring(R=%.17g r=%.17g) rod(r=%.17g at %.17v along %.17v): arc on lane %.17g is %g off the quadric at sample %d",
				ring.MajorRadius, ring.MinorRadius, rod.Radius, rod.Origin, rod.AxisDir, arc.UA, off, i)
		}
	}
}

// quadricSurfaceGap is the first-order distance from a point to the quadric — the Newton step |F|/|∇F|,
// which is a LENGTH and so comparable with the model's own weld, where the raw value |F| is not.
func quadricSurfaceGap(q Quadric, p math.Point3) float64 {
	grad := q.M.Apply(q.Anchor.VectorTo(p)).Scale(2).Add(q.G.Scale(2))
	return float64(stdmath.Abs(q.ValueAt(p)) / stdmath.Max(float64(grad.Length()), stdmath.SmallestNonzeroFloat64))
}

// wrapCorpusPair draws one random ring and one random rod, sized and placed so that a good share of the
// draws actually cross. The seed is fixed, so the corpus is the same on every run and platform.
func wrapCorpusPair(rng *rand.Rand) (Torus, Cylinder, bool) {
	majorR := 1 + 9*rng.Float64()
	ring, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), majorR, wrapCorpusRadius(rng, majorR))
	if err != nil {
		return Torus{}, Cylinder{}, false
	}
	base := math.P3(
		math.Scalar(float64(2*majorR*(rng.Float64()-0.5))),
		math.Scalar(float64(2*majorR*(rng.Float64()-0.5))),
		math.Scalar(float64(2*float64(ring.MinorRadius)*(rng.Float64()-0.5))))
	dir := math.V3(math.Scalar(rng.NormFloat64()), math.Scalar(rng.NormFloat64()), math.Scalar(rng.NormFloat64()))
	rod, err := NewCylinder(base, dir, wrapCorpusRadius(rng, majorR))
	if err != nil {
		return Torus{}, Cylinder{}, false
	}
	return ring, rod, true
}

// wrapCorpusRadius draws a radius between a twentieth and three quarters of the ring's major radius —
// the band in which a rod both reaches the tube and can be fatter than it.
func wrapCorpusRadius(rng *rand.Rand, majorR float64) float64 {
	return float64(0.05*majorR) + float64(0.7*majorR*rng.Float64())
}

const (
	// wrapCorpusRows is how many random pairs the sweep draws. At this count it produces both shapes the
	// reduction builds and the near-degenerate configurations between them.
	wrapCorpusRows = 4000
	// wrapCorpusSamples is how many parameters each arc is read at — prime, so it shares no factor with
	// the construction's 720 stations or the census's 1440.
	wrapCorpusSamples = 997
	// wrapCorpusOffset shifts every sample off the grid by an irrational fraction of a step (1/π), so a
	// curve that is wrong only BETWEEN the stations it was built from still fails here.
	wrapCorpusOffset = 0.31830988618379067
)

// TestAPartialStationIsRefusedByItsPostCondition drives torusStationRootsAreComplete both ways. The
// chart shift means it does not fire on any real station any more (0 in 1 920 000), so the only way to
// exercise it is to take a station apart — which is what a post-condition is for.
func TestAPartialStationIsRefusedByItsPostCondition(t *testing.T) {
	t.Parallel()
	ring, _ := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 1.5)
	fat, _ := NewCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 2)
	h := torusSecondHarmonicAt(ring, fat.QuadricForm(), 0)
	ex, roots := h.extrema(), h.azimuths()
	if len(roots) != 4 {
		t.Fatalf("the fixture station carries %d azimuths, want the four this row takes apart", len(roots))
	}
	if !torusStationRootsAreComplete(h, ex, roots) {
		t.Fatal("a whole station is reported as missing a root")
	}
	for i := range roots {
		short := append(append([]float64{}, roots[:i]...), roots[i+1:]...)
		if torusStationRootsAreComplete(h, ex, short) {
			t.Errorf("dropping azimuth %d of %v goes unnoticed", i, roots)
		}
		if _, ok := torusLaneFrom(h, ex, short, ex[0]); ok {
			t.Errorf("a lane was read on a station missing azimuth %d", i)
		}
	}
}

// TestAnArcIsRefusedWhereAnExtremumCouldBeBorn pins the two rows that beat every certificate before it,
// and the property that now stops them: an extremum PAIR born between two construction stations.
//
// Both are pairs the wave base refused ("the torus section's branch pair never folds") and the
// full-period branches newly admit. At a tube angle BETWEEN two construction stations — each of which
// carries two extrema — the station carries FOUR, three of them inside 0.13 rad. The anchor's lane then
// lands on a track whose discriminant is negative, arcRootFrom finds no root inside it and returns the
// lane's own extremum (the FOLD answer, at a station that has no fold), and the arc leaves the rod by
// 1.48 and 0.91 units inside a band ONE sample wide in 5761 — far narrower than the 257-sample
// post-condition reads, and narrower than any grid this reduction could afford
// (Oblikovati/Oblikovati#3515, review round 4).
//
// The row asserts the PROPERTY, not the refusal: whatever the reduction returns, no point of a curve it
// built may sit off the rod. A later tightening of stationSlopeLipschitz that let these build correctly
// would pass it, which is what it is for; a fix that only renamed the refusal would not.
func TestAnArcIsRefusedWhereAnExtremumCouldBeBorn(t *testing.T) {
	t.Parallel()
	for _, row := range []struct {
		name                      string
		major, minor              float64
		px, py, pz                float64
		dx, dy, dz, radius        float64
		wasOffTheRodBeforeTheGate float64
	}{
		{"row 311", 4.2147734347467933, 1.3463269166692093,
			-3.9589846013182339, 3.4109907944407767, 0.46177466702321185,
			0.10609082791453334, 0.90368294223232937, 0.41485163149098164, 2.394949210933234, 1.4843},
		{"row 1090", 8.6786063604031458, 0.45915108170638086,
			2.8110130560228153, -6.5566016295416834, 0.067661675240782829,
			0.79385336232074843, 0.38981651894851882, -0.46673324360591856, 2.9092491890836212, 0.9135},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			ring, err := NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), row.major, row.minor)
			if err != nil {
				t.Fatalf("NewTorus: %v", err)
			}
			rod, err := NewCylinder(
				math.P3(math.Scalar(row.px), math.Scalar(row.py), math.Scalar(row.pz)),
				math.V3(math.Scalar(row.dx), math.Scalar(row.dy), math.Scalar(row.dz)), row.radius)
			if err != nil {
				t.Fatalf("NewCylinder: %v", err)
			}
			curves, why, ok := TorusSection(ring, rod.QuadricForm(), ResolutionForSize(20))
			if !ok && !why.IsConditioning() {
				t.Fatalf("refused %q, which is not a conditioning demotion, so no defect is recorded", why)
			}
			assertNoCurveLeavesTheRod(t, rod, curves, row.wasOffTheRodBeforeTheGate)
		})
	}
}

// assertNoCurveLeavesTheRod walks every built curve at EIGHT times the construction's station step and
// requires each point to be on the rod. The excursion this catches is one sample wide at that density,
// which is why the walk is not the section's own 257-point post-condition.
func assertNoCurveLeavesTheRod(t *testing.T, rod Cylinder, curves []Curve3, wasOff float64) {
	t.Helper()
	const walk = 8 * torusStationProbes
	worst := 0.0
	for _, c := range curves {
		for i := range walk + 1 {
			worst = stdmath.Max(worst, distanceFromRodSurface(rod, c.PointAt(float64(i)/walk)))
		}
	}
	if stdmath.IsNaN(worst) || worst > torusStationRootDistanceBound { // NaN fails this, and must
		t.Errorf("a built curve leaves the rod by %.4e (it was %.4g before the gate); the bound is %.1e",
			worst, wasOff, torusStationRootDistanceBound)
	}
}

// distanceFromRodSurface is the point's distance from the infinite rod's surface, read off the cylinder
// rather than through the quadric's first-order form: at a fold the implicit residual falls off
// quadratically in the azimuth error while the POSITION error does not, so only a length sees a branch
// that has wandered off (ADR-0066's torusSectionSatisfiesItsForm makes the same argument).
func distanceFromRodSurface(rod Cylinder, p math.Point3) float64 {
	w := rod.Origin.VectorTo(p)
	axis := rod.AxisDir.AsVector()
	perp := w.Sub(axis.Scale(math.Scalar(float64(w.Dot(axis)))))
	return stdmath.Abs(float64(perp.Length()) - rod.Radius)
}

// TestTheStationValueBoundHoldsEverywhere is the property stationValueLipschitz asserts: no
// measured |∂f/∂v| anywhere on the torus may exceed it. The bound is what lets a sampled sweep say
// anything about the tube angles BETWEEN its samples, so a bound that is not a bound is worse than none.
func TestTheStationValueBoundHoldsEverywhere(t *testing.T) {
	t.Parallel()
	rng, checked := rand.New(rand.NewSource(3)), 0
	for range 200 {
		ring, rod, ok := wrapCorpusPair(rng)
		if !ok {
			continue
		}
		q := rod.QuadricForm()
		bound := q.stationValueLipschitz(ring)
		for i := range 97 {
			v := twoPi * float64(i) / 97
			for j := range 97 {
				checked++
				assertStationSlopeIsUnderTheBound(t, ring, q, twoPi*float64(j)/97, v, bound)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no station was measured; this row proves nothing")
	}
}

// TestTheStationSlopeBoundHoldsEverywhere is the property stationSlopeLipschitz asserts, for both forms
// in the class: no measured |∂(∂f/∂u)/∂v| anywhere on the chart may exceed it. It is the bound that lets
// the wrap sweep say an extremum cannot be BORN between two stations, so a bound that is not a bound
// would put the certificate's whole claim back where review round 4 found it.
func TestTheStationSlopeBoundHoldsEverywhere(t *testing.T) {
	t.Parallel()
	t.Run("quadric co-form", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewSource(7))
		for range 120 {
			ring, rod, ok := wrapCorpusPair(rng)
			if !ok {
				continue
			}
			q := rod.QuadricForm()
			assertSlopeBoundDominates(t, ring, q, q.stationSlopeLipschitz(ring))
		}
	})
	t.Run("torus co-form", func(t *testing.T) {
		t.Parallel()
		rng := rand.New(rand.NewSource(11))
		for range 120 {
			chart, co := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
			assertSlopeBoundDominates(t, chart, co, co.stationSlopeLipschitz(chart))
		}
	})
}

// assertSlopeBoundDominates differences the station polynomial's azimuth SLOPE along the tube angle over
// a (u, v) grid and requires the bound to dominate every reading.
func assertSlopeBoundDominates(t *testing.T, chart Torus, co TorusCoForm, bound float64) {
	t.Helper()
	const step = 1e-6 // tol:numeric — a central difference in the tube angle
	for i := range 41 {
		v := twoPi * float64(i) / 41
		hi := torusSecondHarmonicAt(chart, co, v+step)
		lo := torusSecondHarmonicAt(chart, co, v-step)
		for j := range 41 {
			u := twoPi * float64(j) / 41
			if d := (hi.slopeAt(u) - lo.slopeAt(u)) / (2 * step); stdmath.Abs(d) > bound {
				t.Fatalf("chart(R=%g r=%g): |d(df/du)/dv| at (u=%g, v=%g) is %g, over the bound %g",
					chart.MajorRadius, chart.MinorRadius, u, v, d, bound)
			}
		}
	}
}

// TestTheStationValueBoundHoldsForATorusCoForm is the same property for the OTHER form in the class
// (ADR-0066): a second torus's quartic restricted to the chart. The wrap certificate is a statement
// about the STATION POLYNOMIAL's degree, not about which surface produced it, so a bound that held only
// for a quadric would leave every torus × torus wrap resting on a sampled sweep
// (Oblikovati/Oblikovati#3515, review round 4).
//
// The slope is differenced on the station polynomial itself rather than on a surface's value function,
// which is the one reading both forms share: valueAt(u) at tube angle v IS F(P(u, v)).
func TestTheStationValueBoundHoldsForATorusCoForm(t *testing.T) {
	t.Parallel()
	rng, checked := rand.New(rand.NewSource(5)), 0
	for range 120 {
		chart, co := randomRingTorus(t, rng, 4), randomRingTorus(t, rng, 4)
		bound := co.stationValueLipschitz(chart)
		for i := range 41 {
			v := twoPi * float64(i) / 41
			for j := range 41 {
				checked++
				assertCoFormSlopeIsUnderTheBound(t, chart, co, twoPi*float64(j)/41, v, bound)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no station was measured; this row proves nothing")
	}
}

// assertCoFormSlopeIsUnderTheBound differences the station polynomial along the tube angle at one (u, v)
// and requires the co-form's own bound to dominate it.
func assertCoFormSlopeIsUnderTheBound(t *testing.T, chart, co Torus, u, v, bound float64) {
	t.Helper()
	const step = 1e-6 // tol:numeric — a central difference in the tube angle
	hi := torusSecondHarmonicAt(chart, co, v+step).valueAt(u)
	lo := torusSecondHarmonicAt(chart, co, v-step).valueAt(u)
	if slope := (hi - lo) / (2 * step); stdmath.Abs(slope) > bound {
		t.Fatalf("chart(R=%g r=%g) co(R=%g r=%g): |df/dv| at (u=%g, v=%g) is %g, over the bound %g",
			chart.MajorRadius, chart.MinorRadius, co.MajorRadius, co.MinorRadius, u, v, slope, bound)
	}
}

// assertStationSlopeIsUnderTheBound differences the quadric's value along the tube angle at one (u, v)
// and requires the bound to dominate it.
func assertStationSlopeIsUnderTheBound(t *testing.T, ring Torus, q Quadric, u, v, bound float64) {
	t.Helper()
	const step = 1e-6 // tol:numeric — a central difference in the tube angle
	slope := (q.ValueAt(ring.PointAt(u, v+step)) - q.ValueAt(ring.PointAt(u, v-step))) / (2 * step)
	if stdmath.Abs(slope) > bound {
		t.Fatalf("ring(R=%g r=%g): |df/dv| at (u=%g, v=%g) is %g, over the bound %g",
			ring.MajorRadius, ring.MinorRadius, u, v, slope, bound)
	}
}
