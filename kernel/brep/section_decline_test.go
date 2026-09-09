// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The three refusals of Oblikovati/Oblikovati#3525. A torus pair, an ill-conditioned lane and a section
// that does not close all reached the user as one generic "no exact analytic path claims this
// configuration": the first was never recorded at all, the second only through two of the four
// pairings, and the third came back as DeclineNone — a refusal that names nothing. Each must now say
// WHICH gate refused, on the recorder brep.BooleanDiag was handed.

// ringAndGrazingRod is the (torus, cylinder) fixture the lane cases are read on: a ring inside a rod
// whose wall is TANGENT to the ring's outer equator. Their surfaces touch at two points and cross
// nowhere, so the section's azimuths meet without separating and the closed form cannot name the set it
// would build — an ill-conditioned section by construction, at any (major, minor).
//
// It used to be a rod merely THICKER than the ring's tube, which is not ill-conditioned at all: that is
// four full-period branches, and they are built now (Oblikovati/Oblikovati#3515). The gate has to be
// driven by a section the closed form genuinely cannot name, or it stops testing the gate.
func ringAndGrazingRod(t *testing.T, major, minor float64) (*topo.Body, *topo.Body) {
	t.Helper()
	ring, err := SolidTorus(math.P3(0, 0, 0), math.V3(1, 0, 0), major, minor, "ring")
	if err != nil {
		t.Fatalf("SolidTorus(%g, %g): %v", major, minor, err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, -10), math.V3(0, 0, 1), major+minor, 20)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	return ring, rod
}

// TestAnOrdinaryRefusalIsRecordedAsInfo: a refusal no closed form CLAIMS is an Info under
// CodeSectionUnclaimedPair, naming both faces; a CONDITIONING demotion of the same pair is a Defect
// under a different code. Two refusals that read alike is what #3525 was about, and this is the one
// place the two are told apart.
//
// It reads recordSectionDecline directly, and that is not a shortcut — it is the only level left where
// the ordinary route can be driven. The fixture used to be a TORUS PAIR, the last surface pair in the
// kernel's primitive vocabulary that no closed form claimed; ADR-0066 (#3514) claims it, and a sweep of
// {torus, sphere, block, cylinder, cone} in all three operations against each other now reaches
// CodeSectionUnclaimedPair from nothing at all. The behaviour under test is the ROUTING of a reason to a
// severity, so it is tested where that decision is made rather than through a fixture chosen to provoke
// it — and it keeps holding when the vocabulary gains a pair the intersector does not claim.
func TestAnOrdinaryRefusalIsRecordedAsInfo(t *testing.T) {
	t.Parallel()
	ringA, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 4, 1)
	ringB, _ := geom.NewTorus(math.P3(4, 0, 0), math.V3(1, 0, 0), 4, 1)
	a, b := curvedFace{surface: ringA}, curvedFace{surface: ringB}
	rec := &diag.Recorder{}
	recordSectionDecline(rec, refusal(geom.DeclineNoClosedForm), a, b)
	d := onlyDiagWithCode(t, rec, CodeSectionUnclaimedPair)
	for _, want := range []string{"geom.Torus ∩ geom.Torus", "no closed form claims this surface pair"} {
		if !strings.Contains(d.Detail, want) {
			t.Errorf("the ordinary refusal does not name %q: %s", want, d.Detail)
		}
	}
	if d.Severity != diag.Info {
		t.Errorf("an unclaimed pair is %v; nothing degraded here, the caller's own decline is the degradation", d.Severity)
	}
	demoted := &diag.Recorder{}
	recordSectionDecline(demoted, refusal(geom.DeclineTorusLaneTracks), a, b)
	if got := onlyDiagWithCode(t, demoted, CodeSectionConditioningDemotion); got.Severity != diag.Defect {
		t.Errorf("a conditioning demotion is %v, want a Defect — it is where the exact pipeline gave up ground", got.Severity)
	}
}

// TestAnIllConditionedLaneDeclinesByName: the torus∩cylinder closed form APPLIES to a ring grazing the
// inside of a rod and still cannot use its answer at these numbers. That is a degradation — a Defect,
// and a different one from the torus pair above, which is the whole point of naming the gate.
func TestAnIllConditionedLaneDeclinesByName(t *testing.T) {
	t.Parallel()
	ring, rod := ringAndGrazingRod(t, 6, 1.5)
	rec := &diag.Recorder{}
	if _, err := BooleanDiag(Difference, rod, ring, rec); err == nil {
		t.Fatal("the grazing ring-in-rod cut built; the fixture no longer exercises the lane decline")
	}
	d := onlyDiagWithCode(t, rec, CodeSectionConditioningDemotion)
	for _, want := range []string{"geom.Cylinder cylinder:f#2 ∩ geom.Torus ring:face#0", "the torus section's"} {
		if !strings.Contains(d.Detail, want) {
			t.Errorf("the lane decline does not name %q: %s", want, d.Detail)
		}
	}
	if d.Severity != diag.Defect {
		t.Errorf("a conditioning demotion is %v; the exact path was available and given up", d.Severity)
	}
}

