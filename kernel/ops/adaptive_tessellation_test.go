// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// offsetBumpSurface is a 4×4 biquadratic patch that is flat except for one raised control point near a
// corner — a curvature feature OFF the centre, the case a single mid-line / diagonal sagitta sample
// misses and a uniform global step over/under-tessellates (#1412).
func offsetBumpSurface(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := make([][]math.Point3, 4)
	w := make([][]float64, 4)
	for i := 0; i < 4; i++ {
		ctrl[i] = make([]math.Point3, 4)
		w[i] = []float64{1, 1, 1, 1}
		for j := 0; j < 4; j++ {
			z := math.Scalar(0)
			if i == 1 && j == 1 { // raise one control point toward the (≈0.25,0.25) corner region
				z = 4
			}
			ctrl[i][j] = math.Point3{X: math.Scalar(i), Y: math.Scalar(j), Z: z}
		}
	}
	knots := []float64{0, 0, 0, 0.5, 1, 1, 1}
	s, err := geom.NewBSplineSurface(2, 2, ctrl, w, knots, knots)
	if err != nil {
		t.Fatalf("offset bump surface: %v", err)
	}
	return s
}

// tightDomeSurface is a centred dome raised so extremely steeply (z far beyond any real model) that even the finest grid (the maxInteriorCells
// floor) cannot meet the chord tolerance — the saturation case #1412 must report.
func tightDomeSurface(t *testing.T) geom.BSplineSurface {
	t.Helper()
	ctrl := [][]math.Point3{
		{math.P3(0, 0, 0), math.P3(0, 1, 0), math.P3(0, 2, 0)},
		{math.P3(1, 0, 0), math.P3(1, 1, 3000), math.P3(1, 2, 0)},
		{math.P3(2, 0, 0), math.P3(2, 1, 0), math.P3(2, 2, 0)},
	}
	w := [][]float64{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}
	s, err := geom.NewBSplineSurface(2, 2, ctrl, w, []float64{0, 0, 0, 1, 1, 1}, []float64{0, 0, 0, 1, 1, 1})
	if err != nil {
		t.Fatalf("tight dome: %v", err)
	}
	return s
}

// TestLocalRefinementConcentratesAtBump is #1412's core criterion: interior refinement adds nodes WHERE
// the curvature is (the off-centre bump), not uniformly — the bump quadrant ends up denser than the far
// flat quadrant, and the refinement adds nodes the base grid alone did not.
func TestLocalRefinementConcentratesAtBump(t *testing.T) {
	s := offsetBumpSurface(t)
	outer := uvSquare(0.02, 0.98, 12)
	base, _ := adaptiveInteriorNodes(s, outer, nil, DefaultQuality(), 1, false)
	refined, _ := adaptiveInteriorNodes(s, outer, nil, DefaultQuality(), 1, true)

	if len(refined) <= len(base) {
		t.Fatalf("local refinement added no nodes: base %d, refined %d", len(base), len(refined))
	}
	bump, far := 0, 0
	for _, n := range refined {
		switch {
		case n[0] < 0.5 && n[1] < 0.5:
			bump++
		case n[0] > 0.5 && n[1] > 0.5:
			far++
		}
	}
	if bump <= far {
		t.Errorf("refinement did not concentrate at the bump: bump quadrant %d nodes, far flat quadrant %d", bump, far)
	}
}

// TestLocalRefinementLeavesFlatFaceUnchanged is #1412 criterion 3 (no flat regression) at its sharpest:
// a flat face has zero curvature everywhere, so local refinement adds exactly nothing on top of the
// base grid — the broader no-regression guarantee is the unchanged golden suite.
func TestLocalRefinementLeavesFlatFaceUnchanged(t *testing.T) {
	flat := unitPatch(t)
	outer := uvSquare(0.05, 0.95, 8)
	base, _ := adaptiveInteriorNodes(flat, outer, nil, DefaultQuality(), 1, false)
	refined, satur := adaptiveInteriorNodes(flat, outer, nil, DefaultQuality(), 1, true)
	if len(refined) != len(base) {
		t.Errorf("flat face densified by local refinement: base %d → refined %d nodes", len(base), len(refined))
	}
	if satur {
		t.Error("a flat face cannot saturate the cap")
	}
}

// TestInteriorRefinementReportsSaturation is #1412's saturation criterion: a face curved too steeply for
// the cell-size floor to meet the chord tolerance reports saturated, and recordCapSaturation surfaces it
// as a searchable diagnostic on the mesh — instead of silently exceeding tolerance.
func TestInteriorRefinementReportsSaturation(t *testing.T) {
	s := tightDomeSurface(t)
	outer := uvSquare(0.02, 0.98, 12)
	_, saturated := adaptiveInteriorNodes(s, outer, nil, DefaultQuality(), 1, true)
	if !saturated {
		t.Fatal("a face too curved for the cell floor should report saturation")
	}

	m := &Mesh{}
	recordCapSaturation(m, saturated, DefaultQuality())
	if !hasDiag(m.Diagnostics, CodeTessellateCapSaturated) {
		t.Errorf("saturation was not recorded as a %q diagnostic: %v", CodeTessellateCapSaturated, m.Diagnostics)
	}
}

// hasDiag reports whether the diagnostics carry the given code.
func hasDiag(ds []diag.Diagnostic, code diag.Code) bool {
	for _, d := range ds {
		if d.Code == code {
			return true
		}
	}
	return false
}
