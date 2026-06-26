// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// General (u,v) arrangement trim of a ruled side (Oblikovati#1405). The analytic walk in
// curved_halfspace_ruled_boundary.go reduces the kept region to a single-valued v-interval [lo(u),hi(u)]
// per azimuth — it reads the surface only through keptV/sectionV, so it cannot trim by a MULTI-valued or
// CLOSED-island imprint (the general curved∩curved case). This file replaces that walk with a parameter-
// space arrangement: project the imprint into the side's (u,v) band, subdivide the band into cells along
// the imprint, classify each cell kept/dropped by a material predicate, and re-emit the kept region's
// boundary as exact analytic edges. A plane cut becomes the special case where the imprint is one conic.
//
// The arrangement reuses the planar subdivision engine (Arrange, arrange2d.go) on the (u,v) point cloud;
// curve identity is carried on the sampled segments so the re-emitted boundary keeps exact arcs/conics
// rather than the sampled polyline. This file builds the (u,v) projection and the tagged imprint sampling;
// the cell classification and boundary re-emission follow in the same arrangement pipeline.

// paramOf returns the (u, v) = (azimuth, axial distance) of a 3D point on the ruled side — the inverse of
// point3. v is the signed distance along the axis from the v=0 base; the radial remainder gives the azimuth
// in [0, 2π) measured from the ref direction. A point off the surface projects to its nearest (u,v) (the
// axial/azimuth components are still well defined), which is what sampling an imprint that lies ON the
// surface to tolerance needs.
func (c ruledUV) paramOf(p math.Point3) math.Point2 {
	d := c.base.VectorTo(p)
	v := float64(d.Dot(c.axis))
	radial := d.Sub(c.axis.Scale(math.Scalar(v)))
	u := stdmath.Atan2(float64(radial.Dot(c.binor)), float64(radial.Dot(c.ref)))
	if u < 0 {
		u += 2 * stdmath.Pi
	}
	return math.P2(u, v)
}

// segKind tags what a (u,v) segment re-emits to when it bounds a kept cell: an imprint sub-arc (the cut),
// a band rim (constant v, a circle arc), or the azimuth SEAM ruling (u=0≡2π — an ARTIFICIAL edge that
// closes the parameter rectangle for the arrangement and dissolves wherever the kept region wraps it).
type segKind int

const (
	segImprint segKind = iota
	segRim
	segSeam
)

// uvSeg is one sampled segment of an imprint curve in the side's (u,v) band, tagged with the analytic
// curve it came from and the curve parameters at its endpoints, so the arrangement can re-emit the exact
// curve sub-arc (an ellipse/hyperbola/parabola arm) for a boundary chain rather than the sampled polyline.
type uvSeg struct {
	a, b   math.Point2 // (u,v) endpoints
	curve  geom.Curve3 // the source analytic curve this segment re-emits to
	tA, tB float64     // its parameters at a and b
	kind   segKind
}

// imprintSampleCount is the number of segments one analytic imprint curve is sampled into across its
// parameter domain. It is dense enough to resolve the curve's crossings with the rims and with other
// imprint arcs to the arrangement weld (1e-7) at part scale, while keeping the planar arrangement small.
const imprintSampleCount = 256

// sampleImprintUV samples an analytic imprint curve over [t0, t1] into tagged (u,v) segments on the side.
// Each sample inverts the 3D curve point to (u,v) via paramOf; consecutive samples become one uvSeg
// carrying the curve and its endpoint parameters, so a later boundary walk re-emits exact sub-arcs. The
// azimuth seam (u wrapping 2π↔0) is left for the arrangement to resolve — sampleImprintUV reports the raw
// branch [0,2π); seam unwrapping is applied when segments are assembled into the band.
func (c ruledUV) sampleImprintUV(curve geom.Curve3, t0, t1 float64) []uvSeg {
	segs := make([]uvSeg, 0, imprintSampleCount)
	prevT := t0
	prevP := c.paramOf(curve.PointAt(t0))
	for i := 1; i <= imprintSampleCount; i++ {
		t := t0 + (t1-t0)*float64(i)/imprintSampleCount
		p := c.paramOf(curve.PointAt(t))
		segs = append(segs, uvSeg{a: prevP, b: p, curve: curve, tA: prevT, tB: t, kind: segImprint})
		prevT, prevP = t, p
	}
	return segs
}

