// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	"fmt"
	stdmath "math"
	"strings"
	"testing"
	"time"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"

	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The sweep behind the resolution floor (finding 4 of the stage-6 review: the floor was asserted, not
// measured). It drives the RD- corpus family — an axial drill through the RING — over the tool
// thickness, and classifies each point against an INDEPENDENT oracle rather than against the kernel.
//
// The regimes it establishes, and which one the size classification may claim, are recorded in
// boolean_resolution_decline.go and ADR-0061. In short: only the SILENT band is a resolution limit.

// TestTheAxialDrillSweepPinsTheResolutionFloor is the measurement the floor is set from, kept as a
// regression so the boundary cannot drift unnoticed. It asserts the SHAPE of the outcome curve, not a
// single point: every tool at or below the floor is refused, every tool at the exact plateau is exact,
// and — the property the whole stage exists for — NO point anywhere in the sweep is silent.
func TestTheAxialDrillSweepPinsTheResolutionFloor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		bore float64
		want drillOutcome
	}{
		// BELOW the floor: the size classification refuses these before any geometry is built. The floor
		// at this pair is thickness < Weld = 2.00499e-8, i.e. RADIUS < 1.0025e-8 — so 1e-8 is below it,
		// not above. (The silent band the floor exists for ends much lower, at ~0.0998 x Weld; the floor
		// covers it with margin. An earlier version of this comment filed 1e-8 as "above the floor",
		// which mis-stated the classification's reach by an order of magnitude.)
		{1e-11, drillRefusedTooThin}, {1e-10, drillRefusedTooThin}, {1e-9, drillRefusedTooThin},
		{1e-8, drillRefusedTooThin},
		// ABOVE the floor: the size classification does not answer, and the outcome is the pipeline's
		// own. Below ~4e-4 the general per-face boolean declines the pair BY NAME
		// (boolean.no-exact-curved-path) and no geometry is built. "Below" and not "up to": the
		// refusals do not form an interval — 3.98e-4 and 7.94e-4 refuse while both of their
		// neighbours build — so these rows pin two radii, not a boundary.
		{1e-6, drillRefusedNoPath}, {1e-4, drillRefusedNoPath},
		// From ~6.3e-4 the pipeline BUILDS the exact section — valid, one torus, one cylinder, four
		// loops, both intersection edges on both surfaces to 4.4e-16 (torus) and 4.4e-12 (cylinder).
		// What separates this row from the plateau is not the body but the number: the section curve's
		// tangent is a fixed-step central difference of an azimuth root whose own error grows as the
		// section's azimuth swing shrinks, so the Green face integral reads this bore's wall 43% light
		// and the removed volume misses the oracle (Oblikovati/Oblikovati#3516; the measured law and
		// the two plateau sweeps that refute a tuned step are in that issue).
		// Measured on this pair: the removed volume misses the oracle by 42.9% at 1e-3 and 9.4% at
		// 2e-3. The miss is NOISE and not a law — over 20 radii per decade it runs from -105% (a cut
		// measuring LARGER than its target) to +252% with no monotone trend, and both extremes are
		// recorded by CodeBooleanMovedVolumeOutOfToolBracket. Nor is the plateau's edge a point: rows
		// at 2.24e-3, 3.16e-3 and 4.47e-3 already measure exact while 2.51e-3, 3.98e-3 between them do
		// not. These two rows are pinned deep inside the coarse side, where the outcome is stable.
		{1e-3, drillCoarse}, {0.002, drillCoarse},
		// The exact plateau, whose lower edge is a DECADE below where ADR-0061 G8 measured it. It moved
		// because the analytic integrator stopped declining the bored torus face, not because the
		// boolean changed: the face's interior probe was a fixed 33x33 grid over the bounding box of
		// ALL its loops, and the two bore mouths sit half a tube-turn apart, so no grid point landed
		// inside either one below bore ~0.0713. The tessellated fallback then measured the ring 2.839
		// light — 300000x a 1e-3 bore's own material — and the Requicha bracket rejected a correct body.
		{0.02, drillExact}, {0.0631, drillExact},
		{0.1, drillExact}, {0.2, drillExact}, {0.4, drillExact}, {0.631, drillExact}, {0.8, drillExact},
	} {
		if got := sweepDrill(t, tc.bore); got != tc.want {
			t.Errorf("bore %g: outcome %q, want %q", tc.bore, got, tc.want)
		}
	}
}

