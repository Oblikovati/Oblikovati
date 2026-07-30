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
//
// ★★ CAPTURING THE ORACLE — IDENTIFY A FACE BY AREA AND CLOSED FORM, NEVER BY `bounding`.
//
// DRAWEXE's `bounding <face>` on a TRIMMED face does NOT return a tight box of the trimmed region. It
// returns the box of the underlying surface's POLE NET (its control hull / natural parametric extent),
// which for a trimmed analytic face can be dramatically larger than the face itself, and for a periodic
// surface can extend to the whole revolution. It is a fast reject box, not a measurement, and reading it
// as one has already nearly caused a mis-attribution here: simple/W2's blend band reports
// z ∈ [−0.29884, 1.00003] from `bounding` while its true z extent is a small fraction of that, which read
// as OCCT terminating the band somewhere it does not.
//
// So when you run a case and have to say WHICH result_i is which face, use:
//   - `sprops result_i 1.e-9` — the area, matched against the closed form of the face you expect; and
//   - `mksurface s result_i ; dump s` — the exact surface (plane origin/axis, cylinder radius, torus
//     Origin/Axis/Radii), which is what actually names the face; and
//   - `dump result_i` for the wire — its edges' UV points on BOTH pcurves give the trim endpoints exactly,
//     which is how simple/W2's run-out boundary was read off OCCT (torus (0.98506, π/2) → (1.55675, π),
//     plane z=0 (2.33652, 0) → (2.98586, 0.2)).
//
// ★★ AND READ A PER-FACE AREA AT `1.e-12` OR TIGHTER, THEN CHECK IT AGAINST A CLOSED FORM.
//
// `sprops`'s second argument is not a print precision — it is the target relative error of DRAWEXE's own
// adaptive QUADRATURE, and on a trimmed oblique quadric that quadrature is UNCONVERGED at the 1e-6 this
// harness used for years. Measured live on simple/T6's obstacle wall (`blend result s 6 s_7`,
// `explode result F`), the SAME face `result_10`:
//
//	1.e-6  -> 2355.61     1.e-9  -> 2393.32     1.e-12 -> 2384.17     1.e-13 -> (no output at all)
//
// a 1.6 % spread, straddling the closed form 2381.677340 (right-section perimeter 79.213410 x axial
// extent sqrt(904) = 30.066593) from both sides. Every one of T6's other ten faces reads IDENTICALLY at
// all three tolerances — the sensitivity is the trimmed oblique elliptical cylinder alone — so the
// whole-body number inherits it: `sprops result` reads 6845.4 / 6883.11 / 6873.96 at the same three.
// Two reports and one oracle comment in this tree each recorded one of those three as "the live OCCT
// value" and drew opposite conclusions about who was right.
//
// So: read per-face areas at >= 1.e-12 (tighter emits nothing, silently), state the tolerance with the
// number, and adjudicate a disagreement against a CLOSED FORM rather than against another DRAWEXE run at
// another tolerance. A flat analytic face (plane, right cylinder, cone, sphere) is stable at any of them
// and needs no ceremony; a trimmed oblique/swept quadric does.
//
// `-b -f` DOES interleave `sprops` output with `puts` markers, and MANY `sprops` calls in one process are
// fine — all eleven of T6's faces above were read at three tolerances each in a SINGLE invocation, and
// `dlog reset` / `dlog on` / `dlog get` captures the full blocks, not merely the command echo. (An earlier
// report recorded the opposite on all three counts and prescribed one process per face; it is wrong.)
// `CSF_TestDataPath` must be exported by hand — `drawenv.sh` does not set it — and `dprecision` does not
// exist. The environment is `test-utilities/occt-blend/oracle/drawenv.sh`.
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
