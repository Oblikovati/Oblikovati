// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Frame assembly for the loop-framed ruled chart (ADR-0060): the closed-form incidences between the
// face's boundary edges, the imprint curves and the artificial seam, and the sampled segments that
// carry them into the arrangement with those incidences as shared vertices.

// frameCrossing is one incidence of a frame edge (loop, edge) with an imprint curve (imp ≥ 0) or with
// the seam ruling (imp < 0), in each curve's own parameter. The shared arrangement vertex is the frame
// edge's point at tEdge.
type frameCrossing struct {
	loop, edge int
	tEdge      float64
	imp        int
	tImp       float64
}

const seamIncidence = -1

// solveFrameCrossings intersects every frame edge with every imprint curve in closed form. ok=false when
// an imprint coincides with a frame edge (one section plane) — admits has already dropped those.
func (c *ruledFaceUV) solveFrameCrossings(imprint []geom.Curve3) ([]frameCrossing, bool) {
	var out []frameCrossing
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			for ii, imp := range imprint {
				lo, hi := imp.Domain()
				pts, coincident := geom.SectionCrossingCandidates(c.face.surface, e.curve, imp)
				if coincident {
					return nil, false
				}
				for _, p := range pts {
					tE, okE := c.paramWithin(e.curve, e.t0, e.t1, p)
					tI, okI := c.paramWithin(imp, lo, hi, p)
					if okE && okI {
						out = append(out, frameCrossing{loop: li, edge: ei, tEdge: tE, imp: ii, tImp: tI})
					}
				}
			}
		}
	}
	return out, true
}

// solveSeamCrossings intersects the placed seam ruling with every frame edge and every imprint curve.
// Rulings are skipped: the seam was placed clear of every ruling edge and of every imprint azimuth.
func (c *ruledFaceUV) solveSeamCrossings(imprint []geom.Curve3) []frameCrossing {
	seam := c.frame.Ruling(c.seamU)
	var out []frameCrossing
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			if t, ok := c.seamHit(seam, e.curve, e.t0, e.t1); ok {
				out = append(out, frameCrossing{loop: li, edge: ei, tEdge: t, imp: seamIncidence})
			}
		}
	}
	for ii, imp := range imprint {
		lo, hi := imp.Domain()
		if t, ok := c.seamHit(seam, imp, lo, hi); ok {
			out = append(out, frameCrossing{loop: seamIncidence, edge: ii, tEdge: t, imp: seamIncidence})
		}
	}
	return out
}

// seamHit is the parameter where a section curve meets the seam ruling, when it does within its span.
func (c *ruledFaceUV) seamHit(seam geom.Line, cv geom.Curve3, t0, t1 float64) (float64, bool) {
	if geom.IsStraightCurve(cv) {
		return 0, false
	}
	pts, _ := geom.SectionCrossingCandidates(c.face.surface, geom.NewLineSegment(seam.PointAt(0), seam.PointAt(1)), cv)
	for _, p := range pts {
		if t, ok := c.paramWithin(cv, t0, t1, p); ok {
			return t, true
		}
	}
	return 0, false
}

// paramWithin inverts a curve at a point and accepts it when the point lies on the curve within the
// sew tolerance and its parameter falls in [t0, t1] (either order; a whole closed curve accepts every
// parameter). A parameter a hair outside the span is clamped to the end it overshoots: the incidence
// is then a frame vertex, which the sampling emits anyway.
func (c *ruledFaceUV) paramWithin(cv geom.Curve3, t0, t1 float64, p math.Point3) (float64, bool) {
	t, ok := geom.CurveParamAt(cv, p)
	if !ok {
		return 0, false
	}
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	if dlo, dhi := cv.Domain(); geom.CurveIsClosed(cv) && hi-lo >= (dhi-dlo)-paramSlack(cv, dhi-dlo, c.res) {
		return t, float64(cv.PointAt(t).DistanceTo(p)) <= c.res.Sew()
	}
	slack := paramSlack(cv, hi-lo, c.res)
	if t < lo-slack || t > hi+slack {
		return 0, false
	}
	t = stdmath.Max(lo, stdmath.Min(hi, t))
	return t, float64(cv.PointAt(t).DistanceTo(p)) <= c.res.Sew()
}

// paramSlack converts the sew tolerance to a parameter slack through the curve's speed over the span.
func paramSlack(cv geom.Curve3, span float64, res geom.Resolution) float64 {
	speed := float64(cv.TangentAt(0.5 * span).Length())
	if speed <= 0 {
		return 0
	}
	return res.Sew() / speed
}

// frameSegments samples every frame edge in its traversal order with the imprint and seam incidences
// injected, tagged segPolygon so a boundary run re-emits the exact sub-edge.
func (c *ruledFaceUV) frameSegments(seamHits []frameCrossing) []uvSeg {
	var out []uvSeg
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			var inject, atSeam []float64
			for _, cr := range c.crossings {
				if cr.loop == li && cr.edge == ei {
					inject = append(inject, cr.tEdge)
				}
			}
			for _, cr := range seamHits {
				if cr.loop == li && cr.edge == ei {
					atSeam = append(atSeam, cr.tEdge)
				}
			}
			out = append(out, c.sampledPolyline(e.curve, e.t0, e.t1, inject, atSeam, segPolygon)...)
		}
	}
	return out
}

