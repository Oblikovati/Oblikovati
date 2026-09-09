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

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// The sweep behind the resolution floor (finding 4 of the stage-6 review: the floor was asserted, not
// measured). It drives the RD- corpus family — an axial drill through the RING — over the tool
// thickness, and classifies each point against an INDEPENDENT oracle rather than against the kernel.
//
// The regimes it establishes, and which one the size classification may claim, are recorded in
// boolean_resolution_decline.go and ADR-0061. In short: only the SILENT band is a resolution limit.

// boreRemovalOracle integrates the material an axial drill of radius r centred at (at,0) removes from
// a torus of the given major/minor radii, by polar quadrature over the drill's disc. It is
// independent of the kernel: at distance d from the axis the tube runs from -h to +h with
// h = sqrt(minor^2 - (d-major)^2), so the removed volume is the integral of 2h over the disc.
// Verified against the shipped RD- row: at r=0.8 it agrees with the built solid to 3.5e-7 relative.
func boreRemovalOracle(major, minor, at, r float64, n int) float64 {
	sum, dr, dth := 0.0, r/float64(n), 2*stdmath.Pi/float64(n)
	for i := range n {
		rho := r * (float64(i) + 0.5) / float64(n)
		for j := range n {
			th := 2 * stdmath.Pi * (float64(j) + 0.5) / float64(n)
			d := stdmath.Hypot(at+rho*stdmath.Cos(th), rho*stdmath.Sin(th))
			if s := minor*minor - (d-major)*(d-major); s > 0 {
				sum += 2 * stdmath.Sqrt(s) * rho * dr * dth
			}
		}
	}
	return sum
}

// boreQuadratureCells is the polar quadrature's resolution per axis. 240x240 puts the oracle three
// orders inside the 1% band the exactness test uses, measured at r=0.8 (rel 3.5e-7).
const boreQuadratureCells = 240 // tol:numeric — a quadrature cell count, not a model length

// drillOutcome is what one sweep point did: refused by name, silently unchanged, or exact.
type drillOutcome string

const (
	drillRefused   drillOutcome = "refused"
	drillSilent    drillOutcome = "silent"    // returned the target untouched, err=nil, nothing recorded
	drillExact     drillOutcome = "exact"     // torus + cylinder, 4 loops, volume matches the oracle
	drillWrongBody drillOutcome = "wrongbody" // a body that is not the exact answer, returned anyway
	// drillCoarse is the exact SECTION — valid, torus + cylinder, 4 loops — whose analytic removed
	// volume misses the oracle. It is a separate outcome from drillWrongBody because the two are
	// separate defects: a wrong body is a modelling failure, this is a MEASUREMENT one, and calling
	// them the same thing is what let the RING band be filed as "the section is wrong" for a whole
	// milestone when the section was right and only the number was not (Oblikovati/Oblikovati#3516).
	drillCoarse drillOutcome = "coarse"
)

// sweepDrill runs one point of the sweep and classifies the outcome against the oracle.
func sweepDrill(t *testing.T, bore float64) drillOutcome {
	t.Helper()
	ring, drill := ringAndDrill(t, bore)
	body, ok, err := booleanWithinDeadline(t, ring, drill)
	if !ok {
		t.Fatalf("bore %g: the boolean did not terminate within %s", bore, drillDeadline(t))
	}
	if err != nil || body == nil {
		return drillRefused
	}
	if len(body.Faces()) == 1 {
		return drillSilent // the ring came back with no bore in it at all
	}
	return classifyBoredRing(ring, body, bore)
}

// booleanWithinDeadline runs one cut under a deadline, so a pipeline that stops terminating fails the
// sweep as a test rather than hanging the whole suite. It is a STANDING guard, not a description of any
// row: the r=1.585e-7 row was the case it was written for, and since #3513 that row returns promptly
// like every other. The deadline stays because "does not answer" is the outcome the ground rules do not
// admit, and only a deadline can tell it from a slow one.
func booleanWithinDeadline(t *testing.T, ring, drill *topo.Body) (*topo.Body, bool, error) {
	t.Helper()
	type result struct {
		body *topo.Body
		err  error
	}
	done := make(chan result, 1)
	go func() {
		b, err := Boolean(Cut, ring, drill)
		done <- result{b, err}
	}()
	select {
	case r := <-done:
		return r.body, true, r.err
	case <-time.After(drillDeadline(t)):
		return nil, false, nil
	}
}

