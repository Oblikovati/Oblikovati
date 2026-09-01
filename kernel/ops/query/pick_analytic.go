// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Analytic pick and nearest-geometry queries (M48/C3, Oblikovati/Oblikovati#3468–#3471). A
// pick → reference-key resolution must read the EXACT B-rep, never a face tessellation: a
// tessellated hit lands on a facet chord, inside the true surface by the sagitta, so the resolved
// point (and any distance gate on it) drifts with the display Quality. These helpers ray-cast and
// project against the analytic surface (geom.RaySurfaceHits / geom.ClosestPointOnSurface) and
// confirm the hit lands on THIS face's trimmed region with the exact parameter-space classifier
// (brep.PointInFaceTrim) — no TessellateFace, and identical at every Quality.

// analyticFaceRayHit returns the nearest FORWARD pierce of the ray (origin, dir) through face f's
// analytic surface that lies within f's trim: the parameter t along dir (so the pierce is
// origin + t·dir, matching the legacy mesh semantics where dir need not be unit) and the pierce
// point. RaySurfaceHits returns crossings ascending in ray parameter, so the first in-trim crossing
// is the nearest. Curved/NURBS surfaces resolve through the same primitive's numeric tracer — a
// derived computation of the analytic surface, not a tessellation read.
func analyticFaceRayHit(f *topo.Face, origin math.Point3, dir math.Vector3) (float64, math.Point3, bool) {
	l := float64(dir.Length())
	if l == 0 {
		return 0, math.Point3{}, false
	}
	box := geom.SurfaceRangeBox(f.Geometry())
	if _, ok := box.IntersectsRay(origin, dir); !ok {
		return 0, math.Point3{}, false // broad-phase: the ray cannot reach this face's surface (no narrow-phase alloc)
	}
	line, err := geom.NewLine(origin, dir)
	if err != nil {
		return 0, math.Point3{}, false
	}
	for _, h := range geom.RaySurfaceHits(f.Geometry(), line, boxRayReach(box, origin)) {
		if h.T > 0 && pierceOnFace(f, h.Point) {
			return h.T / l, h.Point, true // h.T is arc length (unit line dir); rescale to dir-parameter
		}
	}
	return 0, math.Point3{}, false
}

// pierceOnFace reports whether a ray's surface pierce p lies on face f: strictly inside its trim
// (brep.PointInFaceTrim), OR exactly on the trim boundary. The boundary case is a ray grazing a
// shared edge — the pierce lands on the edge between two faces (e.g. a ray travelling along a box's
// bottom-face plane strikes the adjacent side face precisely on their common edge). The strict
// even-odd interior test rejects a boundary point, but the solid IS entered there, so the mesh pick
// counted it via inclusive triangle edges; matching that keeps RayCastFaces' nearest-pierce identical
// to the mesh path (rib to-next depth, Oblikovati/Oblikovati#1882). A pierce within modelling
// tolerance of a boundary edge is treated as on the face.
func pierceOnFace(f *topo.Face, p math.Point3) bool {
	if brep.PointInFaceTrim(f, p) {
		return true
	}
	_, d := closestOnFaceBoundary(f, p, DefaultQuality())
	return d <= math.DefaultTolerance // tol:model — pierce coincident with the trim boundary edge
}

// boxRayReach bounds how far along the ray a hit inside the surface range box can lie. The box is the
// SURFACE's own range box (geom.SurfaceRangeBox), not the face's vertex/edge box, because a
// boundary-less closed face (a whole sphere or torus) has no vertices and so an empty vertex box. An
// unbounded surface (plane, cylinder, cone) yields ±Inf corners → +Inf, which the exact closed-form
// ray solver accepts; a finite surface (sphere, torus, NURBS) — the ones whose numeric tracer needs a
// finite tMax — yields a real bound: distance to the box centre plus its full space diagonal covers
// every point of the box with margin and scales with the model, so it introduces no absolute
// tolerance. A crossing beyond that bound cannot be on the face anyway.
func boxRayReach(box math.Box, origin math.Point3) float64 {
	if box.IsEmpty() {
		return 0
	}
	reach := float64(origin.DistanceTo(box.Center())) + float64(box.Diagonal().Length())
	if stdmath.IsNaN(reach) {
		return stdmath.Inf(1) // an unbounded surface box (±Inf corners) — the closed-form solver takes +Inf
	}
	return reach
}

// analyticClosestOnFace returns the closest point ON face f to p and its distance. It drops the
// perpendicular foot onto f's analytic surface (geom.ClosestPointOnSurface); when that foot lies
// within f's trim it is the closest face point, otherwise the closest point is on f's boundary and
// is found on the face's boundary edges (the same curve discretization closerEdge uses — a curve
// discretizer, not a face tessellation).
func analyticClosestOnFace(f *topo.Face, p math.Point3, q Quality) (math.Point3, float64) {
	_, _, foot := geom.ClosestPointOnSurface(f.Geometry(), p)
	if brep.PointInFaceTrim(f, foot) {
		return foot, float64(p.DistanceTo(foot))
	}
	return closestOnFaceBoundary(f, p, q)
}

// closestOnFaceBoundary returns the closest point to p on f's boundary edges and its distance —
// the nearest face point when the perpendicular foot falls outside the trim.
func closestOnFaceBoundary(f *topo.Face, p math.Point3, q Quality) (math.Point3, float64) {
	best := math.Point3{}
	bestD := stdmath.Inf(1)
	for _, e := range f.Edges() {
		pl := tessellate.DiscretizeEdge(e, q)
		for i := 0; i+1 < len(pl); i++ {
			cp := ClosestPointOnSegment(p, pl[i], pl[i+1])
			if d := float64(p.DistanceTo(cp)); d < bestD {
				best, bestD = cp, d
			}
		}
	}
	return best, bestD
}
