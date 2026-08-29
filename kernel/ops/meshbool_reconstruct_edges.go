// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Original-edge reuse for reconstruction (ADR-0056 Layer 2c). A split face's boundary
// run is, in a gluing boolean (union along coincident faces, the #2167 family), always
// a tessellation of an ORIGINAL edge of one operand — a rim circle, a chord line, a
// profile arc. Reusing that edge's exact analytic curve makes the rebuilt face's
// boundary EXACT and, crucially, identical to the untouched neighbour's copy of the
// same edge, so the stitch welds them and the solid closes. A run that matches no
// original edge is a genuine surface-surface intersection curve, handled by the SSI
// layer; here it fails the match so the caller declines to the faceted fallback.

// origEdge is one operand edge available for reuse: its analytic curve, endpoints, and
// whether it is a closed seam (a full circle).
type origEdge struct {
	curve  geom.Curve3
	p0, p1 math.Point3
	closed bool
}

// origEdgeCatalog holds every original edge of both operands.
type origEdgeCatalog struct {
	edges []origEdge
}

// catalogOriginalEdges collects the edges of both operand bodies for reuse.
func catalogOriginalEdges(bodies ...*topo.Body) *origEdgeCatalog {
	cat := &origEdgeCatalog{}
	for _, b := range bodies {
		for _, e := range b.Edges() {
			cat.edges = append(cat.edges, origEdge{
				curve:  e.Geometry(),
				p0:     e.StartVertex().Point(),
				p1:     e.EndVertex().Point(),
				closed: e.StartVertex() == e.EndVertex(),
			})
		}
	}
	return cat
}

// reconstructRun builds the analytic boundary edge a run traces. It first reuses an
// operand's ORIGINAL edge (exact, and identical to an untouched neighbour's copy, so
// the stitch welds them) and otherwise falls to the analytic surface-surface
// intersection of the two faces meeting along the run. It fails when neither is
// available (a run whose curve no closed-form path yields — left to the faceted
// fallback).
func reconstructRun(run meshbool.ArrangementRun, surface, neighbor geom.Surface, verts []meshbool.Point,
	cat *origEdgeCatalog, res geom.Resolution) (brep.ReconEdge, bool) {
	if e, ok := cat.matchRun(run, verts, res.Weld()); ok {
		return e, true
	}
	return intersectionRunEdge(run, surface, neighbor, verts, res)
}

// matchRun finds the original edge a boundary run traces and returns it as a ReconEdge
// oriented along the run. A closed run (its endpoints coincide) matches a closed seam
// edge; an open run matches an edge with the same endpoints. An interior sample must
// lie on the curve, so a circle and its chord (shared endpoints) are told apart.
func (cat *origEdgeCatalog) matchRun(run meshbool.ArrangementRun, verts []meshbool.Point, tol float64) (brep.ReconEdge, bool) {
	n := len(run.Verts)
	if n < 2 {
		return brep.ReconEdge{}, false
	}
	p0, pn := runPoint(verts, run, 0), runPoint(verts, run, n-1)
	mid := runPoint(verts, run, n/2)
	closed := p0.DistanceTo(pn) <= tol
	for _, oe := range cat.edges {
		if closed {
			if e, ok := oe.matchClosed(p0, mid, runPoint(verts, run, 1), tol); ok {
				return e, true
			}
			continue
		}
		if e, ok := oe.matchOpen(p0, pn, mid, tol); ok {
			return e, true
		}
		if e, ok := oe.matchSubArc(p0, mid, pn, tol); ok {
			return e, true
		}
		if e, ok := oe.matchSubSegment(p0, mid, pn, tol); ok {
			return e, true
		}
	}
	return brep.ReconEdge{}, false
}

