//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	stdmath "math"
	"strings"

	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/linetype"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// sketchOverlayCache memoises the finished-sketch wireframe. Sampling every curve
// of every finished sketch into line segments each frame makes a dense sketch — a
// DWG import can be hundreds of thousands of segments — redraw at a crawl. The
// overlay geometry is camera-independent (it is mapped through the sketch plane
// into model space), so it holds across camera moves and rebuilds only when the
// key changes. The render loop is single-threaded, so a package-level cache is safe.
var sketchOverlayCache struct {
	key   string
	items []renderer.DrawItem
	// bounds is the model-space extent of the cached line work, computed once per rebuild so
	// the viewport's adaptive far plane can enclose a sketch-only drawing (DWG/DXF imports are
	// all OnTop line primitives, which DrawListBounds excludes). hasBounds is false for an
	// empty overlay.
	boundsMin, boundsMax [3]float32
	hasBounds            bool
}

// cachedPartSketchOverlays returns the finished-sketch overlay, rebuilding it only
// when the sketch state changes (see sketchOverlayKey). The returned slice is a
// shallow copy so the caller's appends never mutate the cached list. When there is
// no part to key on it falls back to an uncached build.
func cachedPartSketchOverlays(s *app.Session) []renderer.DrawItem {
	key, ok := sketchOverlayKey(s)
	if !ok {
		return partSketchOverlays(s)
	}
	if key != sketchOverlayCache.key {
		sketchOverlayCache.key = key
		sketchOverlayCache.items = partSketchOverlays(s)
		sketchOverlayCache.boundsMin, sketchOverlayCache.boundsMax, sketchOverlayCache.hasBounds =
			drawItemsBounds(sketchOverlayCache.items)
	}
	return append([]renderer.DrawItem(nil), sketchOverlayCache.items...)
}

// cachedSketchOverlayBounds returns the model-space extent of the finished-sketch overlay
// computed at the last cache rebuild, so the viewport far plane encloses it without rescanning
// hundreds of thousands of line vertices every frame. ok is false when there is no line work.
func cachedSketchOverlayBounds() (min, max [3]float32, ok bool) {
	return sketchOverlayCache.boundsMin, sketchOverlayCache.boundsMax, sketchOverlayCache.hasBounds
}

// drawItemsBounds is the axis-aligned model-space box over every vertex of items (all
// primitives, including the OnTop lines a sketch overlay is made of). ok is false when items
// carry no positions.
func drawItemsBounds(items []renderer.DrawItem) (min, max [3]float32, ok bool) {
	min = [3]float32{stdmath.MaxFloat32, stdmath.MaxFloat32, stdmath.MaxFloat32}
	max = [3]float32{-stdmath.MaxFloat32, -stdmath.MaxFloat32, -stdmath.MaxFloat32}
	for _, it := range items {
		for _, p := range it.Positions {
			min[0], max[0] = minF32(min[0], float32(p.X)), maxF32(max[0], float32(p.X))
			min[1], max[1] = minF32(min[1], float32(p.Y)), maxF32(max[1], float32(p.Y))
			min[2], max[2] = minF32(min[2], float32(p.Z)), maxF32(max[2], float32(p.Z))
		}
	}
	return min, max, min[0] <= max[0]
}

// sketchOverlayKey identifies the finished-sketch overlay geometry. Sketch edits do
// not bump the model geometry version, so the key also folds in the selected sketch
// (it recolours) and each sketch's seq, visibility, edit state, edit-scope hiding
// and entity count — which together change on import (new sketch, new count), the
// edit cycle (IsEditing flips on Finish), and add/remove.
//
// It also folds in Show Format and each sketch's format revision, because recolouring an entity
// changes neither the geometry version nor the entity count: without them a finished sketch would
// keep its old colours until something unrelated happened to change the key (#2015).
func sketchOverlayKey(s *app.Session) (string, bool) {
	version, ok := activeModelGeometryVersion(s)
	if !ok {
		return "", false
	}
	part := activePart(s)
	if part == nil {
		return "", false
	}
	var b strings.Builder
	b.WriteString(version)
	fmt.Fprintf(&b, "|sel=%p|fmt=%t", selectedSketch(s), s.ShowFormat())
	for i := 0; i < part.Sketches().Count(); i++ {
		sk := part.Sketches().Item(i)
		fmt.Fprintf(&b, "|%d:%t%t%t:%d:%d", sk.Seq(), sk.Visible(), sk.IsEditing(),
			s.EditScopeHides(sk.Seq()), sketchEntityCount(sk), sk.FormatRevision())
	}
	return b.String(), true
}