// TestTheThreeRefusalsReadDifferently is the statement of #3525 itself: the user must be able to tell
// the three apart. Before, all three arrived as the same generic message.
func TestTheThreeRefusalsReadDifferently(t *testing.T) {
	t.Parallel()
	ball, err := geom.NewSphere(math.P3(0, 0, 0), 5)
	if err != nil {
		t.Fatalf("NewSphere: %v", err)
	}
	rod, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	seat := curvedFace{surface: ball, lineage: topo.NewLineage(topo.Tok("ball", "face", 0))}
	bore := curvedFace{surface: rod, lineage: topo.NewLineage(topo.Tok("rod", "face", 1))}
	rec := &diag.Recorder{}
	recordSectionDecline(rec, refusal(geom.DeclineNoClosedForm), seat, bore)
	recordSectionDecline(rec, refusal(geom.DeclineTorusLaneTracks), seat, bore)
	recordSectionDecline(rec, refusalf(geom.DeclineOpenSection, "endpoint gap %g > sew %g", 0.25, 0.001), seat, bore)
	seen := map[string]bool{}
	for _, d := range rec.Records() {
		if seen[d.Detail] {
			t.Errorf("two of the three refusals read identically: %s", d.Detail)
		}
		seen[d.Detail] = true
		// The faces, not just their kinds: "failure is local — naming the faulty entity".
		if !strings.Contains(d.Detail, "geom.Cylinder rod:face#1 ∩ geom.Sphere ball:face#0") {
			t.Errorf("a refusal does not name the two FACES that refused: %s", d.Detail)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("recorded %d refusals, want 3: %v", len(seen), rec.Records())
	}
	if !strings.Contains(strings.Join(detailsOf(rec), " "), "endpoint gap 0.25 > sew 0.001") {
		t.Error("the open-section refusal dropped the value it measured")
	}
}

// TestAnOpenSectionIsRefusedByName: the closure gate is the one that returned DeclineNone with ok=false
// (#3525). It must name itself AND report the gap it measured — the exception-message rule.
func TestAnOpenSectionIsRefusedByName(t *testing.T) {
	t.Parallel()
	res := geom.ResolutionForSize(10)
	open, err := geom.NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("NewLine: %v", err)
	}
	closed, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("NewCircle: %v", err)
	}
	if gap, why := declineOpenSection([]geom.Curve3{closed}, res); why != geom.DeclineNone {
		t.Errorf("a circle is refused as open (%v, gap %g); it closes on itself", why, gap)
	}
	// A BOUNDED open arc: the gap is a real number the message can name.
	arc := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(4, 0, 0))
	gap, why := declineOpenSection([]geom.Curve3{closed, arc}, res)
	if why != geom.DeclineOpenSection {
		t.Fatalf("a bounded open section is refused as %v, want DeclineOpenSection", why)
	}
	if stdmath.Abs(gap-4) > 1e-9 {
		t.Errorf("the open-section gate measured a gap of %g, want the segment's own length 4", gap)
	}
	// An UNBOUNDED curve: its endpoint distance is NaN, and `NaN > sew` is false — the comparison the
	// gate used to make let the widest refusal through the narrowest gate.
	if _, why := declineOpenSection([]geom.Curve3{open}, res); why != geom.DeclineOpenSection {
		t.Errorf("an unbounded section is refused as %v, want DeclineOpenSection", why)
	}
}

