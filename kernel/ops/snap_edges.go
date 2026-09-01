// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Imported B-rep edges (SolidWorks STEP, ADR-0030) sit up to several mm OFF their adjacent faces'
// surfaces: the authoring kernel stores each edge's 3D curve to its own tolerance, not exactly on
// either neighbour. The grid meshers (structuredGridMesh, gridPatchMesh) re-derive an analytic
// face's boundary as s.PointAt(s.ParamAt(edge)) — equal to the edge point only when the edge lies ON
// s. Off-surface it diverges, so the analytic face and its neighbour tessellate DIFFERENT boundary
// points and leave free edges (a non-watertight shell: wrong mass-properties, no boolean/STL).
// SnapEdgesToSurfaces projects each edge back onto its surfaces so the discretization lies on them
// and both faces mesh the SAME boundary (M25 PBI-324). It must run BEFORE ReconstructPcurves so the
// pcurves are reconstructed from the on-surface (snapped) polyline.
//
// Snapping is done with the surfaces' ParamAt inverse (geom.ProjectCurveToSurface), NOT the geometric
// foot, on purpose: the grid mesher reproduces s.PointAt(s.ParamAt(p)), and ParamAt∘PointAt is the
// identity on parameters, so a point snapped via ParamAt is a fixed point the grid reproduces exactly.

// SnapEdgesToSurfaces snaps every edge of an imported body onto its adjacent face surfaces (file doc).
func SnapEdgesToSurfaces(b *topo.Body, q Quality) {
	for _, e := range b.Edges() {
		snapEdge(e, q)
	}
}

// snapEdge replaces an edge's discretization with one lying on its adjacent surfaces, recording the
// off-surface gap as the edge tolerance. The WHOLE polyline (endpoints included) is snapped so it lies
// on the surface — a closed circle's seam point would otherwise kink back to its off-surface vertex.
// Welding the shared vertices that incident snapped edges now disagree on is PBI-325 (vertex merge).
//
// An edge bounding a B-spline face is left NATIVE: that face's mesher (nurbsPcurveMesh) triangulates in
// the surface's own metric (u,v) and needs its boundary ON the B-spline, so snapping the shared edge
// onto an analytic neighbour pulls the freeform boundary off its surface and folds the CDT (measured on
// EDF: free edges 69→75). Freeform watertightness is the NURBS-mesher's own problem (M25 F03 / PBI-330),
// not edge snapping — verified that imported B-spline edges already sit ~sub-µm on their surfaces.
func snapEdge(e *topo.Edge, q Quality) {
	surfs := adjacentSurfaces(e)
	if len(surfs) == 0 {
		return // a wireframe/dangling edge with no face: nothing to snap onto
	}
	if boundsBSplineFace(surfs) {
		e.SetSnappedCurve(nil, 0)
		return
	}
	raw := tessellate.SampleEdgeCurve(e, q)
	snapped, residual := reconcileOntoSurfaces(raw, surfs)
	if residual < snapResidualFloor {
		e.SetSnappedCurve(nil, 0) // already on its surfaces (a native/accurate edge): leave it untouched
		return
	}
	e.SetSnappedCurve(snapped, residual)
}

// boundsBSplineFace reports whether any adjacent surface is a B-spline (a freeform face whose mesher
// must keep its own on-surface boundary — see snapEdge).
func boundsBSplineFace(surfs []geom.Surface) bool {
	for _, s := range surfs {
		if _, isSpline := s.(geom.BSplineSurface); isSpline {
			return true
		}
	}
	return false
}

// snapResidualFloor is the off-surface gap below which an edge is considered already on its surfaces
// and is left native — so accurately-authored imports (OpenCASCADE) and modelled solids do not move,
// while the ~mm SolidWorks gap (ADR-0030) is well above it. Tied to the mesh weld tolerance.
const snapResidualFloor = 1e-6

// adjacentSurfaces returns the surfaces of the distinct faces the edge bounds (1 for a boundary edge,
// 2 for a manifold edge).
func adjacentSurfaces(e *topo.Edge) []geom.Surface {
	faces := e.Faces()
	out := make([]geom.Surface, len(faces))
	for i, f := range faces {
		out[i] = f.Geometry()
	}
	return out
}

