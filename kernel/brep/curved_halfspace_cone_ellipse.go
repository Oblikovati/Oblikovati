// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone-side elliptic split (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). An OBLIQUE box face
// tilted steeper than the cone's generators cuts the frustum side in a closed ELLIPSE that encircles
// the axis once (geom.planeConeCurve returns it analytically). The kept side of the band is then a
// periodic cone band whose two rims are one full circle (the rim the cut leaves intact) and the
// ellipse — a seam-wrapping band exactly like a frustum side but with one circular and one elliptical
// rim. This slice handles the clean arrangement where the ellipse lies wholly BETWEEN the two rims
// (the cut does not clip a rim): the kept band is the rim on the plane's negative side plus the
// ellipse, and the planar elliptical lid (built by the general arrangement) caps it. A cut whose
// ellipse clips a rim — the section becomes an elliptical arc plus a rim arc — defers to CSG.

// coneSideEllipseSplit splits a full periodic frustum side by an oblique plane whose section is one
// ellipse wholly inside the band. It returns the kept seam-wrapping band (kept circle rim + ellipse
// rim) and the ellipse section (for the lid). Defers when the ellipse clips a rim or the rims do not
// straddle the plane.
func coneSideEllipseSplit(f curvedFace, curves []geom.Curve3, cone geom.Cone, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	el, ok := curves[0].(geom.EllipseFull)
	if !ok || len(curves) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	band, ok := coneSideBand(f, cone)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	switch ellipseBandPosition(el, cone, band) {
	case ellipseOutsideBand: // the infinite-cone section misses the frustum side: keep whole or drop it
		return coneSideWholeOrEmpty(f, signedDistance(rimPoint(band.bottomCirc), plane, n)), nil, nil
	case ellipseWithinBand:
		break
	default: // ellipseClipsRim — the section becomes an elliptical arc plus a rim arc: deferred to CSG
		return nil, nil, ErrUnsupportedHalfSpace
	}
	keepBottom, ok := keptEllipseRim(band, plane, n)
	if !ok {
		return nil, nil, ErrUnsupportedHalfSpace
	}
	loop, section := ellipseBandLoop(cone, el, band, keepBottom)
	kept := curvedFace{surface: cone, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	return []curvedFace{kept}, section, nil
}

// ellipseBandPos classifies where the section ellipse sits relative to the frustum's axial band.
type ellipseBandPos int

const (
	ellipseWithinBand  ellipseBandPos = iota // every point strictly between the two rims: a clean split
	ellipseOutsideBand                       // every point beyond a rim: the plane clears the frustum side
	ellipseClipsRim                          // some in, some out: the section clips a rim (deferred)
)

// ellipseBandPosition samples the ellipse and reports whether it lies wholly within, wholly outside, or
// straddling the band's apex-distance range [vMin, vMax].
func ellipseBandPosition(el geom.EllipseFull, cone geom.Cone, band coneSideBand_) ellipseBandPos {
	axis := cone.AxisDir.AsVector()
	in, out := 0, 0
	for i := 0; i < 24; i++ {
		v := float64(cone.Apex.VectorTo(el.PointAt(float64(i) / 24)).Dot(axis))
		if v <= band.vMin+cylinderAxisTol || v >= band.vMax-cylinderAxisTol {
			out++
		} else {
			in++
		}
	}
	if in == 0 {
		return ellipseOutsideBand
	}
	if out == 0 {
		return ellipseWithinBand
	}
	return ellipseClipsRim
}

// keptEllipseRim reports which rim lies on the plane's negative (kept) side — true for the bottom rim,
// false for the top. ok=false unless exactly one rim is kept (the ellipse must separate the two rims).
func keptEllipseRim(band coneSideBand_, plane geom.Plane, n math.Vector3) (keepBottom, ok bool) {
	bot := signedDistance(rimPoint(band.bottomCirc), plane, n) < 0
	top := signedDistance(rimPoint(band.topCirc), plane, n) < 0
	if bot == top {
		return false, false // both rims on the same side: the ellipse does not separate them
	}
	return bot, true
}

// rimPoint returns a representative point on a rim circle (its parameter-0 point).
func rimPoint(c geom.Circle) math.Point3 { return c.PointAt(0) }

// ellipseBandLoop builds the kept band's seam-wrapping loop and the ellipse section for the lid. The
// loop mirrors a frustum side (Fwd lower rim, Fwd seam, Rev upper rim, Rev seam) with one rim the kept
// circle and the other the ellipse. The kept circle is the SOURCE rim circle (so the band welds to the
// cap that shares it); the seam is the straight cone ruling at that circle's own seam azimuth, and the
// ellipse is re-anchored as an EllipticalArc starting at the same azimuth so the seam lies on the cone.
func ellipseBandLoop(cone geom.Cone, el geom.EllipseFull, band coneSideBand_, keepBottom bool) (loop, section []loopEdge) {
	circle := band.bottomCirc
	if !keepBottom {
		circle = band.topCirc
	}
	circSeam := circle.PointAt(0) // the source circle's natural seam point (so the band reuses this edge)
	uSeam := coneAngleOf(cone, cone.Apex.VectorTo(circSeam))
	ellipse := anchorEllipse(el, cone, uSeam)
	ellipseSeam := ellipse.PointAt(0)
	seam := loopEdge{curve: geom.NewLineSegment(circSeam, ellipseSeam), t0: 0, t1: 1}
	circEdge := loopEdge{curve: circle, t0: 0, t1: 1}
	ellipseFwd := loopEdge{curve: ellipse, t0: 0, t1: 1}
	ellipseRev := loopEdge{curve: ellipse, t0: 1, t1: 0}
	if keepBottom { // kept circle is the lower rim; the ellipse is above it
		return []loopEdge{circEdge, seam, ellipseRev, reverseEdge(seam)}, []loopEdge{ellipseFwd}
	}
	// kept circle is the upper rim; the ellipse is the lower rim
	return []loopEdge{ellipseFwd, seam, reverseEdge(circEdge), reverseEdge(seam)}, []loopEdge{ellipseRev}
}

// anchorEllipse re-expresses the full ellipse as an EllipticalArc whose parameter 0 sits at cone
// azimuth uSeam, full 2π sweep — so its seam point lies on the same cone ruling as the rim circle's
// seam and the connecting seam line lies on the cone.
func anchorEllipse(el geom.EllipseFull, cone geom.Cone, uSeam float64) geom.EllipticalArc {
	start := ellipseAngleAtAzimuth(el, cone, uSeam)
	return geom.EllipticalArc{
		Center: el.Center, Normal: el.Normal, MajorAxis: el.MajorAxis,
		MajorRadius: el.MajorRadius, MinorRadius: el.MinorRadius,
		StartAngle: start, SweepAngle: 2 * stdmath.Pi,
	}
}

// ellipseAngleAtAzimuth returns the ellipse angle (radians) whose point sits at cone azimuth uSeam.
// The ellipse encircles the axis once, so the azimuth offset az(a) crosses zero exactly once as a runs
// the ellipse — plus a π→−π WRAP half a turn away, which is NOT the seam. A coarse scan brackets the
// true zero crossing (a sign change with a small jump, distinguishing it from the wrap) and bisection
// pins it; the seam ruling then lies on the cone.
func ellipseAngleAtAzimuth(el geom.EllipseFull, cone geom.Cone, uSeam float64) float64 {
	az := func(a float64) float64 {
		p := el.PointAt(a / (2 * stdmath.Pi))
		return wrapToPi(coneAngleOf(cone, cone.Apex.VectorTo(p)) - uSeam) // SIGNED offset, crosses 0 at the seam
	}
	const steps = 720
	prev := az(0)
	for i := 1; i <= steps; i++ {
		a := 2 * stdmath.Pi * float64(i) / steps
		cur := az(a)
		if prev*cur <= 0 && stdmath.Abs(prev-cur) < stdmath.Pi { // a true zero crossing, not the π→−π wrap
			return bisectAzimuth(az, 2*stdmath.Pi*float64(i-1)/steps, a)
		}
		prev = cur
	}
	return 0
}

// wrapToPi reduces an angle to (−π, π].
func wrapToPi(d float64) float64 {
	for d > stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	for d <= -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	return d
}

// bisectAzimuth refines the ellipse angle where az crosses zero (the seam azimuth) to a tight tolerance.
func bisectAzimuth(az func(float64) float64, lo, hi float64) float64 {
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		if az(lo)*az(mid) <= 0 {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}

// allEllipses reports whether every imprint curve is a full ellipse — the oblique cone cut whose
// section encircles the axis, routed to coneSideEllipseSplit.
func allEllipses(curves []geom.Curve3) bool {
	for _, c := range curves {
		if _, ok := c.(geom.EllipseFull); !ok {
			return false
		}
	}
	return len(curves) > 0
}
