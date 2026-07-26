// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The shared PER-FACE DRAWEXE reconciliation. The corpus scoreboard gates on WHOLE-BODY area, which is
// structurally blind to a compensating pair of per-face errors — it has now hidden a gross defect three
// times (F7 shipped a chord across its elliptic wall at +50.9% with its top plane at −73.4% and still
// measured PASS at −0.1234%; S7's run-out patch was 18.5% of r wrong in SHAPE at 0.03% in area; N4's
// corner patch was 27% short with the vertical plane over-reading by nearly the same amount). Ranking
// our per-face mesh areas against OCCT's own `explode result F` + `sprops result_i 1e-6` numbers is what
// actually sees that class, so the oracle gates for individual roots share these helpers.
type drawexeFaceCase struct {
	name string
	// drawexe are DRAWEXE 8.0.0's per-face `sprops result_i 1e-6` areas, sorted DESCENDING (rank-paired
	// against ours, which is sound only when the case's face areas are well separated — state the margin
	// where the case is declared).
	drawexe []float64
	// totalArea is DRAWEXE's `sprops result 1e-6` whole-body number.
	totalArea float64
	// perFaceTol is the per-face RELATIVE budget: mesh quantization only, decades tighter than the defect
	// the gate protects against.
	perFaceTol float64
}

// assertPerFaceAgainstDrawexe fails when the body's face count differs from DRAWEXE's, when any
// rank-paired face area is outside tc.perFaceTol, or when the summed area misses DRAWEXE's total.
func assertPerFaceAgainstDrawexe(t *testing.T, tc drawexeFaceCase, b *topo.Body) {
	t.Helper()
	got := sortedFaceMeshAreas(b)
	if len(got) != len(tc.drawexe) {
		t.Fatalf("%s: %d faces, DRAWEXE has %d", tc.name, len(got), len(tc.drawexe))
	}
	sum := 0.0
	for i, want := range tc.drawexe {
		sum += got[i]
		if rel := stdmath.Abs(got[i]-want) / want; rel > tc.perFaceTol {
			t.Errorf("%s: face #%d (by size) meshes %.6g, DRAWEXE %.6g (rel %+.4f%%, tol %.1g)",
				tc.name, i+1, got[i], want, (got[i]-want)/want*100, tc.perFaceTol)
		}
	}
	if rel := stdmath.Abs(sum-tc.totalArea) / tc.totalArea; rel > tc.perFaceTol {
		t.Errorf("%s: summed face area %.6g, DRAWEXE %.6g (rel %+.4f%%)", tc.name, sum, tc.totalArea,
			(sum-tc.totalArea)/tc.totalArea*100)
	}
}

// sortedFaceMeshAreas returns the body's per-face mesh areas at PropertyQuality, largest first.
func sortedFaceMeshAreas(b *topo.Body) []float64 {
	out := make([]float64, 0, len(b.Faces()))
	for _, f := range b.Faces() {
		out = append(out, ops.MeshArea(ops.TessellateFace(f, ops.PropertyQuality())))
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(out)))
	return out
}

// shippedFaceAreasDesc returns the body's per-face areas AS SHIPPED — through CalculateBodyFacets, so
// the cross-face conformance repair's adoption decisions are included — largest first. The solo
// variant (sortedFaceMeshAreas) is blind to the repair: a face whose solo mesh is fine but whose
// SHIPPED mesh was swapped for a worse one reads clean there.
func shippedFaceAreasDesc(b *topo.Body) []float64 {
	facets := ops.CalculateBodyFacets(b, ops.PropertyQuality())
	out := make([]float64, 0, len(facets.FaceMeshes))
	for _, m := range facets.FaceMeshes {
		out = append(out, ops.MeshArea(m))
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(out)))
	return out
}

// assertShippedPerFaceAgainstDrawexe is assertPerFaceAgainstDrawexe on the SHIPPED meshes, with a
// per-RANK debt ceiling for faces that carry a measured, separately-rooted shortfall. debt[i] (0 =
// none) is a RATCHET on rank i: it may shrink freely, must never grow, and every entry names its root
// where it is declared. totalTol gates the summed area (a per-face debt necessarily shows there too).
func assertShippedPerFaceAgainstDrawexe(t *testing.T, tc drawexeFaceCase, b *topo.Body, debt map[int]float64, totalTol float64) {
	t.Helper()
	got := shippedFaceAreasDesc(b)
	if len(got) != len(tc.drawexe) {
		t.Fatalf("%s: %d faces, DRAWEXE has %d", tc.name, len(got), len(tc.drawexe))
	}
	sum := 0.0
	for i, want := range tc.drawexe {
		sum += got[i]
		budget := tc.perFaceTol
		if d, ok := debt[i]; ok {
			budget = d
		}
		if rel := stdmath.Abs(got[i]-want) / want; rel > budget {
			t.Errorf("%s: SHIPPED face #%d (by size) meshes %.6g, DRAWEXE %.6g (rel %+.4f%%, budget %.3g)",
				tc.name, i+1, got[i], want, (got[i]-want)/want*100, budget)
		}
	}
	if rel := stdmath.Abs(sum-tc.totalArea) / tc.totalArea; rel > totalTol {
		t.Errorf("%s: summed SHIPPED face area %.6g, DRAWEXE %.6g (rel %+.4f%%, budget %.3g)", tc.name, sum,
			tc.totalArea, (sum-tc.totalArea)/tc.totalArea*100, totalTol)
	}
}
