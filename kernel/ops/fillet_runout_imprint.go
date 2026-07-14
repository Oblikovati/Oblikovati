// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// imprintGrazeEps is the dimensionless grazing-tangency threshold for the circle∩band-line
// solve below: a discriminant smaller than (scale·imprintGrazeEps)² is treated as a tangent
// (or numerically indistinguishable-from-tangent) chord, not a genuine two-point crossing —
// the "Numerical pitfalls" tangency guard, mirrored from rimCrossings' weld-tolerance guard
// but expressed relative to the host's own size (ADR-0042) since this solve works in exact
// (unsampled) conic space where there is no polyline weld to reuse.
const imprintGrazeEps = 1e-6

// imprintCut is the exact circle∩band-line imprint solve result: the two crossing points in 3D
// (on the host plane) and the footprint sub-arc between them on the OUTBOARD side — the piece a
// later task merges into the host loop and trims the fillet against (plan
// docs/superpowers/plans/2026-07-14-curved-runout-imprint-fillet.md).
type imprintCut struct {
	pMinus, pPlus math.Point3
	arc           geom.Curve3
}

// solveImprint computes the exact crossing of im's footprint against the receded fillet band
// (reconstructed from im.nodes) and the outboard sub-arc between the crossings. It is scoped to
// a circular footprint for now — ellipse/b-spline conics are Tasks 7/9 and honest-reject here
// (ok=false), same as a tangential/grazing crossing.
//
// Example: a boss footprint circle centered at the origin (r=8) crossing the band at y=-4
// crosses at (±√48,−4); solveImprint returns those points and the ~300° arc that stays above
// the band (geom.Arc3d, PointAt/TangentAt/Domain).
func solveImprint(im runoutImprint, res Resolution) (imprintCut, bool) {
	circle, ok := im.footprintEdge.Geometry().(geom.Circle)
	if !ok {
		return imprintCut{}, false // ellipse/b-spline footprint: not this task's scope
	}
	if im.nodes[0].P.DistanceTo(im.nodes[1].P) <= res.Weld() {
		return imprintCut{}, false // nodes too close to fix a band direction from
	}
	band := bandLineFromNodes(im.nodes)
	center2 := im.flat(circle.Center)
	t0, t1, ok := lineCircleRoots(band, center2, circle.Radius, hostBoundingDiag(im.host))
	if !ok {
		return imprintCut{}, false
	}
	p0, p1 := im.back(band.origin.TranslateBy(band.dir.Scale(t0))), im.back(band.origin.TranslateBy(band.dir.Scale(t1)))
	return imprintCut{pMinus: p0, pPlus: p1, arc: outboardArc(circle, p0, p1)}, true
}

// bandLineFromNodes rebuilds the receded fillet band as a 2D line through both crossing nodes:
// each is a signedDist==0 point on that band by construction (Task 2's bandCrossings found
// them there), so the pair determines the exact same line without re-deriving it from
// ef/boundaryFromTangents — solveImprint only ever sees the packaged runoutImprint.
func bandLineFromNodes(nodes [2]crossing) boundaryLine2 {
	d := nodes[0].P.VectorTo(nodes[1].P)
	return boundaryLine2{origin: nodes[0].P, dir: d.Scale(1 / d.Length())}
}

// lineCircleRoots solves for the two line-parameter roots where the band line P(t)=origin+t·dir
// crosses the circle (center c, radius r): substituting P(t) into |P−c|²=r² gives
// t² + 2t·(dir·(origin−c)) + (|origin−c|²−r²) = 0. scale is the host's model-relative size
// (its vertex bounding-box diagonal, ADR-0042); a discriminant below (scale·imprintGrazeEps)²
// is a tangential/grazing chord, not a genuine crossing.
func lineCircleRoots(b boundaryLine2, c math.Point2, r, scale float64) (t0, t1 float64, ok bool) {
	w := c.VectorTo(b.origin) // origin − c
	bb := b.dir.Dot(w)
	cc := w.Dot(w) - r*r
	disc := bb*bb - cc
	eps := scale * imprintGrazeEps
	if disc < eps*eps {
		return 0, 0, false
	}
	s := stdmath.Sqrt(disc)
	return -bb - s, -bb + s, true
}

// hostBoundingDiag is host's characteristic model size (its vertex bounding-box diagonal),
// mirroring occtparity.boundingDiag / the body-scale pattern used elsewhere in kernel/ops — the
// grazing guard above must scale with the model, not a hard-coded constant.
func hostBoundingDiag(host *topo.Face) float64 {
	verts := host.Vertices()
	if len(verts) == 0 {
		return 1
	}
	lo, hi := verts[0].Point(), verts[0].Point()
	for _, v := range verts[1:] {
		lo, hi = minPoint3(lo, v.Point()), maxPoint3(hi, v.Point())
	}
	return lo.DistanceTo(hi)
}

// minPoint3 and maxPoint3 are the per-axis min/max of two points — the bounding-box extend step
// hostBoundingDiag folds over a face's vertices.
func minPoint3(a, b math.Point3) math.Point3 {
	return math.P3(stdmath.Min(a.X, b.X), stdmath.Min(a.Y, b.Y), stdmath.Min(a.Z, b.Z))
}

func maxPoint3(a, b math.Point3) math.Point3 {
	return math.P3(stdmath.Max(a.X, b.X), stdmath.Max(a.Y, b.Y), stdmath.Max(a.Z, b.Z))
}

// outboardArc builds the footprint circle's OUTBOARD sub-arc between p0 and p1 as an exact
// geom.Arc3d (no re-sampling). The runout-imprint trigger (fillet_runout_detect.go) only admits
// a footprint that genuinely DIPS into the band — a small cap clipped off one side — so the
// larger of the two arcs the chord cuts is always the one that stays outboard; picking by sweep
// magnitude avoids needing the (here-unavailable, see bandLineFromNodes) original signed-side
// convention.
func outboardArc(c geom.Circle, p0, p1 math.Point3) geom.Arc3d {
	binormal := c.Normal.Cross(c.RefDir)
	a0 := circleAngleOf(c, binormal, p0)
	a1 := circleAngleOf(c, binormal, p1)
	sweep := majorSweep(a1 - a0)
	arc, _ := geom.NewArc3d(c.Center, c.Normal.AsVector(), c.RefDir.AsVector(), c.Radius, a0, sweep)
	return arc
}

// circleAngleOf returns p's parameter angle on circle c — the inverse of geom's internal
// pointOnCircle — given c's precomputed binormal (Normal × RefDir).
func circleAngleOf(c geom.Circle, binormal math.Vector3, p math.Point3) float64 {
	d := c.Center.VectorTo(p)
	return stdmath.Atan2(d.Dot(binormal), d.Dot(c.RefDir.AsVector()))
}

// majorSweep normalizes a raw a1−a0 angle difference to the LARGER of the two arcs it could
// describe (magnitude > π), signed so StartAngle+SweepAngle lands exactly on a1 (mod 2π).
func majorSweep(raw float64) float64 {
	const twoPi = 2 * stdmath.Pi
	ccw := stdmath.Mod(raw, twoPi)
	if ccw < 0 {
		ccw += twoPi
	}
	if ccw > stdmath.Pi {
		return ccw
	}
	return ccw - twoPi
}
