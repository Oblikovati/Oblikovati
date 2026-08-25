// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// Analytic surface-surface intersection edges for reconstruction (ADR-0054 Layer 2c/3).
// When a split face's boundary run is not an operand's original edge, it is a NEW
// intersection curve where this face's surface meets the run's neighbour surface. For
// the closed-form pairs geom.IntersectSurfacesAnalytic solves (plane∩plane = line,
// plane∩cylinder = circle/ellipse/line, plane∩sphere = circle, plane∩cone = conic), we
// take the exact curve, choose the branch the run's vertices lie on, and restrict it to
// the run's extent. Pairs with no closed form fail here, leaving the caller on the
// faceted fallback until the numeric-SSI layer lands.

// intersectionRunEdge reconstructs a run as the analytic intersection curve of surface
// and neighbor, oriented along the run. It fails when either surface is nil, the pair
// has no closed-form intersection, or no branch fits the run's vertices.
func intersectionRunEdge(run meshbool.ArrangementRun, surface, neighbor geom.Surface,
	verts []meshbool.Point, res geom.Resolution) (brep.ReconEdge, bool) {
	if surface == nil || neighbor == nil {
		return brep.ReconEdge{}, false
	}
	curves, handled := geom.IntersectSurfacesAnalytic(surface, neighbor, res)
	if !handled || len(curves) == 0 {
		return brep.ReconEdge{}, false
	}
	if c, ok := branchThroughRun(curves, run, verts, runMatchTol(run, verts, res)); ok {
		return orientRunEdge(c, run, verts, res.Weld()), true
	}
	return brep.ReconEdge{}, false
}

// branchThroughRun picks the intersection curve the run lies on: the one minimising the
// worst distance from the run's sampled vertices, accepted only within tol (so a facet
// chord's deviation from a curved section is tolerated but a wrong branch is rejected).
func branchThroughRun(curves []geom.Curve3, run meshbool.ArrangementRun, verts []meshbool.Point, tol float64) (geom.Curve3, bool) {
	best, bestErr := geom.Curve3(nil), 0.0
	for _, c := range curves {
		e := worstRunDistance(c, run, verts)
		if best == nil || e < bestErr {
			best, bestErr = c, e
		}
	}
	return best, best != nil && bestErr <= tol
}

// worstRunDistance is the largest distance from any run vertex to curve c.
func worstRunDistance(c geom.Curve3, run meshbool.ArrangementRun, verts []meshbool.Point) float64 {
	worst := 0.0
	for _, vi := range run.Verts {
		p := verts[vi].Round()
		t, _ := geom.CurveParamAtPoint3(c, p)
		if d := c.PointAt(t).DistanceTo(p); d > worst {
			worst = d
		}
	}
	return worst
}

// runMatchTol tolerates a facet chord's deviation from a curved section: half the
// longest run segment (the chord's sagitta is well under this), floored at the weld
// tolerance so a straight run still matches tightly.
func runMatchTol(run meshbool.ArrangementRun, verts []meshbool.Point, res geom.Resolution) float64 {
	longest := 0.0
	for i := 1; i < len(run.Verts); i++ {
		if d := verts[run.Verts[i-1]].Round().DistanceTo(verts[run.Verts[i]].Round()); d > longest {
			longest = d
		}
	}
	if half := 0.5 * longest; half > res.Weld() {
		return half
	}
	return res.Weld()
}

// orientRunEdge restricts c to the run's extent, oriented so the loop walks it start to
// end. A closed run (endpoints coincide) spans the whole domain, swept in the run's
// direction; an open run runs between its endpoint parameters.
func orientRunEdge(c geom.Curve3, run meshbool.ArrangementRun, verts []meshbool.Point, tol float64) brep.ReconEdge {
	n := len(run.Verts)
	p0, pn := verts[run.Verts[0]].Round(), verts[run.Verts[n-1]].Round()
	if p0.DistanceTo(pn) <= tol {
		return closedRunEdge(c, p0, verts[run.Verts[1]].Round())
	}
	t0, _ := geom.CurveParamAtPoint3(c, p0)
	t1, _ := geom.CurveParamAtPoint3(c, pn)
	return brep.ReconEdge{Curve: c, T0: t0, T1: t1}
}

// closedRunEdge spans a closed curve's whole domain, swept in the run's direction (set
// by the run's first step against the curve tangent).
func closedRunEdge(c geom.Curve3, p0, second math.Point3) brep.ReconEdge {
	lo, hi := c.Domain()
	t0, _ := geom.CurveParamAtPoint3(c, p0)
	if c.TangentAt(t0).Dot(p0.VectorTo(second)) >= 0 {
		return brep.ReconEdge{Curve: c, T0: lo, T1: hi}
	}
	return brep.ReconEdge{Curve: c, T0: hi, T1: lo}
}