// unwrapAzimuthNear shifts x by whole turns (2π) so it lands within ±π of ref, turning a wrapped azimuth
// pair into a monotone run — so a segment's true u-extent (and whether it crosses the seam) is unambiguous.
func unwrapAzimuthNear(ref, x float64) float64 {
	for x-ref > stdmath.Pi {
		x -= 2 * stdmath.Pi
	}
	for x-ref < -stdmath.Pi {
		x += 2 * stdmath.Pi
	}
	return x
}

// splitSeamCrossing splits an imprint segment whose endpoints straddle the azimuth seam (the shorter arc
// between them crosses u=0≡2π) into two segments meeting AT the seam, so no segment spans the discontinuity
// — the arrangement sees a clean parameter rectangle. v and the curve parameter are interpolated to the
// seam crossing. A segment that does not straddle the seam is returned unchanged.
func splitSeamCrossing(s uvSeg) []uvSeg {
	ub := unwrapAzimuthNear(s.a.X, s.b.X)
	if ub >= 0 && ub <= 2*stdmath.Pi {
		return []uvSeg{s} // wholly inside the band, no seam crossing
	}
	seamU, otherU := 0.0, 2*stdmath.Pi
	if ub > 2*stdmath.Pi { // the run climbs past 2π: a → 2π, then 0 → b
		seamU, otherU = 2*stdmath.Pi, 0
	}
	f := (seamU - s.a.X) / (ub - s.a.X)
	vSeam := s.a.Y + f*(s.b.Y-s.a.Y)
	tSeam := s.tA + f*(s.tB-s.tA)
	return []uvSeg{
		{a: s.a, b: math.P2(seamU, vSeam), curve: s.curve, tA: s.tA, tB: tSeam, kind: s.kind},
		{a: math.P2(otherU, vSeam), b: s.b, curve: s.curve, tA: tSeam, tB: s.tB, kind: s.kind},
	}
}

// bandFrameSegments returns the four straight (u,v) edges that bound the parameter rectangle for the
// arrangement: the bottom rim (v=vMin) and top rim (v=vMax) — each a single horizontal segment in (u,v),
// tagged to its rim circle — and the two seam verticals (u=0 and u=2π), tagged to the shared seam ruling.
// The rims are genuine surface boundaries; the seam verticals are artificial and dissolve where the kept
// region wraps them (resolved when cells are merged across the seam).
func (c ruledUV) bandFrameSegments() []uvSeg {
	twoPi := 2 * stdmath.Pi
	seam := geom.NewLineSegment(c.point3(0, c.band.vMin), c.point3(0, c.band.vMax))
	return []uvSeg{
		{a: math.P2(0, c.band.vMin), b: math.P2(twoPi, c.band.vMin), curve: c.band.bottomCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMax), b: math.P2(twoPi, c.band.vMax), curve: c.band.topCirc, tA: 0, tB: 1, kind: segRim},
		{a: math.P2(0, c.band.vMin), b: math.P2(0, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
		{a: math.P2(twoPi, c.band.vMin), b: math.P2(twoPi, c.band.vMax), curve: seam, tA: 0, tB: 1, kind: segSeam},
	}
}

// assembleBandSegments builds the full tagged (u,v) segment set the arrangement subdivides: every imprint
// segment (seam-split so none spans the azimuth discontinuity) plus the rim+seam frame that closes the
// parameter rectangle. Degenerate segments (both endpoints welded by the seam split) are dropped.
func (c ruledUV) assembleBandSegments(imprint []uvSeg) []uvSeg {
	out := make([]uvSeg, 0, len(imprint)+4)
	for _, s := range imprint {
		for _, split := range splitSeamCrossing(s) {
			if split.a.DistanceTo(split.b) > arrTol {
				out = append(out, split)
			}
		}
	}
	return append(out, c.bandFrameSegments()...)
}
