// SPDX-License-Identifier: GPL-2.0-only

package pdf

import "oblikovati.org/kernel/exchange/drawing"

// subpathEntities converts one device-space subpath to drawing entities, in millimetres. A
// straight-only subpath is a single LwPolyline; a subpath containing a cubic Bézier is split
// into polyline runs and one degree-3 Spline per Bézier — consecutive entities share their
// endpoint coordinates, so the chain stays connected (the DWG/DXF importers emit per-entity
// too). A cubic Bézier maps exactly onto a 4-control-point clamped degree-3 NURBS, which is
// how the sketch converter rebuilds it (model/exchange add2DSpline).
func subpathEntities(sp subpath) []drawing.Entity {
	if !hasCurve(sp) {
		pts := linePoints(sp)
		if len(pts) < 2 {
			return nil
		}
		return []drawing.Entity{&drawing.LwPolyline{Points: pts, Closed: sp.closed}}
	}
	return mixedEntities(sp)
}

// hasCurve reports whether the subpath contains any Bézier segment.
func hasCurve(sp subpath) bool {
	for _, s := range sp.segs {
		if s.kind == segCurve {
			return true
		}
	}
	return false
}

// linePoints returns the polyline vertices (start plus each line end) in millimetres.
func linePoints(sp subpath) [][2]float64 {
	pts := make([][2]float64, 0, len(sp.segs)+1)
	pts = append(pts, mm2(sp.start))
	for _, s := range sp.segs {
		pts = append(pts, mm2(s.pts[0]))
	}
	return pts
}

// mixedEntities splits a curve-bearing subpath into polyline runs and per-Bézier splines. A
// closed subpath gets an explicit closing line so the closing edge is preserved.
func mixedEntities(sp subpath) []drawing.Entity {
	segs := sp.segs
	if sp.closed {
		segs = append(append([]segment{}, segs...), segment{kind: segLine, pts: []pdfPoint{sp.start}})
	}
	var out []drawing.Entity
	cur := sp.start
	run := [][2]float64{mm2(cur)}
	for _, s := range segs {
		if s.kind == segLine {
			cur = s.pts[0]
			run = append(run, mm2(cur))
			continue
		}
		out = appendPolyline(out, run)
		out = append(out, bezierSpline(cur, s.pts))
		cur = s.pts[2]
		run = [][2]float64{mm2(cur)}
	}
	return appendPolyline(out, run)
}

// appendPolyline appends an open LwPolyline for a run of two or more vertices, dropping a
// degenerate single-vertex run.
func appendPolyline(out []drawing.Entity, run [][2]float64) []drawing.Entity {
	if len(run) < 2 {
		return out
	}
	return append(out, &drawing.LwPolyline{Points: run, Closed: false})
}

// bezierSpline builds a degree-3 NURBS from a cubic Bézier's start point and its [c1, c2,
// end] control points (the four-control-point, clamped-knot form that is the Bézier exactly).
func bezierSpline(start pdfPoint, ctrl []pdfPoint) *drawing.Spline {
	return &drawing.Spline{
		Degree:        3,
		ControlPoints: [][3]float64{mm3(start), mm3(ctrl[0]), mm3(ctrl[1]), mm3(ctrl[2])},
	}
}

// mm2 converts a device point to a 2-D millimetre coordinate.
func mm2(p pdfPoint) [2]float64 { return [2]float64{toMM(p.x), toMM(p.y)} }

// mm3 converts a device point to a 3-D millimetre coordinate (Z = 0; PDF pages are planar).
func mm3(p pdfPoint) [3]float64 { return [3]float64{toMM(p.x), toMM(p.y), 0} }