// TestEveryImprintRefusalIsNamed sweeps the pairings over a corpus of pairs and asserts the invariant
// the AST guard cannot see: a refusal decided at RUNTIME never carries DeclineNone. The AST reads
// `return nil, why, false` and cannot know what `why` holds there — which is exactly how #3525's
// non-closing crossing shipped.
func TestEveryImprintRefusalIsNamed(t *testing.T) {
	t.Parallel()
	base, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder base: %v", err)
	}
	pb := partitionFaces(base)
	if len(pb.wall) != 1 {
		t.Fatalf("fixture: the base has %d walls, want 1", len(pb.wall))
	}
	refusals := 0
	raised := map[string]int{}
	check := func(what string, why sectionRefusal, ok bool) {
		if ok {
			return
		}
		refusals++
		raised[why.why.String()]++
		if why.why == geom.DeclineNone {
			t.Errorf("%s refused with no reason (DeclineNone)", what)
		}
		if strings.Contains(why.String(), "SectionDecline(?)") {
			t.Errorf("%s refused with an unnamed reason %q", what, why)
		}
	}
	for _, pair := range imprintSweepSurfaces(t) {
		_, why, ok := closedSurfacePairImprint(curvedFace{surface: pair.a}, curvedFace{surface: pair.b})
		check("closedSurfacePairImprint "+pair.name, why, ok)
		_, why, ok = closedSurfaceWallImprint(curvedFace{surface: pair.a}, pb.wall[0])
		check("closedSurfaceWallImprint "+pair.name, why, ok)
	}
	for _, tool := range imprintSweepWalls(t) {
		_, why, ok := wallWallImprint(pb.wall[0], tool)
		check("wallWallImprint", why, ok)
	}
	if refusals == 0 {
		t.Fatal("the sweep produced no refusal at all — it is passing vacuously")
	}
	// The names this corpus actually RAISED, against the ones geom declares. geom's
	// TestEveryDeclineNameIsReachable proves only a syntactic mention; this is the live half, and the
	// difference is which reasons a user can meet today (review round 1, finding 11).
	t.Logf("swept %d named refusals across %d reasons: %v", refusals, len(raised), raised)
}

// surfacePair is one entry of the imprint sweep's corpus.
type surfacePair struct {
	name string
	a, b geom.Surface
}

// imprintSweepSurfaces is the closed-surface corpus: spheres and tori at radii that straddle the base
// cylinder's own radius, plus the torus pair that has no closed form at all.
func imprintSweepSurfaces(t *testing.T) []surfacePair {
	t.Helper()
	var out []surfacePair
	for _, rad := range []float64{0.5, 2, 3, 6, 12} {
		for _, dz := range []float64{-9, 0, 9} {
			// The two surfaces of a pair are DISTINCT objects at distinct places. Handing the same
			// object twice short-circuits on SurfacesCoincide, so such an entry can never refuse and
			// pads the count without testing anything (review round 1, finding 14).
			ball := mustSphere(t, math.P3(0, 0, math.Scalar(dz)), rad)
			other := mustSphere(t, math.P3(math.Scalar(rad/2), 0, math.Scalar(dz+1)), rad*0.75)
			out = append(out, surfacePair{"sphere/sphere", ball, other})
			ring, err := geom.NewTorus(math.P3(0, 0, math.Scalar(dz)), math.V3(1, 0, 0), math.Scalar(rad), 0.4)
			if err != nil {
				continue // a minor radius the major cannot carry: not a ring fixture, the spheres stand
			}
			linked, err := geom.NewTorus(math.P3(math.Scalar(rad), 0, math.Scalar(dz)), math.V3(0, 0, 1), math.Scalar(rad), 0.4)
			if err != nil {
				continue
			}
			out = append(out, surfacePair{"sphere/ring", ball, ring}, surfacePair{"ring/ring", ring, linked})
		}
	}
	return out
}

// mustSphere builds a sphere for a fixture, failing the test rather than returning a zero value.
func mustSphere(t *testing.T, centre math.Point3, radius float64) geom.Sphere {
	t.Helper()
	s, err := geom.NewSphere(centre, math.Scalar(radius))
	if err != nil {
		t.Fatalf("NewSphere(%v, %g): %v", centre, radius, err)
	}
	return s
}

// imprintSweepWalls is the ruled-wall corpus: rods at angles and offsets that cross the base's wall,
// graze its rim and clear it entirely.
func imprintSweepWalls(t *testing.T) []curvedFace {
	t.Helper()
	var out []curvedFace
	for _, ang := range []float64{0, 25, 55, 89} {
		for _, dx := range []float64{-6, -3, 0, 2} {
			th := ang * stdmath.Pi / 180
			tool, err := SolidCylinder(math.P3(math.Scalar(dx), 0, 5),
				math.V3(math.Scalar(stdmath.Cos(th)), 0, math.Scalar(stdmath.Sin(th))), 1, 12)
			if err != nil {
				t.Fatalf("SolidCylinder(%g, %g): %v", ang, dx, err)
			}
			p := partitionFaces(tool)
			if len(p.wall) > 0 {
				out = append(out, p.wall[0])
			}
		}
	}
	return out
}

// onlyDiagWithCode returns the recorder's first diagnostic carrying code, failing when there is none.
func onlyDiagWithCode(t *testing.T, rec *diag.Recorder, code diag.Code) diag.Diagnostic {
	t.Helper()
	for _, d := range rec.Records() {
		if d.Code == code {
			return d
		}
	}
	t.Fatalf("no %s diagnostic; the recorder holds %v", code, rec.Records())
	return diag.Diagnostic{}
}

// detailsOf is the recorder's messages, for a whole-report assertion.
func detailsOf(rec *diag.Recorder) []string {
	var out []string
	for _, d := range rec.Records() {
		out = append(out, d.Detail)
	}
	return out
}
