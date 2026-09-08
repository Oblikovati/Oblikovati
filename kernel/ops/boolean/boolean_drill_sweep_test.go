// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
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
// sweep as a test rather than hanging the whole suite (the r=1.585e-7 row is exactly that case).
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
		return drillWrongBody
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
		// ABOVE the floor, inside the capability gap: the size classification does not answer, and these
		// are refused on the pipeline's own merits — by name up to ~6.3e-5 of the extent, and by the
		// post-hoc Requicha volume bracket above that (see TestASmallBoreIsRefusedNotShippedWrong).
		{1e-6, drillRefused}, {1e-4, drillRefused}, {1e-3, drillRefused}, {0.0631, drillRefused},
		// The exact plateau.
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

// TestASmallBoreIsRefusedNotShippedWrong is the corpus row for the CAPABILITY gap the sweep exposed,
// pinned here rather than deferred to a ticket. At r=1e-3 the pipeline builds a valid closed solid
// that removes 2.84 of material where the true bore is 9.4e-6 — a factor of 300000 — and only the
// Requicha volume bracket stands between it and the model. The row asserts the refusal, so the day
// the section is fixed this test says so, and until then a wrong body cannot start shipping.
func TestASmallBoreIsRefusedNotShippedWrong(t *testing.T) {
	t.Parallel()
	ring, drill := ringAndDrill(t, 1e-3)
	body, err := Boolean(Cut, ring, drill)
	if err == nil {
		q := DefaultQuality()
		removed := query.BodyGeometryProperties(ring, q).Volume - query.BodyGeometryProperties(body, q).Volume
		oracle := boreRemovalOracle(5, 1.5, 5, 1e-3, boreQuadratureCells)
		t.Fatalf("a 1e-3 bore built a body removing %g where the oracle says %g: either the section is "+
			"fixed (re-point this row at drillExact) or a wrong body is shipping", removed, oracle)
	}
	// It is a NAMED refusal, and NOT the size one: the feature is resolvable, the pipeline just
	// cannot build it, and mislabelling that as sub-resolution would hide the defect.
	if errors.Is(err, ErrSubResolutionOperand) {
		t.Errorf("a 1e-3 bore is 100x above the resolution floor; refusing it on SIZE would relabel a "+
			"capability gap as a policy: %v", err)
	}
	if !errors.Is(err, ErrUnmodelledBoolean) {
		t.Errorf("want the boolean's named refusal; got %v", err)
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
// subdivides "until stable", and at a scale comparable to its absolute 1e-7 tolerance it never became
// stable, so the operation hung — neither a refusal nor a wrong body. splitTJunctions now stops at a
// provable budget (brep.tjSplitBudget) and the refusal is named end to end.
//
// The row asserts BOTH halves, because either alone can be met dishonestly: a silent break would
// terminate without a name, and a name without a bound would still hang.
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
	if !rec.Has(brep.CodeArrangementUnconverged) {
		t.Fatalf("the unconverged subdivision must be REPORTED, not silently broken out of; got %v", rec.Records())
	}
	// The report must name WHICH of the four splits declined. There are four sites that arrange a
	// face and they fail for the same reason, so a diagnostic that did not distinguish them would
	// send the reader to the wrong one (review round 3).
	var detail string
	for _, d := range rec.Records() {
		if d.Code == brep.CodeArrangementUnconverged {
			detail = d.Detail
		}
	}
	if !strings.Contains(detail, "closed-surface") {
		t.Errorf("the decline must name the split that made it; got %q", detail)
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