// sketchEntityCount sums a sketch's drawable geometry collections (each Count is
// O(1)) for the overlay cache key.
func sketchEntityCount(sk *sketch.Sketch) int {
	return sk.Lines().Count() + sk.Arcs().Count() + sk.Circles().Count() +
		sk.Ellipses().Count() + sk.EllipticalArcs().Count() + sk.Splines().Count() + sk.Points().Count()
}

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
		items = append(items, sketchOverlay(sk, allEntitiesWhenSelected(sk, selected), nil, s.ShowFormat())...)
		items = append(items, projectedCurveOverlay(sk)...)
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
// suppress is the Show Format toggle: when set, per-entity overrides are ignored and every entity
// draws with default attributes (the button's documented, name-inverted behaviour).
func sketchOverlay(sk *sketch.Sketch, selected func(sketch.Entity) bool, candidate sketch.Entity, suppress bool) []renderer.DrawItem {
	if sk == nil {
		return nil
	}
	return sketchItems(sketchSegmentsFor(sk, selected, candidate, suppress))
}

// projectedCurveOverlay draws a sketch's projected reference curves — the lines projected from
// edges or datum geometry (axes, plane↔sketch intersections, #1262). They live in the entity
// list rather than the typed Lines/Arcs collections, so sketchOverlay misses them; this draws
// each as a polyline in the sketch colour. Returns nil when the sketch projects no curves.
func projectedCurveOverlay(sk *sketch.Sketch) []renderer.DrawItem {
	if sk == nil {
		return nil
	}
	plane := sk.Plane()
	acc := &segAccum{}
	for _, e := range sk.Entities() {
		if pc, ok := e.(*sketch.ProjectedCurve); ok {
			acc.polyline(plane, pc.Points(), false)
		}
	}
	return appendGrid(nil, acc, chromeTheme.sketchColor)
}

func sketchSegmentsFor(sk *sketch.Sketch, selected func(sketch.Entity) bool, candidate sketch.Entity, suppress bool) *sketchStyleBuckets {
	b := newSketchStyleBuckets()
	plane := sk.Plane()
	// Every entity keeps its line type in every state — construction dashed, centerlines the
	// centre pattern, per-entity or sketch-level override otherwise (#161). Selection used to
	// force geometry solid, which made a selected centerline indistinguishable from a selected
	// normal line: the dash pattern is what identifies it, so dropping it hid what was picked.
	// Colour still marks state, since the user must be able to see what is selected.
	pick := func(e sketch.Entity) (*segAccum, []float64) {
		style := app.SketchEntityStyle(sk, e, suppress)
		w := sketchStrokeWidth(style)
		switch {
		case candidate != nil && e == candidate: // what the active constraint tool would accept on hover
			return b.cand.forStroke(strokeKey{chromeTheme.sketchCandidateColor, w}), style.Pattern
		case selected != nil && selected(e):
			return b.sel.forStroke(strokeKey{chromeTheme.sketchSelectedColor, w}), style.Pattern
		default:
			return b.normal.forStroke(strokeKey{sketchEntityColor(style), w}), style.Pattern
		}
	}
	addLines(pick, plane, sk)
	addCircles(pick, plane, sk)
	addArcs(pick, plane, sk)
	addEllipses(pick, plane, sk)
	addEllipticalArcs(pick, plane, sk)
	addSplines(pick, plane, sk)
	addBlockInstances(pick, plane, sk)
	return b
}

// strokeKey is what makes two entities shareable in one draw item: same colour, same stroke width.
// A DrawItem carries a single colour and width, so anything that differs in either must be its own
// item.
type strokeKey struct {
	color [4]float32
	width float32
}

// strokeLane accumulates segments per stroke within one draw-order layer.
type strokeLane struct {
	byStroke map[strokeKey]*segAccum
	order    []strokeKey // first-seen, so the draw list is deterministic frame to frame
}

