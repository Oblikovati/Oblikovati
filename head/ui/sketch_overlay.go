//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// partSketchOverlays renders the active part's finished, visible sketches in the 3D
// view (the one being edited is drawn by the in-sketch overlay instead), so a sketch
// stays visible after Finish Sketch and its profile can be clicked to extrude. Returns
// nil when there is no active part.
func partSketchOverlays(s *app.Session) []renderer.DrawItem {
	part := activePart(s)
	if part == nil {
		return nil
	}
	selected := selectedSketch(s)
	var items []renderer.DrawItem
	for i := 0; i < part.Sketches().Count(); i++ {
		sk := part.Sketches().Item(i)
		if !sk.Visible() || sk.IsEditing() || s.EditScopeHides(sk.Seq()) {
			continue
		}
		items = append(items, sketchOverlay(sk, allEntitiesWhenSelected(sk, selected), nil)...)
	}
	return items
}

// partSketchPoints draws the target glyph at every point of the active part's finished,
// visible sketches, so placed points stay visible after Finish Sketch (the in-sketch overlay
// draws them while editing). hWorld is the screen-constant half-size in model units.
func partSketchPoints(s *app.Session, hWorld float64) []renderer.DrawItem {
	part := activePart(s)
	if part == nil {
		return nil
	}
	var items []renderer.DrawItem
	for i := 0; i < part.Sketches().Count(); i++ {
		sk := part.Sketches().Item(i)
		if !sk.Visible() || sk.IsEditing() || s.EditScopeHides(sk.Seq()) {
			continue
		}
		if item, ok := pointsOverlay(sk.Plane(), sk, hWorld); ok {
			items = append(items, item)
		}
	}
	return items
}

// selectedSketch returns the whole sketch selected in the browser, or nil — the input for
// highlighting a finished sketch in the 3D view.
func selectedSketch(s *app.Session) *sketch.Sketch {
	if h, ok := s.Selection().First().(app.SketchHandle); ok {
		return h.Sketch
	}
	return nil
}

// allEntitiesWhenSelected returns a predicate that marks every entity of sk as selected
// when sk is the chosen sketch (so its curves draw cyan), or nil to draw it amber.
func allEntitiesWhenSelected(sk, selected *sketch.Sketch) func(sketch.Entity) bool {
	if sk != selected {
		return nil
	}
	return func(sketch.Entity) bool { return true }
}

// sketchOverlay turns the active sketch's geometry into wireframe line items drawn in
// the viewport, so the user sees what they draw in the sketch environment. Curves are
// sampled into polylines; everything is mapped from sketch 2D to model 3D through the
// sketch plane. Returns nil when there is nothing to draw.
func sketchOverlay(sk *sketch.Sketch, selected func(sketch.Entity) bool, candidate sketch.Entity) []renderer.DrawItem {
	if sk == nil {
		return nil
	}
	normal, sel, cand := sketchSegmentsFor(sk, selected, candidate)
	return sketchItems(normal, sel, cand)
}

func sketchSegmentsFor(sk *sketch.Sketch, selected func(sketch.Entity) bool, candidate sketch.Entity) (*segAccum, *segAccum, *segAccum) {
	normal, sel, cand := &segAccum{}, &segAccum{}, &segAccum{}
	plane := sk.Plane()
	// Selected/candidate geometry stays solid so picks read clearly; normal-state
	// geometry carries its line type (construction dashed, centerlines center
	// pattern, sketch override otherwise — issue #161).
	pick := func(e sketch.Entity) (*segAccum, []float64) {
		switch {
		case candidate != nil && e == candidate:
			return cand, nil // the geometry the active constraint tool would accept on hover
		case selected != nil && selected(e):
			return sel, nil
		default:
			return normal, app.SketchEntityPattern(sk, e)
		}
	}
	addLines(pick, plane, sk)
	addCircles(pick, plane, sk)
	addArcs(pick, plane, sk)
	addEllipses(pick, plane, sk)
	addSplines(pick, plane, sk)
	addBlockInstances(pick, plane, sk)
	return normal, sel, cand
}

// addBlockInstances draws every placed block instance's realized geometry —
// the definition's curves under the placement transform, nesting included
// (M06-F07, #622).
func addBlockInstances(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	blocks := sk.Blocks()
	for i := 0; i < blocks.InstanceCount(); i++ {
		inst := blocks.Item(i)
		acc, pat := pick(inst)
		for _, poly := range inst.ExpandedPolylines() {
			acc.patterned(plane, poly, false, pat)
		}
	}
}

func sketchItems(normal, sel, cand *segAccum) []renderer.DrawItem {
	var items []renderer.DrawItem
	items = appendGrid(items, normal, sketchColor)
	items = appendGrid(items, sel, sketchSelectedColor)
	items = appendGrid(items, cand, sketchCandidateColor)
	return items
}

// sketchSegments is the polyline resolution for sampling sketch curves.
const sketchSegments = 64