// TestNoDrillRadiusIsAnsweredSilently is the stage's actual invariant, swept at five points per decade
// across the five decades that BRACKET the silent band (the measurement put its top edge at
// 0.0998 x Weld, i.e. radius ~1e-9, and these decades run from 1e-12 to 1e-8 either side of it). A
// "silent" outcome — the ring handed back with no bore in it, err=nil, nothing recorded — is the
// defect; every other outcome is honest.
//
// It stops at 1e-8 because that is where the size classification stops answering (the floor is
// radius < 1.0025e-8 at this pair) and each point above it costs a full curved boolean. The upper
// decades are covered instead by TestTheAxialDrillSweepPinsTheResolutionFloor's named points, which
// run in ~1.8 s, and the one radius that used to hang the whole suite has its own row
// (TestTheNonConvergentDrillTerminatesAndIsNamed). Every point here runs under a deadline, so a
// pipeline that stops terminating fails as a test instead of wedging the run.
func TestNoDrillRadiusIsAnsweredSilently(t *testing.T) {
	t.Parallel()
	for e := -12; e <= -8; e++ {
		for k := range 5 {
			bore := stdmath.Pow(10, float64(e)+float64(k)/5.0)
			if got := sweepDrill(t, bore); got == drillSilent {
				t.Errorf("bore %g: the boolean returned the ring untouched with no error and no diagnostic", bore)
			}
		}
	}
}

// The RING volume the rows below measure against: the corpus torus integrated over its own analytic
// B-rep, which is exact for a torus (2·pi^2·R·r^2 = 222.06609902451055...).
const ringAnalyticVolume = 222.06609902451055

// TestASmallBoreBuildsTheExactSectionAndIsMeasuredAnalytically is the corpus row for #3516, and it
// asserts the two things that were wrong there — neither of which was the section.
//
// The row it replaces (TestASmallBoreIsRefusedNotShippedWrong) recorded that a 1e-3 bore "builds a
// valid closed solid that removes 2.84 of material where the true bore is 9.4e-6 — a factor of
// 300000". That premise was measured again and refuted. 2.839 is the DEFICIT of the ring's own
// tessellation at DefaultQuality, not material the boolean removed: the analytic integrator declined
// the bored torus face, BodyGeometryProperties fell back to the mesh for the result while the intact
// ring still integrated analytically, and the bracket then compared two different measurements of the
// same torus. The body was right the whole time, and the Requicha bracket — the "only thing standing
// between it and the model" — was rejecting it for an artefact.
//
// So this row pins the measurement, face by face: the section builds, BOTH of its faces carry an
// interior point the membership certificate can classify (the torus face had none at ANY bore before
// #3516), the body integrates ANALYTICALLY, and its volume is the ring's minus the bore's to within a
// bound far tighter than the tessellated answer's 1.28e-2. It does NOT assert the removed volume
// against the oracle: at this radius that misses by 43%, which is the drillCoarse row above and a
// different defect.
func TestASmallBoreBuildsTheExactSectionAndIsMeasuredAnalytically(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-3)
	body, err := Boolean(Cut, ring, drill)
	if err != nil || body == nil {
		t.Fatalf("a 1e-3 bore in a 5/1.5 ring is an ordinary feature and must build: %v", err)
	}
	if got := classifyBoredRing(ring, body, 1e-3); got != drillCoarse && got != drillExact {
		t.Fatalf("the section must be the exact one (torus + cylinder, 4 loops, valid); got %q", got)
	}
	assertEveryFaceIsProbeable(t, body)
	assertBoredRingRemovesMaterialAnalytically(t, body, drill, 1e-3)
}

// assertEveryFaceIsProbeable is the #3516 regression on the certificate's blind spot: a face with no
// interior point is one certifyBooleanFaces SKIPS, so a result can be "certified" on a strict subset
// of its own faces. On this pair that subset was the bore wall alone — 0.0189 of 296.11 of area.
func assertEveryFaceIsProbeable(t *testing.T, body *topo.Body) {
	t.Helper()
	for _, f := range body.Faces() {
		if _, ok := query.FaceInteriorPoint(f); !ok {
			t.Errorf("face %q (%T, %d loops) has no interior point, so the membership certificate skips it",
				f.ReferenceKey(), f.Geometry(), len(f.Loops()))
		}
	}
}

