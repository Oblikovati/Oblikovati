// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestConformanceRepairNeverLosesFaceArea is the corpus-wide invariant guard for the cross-face
// conformance repair's ADOPTION decision (kernel/ops/conformance_adopt.go).
//
// The repair re-meshes a face so it stops cracking against its neighbour and swaps the result in. It
// used to make that swap on a FOLD count alone, with no fidelity check, so a re-mesh that tiled far
// LESS of the face than the mesh it replaced was adopted silently. Measured across the corpus (a
// provenance sweep of all 151 adoption decisions on the 20 cases that run the repair), NINE such
// adoptions were shipping — complex/D8 lost 39% of one corner cylinder and 8.6% of its fillet band,
// complex/F2 lost 30%/41% of two walls, simple/Y4 26% of a plane, simple/Q5 0.009% of one wall.
// D8's band alone was the whole of that case's −0.90% shipped-vs-closed-form area gap. A crack is a
// hairline T-junction on geometry that is otherwise right; a face that is 39% missing is not.
//
// THE INVARIANT, stated on the SHIPPED body and independently of the kernel helper under test: a
// face's mesh in BODY context (which is the solo mesh, then possibly a conformance re-mesh) may not
// have less area than its SOLO per-face mesh, beyond the only thing that can legitimately move it —
// re-discretizing the face's own boundary. A boundary point the absorbing mesher was entitled to drop
// lies within the chordal tolerance of the segment it splits, so restoring every such point moves the
// enclosed area by at most q.tol() × the face's boundary length. That bound is a length times a
// tolerance, so it scales with the model (ADR-0042), and it is nowhere near tight: every faithful
// adoption in the corpus sits ≥10 decades inside it while every defective one exceeded it by 70×–5000×.
func TestConformanceRepairNeverLosesFaceArea(t *testing.T) {
	t.Parallel()
	dir := CorpusFixtureDir()
	q := ops.PropertyQuality()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		assertNoFaceLostAreaInBodyContext(t, r, body, q)
	}
}

// assertNoFaceLostAreaInBodyContext compares every face's body-context mesh against its solo mesh.
func assertNoFaceLostAreaInBodyContext(t *testing.T, r Record, body *topo.Body, q ops.Quality) {
	t.Helper()
	facets := tessellate.CalculateBodyFacets(body, q)
	for i, f := range facets.Faces {
		solo := ops.MeshArea(tessellate.TessellateFace(f, q))
		got := ops.MeshArea(facets.FaceMeshes[i])
		slack := q.ChordTolerance * faceBoundaryPerimeter(f, q)
		if got >= solo-slack {
			continue
		}
		t.Errorf("%s/%s: face %d (%T) tiles %.6g in body context but %.6g solo — the conformance repair "+
			"destroyed %.6g of it (%.3f%%), %.4g× the %.4g a boundary re-discretization can move",
			r.Grid, r.Case, f.ID(), f.Geometry(), got, solo, solo-got, (solo-got)/solo*100, (solo-got)/slack, slack)
	}
}

// faceBoundaryPerimeter is the total length of f's tessellated boundary edges. Deliberately measured
// from the EDGES rather than through the kernel's own faceBoundaryLength, so the guard does not
// inherit a bug from the code it gates; a seam edge used twice by one loop counts twice, which only
// makes the bound more generous.
func faceBoundaryPerimeter(f *topo.Face, q ops.Quality) float64 {
	total := 0.0
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if e := u.Edge(); e != nil {
				total += polylineLength(tessellate.TessellateEdge(e, q))
			}
		}
	}
	return total
}

// polylineLength is the summed length of an OPEN polyline (an edge's tessellation).
func polylineLength(pts []math.Point3) float64 {
	total := 0.0
	for i := 1; i < len(pts); i++ {
		total += float64(pts[i-1].DistanceTo(pts[i]))
	}
	return total
}
