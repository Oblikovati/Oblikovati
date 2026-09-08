// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// The figure-eight torus band's PER-FACE area gate (ADR-0061 stage 5, Task 7).
//
// torusFigureEightBox cuts a torus R=5, r=2 by the plane y=3, which touches its inner equator: the
// section is a figure eight and the two pieces are the torus split at it. Each piece keeps ONE torus
// face, so the two faces partition the torus's whole 394.784 mm² between them — the strongest oracle
// a single face can have, and one a body volume cannot give.
//
// It is the row that retired the spiric band loft. That mesher meshed the cut piece at 310.800 mm²
// against the analytic 283.100 (+9.8%) and the intersect piece at 215.177 against 111.684 (+92.7%):
// it covered the pinch twice, and the two faces summed to 525.98 where the whole torus is 394.78. The
// chart-driven mesher reads the region the face records and meshes 281.62 and 110.95 — the analytic
// value less a chord deficit — and the two sum to 392.57.
const (
	figureEightTorusArea = 4 * stdmath.Pi * stdmath.Pi * 5 * 2 // 394.78418, the whole torus
	figureEightBelowArea = 283.09969                           // ∫ over (5+2cos v)·sin u < 3, r(R+r cos v) du dv
	figureEightAboveArea = 111.68448                           // the complement; the two sum to the torus
)

// TestTheFigureEightTorusBandsPartitionTheTorus is the per-face gate: each piece's torus face must
// mesh its own share of the torus to within a chord deficit, at BOTH gate qualities, and the two
// shares must add up to the whole torus. A mesher that covers the pinch twice fails the sum even when
// each piece's body volume looks plausible.
func TestTheFigureEightTorusBandsPartitionTheTorus(t *testing.T) {
	t.Parallel()
	cut, intersect := figureEightPiece(t, ops.Cut), figureEightPiece(t, ops.Intersect)
	for _, gq := range figureEightQualities() {
		below, above := torusFaceArea(t, cut, gq.q), torusFaceArea(t, intersect, gq.q)
		assertChordDeficit(t, gq.name+" below y=3", below, figureEightBelowArea)
		assertChordDeficit(t, gq.name+" above y=3", above, figureEightAboveArea)
		assertChordDeficit(t, gq.name+" sum", below+above, figureEightTorusArea)
	}
}

// figureEightQualities is the pair of facetings every gate in this file runs at — a gate that only
// holds at one sampling measures the tessellation, not the geometry.
func figureEightQualities() []struct {
	name string
	q    ops.Quality
} {
	return []struct {
		name string
		q    ops.Quality
	}{
		{"default", ops.DefaultQuality()},
		{"property", ops.PropertyQuality()},
	}
}

// assertChordDeficit fails unless got is want less a chord deficit — under it, and by no more than 1%.
// A faceted mesh of a convex-in-section band is always SHORT of the analytic area; being over it means
// the mesher covered something twice.
func assertChordDeficit(t *testing.T, what string, got, want float64) {
	t.Helper()
	rel := (got - want) / want
	if rel > 0 || rel < -0.01 {
		t.Errorf("%s: torus band meshes %.5f mm², want %.5f less a chord deficit (rel %+.4f, want (-0.01, 0])",
			what, got, want, rel)
	}
}

// figureEightPiece builds one side of the figure-eight split.
func figureEightPiece(t *testing.T, op ops.PartFeatureOperation) *topo.Body {
	t.Helper()
	target, tool := torusFigureEightBox(t)
	res, err := ops.Boolean(op, target, tool)
	if err != nil {
		t.Fatalf("Boolean(%v) on the figure-eight fixture: %v", op, err)
	}
	return res
}

// torusFaceArea sums the mesh area of a piece's torus faces.
func torusFaceArea(t *testing.T, res *topo.Body, q ops.Quality) float64 {
	t.Helper()
	sum := 0.0
	for _, f := range res.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); isTorus {
			sum += tessellate.MeshGeometryProperties(tessellate.TessellateFace(f, q)).Area
		}
	}
	if sum == 0 {
		t.Fatal("the figure-eight piece has no torus face")
	}
	return sum
}

// figureEightTorusFaceCount is asserted so the area gate cannot pass by summing a different face
// inventory than the one it was measured on.
func TestTheFigureEightPiecesKeepOneTorusFaceEach(t *testing.T) {
	t.Parallel()
	for _, op := range []ops.PartFeatureOperation{ops.Cut, ops.Intersect} {
		if n := countTorusFaces(figureEightPiece(t, op)); n != 1 {
			t.Errorf("the figure-eight %v piece has %d torus faces, want 1 (the area gate sums them)", op, n)
		}
	}
}

// countTorusFaces counts a body's torus faces.
func countTorusFaces(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); isTorus {
			n++
		}
	}
	return n
}

