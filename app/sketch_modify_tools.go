// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// pickCollector gathers sketch-entity picks for the modify/pattern tools. It mirrors the
// ConstraintTool flow (tool-first: activate, then feed picks) but commits a model edit
// rather than a constraint.
type pickCollector struct {
	picks []sketch.Entity
	want  int
}

// take is the 3D-viewport pick route (Tool.Pick); PickSnap is the in-sketch route. Both
// funnel through takeEntity so the two entry points share the dedup / capacity rule.
func (c *pickCollector) take(sel Selectable) {
	if h, ok := sel.(SketchEntityHandle); ok {
		c.takeEntity(h.Entity)
	}
}

// PickSnap implements the entity-pick half of [SketchEntityTool]: inside a 2D sketch,
// Session.sketchEntityPointer delivers every entity click HERE, not through Pick. A modify
// tool that had only Pick therefore had its in-sketch clicks silently dropped, so Fillet/
// Chamfer/Offset/Mirror did nothing (#1799). The snap carries nothing a corner blend,
// offset, or mirror needs.
func (c *pickCollector) PickSnap(ent sketch.Entity, _ SnapResult) { c.takeEntity(ent) }

// takeEntity records one entity pick, ignoring a nil, a repeat of an already-picked entity
// (a corner needs two DISTINCT lines), and any pick past `want`.
func (c *pickCollector) takeEntity(ent sketch.Entity) {
	if ent == nil || len(c.picks) >= c.want {
		return
	}
	for _, p := range c.picks {
		if p == ent {
			return
		}
	}
	c.picks = append(c.picks, ent)
}

func (c *pickCollector) ready() bool             { return len(c.picks) == c.want }
func (c *pickCollector) reset()                  { c.picks = nil }
func (c *pickCollector) Picked() []sketch.Entity { return c.picks }

// The modify tools must satisfy SketchEntityTool so in-sketch clicks route to them (#1799).
var (
	_ SketchEntityTool = (*SketchFilletTool)(nil)
	_ SketchEntityTool = (*SketchOffsetTool)(nil)
	_ SketchEntityTool = (*SketchMirrorTool)(nil)
)

// SketchFilletTool rounds the corner between two picked lines with a tangent arc.
type SketchFilletTool struct {
	dialogTool
	pickCollector
	radius math.Scalar
}

// NewSketchFilletTool makes a fillet tool with the given default radius.
func NewSketchFilletTool(radius float64) *SketchFilletTool {
	return &SketchFilletTool{pickCollector: pickCollector{want: 2}, radius: math.Scalar(radius)}
}

func (t *SketchFilletTool) Name() string                  { return "Sketch Fillet" }
func (t *SketchFilletTool) Pick(_ *Session, s Selectable) { t.take(s) }

// Accepts highlights the lines a fillet can blend (the hover-candidate cue).
func (t *SketchFilletTool) Accepts(e sketch.Entity) bool { return entityKindIs(e, sketch.LineKind) }
func (t *SketchFilletTool) CanCommit() bool              { return t.ready() }
func (t *SketchFilletTool) AutoCommitOnPick() bool       { return true }
func (t *SketchFilletTool) Cancel(*Session)              { t.reset() }
func (t *SketchFilletTool) Prompt(*Session) string       { return "Pick two lines to fillet." }

// SetRadius sets the fillet radius.
func (t *SketchFilletTool) SetRadius(r float64) { t.radius = math.Scalar(r) }

// Params exposes the fillet radius to the generic property dialog.
func (t *SketchFilletTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Radius", &t.radius)}}
}

// Commit applies the fillet to the two picked lines.
func (t *SketchFilletTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("sketch fillet: no active sketch")
	}
	l1, l2, err := twoPickedLines(t.picks)
	if err != nil {
		return err
	}
	_, err = sk.AddFillet(l1, l2, t.radius)
	return err
}

// SketchOffsetTool offsets a picked curve — with Loop Select (Inventor's default) the whole
// connected loop, otherwise the single curve — by a distance, keeping the geometry analytic (lines,
// arcs and circles stay themselves). A positive distance offsets inward (shrinks a closed loop).
type SketchOffsetTool struct {
	dialogTool
	pickCollector
	distance        math.Scalar
	loopSelect      bool        // Inventor default: offset the whole connected loop, not just the picked curve
	constrainOffset bool        // Inventor default: constrain the offset associative to its source
	placement       math.Point2 // where the user clicked to place the offset (its side + distance)
	placed          bool
}

// NewSketchOffsetTool makes an offset tool with the given default distance and both Loop Select and
// Constrain Offset on, as Inventor defaults.
func NewSketchOffsetTool(distance float64) *SketchOffsetTool {
	return &SketchOffsetTool{
		pickCollector:   pickCollector{want: 1},
		distance:        math.Scalar(distance),
		loopSelect:      true,
		constrainOffset: true,
	}
}

