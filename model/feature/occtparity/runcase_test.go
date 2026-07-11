// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// midpointOfA1PickedEdge is the locator the oracle recorded for simple/A1's picked edge s_5
// (box 100³, the vertical edge at x=100,y=0). Source: oracle A1.json.
func midpointOfA1PickedEdge() [3]float64 { return [3]float64{100, 0, 50} }

// TestRunCaseSimpleA1 drives the real fillet feature on OCCT's exact simple/A1 input and
// asserts OCCT's reference area (59527.9) within its 1% tolerance. A failure here is a real
// parity signal for the greening backlog — the assertion must never be loosened.
func TestRunCaseSimpleA1(t *testing.T) {
	r := Record{
		Grid: "simple", Case: "A1", Verb: "blend", ExpectedArea: 59527.9, Deps: 0.01,
		InputStep: "A1.step",
		Picks: []Pick{{
			Radius:  10,
			Locator: Locator{Midpoint: midpointOfA1PickedEdge(), Direction: [3]float64{0, 0, 1}},
		}},
	}
	RunCase(t, r, "testdata")
}
