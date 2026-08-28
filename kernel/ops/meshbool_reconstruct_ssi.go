// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// Analytic surface-surface intersection edges for reconstruction (ADR-0056 Layer 2c/3).
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
	c, ok := branchThroughRun(curves, run, verts, runMatchTol(run, verts, res))
	if !ok || !weldableSSICurve(c) {
		return brep.ReconEdge{}, false
	}
	return orientRunEdge(c, run, verts, res.Weld()), true
}

// weldableSSICurve reports whether a synthesized surface-surface intersection curve is one
// reconstruction can reuse and close watertight: a LINE (plane∩plane, plane∩cylinder along a
// ruling), a CIRCLE (cylinder/cone/sphere cut by a plane ⊥ its axis), or an ELLIPSE
// (cylinder∩tilted plane). geom.IntersectSurfacesAnalytic canonicalises the plane∩cylinder
// ellipse — the plane operand is always the first argument, so both incident faces synthesize
// the BIT-IDENTICAL EllipseFull — and the endpoints+midpoint weld key (curved_stitch.edgeKey)
// therefore fuses the two faces' copies into ONE topo.Edge (ADR-0056 Layer 4). The remaining
// closed forms with no consistent weld yet (cone∩plane parabola/hyperbola) still fall through
// to the exact faceted boolean.
func weldableSSICurve(c geom.Curve3) bool {
	switch c.(type) {
	case geom.Line, geom.Circle, geom.EllipseFull, geom.EllipticalArc:
		return true
	default:
		return false
	}
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
// direction; an open run runs between its endpoint parameters. For a PERIODIC curve (a
// closed ellipse) the two endpoints leave the arc ambiguous — the boundary is either the
// direct [t0,t1] interval or its wrapping complement — so the run's interior vertex fixes
// which arc, and thus the sweep sign and extent (ADR-0056 Layer 4; the analogue of
// matchSubArc's three-point arc for original circle edges).
func orientRunEdge(c geom.Curve3, run meshbool.ArrangementRun, verts []meshbool.Point, tol float64) brep.ReconEdge {
	n := len(run.Verts)
	p0, pn := verts[run.Verts[0]].Round(), verts[run.Verts[n-1]].Round()
	if p0.DistanceTo(pn) <= tol {
		return closedRunEdge(c, p0, verts[run.Verts[1]].Round())
	}
	t0, _ := geom.CurveParamAtPoint3(c, p0)
	t1, _ := geom.CurveParamAtPoint3(c, pn)
	if _, periodic := c.(geom.EllipseFull); periodic {
		t1 = ellipseArcEndThroughMid(c, t0, t1, verts[run.Verts[n/2]].Round())
	}
	return brep.ReconEdge{Curve: c, T0: t0, T1: t1}
}

// ellipseArcEndThroughMid returns the end parameter (a signed offset from t0, so the
// sweep sign is right) of the sub-arc of a closed ellipse that RUNS THROUGH the interior
// vertex mid. A closed conic on [0,1] restricted to two endpoints spans either the forward
// interval or its wrapping complement; the mid vertex disambiguates. Without this the raw
// [t0,t1] stores the wrong half (SlottedScrew's slanted bore exit removed too little).
func ellipseArcEndThroughMid(c geom.Curve3, t0, t1 float64, mid math.Point3) float64 {
	tm, _ := geom.CurveParamAtPoint3(c, mid)
	forward := frac01(t1 - t0) // increasing-param distance t0→t1
	toMid := frac01(tm - t0)   // increasing-param distance t0→mid
	if toMid <= forward {
		return t0 + forward // the forward sub-arc holds the mid vertex
	}
	return t0 - (1 - forward) // the boundary runs the wrapping complement (negative sweep)
}

// frac01 maps x to its fractional part in [0,1).
func frac01(x float64) float64 {
	x -= stdmath.Floor(x)
	if x < 0 {
		x++
	}
	return x
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
