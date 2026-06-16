// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// cutRectangle adds a closed rectangle (two opposite corners) on the XY plane and returns the
// sketch — the cut profile the unfold/refold round-trip tests develop.
func cutRectangle(d *PartComponentDefinition, x1, y1, x2, y2 float64) *sketch.Sketch {
	sk := d.Sketches().Add(sketch.XYPlane())
	a := sk.Points().Add(gmath.P2(x1, y1))
	b := sk.Points().Add(gmath.P2(x2, y1))
	c := sk.Points().Add(gmath.P2(x2, y2))
	e := sk.Points().Add(gmath.P2(x1, y2))
	sk.Lines().Add(a, b)
	sk.Lines().Add(b, c)
	sk.Lines().Add(c, e)
	sk.Lines().Add(e, a)
	return sk
}

// unfoldCutRefold runs the cut-while-flat round trip on d: unfold flat, cut the rectangle
// through the developed part, then refold. It fails the test if any step is unhealthy or the
// cut removes no material, and returns the folded, flat-cut, and refolded volumes — so each
// caller asserts the carry-back (refolded ≈ flat-cut) for its own cut placement.
func unfoldCutRefold(t *testing.T, d *PartComponentDefinition, x1, y1, x2, y2 float64) (folded, flatCut, refolded float64) {
	t.Helper()
	folded = flatVolume(d.Features().Result()[0])
	if _, err := d.AddUnfold(); err != nil {
		t.Fatalf("AddUnfold: %v", err)
	}
	d.Recompute()
	assertFlatSolid(t, d.Features().Result()[0]) // the bend developed to a proper flat strip
	cut := feature.NewSheetMetalCutFeatures(d.features).Add(&feature.SheetMetalCutDefinition{Sketch: cutRectangle(d, x1, y1, x2, y2), ProfileIndex: 0})
	d.Recompute()
	if !cut.Health().OK() {
		t.Fatalf("cut on the flat unhealthy: %s", cut.Health().Reason)
	}
	flatCut = flatVolume(d.Features().Result()[0])
	if !(flatCut < folded) {
		t.Fatalf("cut removed no material: %.4f vs %.4f", flatCut, folded)
	}
	if _, err := d.AddRefold(); err != nil {
		t.Fatalf("AddRefold: %v", err)
	}
	d.Recompute()
	assertFlatSolid(t, d.Features().Result()[0])
	return folded, flatCut, flatVolume(d.Features().Result()[0])
}

// TestCutCrossingBendSurvivesRefold a slot cut while flat that STRADDLES the bend line is
// carried fully through the bend on refold — reaching the base, the bend region, and the
// flange. The regression for the rigid-rotation defect, where the bend stayed curved so a
// crossing cut reached only one face: the refolded volume must match the flat-cut volume (if
// the bend region kept its material, the refolded volume would be higher).
func TestCutCrossingBendSurvivesRefold(t *testing.T) {
	d, _ := sheetWithFlange(t) // base y∈[0,4]; flange on the y=0 edge, develops into −Y
	// A slot crossing y=0: from the base (y>0) through the bend region into the flange (y<0).
	_, flatCut, refolded := unfoldCutRefold(t, d, 1.6, -0.6, 2.4, 0.6)
	if math.Abs(refolded-flatCut)/flatCut > 0.02 {
		t.Errorf("refolded volume %.4f != flat-cut %.4f: the slot was not carried through the bend (one face kept its material)", refolded, flatCut)
	}
}

// maxZ returns the body's highest vertex Z — the out-of-plane extent that drops to ~the gauge
// when the part is flat and rises again when refolded.
func maxZ(b *topo.Body) float64 {
	m := math.Inf(-1)
	for _, v := range b.Vertices() {
		if v.Point().Z > m {
			m = v.Point().Z
		}
	}
	return m
}

func assertFlatSolid(t *testing.T, b *topo.Body) {
	t.Helper()
	if !b.IsSolid() {
		t.Error("unfold/refold result is not a solid")
	}
	if open := ops.BoundaryEdges(b); len(open) != 0 {
		t.Errorf("result has %d boundary edges, want 0 (watertight)", len(open))
	}
}

// TestUnfoldRefoldRoundTrip a flanged sheet unfolds to a flat watertight solid (the flange
// lies in the base plane) and refolds back to the folded part — volume preserved throughout
// (the moving flange is rigidly transformed and rejoined, no material added or lost).
func TestUnfoldRefoldRoundTrip(t *testing.T) {
	d, _ := sheetWithFlange(t)
	folded := d.Features().Result()[0]
	v0, z0 := flatVolume(folded), maxZ(folded)
	gauge := d.SheetMetal().Thickness()
	if z0 < 5*gauge {
		t.Fatalf("folded flange should rise well above the gauge; maxZ=%.3f gauge=%.3f", z0, gauge)
	}

	if _, err := d.AddUnfold(); err != nil {
		t.Fatalf("AddUnfold: %v", err)
	}
	d.Recompute()
	flat := d.Features().Result()[0]
	assertFlatSolid(t, flat)
	if mz := maxZ(flat); mz > 2*gauge {
		t.Errorf("unfolded part is not flat: maxZ=%.3f, want ~gauge %.3f", mz, gauge)
	}
	if v := flatVolume(flat); math.Abs(v-v0)/v0 > 0.02 {
		t.Errorf("unfold changed volume: %.4f vs folded %.4f", v, v0)
	}

	if _, err := d.AddRefold(); err != nil {
		t.Fatalf("AddRefold: %v", err)
	}
	d.Recompute()
	refolded := d.Features().Result()[0]
	assertFlatSolid(t, refolded)
	if mz := maxZ(refolded); math.Abs(mz-z0) > 0.05 {
		t.Errorf("refold did not restore the fold: maxZ=%.3f, want %.3f", mz, z0)
	}
	if v := flatVolume(refolded); math.Abs(v-v0)/v0 > 0.02 {
		t.Errorf("refold changed volume: %.4f vs original %.4f", v, v0)
	}
}

// TestCutWhileFlatSurvivesRefold the headline of the body-transform unfold: a hole cut into
// the developed flange while flat is carried back onto the folded part by refold — because
// unfold/refold are body transforms, the cut is part of the topology, not lost.
func TestCutWhileFlatSurvivesRefold(t *testing.T) {
	d, _ := sheetWithFlange(t) // flange on the y=0 top edge; unfolded it extends into −Y
	// A 0.4×0.4 hole wholly on the developed flange (y in [−0.8,−0.4]) — away from the bend.
	folded, flatCut, refolded := unfoldCutRefold(t, d, 1.8, -0.8, 2.2, -0.4)
	if math.Abs(refolded-flatCut)/flatCut > 0.02 {
		t.Errorf("refold volume %.4f != flat-cut volume %.4f (the cut was not carried back)", refolded, flatCut)
	}
	if !(refolded < folded) {
		t.Errorf("refolded part volume %.4f should stay below the uncut folded %.4f", refolded, folded)
	}
}

// TestUnfoldRejectsNoBends unfold on a flat base sheet (no bends) errors.
func TestUnfoldRejectsNoBends(t *testing.T) {
	d := NewPartComponentDefinition()
	if _, err := d.EnableSheetMetal(); err != nil {
		t.Fatalf("EnableSheetMetal: %v", err)
	}
	addRectFace(t, d, 4, 3)
	if _, err := d.AddUnfold(); err == nil {
		t.Error("AddUnfold with no bends must error")
	}
}
