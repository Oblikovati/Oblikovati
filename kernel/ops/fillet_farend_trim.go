// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Far-end trim of a fillet band against the wall it stops on (farend-runon-report.md).
//
// THE DEFECT this fixes. cornerTangents places a terminal corner's whole cross-section in the plane
// through the filleted edge's END VERTEX perpendicular to the edge axis (cen = v + offDir·r, then
// ta/tb = cen ± r·n). That flat cap is the right answer only when the STOP face — the third face meeting
// at that vertex, corner.endFace — is a plane through the vertex whose normal is the edge axis. Against a
// CURVED wall (B5/B1/N5's host cylinder, C4/B9/I3's cone, D7/D3/E2's sphere, F7's elliptic cylinder) or a
// plane OBLIQUE to the axis, the flat cap sits off the wall, so the band RUNS ON past where the solid
// actually ends and the wall's own loop then carries a cross-section arc (or a chord) that does not lie on
// the wall at all. B5's cap band reached x=±50 where the host cylinder ends at √(50²−10²) = 48.9898 —
// DRAWEXE's own vertex — leaving the recurring √(50²+10²)−50 = 0.990195 off-surface residual.
//
// THE TRIM. A constant-radius planar-edge band is a radius-r cylinder about the edge axis, so every one of
// its rulings is a straight line ALONG that axis. Sliding a section point along the axis therefore keeps
// it EXACTLY on the band, and the true far end is the locus of each section point slid until it meets the
// wall: an exact geometric trim of band ∩ wall, with no approximation in the point set. Grounded on
// DRAWEXE 8.0.0 per-face areas — the trimmed cap-band integral ∫(s_wall(φ) − s_corner)·r dφ gives
// B5 624.7357 (OCCT 624.736; the flat cap gives 628.3185), C4 485.7956 (OCCT 485.796),
// D7 1109.3345 (OCCT 1109.33).

// farEndTrimStations is the number of spans the trim curve is interpolated over. band ∩ wall is a quartic
// space curve for a cylinder band against a cylinder/cone/sphere wall, so it is carried as the cubic
// B-spline through these stations — the same "approximate the SSI" choice OCCT itself makes (DRAWEXE dumps
// B5's far-end edge as a degree-7 B-spline, not an analytic conic). 32 spans put the interpolation's own
// deviation at ~2e-6 of the fillet radius, three decades inside the on-surface invariant's
// 1e-6-of-diagonal budget, while every station is on both surfaces to machine precision.
const farEndTrimStations = 32

// farEndNewtonSteps polishes each station from geom.IntersectCurveSurface's bisection (≈1e-13 of the
// bracket) to machine precision on the wall, so the trimmed vertices reproduce the oracle's exact
// coordinates rather than a bisection residual.
const farEndNewtonSteps = 4

// trimBandEndsToWalls slides both of a constant-radius band's terminal section caps onto the walls they
// stop against, so the band ends where the solid does. A corner whose flat cap ALREADY lies on its stop
// face (every stop plane perpendicular to the edge axis — the overwhelming majority) is returned
// untouched, byte-for-byte, so this is invisible to every case that was already correct.
func trimBandEndsToWalls(c0, c1 *corner, in cornerInputs) {
	slide := bandAxialSpan(*c0, *c1, in.axis)
	if slide <= 0 {
		return
	}
	*c0 = trimTerminalSection(*c0, in, slide)
	*c1 = trimTerminalSection(*c1, in, slide)
}

// bandAxialSpan is the band's own extent along the edge axis — the cap on how far a section may slide
// before the trim would invert the band onto its other end.
func bandAxialSpan(c0, c1 corner, axis math.Vector3) float64 {
	return stdmath.Abs(c0.cen.VectorTo(c1.cen).Dot(axis))
}

// trimTerminalSection returns c with its section cap slid along the edge axis onto c.endFace, carrying the
// exact band∩wall curve in endCurve. It declines — returning c unchanged — for every corner whose end is
// not a plain wall stop, whose flat cap is already on the wall, whose wall has no extendable implicit form,
// or whose stations do not all land within the band's own axial span.
func trimTerminalSection(c corner, in cornerInputs, maxSlide float64) corner {
	if !plainWallStop(c) {
		return c
	}
	wall := c.endFace.Geometry()
	if !extendableWall(wall) {
		return c
	}
	arc, err := geom.Arc3dByThreePoints(c.ta, c.mid, c.tb)
	if err != nil {
		return c
	}
	if !sectionLeavesWall(arc, wall, in.weld) {
		return c // already on the wall: keep the flat cap EXACTLY as solved
	}
	pts, ok := slideSectionOntoWall(arc, wall, in.axis, maxSlide)
	if !ok {
		return c
	}
	curve, err := geom.NewFittedBSplineCurve(pts)
	if err != nil {
		return c
	}
	c.ta, c.mid, c.tb = pts[0], pts[len(pts)/2], pts[len(pts)-1]
	c.endCurve = curve
	return c
}

