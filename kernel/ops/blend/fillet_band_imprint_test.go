// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The band∩obstacle imprint walk's acceptance, driven on the REAL corpus bodies (the STEP fixtures the
// scoreboard scores), and held against CLOSED FORMS derived from those bodies' own dimensions — never
// against a captured number.
//
// The family: a 100³ box with a 10×10 through-slot at x ∈ [90,100], z ∈ [80,90], filleted along its
// y = 0 ∧ z = 100 edge. The blend's band is a quarter cylinder of radius r about (y = r, z = 100 − r)
// along x, and the slot decides how much of it survives:
//
//	Y1 r=10 — the band's contact with y = 0 lands at z = 90, the slot's own TOP: the band never enters
//	          the slot, so the walk must find no interior cut at all and DECLINE.
//	Y3 r=20 — contact at z = 80, the slot's BOTTOM: the band is void over x ∈ [90,100] for
//	          sin v < 1/2, an L-shaped survivor.
//	Y2 r=15 — contact at z = 85, INSIDE the slot: void for sin v < 1/3, also L-shaped, and the host's
//	          own contact line is cut short at the slot wall.
//	Y4 r=25 — contact at z = 75, BELOW the slot: void for sin v ∈ [1/5, 3/5], so the slot bites a hole
//	          out of the MIDDLE of the band's end and the survivor is C-shaped.

// bandImprintTol is the acceptance's own comparison budget, relative. The walk's cut coordinates are
// closed-form (an asin, an axial projection), so they land at ~1e-15; 1e-9 separates a construction
// defect from arithmetic without being a tolerance to tune.
const bandImprintTol = 1e-9

// bandCase is one corpus body of the slot family with the closed forms its imprint must produce.
type bandCase struct {
	name string
	r    float64
}

// bandSlotFillet solves one slot-family fillet from its real STEP fixture, exactly as the corpus does.
func bandSlotFillet(t *testing.T, c bandCase) (*topo.Body, edgeFillet, filletRebuildMaps, map[uint64][]cornerPiece) {
	t.Helper()
	b := importCornerWeldFixture(t, c.name)
	picks := []filletPick{{edge: edgeNearMidpoint(t, b, math.P3(50, 0, 100)), r0: c.r, r1: c.r}}
	blends, miters, err := computeCorners(b, picks)
	if err != nil {
		t.Fatalf("%s: computeCorners: %v", c.name, err)
	}
	fils, err := computeFillets(b, picks, blends, miters, ConcaveFill(0), nil)
	if err != nil {
		t.Fatalf("%s: computeFillets: %v", c.name, err)
	}
	maps, caps := filletBuildMaps(b, fils)
	return b, fils[0], maps, caps
}

// TestBandImprintChartIsTheBlendsOwnRectangle pins the chart the walk draws in: the axial span is the
// filleted edge's own length and the sweep is the quarter turn from host A's contact to host B's, with
// the four box corners answered by the fillet's own tangent points bit for bit (bandBoxCorner) — which
// is what lets a face the walk does not rebuild keep the vertices it has today.
func TestBandImprintChartIsTheBlendsOwnRectangle(t *testing.T) {
	t.Parallel()
	for _, c := range []bandCase{{"Y2", 15}, {"Y3", 20}, {"Y4", 25}} {
		_, ef, _, _ := bandSlotFillet(t, c)
		chart, ok := newBandChart(ef)
		if !ok {
			t.Fatalf("%s: newBandChart declined", c.name)
		}
		assertBandNear(t, c.name+" uMax", chart.uMax, 100)
		assertBandNear(t, c.name+" vMax", chart.vMax, stdmath.Pi/2)
		assertBandNear(t, c.name+" radius", chart.radius, c.r)
		for _, corner := range [][3]float64{{0, 0, 0}, {100, 0, 0}, {0, chart.vMax, 0}, {100, chart.vMax, 0}} {
			p := chart.bandBoxCorner(corner[0], corner[1])
			if d := float64(p.DistanceTo(chart.bandPointAt(corner[0], corner[1]))); d > bandImprintTol*c.r {
				t.Errorf("%s: box corner (%.4g,%.4g) is %.4g off the chart's own point", c.name, corner[0], corner[1], d)
			}
		}
	}
}

