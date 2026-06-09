// SPDX-License-Identifier: GPL-2.0-only

package topomap

import (
	stdmath "math"

	"oblikovati.org/kernel/exchange/step/geommap"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// buildEdge constructs the shared kernel Edge for an EDGE_CURVE, with its curve
// trimmed so PointAt over the curve's domain runs start→end. The trim already bakes
// in EDGE_CURVE.same_sense, so the downstream ORIENTED_EDGE orientation alone
// decides each use's Reversed flag.
func (a *assembler) buildEdge(edgeCurveID int) (*topo.Edge, error) {
	refs, err := a.readEdgeCurve(edgeCurveID)
	if err != nil {
		return nil, err
	}
	start, err := a.vertex(refs.startVertexID)
	if err != nil {
		return nil, err
	}
	end, err := a.vertex(refs.endVertexID)
	if err != nil {
		return nil, err
	}
	curve, err := a.trimmedCurve(refs, start.Point(), end.Point())
	if err != nil {
		return nil, err
	}
	return a.builder.AddEdge(curve, start, end, a.edgeLineage()), nil
}

// edgeLineage mints the next stable imported edge lineage.
func (a *assembler) edgeLineage() topo.Lineage {
	l := topo.NewLineage(topo.Tok(a.feat, "edge", a.nextE))
	a.nextE++
	return l
}

// trimmedCurve maps the EDGE_CURVE's curve and trims it to run start→end. A LINE
// becomes the start→end LineSegment; a CIRCLE becomes the Arc3d (or full Circle when
// the endpoints coincide); a B-spline is used as-is.
func (a *assembler) trimmedCurve(refs edgeCurveRefs, start, end math.Point3) (geom.Curve3, error) {
	mapped, err := geommap.Curve(a.g, refs.curveID, a.scale)
	if err != nil {
		return nil, err
	}
	switch mapped.Kind {
	case geommap.CurveLine:
		return geom.NewLineSegment(start, end), nil
	case geommap.CurveCircle:
		return circleEdge(mapped.Circle, start, end, refs.sameSense)
	case geommap.CurveEllipse:
		return ellipseEdge(mapped.Ellipse, start, end, refs.sameSense)
	case geommap.CurvePolyline:
		return mapped.Polyline, nil
	default:
		return mapped.BSpline, nil
	}
}

// ellipseEdge trims an ELLIPSE to the elliptical arc/full ellipse between two vertices,
// mirroring circleEdge: coincident endpoints (a seam) keep the full ellipse; otherwise it
// builds an EllipticalArc whose parametric sweep runs start→end in the sense same_sense implies.
func ellipseEdge(e geommap.EllipseParams, start, end math.Point3, sameSense bool) (geom.Curve3, error) {
	if start.DistanceTo(end) < seamTol {
		return geom.NewEllipseFull(e.Center, e.Normal, e.RefDir, e.Major, e.Minor)
	}
	startAng := ellipseAngle(e, start)
	ccw := positiveSweep(startAng, ellipseAngle(e, end))
	sweep := ccw
	if !sameSense {
		sweep = ccw - twoPi
	}
	return geom.NewEllipticalArc(e.Center, e.Normal, e.RefDir, e.Major, e.Minor, startAng, sweep)
}

// ellipseAngle returns the parametric angle of p on the ellipse (point = center +
// Major·cosθ·RefDir + Minor·sinθ·(Normal×RefDir)), the convention geom EllipticalArc uses.
func ellipseAngle(e geommap.EllipseParams, p math.Point3) float64 {
	ref := unit(e.RefDir)
	bi := unit(cross(e.Normal, e.RefDir))
	d := e.Center.VectorTo(p)
	ang := stdmath.Atan2(d.Dot(bi)/e.Minor, d.Dot(ref)/e.Major)
	if ang < 0 {
		ang += twoPi
	}
	return ang
}

// circleEdge trims a CIRCLE to the arc/circle between two vertices. Coincident
// endpoints (a seam vertex) keep the full circle; otherwise it builds an Arc3d whose
// sweep runs start→end in the sense the circle's frame (and same_sense) imply.
func circleEdge(c geommap.CircleParams, start, end math.Point3, sameSense bool) (geom.Curve3, error) {
	if start.DistanceTo(end) < seamTol {
		// A full-circle seam. Parametrize the circle FROM the seam vertex (RefDir aimed at
		// `start`), so PointAt(0) == the vertex and discretizeEdge's endpoint snap is a no-op.
		// A plain NewCircle uses an arbitrary RefDir, so PointAt(0) lands elsewhere and snapping
		// both endpoints to the seam vertex yields a self-touching ring — which earcut fills with
		// a stray wedge across the hole on the planar face that borders this loop.
		return geom.NewArc3d(c.Center, c.Normal, c.Center.VectorTo(start), c.Radius, 0, twoPi)
	}
	startAng := circleAngle(c, start)
	ccw := positiveSweep(startAng, circleAngle(c, end)) // CCW arc start→end
	// same_sense=true: edge runs in the curve's natural CCW direction → use the CCW
	// sweep. same_sense=false: edge runs CW → the same endpoints, swept the short way
	// the other direction (a negative sweep of the complementary arc).
	sweep := ccw
	if !sameSense {
		sweep = ccw - twoPi // negative CW sweep landing on the same end point
	}
	return geom.NewArc3d(c.Center, c.Normal, c.RefDir, c.Radius, startAng, sweep)
}

// seamTol is the distance under which a circle's two edge vertices are the same
// point (a full-circle seam); below it we keep the unbounded full Circle.
const seamTol = 1e-9

// twoPi is the full-circle angle.
const twoPi = 2 * stdmath.Pi

// circleAngle returns the angle of p about the circle, measured from RefDir toward
// Normal×RefDir (the same convention geom.Circle/Arc3d evaluate).
func circleAngle(c geommap.CircleParams, p math.Point3) float64 {
	ref := unit(c.RefDir)
	bi := unit(cross(c.Normal, c.RefDir))
	d := c.Center.VectorTo(p)
	ang := stdmath.Atan2(d.Dot(bi), d.Dot(ref))
	if ang < 0 {
		ang += twoPi
	}
	return ang
}

// positiveSweep returns the CCW sweep from a to b in [0, 2π).
func positiveSweep(a, b float64) float64 {
	s := b - a
	for s < 0 {
		s += twoPi
	}
	for s >= twoPi {
		s -= twoPi
	}
	return s
}

// unit returns v normalized (v assumed nonzero — directions are validated upstream).
func unit(v math.Vector3) math.Vector3 { return v.Scale(1 / v.Length()) }

// cross returns a×b.
func cross(a, b math.Vector3) math.Vector3 { return a.Cross(b) }
