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
	foldedVol := flatVolume(d.Features().Result()[0])

	if _, err := d.AddUnfold(); err != nil {
		t.Fatalf("AddUnfold: %v", err)
	}
	d.Recompute()

	// Cut a 0.4×0.4 square hole through the developed flange (y in [−0.8,−0.4] lies on it).
	sk := d.Sketches().Add(sketch.XYPlane())
	a := sk.Points().Add(gmath.P2(1.8, -0.8))
	b := sk.Points().Add(gmath.P2(2.2, -0.8))
	c := sk.Points().Add(gmath.P2(2.2, -0.4))
	e := sk.Points().Add(gmath.P2(1.8, -0.4))
	sk.Lines().Add(a, b)
	sk.Lines().Add(b, c)
	sk.Lines().Add(c, e)
	sk.Lines().Add(e, a)
	cut := feature.NewSheetMetalCutFeatures(d.features).Add(&feature.SheetMetalCutDefinition{Sketch: sk, ProfileIndex: 0})
	d.Recompute()
	if !cut.Health().OK() {
		t.Fatalf("cut on the flat flange unhealthy: %s", cut.Health().Reason)
	}
	flatCutVol := flatVolume(d.Features().Result()[0])
	if !(flatCutVol < foldedVol) {
		t.Fatalf("cut did not remove material: %.4f vs %.4f", flatCutVol, foldedVol)
	}

	if _, err := d.AddRefold(); err != nil {
		t.Fatalf("AddRefold: %v", err)
	}
	d.Recompute()
	refolded := d.Features().Result()[0]
	assertFlatSolid(t, refolded)
	// The hole survives refold: volume stays reduced (matches the flat-cut volume) and the
	// flange is folded up again.
	if v := flatVolume(refolded); math.Abs(v-flatCutVol)/flatCutVol > 0.02 {
		t.Errorf("refold volume %.4f != flat-cut volume %.4f (the cut was not carried back)", v, flatCutVol)
	}
	if v := flatVolume(refolded); !(v < foldedVol) {
		t.Errorf("refolded part volume %.4f should stay below the uncut folded %.4f", v, foldedVol)
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
