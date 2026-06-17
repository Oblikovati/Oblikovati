// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	gmath "oblikovati.org/math"
)

// Derived drawing views (M14-F02 PBI-140, #387): view kinds projected from a parent view
// rather than from a standard orientation. An auxiliary view folds the model about a line
// drawn on the parent and projects perpendicular to it, so an inclined face appears true-size.
// Section/detail/break build on the same parent-derived machinery in later increments.

// auxiliaryBasis derives an auxiliary view's projection frame from its parent frame and the
// fold-line angle (radians, measured from the parent's screen X axis). The fold line is an
// in-plane axis of the parent; the auxiliary looks perpendicular to it (rotating the parent's
// view direction 90° about the fold axis), with the fold line as the auxiliary's screen X.
//
// A fold angle of 0 (fold line along the parent's X) yields the downward projection (the top
// view of a front parent); π/2 yields the sideways projection — so it generalises the four
// discrete projected directions to any angle.
func auxiliaryBasis(parent hlr.View, angleRad float64, origin gmath.Point3) hlr.View {
	foldAxis := parent.XAxis.Scale(math.Cos(angleRad)).Add(parent.YAxis.Scale(math.Sin(angleRad)))
	viewDir := foldAxis.Cross(parent.ViewDir) // ⟂ fold line, rotated 90° from the parent's view dir
	// upHint = parent.ViewDir makes the fold axis the auxiliary's screen X (see hlr.NewView).
	return hlr.NewView(origin, viewDir, parent.ViewDir)
}

// sectionLine is a section view's cut line on its parent, in sheet millimetres.
type sectionLine struct{ x1, y1, x2, y2 float64 }

// detailBoundary is a detail view's circular region. cx/cy/r are in the parent's projection
// space (model-2D, where the parent's curves are projected before placement) so the clip is
// independent of either view's placement; sheetCX/CY/R keep the parent-sheet-mm boundary the
// user drew, for the annotation and persistence.
type detailBoundary struct {
	cx, cy, r                float64 // parent model-2D (clip space)
	sheetCX, sheetCY, sheetR float64 // parent sheet mm (as drawn)
}

// detailBoundaryOf maps a circle drawn on the parent (sheet mm) into the parent's projection
// space, inverting the parent's placement (sheet = centre + model·scale·cmToMM).
func detailBoundaryOf(parent *DrawingView, sheetCX, sheetCY, sheetR float64) detailBoundary {
	s := cmToMM * parent.scale
	return detailBoundary{
		cx: (sheetCX - parent.centerX) / s, cy: (sheetCY - parent.centerY) / s, r: sheetR / s,
		sheetCX: sheetCX, sheetCY: sheetCY, sheetR: sheetR,
	}
}

// clipToCircle restricts segment a→b to the disk (cx, cy, r), returning the inside sub-segment;
// ok is false when the segment lies wholly outside. Solves |a + t·(b−a)|² = r² for the inside
// parameter interval ∩ [0, 1].
func clipToCircle(a, b gmath.Point2, cx, cy, r float64) (gmath.Point2, gmath.Point2, bool) {
	ax, ay := float64(a.X)-cx, float64(a.Y)-cy
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	aa := dx*dx + dy*dy
	cc := ax*ax + ay*ay - r*r
	if aa < 1e-18 { // a zero-length segment: in iff its point is inside
		return a, b, cc <= 0
	}
	bb := 2 * (ax*dx + ay*dy)
	disc := bb*bb - 4*aa*cc
	if disc <= 0 { // no real crossing: wholly inside iff an endpoint is inside
		return a, b, cc < 0
	}
	sq := math.Sqrt(disc)
	t0, t1 := maxf2((-bb-sq)/(2*aa), 0), minf2((-bb+sq)/(2*aa), 1)
	if t0 >= t1 {
		return a, b, false
	}
	return lerp2(a, dx, dy, t0), lerp2(a, dx, dy, t1), true
}

func lerp2(a gmath.Point2, dx, dy, t float64) gmath.Point2 {
	return gmath.P2(a.X+gmath.Scalar(dx*t), a.Y+gmath.Scalar(dy*t))
}