func (t *SketchOffsetTool) Name() string                  { return "Offset" }
func (t *SketchOffsetTool) Pick(_ *Session, s Selectable) { t.take(s) }

// ClickAt drives Inventor's two-step flow: the first click selects the geometry to offset, then the
// user moves the cursor and clicks to place the offset copy — the placement click's side and its
// distance from the curve set the offset. Selecting first through ClickAt (rather than an ambient
// entity pick) is what lets the second, empty-space click become the placement.
func (t *SketchOffsetTool) ClickAt(s *Session, px, py float64) {
	if t.placed {
		return
	}
	if len(t.picks) == 0 {
		if ent, ok := s.pickSketchEntity(px, py); ok && t.Accepts(ent) {
			t.takeEntity(ent)
		}
		return
	}
	if p, ok := s.sketchClickPoint(px, py); ok {
		t.placement, t.placed = p, true
	}
}

// AutoCommits finishes the tool on the placement click — only then, so the first (selection) click
// does not commit. CanCommit stays true once a curve is selected so OK/Enter can finish with the
// default distance (the API path used by tests).
func (t *SketchOffsetTool) AutoCommits() bool { return t.placed }

// LoopSelect reports whether the whole connected loop is offset (Inventor's default); the right-click
// menu toggles it. With it off, only the picked curve is offset.
func (t *SketchOffsetTool) LoopSelect() bool      { return t.loopSelect }
func (t *SketchOffsetTool) SetLoopSelect(on bool) { t.loopSelect = on }
func (t *SketchOffsetTool) ToggleLoopSelect()     { t.loopSelect = !t.loopSelect }

// ConstrainOffset reports whether the offset is constrained associative to its source (Inventor's
// default — parallel lines, concentric arcs, joined corners); the right-click menu toggles it.
func (t *SketchOffsetTool) ConstrainOffset() bool      { return t.constrainOffset }
func (t *SketchOffsetTool) SetConstrainOffset(on bool) { t.constrainOffset = on }
func (t *SketchOffsetTool) ToggleConstrainOffset()     { t.constrainOffset = !t.constrainOffset }

// Accepts highlights the curves OffsetEntity handles: line, circle, arc, and a projected reference
// curve (a projected face perimeter or edge, offset as a polyline — #2158 follow-up). Uses the
// entity's Kind() capability via entityKindIs, not a type switch, per the sketch-entity seam (#1624).
func (t *SketchOffsetTool) Accepts(e sketch.Entity) bool {
	return entityKindIs(e, sketch.LineKind, sketch.CircleKind, sketch.ArcKind, sketch.ProjectedCurveKind)
}
func (t *SketchOffsetTool) CanCommit() bool { return t.ready() }
func (t *SketchOffsetTool) Cancel(*Session) { t.reset(); t.placed = false }
func (t *SketchOffsetTool) Prompt(*Session) string {
	if len(t.picks) == 0 {
		return "Select geometry to offset"
	}
	return "Move the cursor and click to place the offset"
}

// SetDistance sets the offset distance.
func (t *SketchOffsetTool) SetDistance(d float64) { t.distance = math.Scalar(d) }

// Params exposes the offset distance to the generic property dialog.
func (t *SketchOffsetTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{scalarParam("Distance", &t.distance)}}
}

// Commit offsets the picked geometry: with Loop Select the whole connected loop (analytic — arcs
// stay arcs), otherwise the single curve. When the user placed the offset (the second click) the
// placement's side and its distance to the curve set the signed distance directly (Inventor's
// cursor-driven placement); otherwise a positive default distance offsets inward whatever the loop's
// winding, so the API/test path stays predictable.
func (t *SketchOffsetTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("offset: no active sketch")
	}
	path, ok := sk.ConnectedChainFrom(t.picks[0])
	// A lone curve (no connected neighbours) offsets as a single entity even with Loop Select on —
	// OffsetConnectedLoop is for a multi-curve chain, and OffsetEntity keeps a single line/arc/circle
	// analytic. ConnectedChainFrom returns ok for ANY curve, so the count, not ok, gates the loop path.
	if !t.loopSelect || !ok || path.Count() <= 1 {
		off, err := sk.OffsetEntity(t.picks[0], math.Scalar(t.singleDistance()))
		if err != nil {
			return err
		}
		if t.constrainOffset {
			sk.ConstrainOffsetSingle(t.picks[0], off)
		}
		return nil
	}
	offsets, err := sk.OffsetConnectedLoop(path, t.loopDistance(path))
	if err != nil {
		return err
	}
	if t.constrainOffset {
		sk.ConstrainOffsetLoop(path, offsets)
	}
	return nil
}