// assertBoredRingRemovesMaterialAnalytically is the #3516 regression on the integrator: the bored
// torus face must be integrable, so the body's volume comes from its analytic B-rep and not from a
// mesh, AND the material that volume accounts for must be the bore's.
//
// It asserts the REMOVED volume, not the body's. The first cut of this row bounded the body's own
// volume at 1e-6 relative of ring − oracle, which an UNDRILLED ring passes: at 222.06609902451 it
// sits 4.24e-8 from the target, 24x inside the bound, so the row could not fail for the thing it
// claimed to measure. The removal cannot be faked that way — an undrilled ring removes exactly zero
// and fails the first bound below.
//
// The two bounds are different in kind. `0 < removed <= V(tool)` is Requicha's own rule taken at the
// tool's scale; it is an identity, so it carries no tolerance. The comparison against the oracle is
// an accuracy statement and is deliberately loose: at this radius the removal is 42.9% low
// (Oblikovati/Oblikovati#3538), so a factor-of-two bound is what separates "the right feature,
// measured badly" from "no feature" or "twice the feature" without pinning a number that is noise.
func assertBoredRingRemovesMaterialAnalytically(t *testing.T, body, drill *topo.Body, bore float64) {
	t.Helper()
	props, ok := query.AnalyticGeometryProperties(body)
	if !ok {
		t.Fatalf("the bored ring must integrate over its analytic B-rep; the mesh fallback measures "+
			"this torus %g light, which is %gx the bore's own material",
			ringAnalyticVolume-219.2269659, (ringAnalyticVolume-219.2269659)/boreRemovalOracle(5, 1.5, 5, bore, boreQuadratureCells))
	}
	tool, ok := query.AnalyticGeometryProperties(drill)
	if !ok {
		t.Fatalf("the drill must integrate analytically for its volume to bound the removal")
	}
	removed := ringAnalyticVolume - props.Volume
	if removed <= 0 || removed > tool.Volume {
		t.Fatalf("a cut removed %g with a tool holding %g: outside (0, V(tool)], which no cut can be",
			removed, tool.Volume)
	}
	oracle := boreRemovalOracle(5, 1.5, 5, bore, boreQuadratureCells)
	if rel := stdmath.Abs(removed-oracle) / oracle; rel > 1 { // tol:calibrated — measured 0.429
		t.Errorf("removed %.12g against an oracle of %.12g (%.3g relative)", removed, oracle, rel)
	}
}

// TestTheMovedVolumeBracketRefusesOnlyImpossibleMoves is the gate's own truth table, over synthetic
// volumes so every row is exact arithmetic and none of it can drift with the geometry pipeline.
//
// The bracket is one rule for three operations — what a Cut removed, a Join added, an Intersect kept,
// each bounded by the tool that moved it — so the table walks all three at both bounds, at the slack
// on either side of each, and at the guard that keeps a MESH volume out of an analytic comparison.
func TestTheMovedVolumeBracketRefusesOnlyImpossibleMoves(t *testing.T) {
	t.Parallel()
	const tool = 4.0
	for _, tc := range []struct {
		name         string
		op           PartFeatureOperation
		tv, wv, bv   float64
		exact        bool
		wantRecorded bool
	}{
		{"cut removes half its tool", Cut, 100, tool, 98, true, false},
		{"cut removes exactly its tool", Cut, 100, tool, 96, true, false},
		{"cut removes nothing", Cut, 100, tool, 100, true, false},
		{"cut removes an ulp less than nothing", Cut, 100, tool, 100 + tool*movedVolumeSlack/2, true, false},
		{"cut removes measurably less than nothing", Cut, 100, tool, 100 + tool*movedVolumeSlack*10, true, true},
		{"cut removes measurably more than its tool", Cut, 100, tool, 96 - tool*movedVolumeSlack*10, true, true},
		{"join adds half its tool", Join, 100, tool, 102, true, false},
		{"join adds more than its tool", Join, 100, tool, 105, true, true},
		{"join loses material", Join, 100, tool, 99, true, true},
		{"intersect keeps part of the tool", Intersect, 100, tool, 3, true, false},
		{"intersect keeps more than the tool", Intersect, 100, tool, 5, true, true},
		{"an operation with no membership rule", NewBody, 100, tool, 500, true, false},
		// The guard #3516's own root cause demands: one operand measured by mesh and the other
		// analytically is the artefact this issue exists to correct, and a ~1e-2 mesh deficit against
		// a 1e-9 slack would read as a contradiction. The gate does not run — and SAYS so, which is
		// the other half of the row: a gate that silently does not run is the blind spot this issue
		// is about, and it appeared once already inside this very guard.
		{"an impossible move measured by mesh", Cut, 100, tool, 105, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &diag.Recorder{}
			recordMovedVolumeOutOfToolBracket(tc.op, volumeTriple{tc.tv, tc.wv, tc.bv}, volumeSource{true, true, tc.exact}, rec)
			if got := rec.Has(CodeBooleanMovedVolumeOutOfToolBracket); got != tc.wantRecorded {
				t.Errorf("recorded = %v, want %v; records %v", got, tc.wantRecorded, rec.Records())
			}
			// A skipped gate has to say so. Measured over kernel/ops/boolean, 15 of 437 calls skip
			// this way; before this assertion they skipped in silence.
			if got := rec.Has(CodeBooleanVolumeNotBracketed); got != !tc.exact {
				t.Errorf("reported the skipped bracket = %v, want %v; records %v", got, !tc.exact, rec.Records())
			}
		})
	}
}

