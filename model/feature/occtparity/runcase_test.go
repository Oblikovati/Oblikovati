// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// a1PickedEdgeLocator is the locator the oracle recorded for simple/A1's picked edge s_5
// (box 100³, the vertical edge at x=100,y=0, from z=0 to z=100). A straight edge, so its
// arc-length centroid equals its midpoint (100,0,50) and its length is 100. Source: A1.json.
func a1PickedEdgeLocator() Locator {
	return Locator{
		Midpoint:  [3]float64{100, 0, 50},
		Direction: [3]float64{0, 0, 1},
		Centroid:  [3]float64{100, 0, 50},
		Length:    100,
	}
}

// TestRunCaseSimpleA1 drives the real fillet feature on OCCT's exact simple/A1 input and
// asserts OCCT's reference area (59527.9) within its 1% tolerance. A failure here is a real
// parity signal for the greening backlog — the assertion must never be loosened.
func TestRunCaseSimpleA1(t *testing.T) {
	t.Parallel()
	r := Record{
		Grid: "simple", Case: "A1", Verb: "blend", ExpectedArea: 59527.9, Deps: 0.01,
		InputStep: "A1.step",
		Picks:     []Pick{{Radius: 10, Locator: a1PickedEdgeLocator()}},
	}
	RunCase(t, r, "testdata")
}