// imprintSegments samples every imprint curve over its domain with its frame and seam incidences injected.
func (c *ruledFaceUV) imprintSegments(imprint []geom.Curve3, seamHits []frameCrossing) []uvSeg {
	var out []uvSeg
	for ii, imp := range imprint {
		var inject, atSeam []float64
		for _, cr := range c.crossings {
			if cr.imp == ii {
				inject = append(inject, cr.tImp)
			}
		}
		for _, cr := range seamHits {
			if cr.loop == seamIncidence && cr.edge == ii {
				atSeam = append(atSeam, cr.tEdge)
			}
		}
		lo, hi := imp.Domain()
		out = append(out, c.sampledPolyline(imp, lo, hi, inject, atSeam, segImprint)...)
	}
	return out
}

// sampledPolyline samples one curve from t0 to t1 on the curve's own grid with the given parameters
// injected, unwrapping the azimuth along the walk so each segment is continuous, snapping every seam
// incidence to the seam exactly, then folding each segment into the [0, 2π] strip. Any segment that
// still straddles the seam (an incidence the solver did not see) is split there by interpolation.
func (c *ruledFaceUV) sampledPolyline(cv geom.Curve3, t0, t1 float64, inject, atSeam []float64, kind segKind) []uvSeg {
	params := injectedParams(cv, t0, t1, append(append([]float64{}, inject...), atSeam...))
	pts := make([]math.Point2, len(params))
	prevU := 0.0
	for i, t := range params {
		uv := c.paramOf(cv.PointAt(t))
		u := float64(uv.X)
		if i > 0 {
			u = unwrapAzimuthNear(prevU, u)
		}
		if containsParam(atSeam, t) {
			u = 2 * stdmath.Pi * stdmath.Round(u/(2*stdmath.Pi))
		}
		pts[i], prevU = math.P2(u, float64(uv.Y)), u
	}
	var out []uvSeg
	for i := 1; i < len(params); i++ {
		seg := foldIntoStrip(uvSeg{a: pts[i-1], b: pts[i], curve: cv, tA: params[i-1], tB: params[i], kind: kind})
		for _, s := range splitSeamCrossing(seg) {
			if s.a.DistanceTo(s.b) > arrTol {
				out = append(out, s)
			}
		}
	}
	return out
}

// injectedParams is the curve's own sampling grid over [t0, t1] with the extra parameters merged, in
// traversal order (reversed when t0 > t1).
func injectedParams(cv geom.Curve3, t0, t1 float64, extra []float64) []float64 {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	params := curveOwnParams(cv, lo, hi)
	for _, t := range extra {
		if t > lo && t < hi {
			params = append(params, t)
		}
	}
	params = sortedUniqueParams(params)
	if t0 > t1 {
		for i, j := 0, len(params)-1; i < j; i, j = i+1, j-1 {
			params[i], params[j] = params[j], params[i]
		}
	}
	return params
}

// containsParam reports whether t is one of the listed parameters.
func containsParam(ts []float64, t float64) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

// foldIntoStrip shifts an unwrapped segment by whole turns so it lies in the [0, 2π] strip; a segment
// that ends exactly on the seam keeps that end at 0 or 2π.
func foldIntoStrip(s uvSeg) uvSeg {
	twoPi := 2 * stdmath.Pi
	lo := stdmath.Min(float64(s.a.X), float64(s.b.X))
	shift := twoPi * stdmath.Floor(lo/twoPi)
	if stdmath.Max(float64(s.a.X), float64(s.b.X))-shift > twoPi {
		shift = twoPi * stdmath.Floor(stdmath.Max(float64(s.a.X), float64(s.b.X))/twoPi-1)
	}
	s.a = math.P2(float64(s.a.X)-shift, float64(s.a.Y))
	s.b = math.P2(float64(s.b.X)-shift, float64(s.b.Y))
	return s
}

// seamSegments emits the two seam verticals (u=0 and u=2π) over the frame's axial window, split at
// every frame and imprint incidence so the seam shares those vertices. The overrun past the window is
// artificial and bounds no real geometry.
func (c *ruledFaceUV) seamSegments(seamHits []frameCrossing) []uvSeg {
	pad := c.band.vMax - c.band.vMin
	vs := []float64{c.band.vMin - pad, c.band.vMax + pad}
	for _, cr := range seamHits {
		var cv geom.Curve3
		if cr.loop == seamIncidence {
			cv = c.crossingImprint(cr)
		} else {
			cv = c.face.loops[cr.loop].edges[cr.edge].curve
		}
		vs = append(vs, float64(c.paramOf(cv.PointAt(cr.tEdge)).Y))
	}
	sort.Float64s(vs)
	seam := geom.NewLineSegment(c.point3(0, vs[0]), c.point3(0, vs[len(vs)-1]))
	var out []uvSeg
	for i := 1; i < len(vs); i++ {
		if vs[i]-vs[i-1] <= arrTol {
			continue
		}
		for _, u := range []float64{0, 2 * stdmath.Pi} {
			out = append(out, uvSeg{a: math.P2(u, vs[i-1]), b: math.P2(u, vs[i]), curve: seam, tA: 0, tB: 1, kind: segSeam})
		}
	}
	return out
}

// crossingImprint returns the imprint curve a seam incidence refers to; assembleSegments stores the
// imprint on the chart for this lookup.
func (c *ruledFaceUV) crossingImprint(cr frameCrossing) geom.Curve3 { return c.imprint[cr.edge] }
