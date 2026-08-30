// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Acceptance gate for the terminal ear-clip patch fallback (#1605, audit A9).
//
// boundaryPatchMesh is the last line of defense for hard trimmed curved faces — exactly the faces
// most likely to be degenerate — and ear clipping is only guaranteed for SIMPLE polygons (the
// two-ears theorem). On a self-touching or self-crossing trim it either breaks early (a literal
// hole in the body) or emits count-complete but OVERLAPPING triangles (a self-touching pentagon
// measured 3× its area). Both shipped silently. Every ear-clip result is now acceptance-checked,
// recovered once through the true CDT (#1604 — its split-at-vertex recovery handles precisely the
// self-touching case), and any residual defect ships LOUDLY as a diag.Defect on the mesh.

// CodePatchCoverage marks a boundary-patch triangulation that failed coverage acceptance — free
// (unshared) edges beyond the boundary's own, or a triangulated area that disagrees with the trim
// polygon's — after the CDT recovery tier also failed. The mesh still ships (a flagged partial
// covering beats a missing face) but the degradation is counted, never silent (#1605).
const CodePatchCoverage diag.Code = "tessellate.patch-coverage"

// patchCoverageGate runs the two acceptance signals on an ear-clipped patch and recovers through
// the constrained-Delaunay path when they fail, returning the triangles to ship and the acceptance
// verdict of the shipped set. pos2D is the boundary in the ear-clip's own projection plane (outer
// followed by the holes, the order the triangle indices address).
func patchCoverageGate(outer2D []math.Point2, holes2D [][]math.Point2, tris [][3]int) ([][3]int, bool) {
	if patchCovers(outer2D, holes2D, tris) {
		return tris, true
	}
	if ctris, ok := cdtPatchRecovery(outer2D, holes2D); ok && patchCovers(outer2D, holes2D, ctris) {
		return ctris, true
	}
	return tris, false
}

// patchCovers reports whether the 2D triangulation covers the trim polygon exactly: every
// triangle must be positively oriented (a negative one is folded against the covering — and would
// silently CANCEL an overlapping positive one in a signed sum, which is exactly how the
// self-touching pentagon's 3× cover measured "correct"), and the unsigned area sum must equal the
// loops' shoelace area (holes subtracted). The comparison happens in the projection plane where
// both sides are exact polygon arithmetic — a hole under-counts, an overlap over-counts, so the
// bracket is two-sided. The free-edge signal (weldedFreeEdgeCount) is applied by the caller on the
// final welded mesh, where duplicate boundary samples have merged.
func patchCovers(outer2D []math.Point2, holes2D [][]math.Point2, tris [][3]int) bool {
	pts := flattenLoops2D(outer2D, holes2D)
	want := stdmath.Abs(float64(signedArea(outer2D)))
	for _, h := range holes2D {
		want -= stdmath.Abs(float64(signedArea(h)))
	}
	got := 0.0
	for _, tr := range tris {
		if orient2d(pts[tr[0]], pts[tr[1]], pts[tr[2]]) < 0 {
			return false // inverted against the covering: folded or self-crossing input (exact sign)
		}
		got += triTwiceSignedArea(pts[tr[0]], pts[tr[1]], pts[tr[2]]) / 2
	}
	res := geom.ResolutionForPoints2D(outer2D)
	return stdmath.Abs(got-want) <= res.Area()
}

// cdtPatchRecovery retriangulates the boundary loops with the constrained Delaunay path — strictly
// more robust than ear clipping since #1604: a vertex exactly on another loop edge splits the
// constraint instead of defeating it, and coincident duplicate samples weld to representatives.
// ok is false when the CDT itself reports unrecovered constraints or a leak (a genuinely ambiguous
// self-crossing trim), so the caller keeps the ear-clip result and flags it.
func cdtPatchRecovery(outer2D []math.Point2, holes2D [][]math.Point2) ([][3]int, bool) {
	pts2 := flattenLoops2D(outer2D, holes2D)
	loops := make([][]int, 0, 1+len(holes2D))
	next := 0
	for _, n := range append([]int{len(outer2D)}, loopLens(holes2D)...) {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = next + i
		}
		loops = append(loops, idx)
		next += n
	}
	tris, unrecovered, fellBack := constrainedDelaunayChecked(pts2, loops)
	if len(tris) == 0 || len(unrecovered) > 0 || fellBack {
		return nil, false
	}
	return tris, true
}

// flattenLoops2D concatenates the outer loop and holes into the flat [2]float64 point slice the
// CDT and the coverage sum index (the same order the ear-clip triangle indices address).
func flattenLoops2D(outer2D []math.Point2, holes2D [][]math.Point2) [][2]float64 {
	pts := make([][2]float64, 0, len(outer2D))
	for _, p := range outer2D {
		pts = append(pts, [2]float64{float64(p.X), float64(p.Y)})
	}
	for _, h := range holes2D {
		for _, p := range h {
			pts = append(pts, [2]float64{float64(p.X), float64(p.Y)})
		}
	}
	return pts
}

// loopLens returns each hole loop's point count.
func loopLens(holes [][]math.Point2) []int {
	out := make([]int, len(holes))
	for i, h := range holes {
		out[i] = len(h)
	}
	return out
}

// diagnosePatchCoverage records the coverage Defect when the shipped patch failed acceptance and
// the CDT recovery could not repair it (#1605). For a boundary-only patch the 2D bracket subsumes
// hole detection — an interior hole is an area deficit, an overlap a surplus, a fold a negative
// triangle — so acceptance is the verdict; the welded free-edge count (the in-tree watertightness
// metric) is reported as the diagnostic's evidence. The mesh still ships — flagged, never silent.
func diagnosePatchCoverage(m *Mesh, accepted bool) {
	if accepted {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodePatchCoverage,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("boundary-patch triangulation failed coverage acceptance and CDT recovery (%d free edges after welding); "+
			"the trim boundary is degenerate or self-crossing and the face mesh may carry a hole or overlap", weldedFreeEdgeCount(m)),
	})
}