// matchSubSegment reuses a straight original edge for an OPEN run that traces PART of it —
// the straight-line analogue of matchSubArc (#2247). When two coincident-OPPOSITE coplanar
// faces meet along a run (a lap-seam tab whose front face is coplanar with the flange's back
// face, opposite normals), that run lies on a real straight edge but its endpoints fall
// between the edge's own vertices, so matchOpen (which needs equal endpoints) misses it and
// the coincident-plane pair yields no surface-surface line for intersectionRunEdge to fit.
// Here the exact sub-segment between the run's endpoints is rebuilt on the original edge's
// line — identical to the neighbour face's copy of the same edge, so the stitch welds and no
// SSI is needed.
func (oe origEdge) matchSubSegment(p0, mid, pn math.Point3, tol float64) (brep.ReconEdge, bool) {
	if oe.closed {
		return brep.ReconEdge{}, false
	}
	if _, isSeg := oe.curve.(geom.LineSegment); !isSeg {
		return brep.ReconEdge{}, false
	}
	if !onCurve(oe.curve, p0, tol) || !onCurve(oe.curve, mid, tol) || !onCurve(oe.curve, pn, tol) {
		return brep.ReconEdge{}, false
	}
	return brep.ReconEdge{Curve: geom.NewLineSegment(p0, pn), T0: 0, T1: 1}, true
}

// matchSubArc reuses a CLOSED circle edge for an OPEN run that traces part of it — the
// case the two other matchers miss: when a full rim circle is split into sub-arcs by a
// coincident-surface merge (a cocylindrical wall's minor-arc top borders the exposed cap
// on the SAME rim circle), the run's endpoints fall between the circle's own vertices, so
// they match no original edge's endpoints. Here the exact sub-arc is rebuilt on the circle
// through the run's three sample points (start, mid, end), keeping the boundary analytic
// and coincident with the neighbour face's copy of the same rim.
func (oe origEdge) matchSubArc(p0, mid, pn math.Point3, tol float64) (brep.ReconEdge, bool) {
	if !oe.closed {
		return brep.ReconEdge{}, false
	}
	if _, isCircle := oe.curve.(geom.Circle); !isCircle {
		return brep.ReconEdge{}, false
	}
	if !onCurve(oe.curve, p0, tol) || !onCurve(oe.curve, mid, tol) || !onCurve(oe.curve, pn, tol) {
		return brep.ReconEdge{}, false
	}
	arc, err := geom.Arc3dByThreePoints(p0, mid, pn)
	if err != nil {
		return brep.ReconEdge{}, false
	}
	return brep.ReconEdge{Curve: arc, T0: 0, T1: 1}, true
}

// matchOpen returns oe as a run-oriented ReconEdge when its endpoints equal {p0,pn} and
// mid lies on its curve.
func (oe origEdge) matchOpen(p0, pn, mid math.Point3, tol float64) (brep.ReconEdge, bool) {
	if oe.closed || !endpointsEqual(oe.p0, oe.p1, p0, pn, tol) || !onCurve(oe.curve, mid, tol) {
		return brep.ReconEdge{}, false
	}
	t0, _ := geom.CurveParamAtPoint3(oe.curve, p0)
	t1, _ := geom.CurveParamAtPoint3(oe.curve, pn)
	return brep.ReconEdge{Curve: oe.curve, T0: t0, T1: t1}, true
}

// matchClosed returns oe (a full closed edge) as a ReconEdge spanning its whole domain,
// swept in the run's direction (so the loop walks it the way the boundary does). second
// is the run's next vertex after p0, fixing the sweep sign against the curve tangent.
func (oe origEdge) matchClosed(p0, mid, second math.Point3, tol float64) (brep.ReconEdge, bool) {
	if !oe.closed || !onCurve(oe.curve, p0, tol) || !onCurve(oe.curve, mid, tol) {
		return brep.ReconEdge{}, false
	}
	lo, hi := oe.curve.Domain()
	t0, _ := geom.CurveParamAtPoint3(oe.curve, p0)
	if oe.curve.TangentAt(t0).Dot(p0.VectorTo(second)) >= 0 {
		return brep.ReconEdge{Curve: oe.curve, T0: lo, T1: hi}, true
	}
	return brep.ReconEdge{Curve: oe.curve, T0: hi, T1: lo}, true
}

// endpointsEqual reports whether the unordered pairs {a0,a1} and {b0,b1} coincide
// within tol.
func endpointsEqual(a0, a1, b0, b1 math.Point3, tol float64) bool {
	return (a0.DistanceTo(b0) <= tol && a1.DistanceTo(b1) <= tol) ||
		(a0.DistanceTo(b1) <= tol && a1.DistanceTo(b0) <= tol)
}

// onCurve reports whether p lies on curve within tol (via the nearest curve parameter).
func onCurve(c geom.Curve3, p math.Point3, tol float64) bool {
	t, _ := geom.CurveParamAtPoint3(c, p)
	return c.PointAt(t).DistanceTo(p) <= tol
}