// TestAToolTooSmallToAccountForTheRemovalIsRecorded proves the gate on a REAL result, by construction
// rather than by finding a radius that misbehaves. It measures the exact 0.8 bore through the RING —
// 5.809 of material removed — against a stub tool of the same radius but one unit long, which holds
// only 2.011. No cut can move 2.9x its own tool.
//
// The row it replaces pinned two radii where the pipeline's own measurement happens to fall outside
// the bracket, and a 1e-5 RELATIVE change in either bore flips the verdict (measured: +7.45e-6,
// -3.87e-7, +4.83e-6 across a 2e-5 window). Output has to be byte-identical across PLATFORMS, and
// this repo already carries an arm64/amd64 contraction divergence (ADR-0064, #3528); a verdict that
// turns on the last digits of a cancelling difference does not survive one.
func TestAToolTooSmallToAccountForTheRemovalIsRecorded(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 0.8)
	body, err := Boolean(Cut, ring, drill)
	if err != nil {
		t.Fatalf("the RD- row must build: %v", err)
	}
	stub, err := brep.SolidCylinder(math.P3(5, 0, -0.5), math.V3(0, 0, 1), 0.8, 1)
	if err != nil {
		t.Fatalf("stub: %v", err)
	}
	honest := &diag.Recorder{}
	vols, src := boolVolumes(ring, drill, body)
	recordMovedVolumeOutOfToolBracket(Cut, vols, src, honest)
	if honest.Has(CodeBooleanMovedVolumeOutOfToolBracket) {
		t.Fatalf("the genuine pair is out of bracket, so the probe below proves nothing: %v", honest.Records())
	}
	assertThePublicEntryRunsTheBracketSilently(t, ring, drill)
	rec := &diag.Recorder{}
	stubVols, stubSrc := boolVolumes(ring, stub, body)
	recordMovedVolumeOutOfToolBracket(Cut, volumeTriple{vols.target, stubVols.tool, vols.body}, stubSrc, rec)
	if !rec.Has(CodeBooleanMovedVolumeOutOfToolBracket) {
		t.Errorf("a cut removing %g with a tool holding %g was not recorded", vols.target-vols.body, stubVols.tool)
	}
}

// assertThePublicEntryRunsTheBracketSilently drives the honest pair through BooleanWithDiagnostics and
// requires neither of the bracket's two codes.
//
// It is here because of what the rows around it CANNOT see. Both call the predicate directly, so
// deleting `recordMovedVolumeOutOfToolBracket(...)` from curvedResultRejected leaves both green —
// measured. This half shows the public path reaching the gate and finding nothing; it does not, on
// its own, catch a deleted call site, and only the band row below shows the gate firing through that
// path. Saying which is which is the point: an absence assertion is not a wiring proof.
func assertThePublicEntryRunsTheBracketSilently(t *testing.T, ring, drill *topo.Body) {
	t.Helper()
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, ring, drill, rec); err != nil {
		t.Fatalf("the RD- row must build through the public entry: %v", err)
	}
	for _, code := range []diag.Code{CodeBooleanMovedVolumeOutOfToolBracket, CodeBooleanVolumeNotBracketed} {
		if rec.Has(code) {
			t.Errorf("the exact 0.8 bore recorded %q through the public entry: %v", code, rec.Records())
		}
	}
}