// TestBandImprintCutsAreTheSlotsOwnFaces holds the CUT step against closed forms: the slot's inner wall
// x = 90 cuts the band's axial coordinate at 90, and its floor / roof planes cut the sweep exactly where
// the band crosses z = 80 / z = 90 — asin((80 − (100−r))/r) and asin((90 − (100−r))/r), kept only when
// they fall strictly inside the box.
func TestBandImprintCutsAreTheSlotsOwnFaces(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		bandCase
		wantV []float64
	}{
		{bandCase{"Y2", 15}, []float64{stdmath.Asin(1.0 / 3)}},                // z=90 only; z=80 is off the band
		{bandCase{"Y3", 20}, []float64{stdmath.Asin(0.5)}},                    // z=90; z=80 is the v=0 side itself
		{bandCase{"Y4", 25}, []float64{stdmath.Asin(0.2), stdmath.Asin(0.6)}}, // both slot planes cut
		{bandCase{"Y1", 10}, nil},                                             // the slot roof IS the contact line
	} {
		body, ef, _, _ := bandSlotFillet(t, c.bandCase)
		chart, _ := newBandChart(ef)
		us, vs, ok := bandImprintCuts(body, ef, chart, opstol.ForBody(body).Weld())
		if !ok {
			t.Fatalf("%s: bandImprintCuts refused", c.name)
		}
		assertBandCuts(t, c.name+" axial", us, bandExpectedAxial(c.wantV), 100)
		assertBandCuts(t, c.name+" sweep", vs, c.wantV, stdmath.Pi/2)
	}
}

// bandExpectedAxial is the slot wall's own cut: present exactly when the slot cuts the sweep at all.
func bandExpectedAxial(wantV []float64) []float64 {
	if len(wantV) == 0 {
		return nil
	}
	return []float64{90}
}

// assertBandCuts checks a coordinate's grid lines are the two box sides plus exactly the wanted cuts.
func assertBandCuts(t *testing.T, what string, got, want []float64, span float64) {
	t.Helper()
	if len(got) != len(want)+2 || got[0] != 0 || got[len(got)-1] != span {
		t.Fatalf("%s: got %v, want [0 %v %g]", what, got, want, span)
	}
	for i, w := range want {
		assertBandNear(t, what, got[i+1], w)
	}
}

// TestBandImprintDeclinesAnUnobstructedBand is the walk's inert half, and the reason 474 of the 475
// corpus cases cannot move: Y1's contact line lands exactly ON the slot's roof, so no cut falls
// strictly inside the box and solveBandImprint declines before it classifies anything.
func TestBandImprintDeclinesAnUnobstructedBand(t *testing.T) {
	t.Parallel()
	body, ef, maps, caps := bandSlotFillet(t, bandCase{"Y1", 10})
	if imp, ok := solveBandImprint(body, ef, opstol.ForBody(body).Weld()); ok {
		t.Fatalf("Y1: the walk claimed an obstacle it does not have: %+v", imp.runs)
	}
	if set, ok := bandImprintFacesFor(body, ef, maps, caps); ok {
		t.Errorf("Y1: the router rebuilt %d faces on an unobstructed band", len(set.replace))
	}
}

// TestBandImprintTracesTheSurvivingRegion holds the whole walk against the region's closed-form
// DEVELOPED AREA — the band area the blend must keep — measured off the traced runs themselves rather
// than off any mesh: r·(π/2)·100 minus the slot's own bite r·Δv·10.
func TestBandImprintTracesTheSurvivingRegion(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		bandCase
		runs int
		bite float64 // the swept angle the slot removes over x ∈ [90,100]
	}{
		{bandCase{"Y2", 15}, 6, stdmath.Asin(1.0 / 3)},
		{bandCase{"Y3", 20}, 6, stdmath.Asin(0.5)},
		{bandCase{"Y4", 25}, 8, stdmath.Asin(0.6) - stdmath.Asin(0.2)},
	} {
		body, ef, _, _ := bandSlotFillet(t, c.bandCase)
		imp, ok := solveBandImprint(body, ef, opstol.ForBody(body).Weld())
		if !ok {
			t.Fatalf("%s: the walk declined its own slot", c.name)
		}
		if len(imp.runs) != c.runs {
			t.Errorf("%s: traced %d runs, want %d: %+v", c.name, len(imp.runs), c.runs, imp.runs)
		}
		want := c.r*stdmath.Pi/2*100 - c.r*c.bite*10
		assertBandNear(t, c.name+" surviving band area", bandRunsDevelopedArea(imp), want)
	}
}

// bandRunsDevelopedArea is the enclosed area of the traced region IN THE CHART, which for a cylinder is
// the developed area of the band it bounds (the chart's v is an angle, so the shoelace is scaled by the
// radius). It measures the RUNS the walk produced, not a mesh and not a re-derivation of them.
func bandRunsDevelopedArea(imp bandImprint) float64 {
	sum := 0.0
	for _, r := range imp.runs {
		u0, v0, u1, v1 := r.from, r.at, r.to, r.at
		if r.constU {
			u0, v0, u1, v1 = r.at, r.from, r.at, r.to
		}
		sum += u0*v1 - u1*v0
	}
	return stdmath.Abs(sum) / 2 * imp.chart.radius
}