// plainWallStop reports whether c is a terminal corner rounded by a flat cap on a stop FACE — the only
// corner kind this trim owns. A blend/miter/run-out end has no end face, and a variable fillet's chorded
// or conic section is emitted by the ruled-strip path, which carries its own end geometry.
func plainWallStop(c corner) bool {
	return c.endFace != nil && !c.blend && !c.miter && !c.runout && len(c.chords) == 0 && c.crossW == 0
}

// extendableWall reports whether a stop face's surface has an implicit form that extends past the face's
// own trim, which the slide needs: the analytic quadrics and the torus do. A fitted B-spline patch does
// NOT — geom.ClosestPointOnSurface clamps to its parametric box, so a slide off the patch would converge
// onto the patch BOUNDARY rather than the extended wall and silently shorten the band. Those stops keep
// the flat cap and stay in the measured off-surface debt.
func extendableWall(s geom.Surface) bool {
	switch s.(type) {
	case geom.Plane, geom.Cylinder, geom.Cone, geom.Sphere, geom.Torus, geom.EllipticalCylinder:
		return true
	}
	return false
}

// sectionLeavesWall reports whether the flat section cap misses the wall by more than the model-relative
// vertex-coincidence tolerance. This is the gate that makes the trim a no-op wherever the existing
// construction is already right: a stop plane perpendicular to the axis leaves the cap on the wall to
// float noise (~1e-15), decades under Weld = 1e-9·size, so those corners are returned bit-for-bit.
func sectionLeavesWall(arc geom.Arc3d, wall geom.Surface, weld float64) bool {
	lo, hi := arc.Domain()
	for i := 0; i <= farEndTrimStations; i++ {
		p := arc.PointAt(lo + (hi-lo)*float64(i)/float64(farEndTrimStations))
		if stdmath.Abs(geom.SignedDistanceToSurface(wall, p)) > weld {
			return true
		}
	}
	return false
}

// slideSectionOntoWall slides every station of the flat section cap along ±axis onto the wall, returning
// the trim curve's interpolation points. It fails when any station has no landing inside the band's axial
// span — the honest decline for a wall the band's rulings run parallel to or miss entirely.
func slideSectionOntoWall(arc geom.Arc3d, wall geom.Surface, axis math.Vector3, maxSlide float64) ([]math.Point3, bool) {
	lo, hi := arc.Domain()
	pts := make([]math.Point3, 0, farEndTrimStations+1)
	for i := 0; i <= farEndTrimStations; i++ {
		p := arc.PointAt(lo + (hi-lo)*float64(i)/float64(farEndTrimStations))
		q, ok := slideOntoWall(p, axis, wall, maxSlide)
		if !ok {
			return nil, false
		}
		pts = append(pts, q)
	}
	return pts, true
}

// slideOntoWall returns p slid along ±axis to the NEAREST landing on wall within maxSlide, polished to
// machine precision. Nearest is the right branch: the section point is off the wall by the run-on
// residual only, so the wall crossing that bounds the band is the closest one — a sphere wall's second,
// far-side crossing is a whole diameter away.
func slideOntoWall(p math.Point3, axis math.Vector3, wall geom.Surface, maxSlide float64) (math.Point3, bool) {
	seg := geom.NewLineSegment(p.TranslateBy(axis.Scale(-maxSlide)), p.TranslateBy(axis.Scale(maxSlide)))
	best, found := p, false
	for _, h := range geom.IntersectCurveSurface(seg, wall) {
		if !found || h.DistanceTo(p) < best.DistanceTo(p) {
			best, found = h, true
		}
	}
	if !found {
		return p, false
	}
	return polishOntoWall(best, axis, wall), true
}

// polishOntoWall Newton-refines q along axis onto the wall's zero set, using the wall's own normal for the
// derivative of the signed distance along the slide (d/dt = n·axis).
func polishOntoWall(q math.Point3, axis math.Vector3, wall geom.Surface) math.Point3 {
	for i := 0; i < farEndNewtonSteps; i++ {
		u, v, foot := geom.ClosestPointOnSurface(wall, q)
		d := foot.VectorTo(q).Dot(wall.NormalAt(u, v))
		slope := wall.NormalAt(u, v).Dot(axis)
		if stdmath.Abs(slope) < 1e-12 {
			return q
		}
		q = q.TranslateBy(axis.Scale(-d / slope))
	}
	return q
}