// TestTheBandRecordsItsImpossibleRadiiAndThePlateauRecordsNone is the band's own half of the gate,
// written as an EXISTENCE statement so an ulp of drift in the geometry cannot move which radius
// misbehaves and fail the row. What it asserts is the shape the sweep measured: somewhere in the
// coarse band a built result reports a move its tool cannot account for, and nowhere on the exact
// plateau does one.
func TestTheBandRecordsItsImpossibleRadiiAndThePlateauRecordsNone(t *testing.T) {
	t.Parallel()
	fired := 0
	for k := range 20 {
		bore := stdmath.Pow(10, -3.5+float64(k)/20.0) // 3.16e-4 .. 1.78e-3, the coarse band
		if recordsAnImpossibleMove(t, bore) {
			fired++
		}
	}
	if fired == 0 {
		t.Errorf("no radius in the coarse band recorded %q; the band's own measurements were what "+
			"this gate was added for", CodeBooleanMovedVolumeOutOfToolBracket)
	}
	for _, bore := range []float64{0.02, 0.0631, 0.1, 0.4, 0.8} {
		if recordsAnImpossibleMove(t, bore) {
			t.Errorf("bore %g is on the exact plateau and recorded %q", bore, CodeBooleanMovedVolumeOutOfToolBracket)
		}
	}
}

// recordsAnImpossibleMove cuts one bore and reports whether the result's measured move is one its
// tool cannot account for. A refused bore records nothing by definition: no body, no measurement.
func recordsAnImpossibleMove(t *testing.T, bore float64) bool {
	t.Helper()
	ring, drill := ringAndDrill(t, bore)
	rec := &diag.Recorder{}
	if _, err := BooleanWithDiagnostics(Cut, ring, drill, rec); err != nil {
		return false
	}
	return rec.Has(CodeBooleanMovedVolumeOutOfToolBracket)
}

// The oracle must be trustworthy before any row leans on it: it is checked against the shipped RD-
// corpus row, whose exactness is established independently in boolean_torus_section_test.go.
func TestTheBoreOracleAgreesWithTheShippedExactRow(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 0.8)
	body, err := Boolean(Cut, ring, drill)
	if err != nil {
		t.Fatalf("the RD- row must build: %v", err)
	}
	q := DefaultQuality()
	removed := query.BodyGeometryProperties(ring, q).Volume - query.BodyGeometryProperties(body, q).Volume
	oracle := boreRemovalOracle(5, 1.5, 5, 0.8, boreQuadratureCells)
	if rel := stdmath.Abs(removed-oracle) / oracle; rel > 1e-5 { // tol:calibrated — measured 3.5e-7
		t.Errorf("the quadrature oracle disagrees with the exact row by %.3g relative (removed %g, oracle %g)", rel, removed, oracle)
	}
}

