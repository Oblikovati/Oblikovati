// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/math"
)

// Line3DTool is the 3D-sketch line interaction: the user places points in model space and
// each consecutive pair becomes a segment, building a connected polyline rail (Inventor's
// 3D Line). Points arrive via AddPoint (the head feeds model-space picks; e2e tests call
// it directly). Committing adds the segments to the active 3D sketch.
type Line3DTool struct {
	points []math.Point3
}

// NewLine3DTool returns a 3D line tool.
func NewLine3DTool() *Line3DTool { return &Line3DTool{} }

// Name implements [Tool].
func (t *Line3DTool) Name() string { return "3D Line" }

// Start implements [Tool]; the 3D line tool needs no selection filter.
func (t *Line3DTool) Start(*Session) {}

// Pick implements [Tool]; model-space picks arrive via AddPoint, not selectables.
func (t *Line3DTool) Pick(*Session, Selectable) {}

// AddPoint records a model-space vertex of the polyline.
func (t *Line3DTool) AddPoint(p math.Point3) { t.points = append(t.points, p) }

// CanCommit is true once at least two points define a segment.
func (t *Line3DTool) CanCommit() bool { return len(t.points) >= 2 }

// Commit adds the polyline's segments to the active 3D sketch as connected lines (later
// segments reuse the previous endpoint so the rail is a single chain).
func (t *Line3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D line: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("3D line: need at least two points")
	}
	for i := 1; i < len(t.points); i++ {
		sk.AddLine3D(t.points[i-1], t.points[i])
	}
	return nil
}

// Cancel implements [Tool] (no state to restore).
func (t *Line3DTool) Cancel(*Session) {}

// Point3DTool places a single standalone point in the active 3D sketch.
type Point3DTool struct {
	point *math.Point3
}

// NewPoint3DTool returns a 3D point tool.
func NewPoint3DTool() *Point3DTool { return &Point3DTool{} }

// Name implements [Tool].
func (t *Point3DTool) Name() string { return "3D Point" }

// Start/Pick implement [Tool]; the point arrives via SetPoint.
func (t *Point3DTool) Start(*Session)            {}
func (t *Point3DTool) Pick(*Session, Selectable) {}

// SetPoint records the point's model-space position.
func (t *Point3DTool) SetPoint(p math.Point3) { t.point = &p }

// CanCommit is true once a position is set.
func (t *Point3DTool) CanCommit() bool { return t.point != nil }

// Commit adds the point to the active 3D sketch.
func (t *Point3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D point: not editing a 3D sketch")
	}
	if t.point == nil {
		return errors.New("3D point: no position set")
	}
	sk.AddPoint3D(*t.point)
	return nil
}

// Cancel implements [Tool].
func (t *Point3DTool) Cancel(*Session) {}

// Circle3DTool places a full circle in the active 3D sketch from a center, a plane axis
// (defaulting to +Z), and a radius.
type Circle3DTool struct {
	center *math.Point3
	axis   math.Vector3
	radius float64
}

// NewCircle3DTool returns a 3D circle tool (axis defaults to +Z).
func NewCircle3DTool() *Circle3DTool { return &Circle3DTool{axis: math.V3(0, 0, 1)} }

// Name implements [Tool]; Start/Pick are no-ops (inputs arrive via the setters).
func (t *Circle3DTool) Name() string              { return "3D Circle" }
func (t *Circle3DTool) Start(*Session)            {}
func (t *Circle3DTool) Pick(*Session, Selectable) {}

// SetCenter/SetAxis/SetRadius supply the circle's inputs.
func (t *Circle3DTool) SetCenter(p math.Point3) { t.center = &p }
func (t *Circle3DTool) SetAxis(v math.Vector3)  { t.axis = v }
func (t *Circle3DTool) SetRadius(r float64)     { t.radius = r }

// CanCommit is true once a center and a positive radius are set.
func (t *Circle3DTool) CanCommit() bool { return t.center != nil && t.radius > 0 }

// Commit adds the circle to the active 3D sketch.
func (t *Circle3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D circle: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("3D circle: need a center and a positive radius")
	}
	axis, err := math.UnitVector3FromVector(t.axis)
	if err != nil {
		return errors.New("3D circle: degenerate plane axis")
	}
	sk.AddCircle3D(*t.center, axis, t.radius)
	return nil
}

// Cancel implements [Tool].
func (t *Circle3DTool) Cancel(*Session) {}

// Arc3DTool places a circular arc from three picked points: center, start, end.
type Arc3DTool struct {
	points []math.Point3
	ccw    bool
}

// NewArc3DTool returns a 3D arc tool (counter-clockwise by default).
func NewArc3DTool() *Arc3DTool { return &Arc3DTool{ccw: true} }

// Name implements [Tool]; Start/Pick are no-ops.
func (t *Arc3DTool) Name() string              { return "3D Arc" }
func (t *Arc3DTool) Start(*Session)            {}
func (t *Arc3DTool) Pick(*Session, Selectable) {}