// TestBandImprintChainLandsOnEveryFacesOwnRing is the seam contract with chainRetrimLoop: every run the
// walk hands over must already land on the ring of the face it lies on, so the splice has two landings
// to cut at. A run that does not is the "guessed landing" the primitive refuses.
func TestBandImprintChainLandsOnEveryFacesOwnRing(t *testing.T) {
	t.Parallel()
	for _, c := range []bandCase{{"Y2", 15}, {"Y3", 20}, {"Y4", 25}} {
		body, ef, _, _ := bandSlotFillet(t, c)
		imp, ok := solveBandImprint(body, ef, opstol.ForBody(body).Weld())
		if !ok {
			t.Fatalf("%s: the walk declined", c.name)
		}
		segs := bandRunSegs(imp, ef)
		faces, ok := bandRunFaces(body, imp, segs)
		if !ok {
			t.Fatalf("%s: a run did not resolve to exactly one body face", c.name)
		}
		tol := opstol.ForBody(body).Weld()
		for i, f := range faces {
			if !chainLandsOnRing(originalHostSegs(f), []endSeg{segs[i]}, tol) {
				t.Errorf("%s: run %d (%v→%v) does not land on face %d's ring", c.name, i, segs[i].from, segs[i].to, f.ID())
			}
		}
	}
}

// ★ TestBandImprintArrangementIsFalsifiedByAMissingCut is the walk's GUARD, and the invariant it owns:
//
//	a cell of the arrangement is uniformly inside the solid or uniformly outside it.
//
// The arrangement is only a HYPOTHESIS about where the obstacles cut the band; it is right only if no
// cut was missed. Handing bandClassifyCells a grid with the slot wall's own cut REMOVED puts a cell
// astride the slot, and the classifier must refuse it rather than pick whichever side its first probe
// landed on. Without this the walk would ship a band trimmed on a straight guess.
func TestBandImprintArrangementIsFalsifiedByAMissingCut(t *testing.T) {
	t.Parallel()
	body, ef, _, _ := bandSlotFillet(t, bandCase{"Y2", 15})
	chart, _ := newBandChart(ef)
	us, vs, _ := bandImprintCuts(body, ef, chart, opstol.ForBody(body).Weld())
	if _, ok := bandClassifyCells(body, chart, us, vs); !ok {
		t.Fatal("Y2: the complete arrangement was rejected")
	}
	if _, ok := bandClassifyCells(body, chart, []float64{us[0], us[len(us)-1]}, vs); ok {
		t.Error("Y2: an arrangement MISSING the slot wall's cut was accepted — a cell straddling the slot classified uniformly")
	}
}

// TestBandImprintRebuildsEveryFaceItsRunsCross is the ATOMICITY guard. Y2's slot family shares edges
// across the host plane, the slot's walls and the wall above it: re-trimming any subset opens the shell
// (routing Y2's host plane alone takes it 8475 → 8450 while the band still claims x ∈ [0,100] at
// z = 85). So the router must account for EVERY run — either rebuilding that face or proving the
// existing transform already agrees with the walk — and never for only some of them.
func TestBandImprintRebuildsEveryFaceItsRunsCross(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		bandCase
		rebuilt int // faces whose loop the walk DISAGREES with, and therefore replaces
	}{{bandCase{"Y2", 15}, 4}, {bandCase{"Y3", 20}, 4}, {bandCase{"Y4", 25}, 6}} {
		body, ef, maps, caps := bandSlotFillet(t, c.bandCase)
		imp, _ := solveBandImprint(body, ef, opstol.ForBody(body).Weld())
		set, ok := bandImprintFacesFor(body, ef, maps, caps)
		if !ok {
			t.Fatalf("%s: the router declined a solved imprint", c.name)
		}
		if len(set.replace) != c.rebuilt {
			t.Errorf("%s: rebuilt %d faces, want %d (of %d runs)", c.name, len(set.replace), c.rebuilt, len(imp.runs))
		}
		assertBandRunsAccountedFor(t, c.name, body, imp, set)
	}
}

// assertBandRunsAccountedFor checks every traced run's face is either replaced by the walk or left to
// the existing transform — never dropped — which is what makes the rebuild atomic.
func assertBandRunsAccountedFor(t *testing.T, name string, body *topo.Body, imp bandImprint, set bandImprintSet) {
	t.Helper()
	for i, r := range imp.runs {
		f, ok := bandFaceUnderRun(body, bandRunMidpoint(imp.chart, r))
		if !ok {
			t.Fatalf("%s: run %d lies on no single face", name, i)
		}
		if _, replaced := set.replace[f.ID()]; !replaced && !r.bandFullSide(imp.chart) {
			t.Errorf("%s: run %d crosses face %d, which the walk neither rebuilt nor left as a full box side", name, i, f.ID())
		}
	}
}

// assertBandNear fails when got differs from the closed form by more than bandImprintTol relative.
func assertBandNear(t *testing.T, what string, got, want float64) {
	t.Helper()
	if rel := stdmath.Abs(got-want) / stdmath.Max(1, stdmath.Abs(want)); rel > bandImprintTol {
		t.Errorf("%s: got %.12f, closed form %.12f (rel %.3g)", what, got, want, rel)
	}
}
