// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati/math"

// ProjectPointToSurface returns the parameters (u, v) of the point on s closest to p, plus the
// residual distance |s.PointAt(u,v) − p|. It builds on each surface's ParamAt — closed-form for the
// analytic surfaces, a grid-seeded Gauss–Newton inversion for NURBS (verified against a brute-force
// dense-grid closest point: M24 measured them equal to ~2 decimals). The residual is what import
// healing needs: it is the gap between an imported edge and the face's surface (the ~mm STEP
// authoring tolerance, M25), so callers can decide whether the edge must be snapped on.
//
// Example: u, v, gap := ProjectPointToSurface(face.Geometry(), edgePoint).
func ProjectPointToSurface(s Surface, p math.Point3) (u, v, dist float64) {
	u, v = s.ParamAt(p)
	return u, v, float64(s.PointAt(u, v).DistanceTo(p))
}

// ProjectCurveToSurface projects an ordered 3D polyline (an edge's discretization) onto s, returning
// its parameter-space curve — its PCURVE. It MARCHES: the first point is projected from scratch, then
// each subsequent point is seeded from the previous point's (u, v). Seeding keeps the pcurve on one
// smooth branch, so it does not self-intersect where independent per-point projection jitters near a
// near-fold (adjacent points snapping to different local minima — the imported-NURBS reality,
// ADR-0030). Analytic surfaces invert exactly and stably, so seeding is a no-op for them; it matters
// only for NURBS, where it is the prerequisite for a valid trim region in (u, v) (M25 F01).
//
// Example: pcurve := ProjectCurveToSurface(face.Geometry(), edgePolyline).
func ProjectCurveToSurface(s Surface, pts []math.Point3) []math.Point2 {
	out := make([]math.Point2, len(pts))
	if len(pts) == 0 {
		return out
	}
	u, v, _ := ProjectPointToSurface(s, pts[0])
	out[0] = math.P2(math.Scalar(u), math.Scalar(v))
	for i := 1; i < len(pts); i++ {
		u, v = seededParam(s, pts[i], u, v)
		out[i] = math.P2(math.Scalar(u), math.Scalar(v))
	}
	return out
}

// seededParam projects q onto s starting from the seed (u0, v0). For a NURBS surface this is the
// seeded Gauss–Newton march (BSplineSurface.ParamNear) that stays on one branch; the analytic
// surfaces invert exactly from any seed, so they delegate to their closed-form ParamAt.
func seededParam(s Surface, q math.Point3, u0, v0 float64) (u, v float64) {
	if bs, ok := s.(BSplineSurface); ok {
		return bs.ParamNear(q, u0, v0)
	}
	return s.ParamAt(q)
}