// singleDistance is the signed offset for one curve: the placement's signed distance to the seed
// when placed, else the default distance (its own sign, no winding logic — a lone curve has no
// inside).
func (t *SketchOffsetTool) singleDistance() float64 {
	if t.placed {
		return placementSignedOffset(sketch.EntityOutline(t.picks[0]), t.placement)
	}
	return float64(t.distance)
}

// loopDistance is the signed offset for a loop: the placement's signed distance in the seed's
// left-normal frame (the frame OffsetConnectedLoop offsets along) when placed, else the default
// distance flipped so a positive value always shrinks the loop whatever its winding.
func (t *SketchOffsetTool) loopDistance(path *sketch.Path) float64 {
	if t.placed {
		return placementSignedOffset(sketch.EntityOutline(t.picks[0]), t.placement)
	}
	d := float64(t.distance)
	if signedLoopArea(path.Points()) < 0 {
		d = -d // a CW loop offsets inward with a negative d; flip so positive distance always shrinks
	}
	return d
}

// placementSignedOffset returns the offset distance the cursor placement implies: the magnitude is
// the placement's perpendicular distance to the curve, the sign its side (positive when the
// placement lies to the LEFT of the seed's natural traversal, matching offsetLine/OffsetConnectedLoop
// which offset by d along the left normal). This is Inventor's "move the cursor and click to place".
func placementSignedOffset(outline []math.Point2, p math.Point2) float64 {
	if len(outline) < 2 {
		return 0
	}
	best, signed := math.Scalar(-1), 0.0
	for i := 0; i+1 < len(outline); i++ {
		a, b := outline[i], outline[i+1]
		dist := p.DistanceTo(segmentClosestPoint(p, a, b))
		if best >= 0 && dist >= best {
			continue
		}
		best = dist
		cross := (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X) // >0 ⇒ p left of a→b
		if cross < 0 {
			signed = -float64(dist)
		} else {
			signed = float64(dist)
		}
	}
	return signed
}

// signedLoopArea is twice the signed area of the closed polygon (CCW ⇒ positive), used only for its
// SIGN to make a positive offset distance shrink a loop whatever its traversal direction.
func signedLoopArea(pts []math.Point2) float64 {
	a := 0.0
	for i := 0; i < len(pts); i++ {
		p, q := pts[i], pts[(i+1)%len(pts)]
		a += float64(p.X*q.Y - q.X*p.Y)
	}
	return a
}

// SketchMirrorTool mirrors the first picked entity across the second picked line.
type SketchMirrorTool struct {
	dialogTool
	pickCollector
}

// NewSketchMirrorTool makes a mirror tool (pick geometry, then the mirror line).
func NewSketchMirrorTool() *SketchMirrorTool {
	return &SketchMirrorTool{pickCollector: pickCollector{want: 2}}
}

func (t *SketchMirrorTool) Name() string                  { return "Mirror" }
func (t *SketchMirrorTool) Pick(_ *Session, s Selectable) { t.take(s) }

// Accepts highlights any geometry as a mirror candidate — the first pick is the geometry
// to copy, the second the mirror line (Commit validates the second is a line).
func (t *SketchMirrorTool) Accepts(e sketch.Entity) bool { return e != nil }
func (t *SketchMirrorTool) CanCommit() bool              { return t.ready() }
func (t *SketchMirrorTool) AutoCommitOnPick() bool       { return true }
func (t *SketchMirrorTool) Cancel(*Session)              { t.reset() }
func (t *SketchMirrorTool) Prompt(*Session) string       { return "Pick geometry, then a mirror line." }

// Commit mirrors the first picked entity across the second picked line.
func (t *SketchMirrorTool) Commit(s *Session) error {
	sk := s.ActiveSketch()
	if sk == nil {
		return errors.New("mirror: no active sketch")
	}
	line, ok := t.picks[1].(*sketch.Line)
	if !ok {
		return errors.New("mirror: the second pick must be a line")
	}
	if sk.MirrorEntities([]sketch.Entity{t.picks[0]}, line) == nil {
		return errors.New("mirror: produced no copies (zero-length line?)")
	}
	return nil
}

// twoPickedLines casts two picks to lines.
func twoPickedLines(picks []sketch.Entity) (*sketch.Line, *sketch.Line, error) {
	if len(picks) != 2 {
		return nil, nil, errors.New("need two line picks")
	}
	l1, ok1 := picks[0].(*sketch.Line)
	l2, ok2 := picks[1].(*sketch.Line)
	if !ok1 || !ok2 {
		return nil, nil, errors.New("both picks must be lines")
	}
	return l1, l2, nil
}