// TestTheNonConvergentDrillTerminatesAndIsNamed is the corpus row for the third outcome the ground
// rules do not admit. At r = 1.585e-7 the boolean did not RETURN AT ALL: the planar T-junction pass
// subdivides "until stable", and at a scale comparable to its on-edge tolerance it never became
// stable, so the operation hung — neither a refusal nor a wrong body. Two things stand between this
// input and that hang now, and the row asserts both, because either alone can be met dishonestly.
//
//  1. splitTJunctions stops at a provable budget (brep.tjSplitBudget) and says so: a bound without a
//     name would break silently, a name without a bound would still hang. Guarded by the brep-side
//     unit rows on the budget and its decline (kernel/brep/arrange_decline_test.go).
//  2. The pass no longer CHURNS here at all (#3513, ADR-0061 G9). tjTol was one absolute read both as
//     a perpendicular distance to an edge and as a bound on the dimensionless parameter along it; on
//     an edge shorter than a database unit the fixed t-pad excluded no real length near the ends, so a
//     vertex a hair inside an end re-qualified on every shorter half. The parameter reading now
//     converts through the edge's |dP/dt|, and this drill arranges cleanly: measured over
//     kernel/brep + kernel/ops on clean trees, 42 of 24694 arrangements exhausted the budget before,
//     0 after — and all 42 were this drill (20 runs x 2 in the determinism row below, plus 2 here).
//
// So the assertion is INVERTED from what it was: the refusal must still be named and prompt, and it
// must NOT be the arrangement's. Re-introduce the cross-class comparison and this row fails — first
// on the unconverged record, and then, if the bound went too, on the deadline.
func TestTheNonConvergentDrillTerminatesAndIsNamed(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1.585e-7)
	rec := &diag.Recorder{}
	done := make(chan error, 1)
	go func() {
		_, err := BooleanWithDiagnostics(Cut, ring, drill, rec)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, ErrUnmodelledBoolean) {
			t.Fatalf("want the boolean's named refusal; got %v", err)
		}
	case <-time.After(drillDeadline(t)):
		t.Fatalf("the boolean did not terminate within %s: a hang is neither a refusal nor a wrong body",
			drillDeadline(t))
	}
	if rec.Has(brep.CodeArrangementUnconverged) {
		t.Fatalf("the T-junction pass must converge on this drill since #3513 — an unconverged record "+
			"means the on-edge tolerance is being read as a parameter again; got %v", rec.Records())
	}
	// It is refused for the reason its NEIGHBOURS in the sweep are: the capability gap in the
	// torus-cylinder section (G8), not a conditioning failure of the arrangement.
	if !rec.Has(CodeBooleanNoExactCurvedPath) {
		t.Errorf("want the same named refusal the rest of the band gets; got %v", rec.Records())
	}
}

// drillDeadline is the per-boolean budget the sweep rows allow, derived from the test binary's own
// deadline so a slow machine does not turn a correctness row into a flake. Every point the sweep
// measured returns in well under a second; the budget is generous by two orders.
func drillDeadline(t *testing.T) time.Duration {
	t.Helper()
	if d, ok := t.Deadline(); ok {
		if budget := time.Until(d) / 4; budget < 30*time.Second {
			// Never below a second: close to the binary's own deadline the quarter-share collapses
			// (and goes negative once it passes), which would turn every row into an instant failure
			// reported as a hang. A second is still ~14x the slowest point the sweep measured.
			return max(budget, time.Second)
		}
	}
	return 30 * time.Second
}

// TestTheNonConvergentDrillRefusesIdenticallyEveryRun is the determinism half of the row above: this
// drill must refuse the SAME way on every run — one error and one set of diagnostic records, byte for
// byte. It fingerprints whatever refusal occurs, so it holds whichever mechanism produces it.
//
// It was written when that mechanism was the T-junction split budget, whose count depends on the order
// the splits are made; walking the live edge map in its random order made decline-versus-converge a
// run-to-run coin toss, and brep.splitTJunctions walking a SORTED snapshot is what fixed it (final fix
// wave, finding 7). Since #3513 this drill no longer reaches the budget at all — measured, 0 of 24694
// arrangements — and its refusal is boolean.no-exact-curved-path. The sorted snapshot still stands, and
// so does this row: byte-identical refusal is a property of every path, and the ordering it guards is
// still the one the arrangement walks.
func TestTheNonConvergentDrillRefusesIdenticallyEveryRun(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1.585e-7)
	first := drillRefusalFingerprint(t, ring, drill)
	for run := 1; run < 20; run++ {
		if got := drillRefusalFingerprint(t, ring, drill); got != first {
			t.Fatalf("run %d refused differently from run 0:\n%s\nvs\n%s", run, got, first)
		}
	}
}

// drillRefusalFingerprint runs the cut once and prints its error and every recorded diagnostic.
func drillRefusalFingerprint(t *testing.T, ring, drill *topo.Body) string {
	t.Helper()
	rec := &diag.Recorder{}
	_, err := BooleanWithDiagnostics(Cut, ring, drill, rec)
	var b strings.Builder
	fmt.Fprintf(&b, "err=%v\n", err)
	for _, d := range rec.Records() {
		fmt.Fprintf(&b, "%s|%v|%s\n", d.Code, d.Severity, d.Detail)
	}
	return b.String()
}