// AddPoint records the next of the arc's center/start/end points; SetCCW orients it.
func (t *Arc3DTool) AddPoint(p math.Point3) { t.points = append(t.points, p) }
func (t *Arc3DTool) SetCCW(ccw bool)        { t.ccw = ccw }

// CanCommit is true once all three points are placed.
func (t *Arc3DTool) CanCommit() bool { return len(t.points) == 3 }

// Commit adds the arc (center, start, end) to the active 3D sketch.
func (t *Arc3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("3D arc: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("3D arc: need center, start and end points")
	}
	sk.AddArc3D(t.points[0], t.points[1], t.points[2], t.ccw)
	return nil
}

// Cancel implements [Tool].
func (t *Arc3DTool) Cancel(*Session) {}

// Helix3DTool places a cylindrical helix from an axis-base origin, radius, pitch and turn
// count (axis defaults to +Z). It is the spring/thread-path tool.
type Helix3DTool struct {
	origin    *math.Point3
	axis      math.Vector3
	radius    float64
	pitch     float64
	turns     float64
	clockwise bool
}

// NewHelix3DTool returns a 3D helix tool (axis +Z).
func NewHelix3DTool() *Helix3DTool { return &Helix3DTool{axis: math.V3(0, 0, 1)} }

// Name implements [Tool]; Start/Pick are no-ops.
func (t *Helix3DTool) Name() string              { return "Helical Curve" }
func (t *Helix3DTool) Start(*Session)            {}
func (t *Helix3DTool) Pick(*Session, Selectable) {}

// Setters supply the helix's inputs (radius/pitch in cm, turns a revolution count).
func (t *Helix3DTool) SetOrigin(p math.Point3) { t.origin = &p }
func (t *Helix3DTool) SetAxis(v math.Vector3)  { t.axis = v }
func (t *Helix3DTool) SetRadius(r float64)     { t.radius = r }
func (t *Helix3DTool) SetPitch(p float64)      { t.pitch = p }
func (t *Helix3DTool) SetTurns(n float64)      { t.turns = n }
func (t *Helix3DTool) SetClockwise(cw bool)    { t.clockwise = cw }

// CanCommit is true once origin, positive radius/pitch and a positive turn count are set.
func (t *Helix3DTool) CanCommit() bool {
	return t.origin != nil && t.radius > 0 && t.pitch > 0 && t.turns > 0
}

// Commit adds the helix to the active 3D sketch.
func (t *Helix3DTool) Commit(s *Session) error {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return errors.New("helix: not editing a 3D sketch")
	}
	if !t.CanCommit() {
		return errors.New("helix: need origin, radius, pitch and turns")
	}
	axis, err := math.UnitVector3FromVector(t.axis)
	if err != nil {
		return errors.New("helix: degenerate axis")
	}
	sk.AddHelix3D(*t.origin, axis, t.radius, t.pitch, 0, t.turns, t.clockwise)
	return nil
}

// Cancel implements [Tool].
func (t *Helix3DTool) Cancel(*Session) {}

// --- Params (generic property dialog) -------------------------------------
// The 3D tools take their points from viewport clicks; the dialog edits the scalar/bool
// inputs (radius/pitch/turns/winding).

func (t *Circle3DTool) Params() ToolParams {
	return ToolParams{Floats: []FloatParam{{"Radius", func() float64 { return t.radius }, func(r float64) { t.radius = r }}}}
}

func (t *Helix3DTool) Params() ToolParams {
	return ToolParams{
		Floats: []FloatParam{
			{"Radius", func() float64 { return t.radius }, func(r float64) { t.radius = r }},
			{"Pitch", func() float64 { return t.pitch }, func(p float64) { t.pitch = p }},
			{"Turns", func() float64 { return t.turns }, func(n float64) { t.turns = n }},
		},
		Bools: []BoolParam{{"Clockwise", func() bool { return t.clockwise }, func(b bool) { t.clockwise = b }}},
	}
}

func (t *Arc3DTool) Params() ToolParams {
	return ToolParams{Bools: []BoolParam{{"Counter-clockwise", func() bool { return t.ccw }, func(b bool) { t.ccw = b }}}}
}

// compile-time assertions that the 3D sketch tools satisfy the Tool interface (and that the
// parameterized ones expose the property dialog).
var (
	_ Tool = (*Line3DTool)(nil)
	_ Tool = (*Point3DTool)(nil)
	_ Tool = (*Circle3DTool)(nil)
	_ Tool = (*Arc3DTool)(nil)
	_ Tool = (*Helix3DTool)(nil)

	_ ParameterizedTool = (*Circle3DTool)(nil)
	_ ParameterizedTool = (*Helix3DTool)(nil)
	_ ParameterizedTool = (*Arc3DTool)(nil)
)