func maxf2(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf2(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// breakBand is a break view's removed band. g0/g1 bound it (g0<g1) along the break axis in the
// parent's projection space; sheetG0/G1 keep the parent-sheet-mm band the user gave.
type breakBand struct {
	orientation      types.BreakOrientation
	g0, g1           float64 // parent model-2D, along the break axis
	sheetG0, sheetG1 float64 // parent sheet mm
}

// axisX reports whether the break compresses along the X axis (a horizontal break removes a
// vertical band, so it clips/shifts on X).
func (b breakBand) axisX() bool { return b.orientation == types.BreakHorizontal }

// breakBandOf maps a removed band given on the parent (sheet mm, along the break axis) into the
// parent's projection space, inverting the parent's placement.
func breakBandOf(parent *DrawingView, orientation types.BreakOrientation, gapStart, gapEnd float64) breakBand {
	s := cmToMM * parent.scale
	centre := parent.centerX
	if orientation == types.BreakVertical {
		centre = parent.centerY
	}
	g0, g1 := (gapStart-centre)/s, (gapEnd-centre)/s
	if g0 > g1 {
		g0, g1 = g1, g0
	}
	return breakBand{orientation: orientation, g0: g0, g1: g1, sheetG0: gapStart, sheetG1: gapEnd}
}

// recomputeBreak compresses the projected segments by removing the band and butting the two
// sides together, then draws a zigzag break line at the join. The far side is shifted toward
// the near side by the band width; segments inside the band are dropped.
func (v *DrawingView) recomputeBreak(segs []hlr.Segment) {
	wireframe := v.style == types.WireframeViewStyle
	axisX, g0, g1 := v.brk.axisX(), v.brk.g0, v.brk.g1
	w := g1 - g0
	pmin, pmax := perpBounds(segs, axisX)
	for _, s := range segs {
		vis := wireframe || s.Visible
		if a, b, ok := clipHalfAxis(s.A, s.B, axisX, g0, true); ok {
			v.appendCurve(a, b, vis, types.DrawingEdgeCurve, s.EdgeKey)
		}
		if a, b, ok := clipHalfAxis(s.A, s.B, axisX, g1, false); ok {
			v.appendCurve(shiftAxis(a, axisX, -w), shiftAxis(b, axisX, -w), vis, types.DrawingEdgeCurve, s.EdgeKey)
		}
	}
	for _, g := range breakGlyph(axisX, g0, pmin, pmax) {
		v.appendCurve(g[0], g[1], true, types.DrawingBreakCurve, nil)
	}
}

// clipHalfAxis keeps the part of a→b on one side of an axis-aligned line. With keepLE it keeps
// the portion whose axis coordinate is ≤ limit; otherwise ≥ limit. axisX selects which
// coordinate is the axis. ok is false when nothing is kept.
func clipHalfAxis(a, b gmath.Point2, axisX bool, limit float64, keepLE bool) (gmath.Point2, gmath.Point2, bool) {
	sa, sb := axisCoord(a, axisX), axisCoord(b, axisX)
	in := func(s float64) bool {
		if keepLE {
			return s <= limit
		}
		return s >= limit
	}
	ina, inb := in(sa), in(sb)
	if ina && inb {
		return a, b, true
	}
	if !ina && !inb {
		return a, b, false
	}
	t := (limit - sa) / (sb - sa)
	cross := lerp2(a, float64(b.X-a.X), float64(b.Y-a.Y), t)
	if ina {
		return a, cross, true
	}
	return cross, b, true
}

func axisCoord(p gmath.Point2, axisX bool) float64 {
	if axisX {
		return float64(p.X)
	}
	return float64(p.Y)
}

func shiftAxis(p gmath.Point2, axisX bool, delta float64) gmath.Point2 {
	if axisX {
		return gmath.P2(p.X+gmath.Scalar(delta), p.Y)
	}
	return gmath.P2(p.X, p.Y+gmath.Scalar(delta))
}

// perpBounds returns the min/max of the coordinate perpendicular to the break axis, over all
// segment endpoints — the extent the break-line glyph spans.
func perpBounds(segs []hlr.Segment, axisX bool) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, s := range segs {
		for _, p := range [2]gmath.Point2{s.A, s.B} {
			c := axisCoord(p, !axisX)
			lo, hi = minf2(lo, c), maxf2(hi, c)
		}
	}
	return lo, hi
}

// breakGlyph builds the zigzag break line at the join (axis coordinate g0), spanning the
// perpendicular extent [pmin, pmax].
func breakGlyph(axisX bool, g0, pmin, pmax float64) [][2]gmath.Point2 {
	const segments = 6
	amp := 0.06 * (pmax - pmin)
	pts := make([]gmath.Point2, segments+1)
	for i := 0; i <= segments; i++ {
		perp := pmin + (pmax-pmin)*float64(i)/float64(segments)
		axis := g0
		if i%2 == 1 {
			axis = g0 + amp
		}
		pts[i] = axisPoint(axisX, axis, perp)
	}
	out := make([][2]gmath.Point2, 0, segments)
	for i := 0; i+1 < len(pts); i++ {
		out = append(out, [2]gmath.Point2{pts[i], pts[i+1]})
	}
	return out
}

// axisPoint builds a point from an axis coordinate and a perpendicular coordinate.
func axisPoint(axisX bool, axis, perp float64) gmath.Point2 {
	if axisX {
		return gmath.P2(gmath.Scalar(axis), gmath.Scalar(perp))
	}
	return gmath.P2(gmath.Scalar(perp), gmath.Scalar(axis))
}

// sectionBasis derives a section view's cut-plane frame from its parent frame and the cut line
// drawn on the parent. The line (sheet mm) is mapped back through the parent's placement into the
// parent's model plane; the cut plane contains that line and is parallel to the parent's view
// direction (it cuts straight into the part). The section looks along the plane normal — the
// retained half is the side the normal points into.
func sectionBasis(parent hlr.View, line sectionLine, scale, cx, cy float64, origin gmath.Point3) hlr.View {
	// Sheet mm → parent model 2D (cm), inverting place(): (mm - centre) / (scale·cmToMM).
	s := cmToMM * scale
	mx1, my1 := (line.x1-cx)/s, (line.y1-cy)/s
	mx2, my2 := (line.x2-cx)/s, (line.y2-cy)/s
	dirX, dirY := mx2-mx1, my2-my1
	// Cut-plane normal = the line's left normal (-dy, dx), mapped into the parent's plane.
	normal := parent.XAxis.Scale(-dirY).Add(parent.YAxis.Scale(dirX))
	midX, midY := (mx1+mx2)/2, (my1+my2)/2
	planeOrigin := origin.TranslateBy(parent.XAxis.Scale(midX).Add(parent.YAxis.Scale(midY)))
	return hlr.NewView(planeOrigin, normal, parent.ViewDir)
}
