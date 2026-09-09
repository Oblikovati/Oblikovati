// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A trimmed curved face can reach past its own boundary edges, and the edge sweep alone then does not
// bound the body. The smallest case is a hemisphere: its only edge is the equator, so the box came out
// with ZERO height for a ball of radius 5, and a tool built to cover that box missed the body entirely
// (ADR-0061). The face's chart says which side of the equator the cap is on, which no rule about the
// edges can (ADR-0063).
func TestRangeBoxSweepsAChartedFacesWindow(t *testing.T) {
	t.Parallel()
	const r = 5.0
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), r)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	body := hemisphereBodyWithChart(t, sphere)
	box := body.RangeBox()
	if float64(box.Min.Z) > -r+1e-9 { // tol:numeric — the pole is a chart-window corner, hit exactly
		t.Errorf("range box reaches z = %v, want the south pole at %v", box.Min.Z, -r)
	}
	if float64(box.Max.Z) > 1e-9 {
		t.Errorf("range box reaches z = %v above the equator, want 0", box.Max.Z)
	}
}

// The sweep must stay TIGHT: a small cap on a big sphere bounds the cap, never the whole ball.
func TestRangeBoxDoesNotBalloonASmallCap(t *testing.T) {
	t.Parallel()
	const r = 5.0
	sphere, err := geom.NewSphere(math.P3(0, 0, 0), r)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	body := capBodyWithChart(t, sphere, stdmath.Asin(0.8)) // the cap above z = 4
	if got := float64(body.RangeBox().Min.Z); got < 4-0.01 {
		t.Errorf("a cap above z = 4 reports a box reaching z = %v: the whole ball, not the cap", got)
	}
}

// hemisphereBodyWithChart is a one-face body carrying the LOWER half of the sphere as its chart.
func hemisphereBodyWithChart(t *testing.T, sphere geom.Sphere) *Body {
	t.Helper()
	return chartedSphereBody(t, sphere, -stdmath.Pi/2, 0)
}

// capBodyWithChart is a one-face body carrying the cap above latitude v0 as its chart.
func capBodyWithChart(t *testing.T, sphere geom.Sphere, v0 float64) *Body {
	t.Helper()
	return chartedSphereBody(t, sphere, v0, stdmath.Pi/2)
}

// chartedSphereBody builds a body of one boundary-less sphere face whose chart is the latitude band
// [v0, v1] over the whole azimuth — the shape a cap's trim has.
func chartedSphereBody(t *testing.T, sphere geom.Sphere, v0, v1 float64) *Body {
	t.Helper()
	b := NewBuilder(true, NewLineage(Tok("test", "body", 0)))
	f := b.AddFace(sphere, NewLineage(Tok("test", "face", 0)))
	f.SetChart([][]math.Point2{{
		math.P2(0, v0), math.P2(2*stdmath.Pi, v0), math.P2(2*stdmath.Pi, v1), math.P2(0, v1),
	}})
	return b.Build()
}