// classifyBoredRing decides whether a built result IS the exact bore: the RD- row's own shape gate
// (one torus face and one cylinder face, two loops each) plus the removed volume against the oracle.
func classifyBoredRing(ring, body *topo.Body, bore float64) drillOutcome {
	tori, cyls, loops := 0, 0, 0
	for _, f := range body.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
		case geom.Cylinder:
			cyls++
		}
		loops += len(f.Loops())
	}
	if !Validate(body).ValidSolid() || tori != 1 || cyls != 1 || loops != 4 {
		return drillWrongBody
	}
	q := DefaultQuality()
	removed := query.BodyGeometryProperties(ring, q).Volume - query.BodyGeometryProperties(body, q).Volume
	oracle := boreRemovalOracle(5, 1.5, 5, bore, boreQuadratureCells)
	if stdmath.Abs(removed-oracle) > 0.01*oracle { // tol:calibrated — 1%, four orders above the oracle's own 3.5e-7
		return drillCoarse // the shape gate passed: the section is right and the measurement is not
	}
	return drillExact
}

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
		{1e-11, drillRefused}, {1e-10, drillRefused}, {1e-9, drillRefused}, {1e-8, drillRefused},
		// ABOVE the floor: the size classification does not answer, and the outcome is the pipeline's
		// own. Up to ~4e-4 the general per-face boolean declines the pair BY NAME
		// (boolean.no-exact-curved-path) and no geometry is built.
		{1e-6, drillRefused}, {1e-4, drillRefused},
		// From ~6.3e-4 the pipeline BUILDS the exact section — valid, one torus, one cylinder, four
		// loops, both intersection edges on both surfaces to 4.4e-16 (torus) and 4.4e-12 (cylinder).
		// What separates this row from the plateau is not the body but the number: the section curve's
		// tangent is a fixed-step central difference of an azimuth root whose own error grows as the
		// section's azimuth swing shrinks, so the Green face integral reads this bore's wall 43% light
		// and the removed volume misses the oracle (Oblikovati/Oblikovati#3516; the measured law and
		// the two plateau sweeps that refute a tuned step are in that issue).
		// Measured on this pair: the removed volume misses the oracle by 42.9% at 1e-3 and 9.4% at
		// 2e-3, and the transition to the plateau is between 4e-3 (1.17%) and 6e-3 (0.06%). The two
		// rows are chosen away from that edge, where the miss is noise and a pinned row would flake.
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
	assertBoredRingIsIntegratedAnalytically(t, body, 1e-3)
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

// assertBoredRingIsIntegratedAnalytically is the #3516 regression on the integrator: the bored torus
// face must be integrable, so the body's volume comes from its analytic B-rep and not from a mesh.
// The bound is 1e-6 RELATIVE — three orders looser than the 1.8e-8 measured, and four orders tighter
// than the 1.28e-2 the tessellated fallback returns, so it separates the two answers without pinning
// the integrator's own precision.
func assertBoredRingIsIntegratedAnalytically(t *testing.T, body *topo.Body, bore float64) {
	t.Helper()
	props, ok := query.AnalyticGeometryProperties(body)
	if !ok {
		t.Fatalf("the bored ring must integrate over its analytic B-rep; the mesh fallback measures "+
			"this torus %g light, which is %gx the bore's own material",
			ringAnalyticVolume-219.2269659, (ringAnalyticVolume-219.2269659)/boreRemovalOracle(5, 1.5, 5, bore, boreQuadratureCells))
	}
	want := ringAnalyticVolume - boreRemovalOracle(5, 1.5, 5, bore, boreQuadratureCells)
	if rel := stdmath.Abs(props.Volume-want) / want; rel > 1e-6 { // tol:calibrated — measured 1.8e-8
		t.Errorf("bored-ring volume %.12g, want %.12g (%.3g relative)", props.Volume, want, rel)
	}
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