// reconcileOntoSurfaces projects raw onto each adjacent surface and merges the projections into one
// polyline lying on the surface(s), returning it with the residual — the max disagreement between the
// surfaces (the import gap). A boundary edge (one surface) snaps onto it. A manifold edge (two)
// reconciles via mergeProjections; surfaces beyond the first two (rare, non-manifold) are ignored.
func reconcileOntoSurfaces(raw []math.Point3, surfs []geom.Surface) ([]math.Point3, float64) {
	if len(surfs) == 1 {
		on := onSurfacePolyline(surfs[0], raw)
		return on, maxDist(raw, on)
	}
	a, b := surfs[0], surfs[1]
	onA, onB := onSurfacePolyline(a, raw), onSurfacePolyline(b, raw)
	return mergeProjections(a, b, onA, onB), maxDist(onA, onB)
}

// mergeProjections combines the two on-surface polylines into one shared boundary. A GRID-meshed face
// (cylinder/cone/sphere/torus, re-sampled via ParamAt) only reproduces an ON-surface boundary, whereas
// a plane meshes its 3D boundary verbatim (earcut) — so a point on the grid neighbour is watertight on
// BOTH. With two grid faces the boundary must lie on both, so it converges to their intersection; with
// two planar faces either projection works and the midpoint minimises error. (B-spline-adjacent edges
// never reach here — snapEdge leaves them native so the freeform mesher keeps its own boundary.)
func mergeProjections(a, b geom.Surface, onA, onB []math.Point3) []math.Point3 {
	aGrid, bGrid := isGridMeshedSurface(a), isGridMeshedSurface(b)
	switch {
	case aGrid && !bGrid:
		return onA
	case bGrid && !aGrid:
		return onB
	case aGrid && bGrid:
		return intersectionPolyline(a, b, onA, onB)
	default:
		return midPolyline(onA, onB)
	}
}

// onSurfacePolyline projects an ordered polyline onto s, marching each point from the previous one's
// parameters so a NURBS projection stays on one smooth branch (geom.ProjectCurveToSurface), then
// lifts back with PointAt — the points the surface's own ParamAt/PointAt round-trip reproduces.
func onSurfacePolyline(s geom.Surface, pts []math.Point3) []math.Point3 {
	pc := geom.ProjectCurveToSurface(s, pts)
	out := make([]math.Point3, len(pts))
	for i, uv := range pc {
		out[i] = s.PointAt(float64(uv.X), float64(uv.Y))
	}
	return out
}

// intersectionPolyline reconciles two on-surface polylines onto the surfaces' intersection curve,
// point by point (see convergeToIntersection).
func intersectionPolyline(a, b geom.Surface, onA, onB []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(onA))
	for i := range onA {
		out[i] = convergeToIntersection(a, b, onA[i].Midpoint(onB[i]))
	}
	return out
}

// intersectIters is how many alternating projections drive a seed onto both surfaces; a handful
// suffices for the near-coincident imported case (the two projections start ~mm apart).
const intersectIters = 6

// convergeToIntersection alternately projects p onto a then b, driving it toward a point on BOTH
// surfaces (their intersection) — where two analytic grid-meshed faces each reproduce it.
func convergeToIntersection(a, b geom.Surface, p math.Point3) math.Point3 {
	for range intersectIters {
		ua, va := a.ParamAt(p)
		p = a.PointAt(ua, va)
		ub, vb := b.ParamAt(p)
		p = b.PointAt(ub, vb)
	}
	return p
}

// isGridMeshedSurface reports whether the curved-face mesher tessellates s over a (u,v) GRID, which it
// re-samples through s.PointAt(s.ParamAt(boundary)) — so the boundary is faithful only when it lies ON
// s. A plane (earcut) and a B-spline (pcurve mesher) instead mesh the 3D boundary verbatim, so they
// accept any shared polyline; only the grid surfaces constrain where the snap must land.
func isGridMeshedSurface(s geom.Surface) bool {
	switch s.(type) {
	case geom.Cylinder, geom.EllipticalCylinder, geom.Cone, geom.Sphere, geom.Torus:
		return true
	default:
		return false
	}
}

// midPolyline returns the point-wise midpoint of two equal-length polylines.
func midPolyline(a, b []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(a))
	for i := range a {
		out[i] = a[i].Midpoint(b[i])
	}
	return out
}

// maxDist returns the largest point-wise distance between two equal-length polylines.
func maxDist(a, b []math.Point3) float64 {
	var m float64
	for i := range a {
		if d := float64(a[i].DistanceTo(b[i])); d > m {
			m = d
		}
	}
	return m
}
