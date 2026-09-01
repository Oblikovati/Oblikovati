// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestBuildFlatPatternWatertightWithTab a base square plus one developed tab unions into one
// watertight flat solid whose footprint is the square plus the tab, with one fold line.
func TestBuildFlatPatternWatertightWithTab(t *testing.T) {
	t.Parallel()
	const side, thickness, tab = 4.0, 0.2, 1.5
	// Tab on the y=0 edge; BuildFlatPattern orients it outward (−Y, away from the base).
	tabs := []FlatTab{{
		A: math.P2(0, 0), B: math.P2(side, 0), Length: tab, Angle: stdmath.Pi / 2,
	}}

	fp, err := BuildFlatPattern(squareSketch(side), 0, thickness, tabs)
	if err != nil {
		t.Fatalf("BuildFlatPattern: %v", err)
	}
	assertWatertightSolid(t, fp.Body)

	wantVol := (side*side + side*tab) * thickness
	if got := meshVolume(fp.Body); stdmath.Abs(got-wantVol)/wantVol > 1e-3 {
		t.Errorf("flat volume = %.5f, want %.5f", got, wantVol)
	}
	dy := float64(fp.Extents.Diagonal().Y)
	if stdmath.Abs(dy-(side+tab)) > 1e-6 {
		t.Errorf("flat Y-extent = %.5f, want %.5f (side + tab)", dy, side+tab)
	}
	if len(fp.Bends) != 1 || fp.Bends[0].A != tabs[0].A {
		t.Errorf("flat bends = %+v, want one fold line at the tab edge", fp.Bends)
	}
}

// TestBuildFlatPatternRejectsBadGauge a non-positive thickness is rejected with the value.
func TestBuildFlatPatternRejectsBadGauge(t *testing.T) {
	t.Parallel()
	if _, err := BuildFlatPattern(squareSketch(4), 0, 0, nil); err == nil {
		t.Error("BuildFlatPattern with zero thickness must error")
	}
}

// TestBuildFlatPatternBaseOnly with no tabs the flat is just the thickened base plate.
func TestBuildFlatPatternBaseOnly(t *testing.T) {
	t.Parallel()
	const side, thickness = 3.0, 0.1
	fp, err := BuildFlatPattern(squareSketch(side), 0, thickness, nil)
	if err != nil {
		t.Fatalf("BuildFlatPattern: %v", err)
	}
	assertWatertightSolid(t, fp.Body)
	if got, want := meshVolume(fp.Body), side*side*thickness; stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("base-only flat volume = %.5f, want %.5f", got, want)
	}
	if len(fp.Bends) != 0 {
		t.Errorf("base-only flat has %d fold lines, want 0", len(fp.Bends))
	}
}

// TestFlangePlacementCaptured a recomputed flange records its bend geometry: the bend line is
// the picked edge and the outward direction is in the sheet plane (perpendicular to the
// edge), so the flat pattern can lay the tab out.
func TestFlangePlacementCaptured(t *testing.T) {
	t.Parallel()
	fs, edge := seedSheetMetalSheet(t, 4, map[string]string{"BendRadius": "1 mm"})
	pf := NewSheetMetalFlangeFeatures(fs).Add(&SheetMetalFlangeDefinition{
		EdgeKey: edge.ReferenceKey(),
		Height:  func() float64 { return 1 },
	})
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("flange unhealthy: %s", pf.Health().Reason)
	}
	placed, ok := pf.Definition().(PlacedBend)
	if !ok {
		t.Fatal("flange does not implement PlacedBend")
	}
	p, ok := placed.Placement()
	if !ok {
		t.Fatal("flange placement not captured after recompute")
	}
	if p.AxisStart.DistanceTo(p.AxisEnd) <= 0 {
		t.Error("placement bend line is degenerate")
	}
	// Outward must be perpendicular to the bend line (it is the in-plane fold-out direction).
	axis, err := math.UnitVector3FromVector(p.AxisStart.VectorTo(p.AxisEnd))
	if err != nil {
		t.Fatalf("axis dir: %v", err)
	}
	if dot := p.Outward.AsVector().Dot(axis.AsVector()); stdmath.Abs(dot) > 1e-9 {
		t.Errorf("outward·axis = %g, want 0 (perpendicular)", dot)
	}
	if p.Length != 1 || p.Angle <= 0 {
		t.Errorf("placement length/angle = (%g,%g), want (1, >0)", p.Length, p.Angle)
	}
}
