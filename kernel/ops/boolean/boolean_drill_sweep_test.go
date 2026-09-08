// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	stdmath "math"
	"testing"

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
	body, err := Boolean(Cut, ring, drill)
	if err != nil || body == nil {
		return drillRefused
	}
	if len(body.Faces()) == 1 {
		return drillSilent // the ring came back with no bore in it at all
	}
	return classifyBoredRing(ring, body, bore)
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
		// Below the floor: the silent band the size classification now refuses by name.
		{1e-11, drillRefused}, {1e-10, drillRefused}, {1e-9, drillRefused},
		// Above the floor and inside the capability gap: refused by name, on the pipeline's own merits.
		{1e-8, drillRefused}, {1e-6, drillRefused}, {1e-4, drillRefused}, {1e-3, drillRefused}, {0.0631, drillRefused},
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
// 0.0998 x Weld, i.e. r ~ 1e-9, and these decades run from 1e-12 to 1e-8 either side of it). A
// "silent" outcome — the ring handed back with no bore in it, err=nil, nothing recorded — is the
// defect; every other outcome is honest.
//
// It stops at 1e-8 on purpose. Above the floor the classification no longer answers and each point
// costs a full curved boolean, and at least one radius near 1.6e-7 does not terminate in any budget
// this suite can afford — a pathology of the small-radius torus-cylinder section, recorded in
// ADR-0061 alongside the wrong-volume band it sits in. The upper decades are covered instead by
// TestTheAxialDrillSweepPinsTheResolutionFloor's named points, which run in ~1.7 s.
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