// accumFor is the per-entity accumulator + dash-pattern chooser passed to the add*
// helpers (a nil pattern draws solid).
type accumFor func(sketch.Entity) (*segAccum, []float64)

func addLines(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		acc, pat := pick(l)
		acc.patterned(plane, []math.Point2{l.A.Position(), l.B.Position()}, false, pat)
	}
}

func addCircles(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.Circles().Count(); i++ {
		c := sk.Circles().Item(i)
		acc, pat := pick(c)
		acc.patterned(plane, sampleArc(c.Center.Position(), c.Radius, 0, 2*stdmath.Pi), true, pat)
	}
}

func addArcs(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.Arcs().Count(); i++ {
		a := sk.Arcs().Item(i)
		c := a.Center.Position()
		start := angleOf(c, a.Start.Position())
		end := angleOf(c, a.End.Position())
		if a.CounterClockwise && end < start {
			end += 2 * stdmath.Pi
		}
		if !a.CounterClockwise && end > start {
			end -= 2 * stdmath.Pi
		}
		acc, pat := pick(a)
		acc.patterned(plane, sampleArc(c, a.Radius(), start, end), false, pat)
	}
}

func addEllipses(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.Ellipses().Count(); i++ {
		e := sk.Ellipses().Item(i)
		acc, pat := pick(e)
		acc.patterned(plane, sampleEllipse(e), true, pat)
	}
}

func addSplines(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.Splines().Count(); i++ {
		sp := sk.Splines().Item(i)
		pts := make([]math.Point2, sp.PointCount())
		for j, p := range sp.Points {
			pts[j] = p.Position()
		}
		acc, pat := pick(sp)
		acc.patterned(plane, pts, sp.Closed, pat)
	}
}

// sampleArc samples count+1 points of a circular arc from angle a0 to a1.
func sampleArc(center math.Point2, r, a0, a1 float64) []math.Point2 {
	pts := make([]math.Point2, sketchSegments+1)
	for i := range pts {
		a := a0 + (a1-a0)*float64(i)/float64(sketchSegments)
		pts[i] = math.P2(center.X+r*stdmath.Cos(a), center.Y+r*stdmath.Sin(a))
	}
	return pts
}

// sampleEllipse samples the ellipse's perimeter in its major/minor frame.
func sampleEllipse(e *sketch.Ellipse) []math.Point2 {
	c := e.Center.Position()
	ux, uy := unit(e.MajorAxis)
	pts := make([]math.Point2, sketchSegments)
	for i := range pts {
		a := 2 * stdmath.Pi * float64(i) / float64(sketchSegments)
		mx, my := e.MajorRadius*stdmath.Cos(a), e.MinorRadius*stdmath.Sin(a)
		pts[i] = math.P2(c.X+mx*ux-my*uy, c.Y+mx*uy+my*ux)
	}
	return pts
}

func angleOf(center, p math.Point2) float64 {
	return stdmath.Atan2(p.Y-center.Y, p.X-center.X)
}

// unit returns the normalized components of v (falling back to +X for a zero vector).
func unit(v math.Vector2) (float64, float64) {
	l := v.Length()
	if l < math.DefaultTolerance {
		return 1, 0
	}
	return v.X / l, v.Y / l
}

// segAccum accumulates indexed line segments (positions + endpoint index pairs).
type segAccum struct {
	pos []math.Point3
	idx []int
}

func (a *segAccum) add(p math.Point3) int {
	i := len(a.pos)
	a.pos = append(a.pos, p)
	return i
}

func (a *segAccum) seg(plane sketch.Plane, p, q math.Point2) {
	i, j := a.add(plane.ToModel(p)), a.add(plane.ToModel(q))
	a.idx = append(a.idx, i, j)
}

// addSegment adds a model-space line segment (3D, no plane mapping).
func (a *segAccum) addSegment(p, q math.Point3) {
	i, j := a.add(p), a.add(q)
	a.idx = append(a.idx, i, j)
}

// patterned adds a polyline either solid or split into its line-type dashes
// (issue #161); a nil/degenerate pattern falls back to the solid polyline.
func (a *segAccum) patterned(plane sketch.Plane, pts []math.Point2, closed bool, pattern []float64) {
	segs := linetype.DashPolyline(pts, closed, pattern)
	if segs == nil {
		a.polyline(plane, pts, closed)
		return
	}
	for _, s := range segs {
		a.seg(plane, s[0], s[1])
	}
}

// polyline adds a connected chain (optionally closed) of plane points.
func (a *segAccum) polyline(plane sketch.Plane, pts []math.Point2, closed bool) {
	if len(pts) < 2 {
		return
	}
	idxs := make([]int, len(pts))
	for i, p := range pts {
		idxs[i] = a.add(plane.ToModel(p))
	}
	for i := 0; i+1 < len(pts); i++ {
		a.idx = append(a.idx, idxs[i], idxs[i+1])
	}
	if closed {
		a.idx = append(a.idx, idxs[len(pts)-1], idxs[0])
	}
}