func newStrokeLane() strokeLane { return strokeLane{byStroke: map[strokeKey]*segAccum{}} }

// forStroke returns the accumulator for one colour+width, creating it on first use.
func (l *strokeLane) forStroke(k strokeKey) *segAccum {
	acc, ok := l.byStroke[k]
	if !ok {
		acc = &segAccum{}
		l.byStroke[k] = acc
		l.order = append(l.order, k)
	}
	return acc
}

// appendTo emits this lane's accumulators as draw items, in first-seen order.
func (l *strokeLane) appendTo(items []renderer.DrawItem) []renderer.DrawItem {
	for _, k := range l.order {
		items = appendStroke(items, l.byStroke[k], k.color, k.width)
	}
	return items
}

// sketchStyleBuckets groups a sketch's segments by stroke, in three draw-order layers. Normal-state
// geometry is bucketed by its resolved colour and line weight so per-entity overrides render
// (#2015 shipped the model, persistence and panel for these, but the overlay only ever asked for
// the dash pattern, so neither a colour nor a weight set in the Format panel reached the screen).
// Selection and hover are separate layers so they draw last and win the overlap; they take the
// state colour, which must override the entity's, but keep its WIDTH — a selected heavy line that
// snapped back to a hairline would look like the selection had changed the geometry.
type sketchStyleBuckets struct {
	normal, sel, cand strokeLane
}

func newSketchStyleBuckets() *sketchStyleBuckets {
	return &sketchStyleBuckets{normal: newStrokeLane(), sel: newStrokeLane(), cand: newStrokeLane()}
}

// sketchEntityColor is the colour an entity draws in: its per-entity override, or the theme's
// sketch colour when it inherits.
func sketchEntityColor(style app.EntityStyle) [4]float32 {
	if style.Color.IsOverride() {
		return style.Color.Rgba().Array()
	}
	return chromeTheme.sketchColor
}

const (
	// pixelsPerMillimetre converts a line weight to its on-screen stroke. A line weight is a PLOT
	// width, so it is shown at a fixed screen scale rather than scaled with zoom — the reference
	// CSS pixel density (96 dpi) puts a 0.5 mm weight at roughly 2 px, close to how the reference
	// application draws it.
	pixelsPerMillimetre = 96.0 / 25.4
	// maxSketchStrokePixels caps the stroke so a mis-typed or badly imported weight (a DWG layer
	// table can carry anything) cannot paint over the whole viewport.
	maxSketchStrokePixels = 16
)

// sketchStrokeWidth converts an entity's line weight in millimetres to the stroke width in pixels
// the viewport expands it to. 0 (inherit) stays 0, which keeps the entity on the hairline path.
func sketchStrokeWidth(style app.EntityStyle) float32 {
	if style.LineWeight <= 0 {
		return 0
	}
	if px := style.LineWeight * pixelsPerMillimetre; px < maxSketchStrokePixels {
		return float32(px)
	}
	return maxSketchStrokePixels
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

// sketchItems flattens the buckets into draw items: one per distinct stroke, normal-state geometry
// first so the selection and hover lanes draw over it.
func sketchItems(b *sketchStyleBuckets) []renderer.DrawItem {
	var items []renderer.DrawItem
	items = b.normal.appendTo(items)
	items = b.sel.appendTo(items)
	return b.cand.appendTo(items)
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

func addEllipticalArcs(pick accumFor, plane sketch.Plane, sk *sketch.Sketch) {
	for i := 0; i < sk.EllipticalArcs().Count(); i++ {
		e := sk.EllipticalArcs().Item(i)
		acc, pat := pick(e)
		acc.patterned(plane, sampleEllipticalArc(e), false, pat)
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

// sampleEllipticalArc samples an elliptical arc from StartAngle to EndAngle in its
// major/minor frame (an open polyline — endpoints inclusive, not wrapped closed).
func sampleEllipticalArc(e *sketch.EllipticalArc) []math.Point2 {
	c := e.Center.Position()
	ux, uy := unit(e.MajorAxis)
	start := float64(e.StartAngle)
	sweep := float64(e.EndAngle) - start
	pts := make([]math.Point2, sketchSegments+1)
	for i := range pts {
		a := start + sweep*float64(i)/float64(sketchSegments)
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
