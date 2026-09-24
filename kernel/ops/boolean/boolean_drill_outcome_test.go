// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"errors"
	stdmath "math"
	"testing"
	"time"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// What ONE point of the axial-drill sweep DID: the outcome vocabulary, the independent oracle each point
// is classified against, and the classifier itself. The rows that assert the sweep's shape are in
// boolean_drill_sweep_test.go, which this file was split out of to keep both under the 500-line rule
// (#3527 review 1, Minor 4). One responsibility each: what an outcome is here, which outcomes the sweep
// requires there.

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
	// The two refusals are SEPARATE outcomes, and that is the point of naming them (#3527). The rows
	// below the floor are refused by the size classification before any geometry is built, and the rows
	// above it by the general per-face boolean; scoring both as "refused" let either row pass on the
	// other's reason, so a change that moved the floor — or lost one named decline and gained the
	// other — read green. A third value catches a refusal that is neither, which would be an error no
	// caller can act on.
	drillRefusedTooThin drillOutcome = "refused:sub-resolution" // ErrSubResolutionOperand, at classification
	drillRefusedNoPath  drillOutcome = "refused:no-exact-path"  // ErrUnmodelledBoolean, after the pipeline ran
	drillRefusedUnnamed drillOutcome = "refused:unnamed"        // an error neither of the two names
	// drillNilBody is err=nil with no body at all. It is not a refusal (nothing was named) and not
	// drillSilent (the ring did not come back either), and it used to be scored as a refusal.
	drillNilBody   drillOutcome = "nilbody"
	drillSilent    drillOutcome = "silent"    // returned the target untouched, err=nil, nothing recorded
	drillExact     drillOutcome = "exact"     // torus + cylinder, 4 loops, volume matches the oracle
	drillWrongBody drillOutcome = "wrongbody" // a body that is not the exact answer, returned anyway
	// drillCoarse is the exact SECTION — valid, torus + cylinder, 4 loops — whose analytic removed
	// volume misses the oracle. It is a separate outcome from drillWrongBody because the two are
	// separate defects: a wrong body is a modelling failure, this is a MEASUREMENT one, and calling
	// them the same thing is what let the RING band be filed as "the section is wrong" for a whole
	// milestone when the section was right and only the number was not (Oblikovati/Oblikovati#3516).
	//
	// "Coarse" understates it and is kept only because the SHAPE is the thing this outcome asserts:
	// the miss is not a bounded imprecision but noise that can be sign-wrong — at bore 8.913e-4 the
	// measured removal is NEGATIVE, the result reading larger than the ring it was cut from. Those
	// radii are recorded by CodeBooleanMovedVolumeOutOfToolBracket and pinned by
	// TestAnImpossibleRemovalIsRecorded; the root is Oblikovati/Oblikovati#3538.
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
	if err != nil {
		return refusalName(err)
	}
	if body == nil {
		return drillNilBody
	}
	if len(body.Faces()) == 1 {
		return drillSilent // the ring came back with no bore in it at all
	}
	return classifyBoredRing(ring, body, bore)
}

// refusalName says WHICH refusal an error is, so a row pins the reason and not merely the fact.
//
// Measured on this tree: the four radii at and below 1e-8 return ErrSubResolutionOperand naming the
// tool's thickness against the model's 2.0049937655763422e-08 weld, and 1e-6 and 1e-4 return
// ErrUnmodelledBoolean ("no exact path models this contact configuration ... the exact result failed its
// own acceptance gate"), which is the boolean.no-exact-curved-path decline the comments below name.
func refusalName(err error) drillOutcome {
	switch {
	case errors.Is(err, ErrSubResolutionOperand):
		return drillRefusedTooThin
	case errors.Is(err, ErrUnmodelledBoolean):
		return drillRefusedNoPath
	}
	return drillRefusedUnnamed
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
