// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"
	"testing"
)

// Green's theorem's two preconditions, pinned directly (M48/C3, Oblikovati/Oblikovati#3453): the
// contour must CLOSE, and the sign that orients it must come from a quantity that is not degenerate.
// Both were reached by real corpus bodies, both silently, and neither was caught by the vector-area
// closure post-condition — see loopsCloseTheirWalk and enclosedTerms for the forensics.

// squareSamples is a closed uv unit square, the reference polyline a well-formed loop develops into.
var squareSamples = []arcSample{{u: 0, v: 0}, {u: 1, v: 0}, {u: 1, v: 1}, {u: 0, v: 1}}

// walkLoop is one loop with the given net travel and periods over the reference square.
func walkLoop(netU, netV, uPeriod, vPeriod float64) faceLoop {
	return faceLoop{
		netU:  netU,
		netV:  netV,
		edges: []loopEdge{{samples: squareSamples, uPeriod: uPeriod, vPeriod: vPeriod}},
	}
}

func TestLoopsCloseTheirWalkAcceptsAReturnAndRefusesAnOpenContour(t *testing.T) {
	t.Parallel()
	const tau = 2 * stdmath.Pi
	cases := []struct {
		name string
		loop faceLoop
		want bool
	}{
		{"a loop that returns to its start closes", walkLoop(0, 0, 0, 0), true},
		{"a round-trip residue is still a return", walkLoop(3e-14, -1.4e-14, tau, 0), true},
		{"a whole period on a PERIODIC parameter is a return (a seam-wrapping band)",
			walkLoop(tau, 0, tau, 0), true},
		{"a whole period backwards is equally a return", walkLoop(-tau, 0, tau, 0), true},
		// The cylinder arm whose loop came back as (wall line, far rim, wall line, near rim): every
		// edge present, none adjacent to its neighbour, so the walk jumped the arm's height and read
		// a lateral area of -12075.67 where 9629.06 is right.
		{"a jump of the face's whole height on a NON-periodic parameter is an open contour",
			walkLoop(0, -57.5736, tau, 0), false},
		// The pole-degenerate B-spline flank: a whole isoparametric edge collapses to one 3-D point,
		// so ParamAt cannot recover the parameter along it and the walk restarts on the wrong branch.
		{"a jump of the whole [0,1] domain is an open contour", walkLoop(0, -1, 0, 0), false},
		{"a period-sized jump in a parameter that has NO period is still an open contour",
			walkLoop(tau, 0, 0, 0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := loopsCloseTheirWalk([]faceLoop{c.loop}); got != c.want {
				t.Errorf("loopsCloseTheirWalk(netU=%g netV=%g) = %v, want %v", c.loop.netU, c.loop.netV, got, c.want)
			}
		})
	}
}

func TestLoopsCloseTheirWalkRefusesWhenAnyLoopIsOpen(t *testing.T) {
	t.Parallel()
	loops := []faceLoop{walkLoop(0, 0, 0, 0), walkLoop(0, -1, 0, 0)}
	if loopsCloseTheirWalk(loops) {
		t.Error("one open loop makes the whole boundary open; the face must be refused")
	}
}

// TestLoopRegionSignReadsTheMeasureNotTheShoelace is the pole-degenerate regression. A face whose
// surface has a pole develops a loop into a DEGENERATE uv polyline, so its shoelace is rounding noise
// (±1e-17 measured) while the boundary integral it is supposed to orient is perfectly well defined.
// Two congruent B-spline flanks of one body read opposite noise signs and both came out NEGATIVE,
// putting the body's volume 1.7% out — and their vector-area residuals were mirror images, so they
// cancelled and the closure post-condition passed the wrong answer through.
func TestLoopRegionSignReadsTheMeasureNotTheShoelace(t *testing.T) {
	t.Parallel()
	degenerate := []faceLoop{{edges: []loopEdge{{samples: []arcSample{
		{u: 0, v: 0}, {u: 1, v: 0}, {u: 2, v: 0}, {u: 1, v: 0}, // collapsed onto one line: no enclosed area
	}}}}}
	if a := loopSignedArea(degenerate[0]); stdmath.Abs(a) > 1e-9 {
		t.Fatalf("fixture is not degenerate: shoelace %g — the test premise needs a noise-only shoelace", a)
	}
	for _, measure := range []float64{1245.074239, -1245.074239} {
		want := 1.0
		if measure < 0 {
			want = -1
		}
		if got := loopRegionSigns(degenerate, []float64{measure})[0]; got != want {
			t.Errorf("a top-level loop with boundary integral %g signed %g, want %g", measure, got, want)
		}
	}
}

// TestAClosureResidueIsJudgedAgainstTheContourNotAnEpsilon is the arbitration between the two things
// this gate must tell apart, which an absolute parametric epsilon could not.
//
// A spiric section TURNS at each end: the branches meet at w = 1 ± 1e-16 and acos of that puts the
// shared endpoint 2.98e-8 away in u — double rounding amplified by a square root, on a loop that
// closes exactly. An absolute 1e-9 called every torus oval an open contour and refused five corpus
// faces the integrator computes to 1e-9 of their closed forms. The breaks the gate is FOR are six
// orders larger relative to the contour, so judging the residue against the contour's own extent
// separates them with room to spare.
func TestAClosureResidueIsJudgedAgainstTheContourNotAnEpsilon(t *testing.T) {
	t.Parallel()
	const spiricResidue = 2.98e-8 // measured on the torus∩plane oval
	wide := []arcSample{{u: 0, v: 0}, {u: 6, v: 0}, {u: 6, v: 3}, {u: 0, v: 3}}
	loopOf := func(netU float64, samples []arcSample) []faceLoop {
		return []faceLoop{{netU: netU, edges: []loopEdge{{samples: samples, uPeriod: 2 * stdmath.Pi}}}}
	}
	if !loopsCloseTheirWalk(loopOf(spiricResidue, wide)) {
		t.Error("a turning curve's irreducible 3e-8 residue is a closed contour, not an open one")
	}
	if loopsCloseTheirWalk(loopOf(1.0, wide)) {
		t.Error("a jump of a sixth of the contour is an open contour whatever its absolute size")
	}
	// The scale is the CONTOUR's, not the world's: the same absolute residue on a contour a million
	// times smaller is no longer negligible against it.
	tiny := []arcSample{{u: 0, v: 0}, {u: 6e-6, v: 0}, {u: 6e-6, v: 3e-6}, {u: 0, v: 3e-6}}
	if loopsCloseTheirWalk(loopOf(spiricResidue, tiny)) {
		t.Error("a residue larger than 1e-6 of the contour it belongs to must read as open")
	}
}