// TestAnUnchartedPinchedBandIsRefusedAndSaidSo is the gate the ROUTING alone does not give.
//
// A charted spiric band never reaches the tube-wrapping loft — spiricTubeTrimOf hands it to the general
// chart-driven mesher — so the loft's own conditioning matters only for an UNCHARTED one. "No primitive
// boolean in the corpus builds an uncharted torus band" is an observation about today's corpus, not an
// invariant, and a band whose two boundaries MEET is the one shape a single sweep round the tube cannot
// describe: the loft covers the tangency twice (310.800 mm² where the analytic region is 283.100, and
// its two halves summing to 525.98 against a torus of 394.78).
//
// So the figure-eight's own face is stripped of its chart and driven through the router. The loft must
// refuse it, the mesh must not be the loft's, and the refusal must be NAMED — a diag.Defect the feature
// reply, the API and the UI carry — not a silent sweep.
func TestAnUnchartedPinchedBandIsRefusedAndSaidSo(t *testing.T) {
	t.Parallel()
	for _, op := range []ops.PartFeatureOperation{ops.Cut, ops.Intersect} {
		face := unchartedFigureEightTorusFace(t, op)
		mesh := tessellate.TessellateFace(face, ops.DefaultQuality())
		if !meshReportsIgnoredTrim(mesh) {
			t.Errorf("%v: an uncharted pinched band was meshed with no named decline: %v", op, mesh.Diagnostics)
		}
		// The decline must say WHICH shape was refused, not just that the whole domain was used: a reader
		// who cannot tell "nothing recognised it" from "the loft gave it up" cannot act on the report.
		if !meshDeclineNames(mesh, "boundaries MEET") {
			t.Errorf("%v: the decline does not name the pinch: %v", op, mesh.Diagnostics)
		}
		// The reported fallback is the surface's WHOLE domain — measured 392.571 mm², a chord deficit
		// under 394.784. The loft's own answers for these two pieces are 310.800 and 215.177, so
		// anything near either is the sweep this row exists to refuse.
		if area := tessellate.MeshGeometryProperties(mesh).Area; area < 0.95*figureEightTorusArea {
			t.Errorf("%v: the uncharted pinched band meshed %.5f mm² — the loft's own answer, not the "+
				"reported whole-domain fallback (%.5f less a chord deficit)", op, area, figureEightTorusArea)
		}
	}
}

// unchartedFigureEightTorusFace is the figure-eight piece's torus face with its chart removed, which is
// what a producer that recorded none would hand the tessellator.
func unchartedFigureEightTorusFace(t *testing.T, op ops.PartFeatureOperation) *topo.Face {
	t.Helper()
	for _, f := range figureEightPiece(t, op).Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); isTorus {
			f.SetChart(nil)
			return f
		}
	}
	t.Fatalf("the figure-eight %v piece has no torus face", op)
	return nil
}

// meshDeclineNames reports whether the discarded-trim defect's detail contains the given phrase.
func meshDeclineNames(m *tessellate.Mesh, phrase string) bool {
	for _, d := range m.Diagnostics {
		if d.Code == tessellate.CodeTrimIgnoredFullDomain && strings.Contains(d.Detail, phrase) {
			return true
		}
	}
	return false
}

// meshReportsIgnoredTrim reports whether a mesh carries the discarded-trim defect at Defect severity.
func meshReportsIgnoredTrim(m *tessellate.Mesh) bool {
	for _, d := range m.Diagnostics {
		if d.Code == tessellate.CodeTrimIgnoredFullDomain && d.Severity == diag.Defect {
			return true
		}
	}
	return false
}

// TestTheFigureEightPiecesMeshClosedAtBothQualities is the body-level gate the per-face area gate
// above cannot give: each piece's whole mesh is a closed surface and reports no tear, at BOTH facetings.
//
// At DefaultQuality the cut piece meshed with two free edges of degree FOUR — not a crack but a doubled
// surface: the torus chart mesh closed the material corner at the pinch with a triangle whose three
// vertices were all rim points on y=3, the lid's own tip triangle with the opposite normal. The chart
// mesher now splits every rim-only ear at the surface point under its centroid (chart_rim_ear.go), and
// this row holds it at the corpus's coarse faceting, where the ear appeared, as well as the fine one.
func TestTheFigureEightPiecesMeshClosedAtBothQualities(t *testing.T) {
	t.Parallel()
	for _, op := range []ops.PartFeatureOperation{ops.Cut, ops.Intersect} {
		piece := figureEightPiece(t, op)
		for _, gq := range figureEightQualities() {
			mesh, _ := tessellate.TessellateBody(piece, gq.q)
			free := tessellate.FreeEdgeCount(mesh)
			if free != 0 {
				t.Errorf("%s quality, %v piece: the body meshes with %d free edges, want 0", gq.name, op, free)
			}
			assertTearReportAgreesWithTheMesh(t, mesh, free)
		}
	}
}

// assertTearReportAgreesWithTheMesh is the same shape the merged-band gate asserts: a torn mesh carries
// CodeMeshNotWatertight as a Defect, and a watertight one carries none.
func assertTearReportAgreesWithTheMesh(t *testing.T, m *tessellate.Mesh, freeEdges int) {
	t.Helper()
	reported := false
	for _, d := range m.Diagnostics {
		reported = reported || (d.Code == tessellate.CodeMeshNotWatertight && d.Severity == diag.Defect)
	}
	if reported != (freeEdges > 0) {
		t.Errorf("the mesh has %d free edges and reports %q = %v; the two must agree",
			freeEdges, tessellate.CodeMeshNotWatertight, reported)
	}
}
