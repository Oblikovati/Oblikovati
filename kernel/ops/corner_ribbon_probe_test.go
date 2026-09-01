// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
)

// TestRibbonSeamNonFoldingAcceptsOutwardCoons4 proves the probe passes the shipped, correct
// coons4 patch (outward ribbons) built from the quarter-cylinder fixture.
func TestRibbonSeamNonFoldingAcceptsOutwardCoons4(t *testing.T) {
	t.Parallel()
	loop := quarterCylLoop(t, 4)
	fill, rails, sides, ok := coons4Fill(loop)
	if !ok {
		t.Fatal("coons4Fill declined the quarter-cyl fixture")
	}
	if !ribbonSeamNonFolding(fill, rails, sides, blendScale()) {
		t.Fatal("probe rejected the correct outward-ribbon coons4 patch")
	}
}

// TestRibbonSeamNonFoldingRejectsInwardRibbon proves the probe FLAGS a deliberately inward-signed
// ribbon (the latent obstacle defect) — the sign check creaseAngle cannot see.
func TestRibbonSeamNonFoldingRejectsInwardRibbon(t *testing.T) {
	t.Parallel()
	loop := quarterCylLoop(t, 4)
	c0, c1, d0, d1, ok := loopRails(loop)
	if !ok {
		t.Fatal("loopRails declined the fixture")
	}
	c0, c1, d0, d1, _ = refineForG1(c0, c1, d0, d1)
	rails := [4]geom.BSplineCurve{c0, c1, d0, d1}
	base, err := geom.CoonsFill(rails[0], rails[1], rails[2], rails[3])
	if err != nil {
		t.Fatalf("CoonsFill: %v", err)
	}
	// INWARD ribbons: the negation of the shipped outward awayRef (the bug we are guarding against).
	length := loopRibLen(loop)
	inward := invertedCoons4Sides(loop, rails, base, length)
	fill, err := geom.FillSurface(rails[0], rails[1], rails[2], rails[3], inward)
	if err != nil {
		t.Fatalf("FillSurface: %v", err)
	}
	fill, _ = pinFillBoundary(fill, rails[0], rails[1], rails[2], rails[3])
	if ribbonSeamNonFolding(fill, rails, inward, blendScale()) {
		t.Fatal("probe accepted an inward-signed (folded) ribbon patch")
	}
}

// invertedCoons4Sides mirrors coons4Sides but anchors on the INWARD cross-derivative (no Scale(-1)),
// producing the folded patch under test. Named test helper (not an inline stub).
func invertedCoons4Sides(loop RailLoop, rails [4]geom.BSplineCurve, base geom.BSplineSurface, length float64) [4]geom.FillSide {
	fs0, _ := ribbonSide(rails[0], loop.Sides[0], inwardCrossV(base, false), length)
	fs1, _ := ribbonSide(rails[1], loop.Sides[2], inwardCrossV(base, true), length)
	fs2, _ := ribbonSide(rails[2], loop.Sides[3], inwardCrossU(base, false), length)
	fs3, _ := ribbonSide(rails[3], loop.Sides[1], inwardCrossU(base, true), length)
	return [4]geom.FillSide{fs0, fs1, fs2, fs3}
}
