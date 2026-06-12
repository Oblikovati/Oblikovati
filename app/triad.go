// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/math"
)

// The move/rotate triad (M05-F13, #620): a session-owned gizmo the head renders
// and drags; every gesture streams to the owner as events carrying the accumulated
// delta transform. The drag math lives here, headless and unit-tested — the head
// only routes rays.

// The drag phases of triad/manipulator events.
const (
	DragStart = "start"
	DragMove  = "move"
	DragEnd   = "end"
)

// TriadSegmentChanged fires (After) when the hovered segment changes.
type TriadSegmentChanged struct {
	Segment types.TriadSegment
	Hovered bool
}

// EventID implements event.Event.
func (TriadSegmentChanged) EventID() event.TypeID { return tidTriadSegment }

// TriadDragged fires (After) on every drag phase with the accumulated delta.
type TriadDragged struct {
	Phase    string
	Segment  types.TriadSegment
	MoveType types.TriadMoveType
	Delta    [16]float64
	Context  wire.DragContext
}

// EventID implements event.Event.
func (TriadDragged) EventID() event.TypeID { return tidTriadDrag }

// triadDrag is one in-flight gesture: the constraint resolved at grab time.
type triadDrag struct {
	segment  types.TriadSegment
	moveType types.TriadMoveType
	origin   math.Point3  // triad position at grab
	axis     math.Vector3 // axial/ring constraint axis (unit)
	normal   math.Vector3 // planar/free/ring working plane normal (unit)
	grab     math.Point3  // grab point on the constraint
	delta    [16]float64  // accumulated transform (row-major)
	snapTol  float64      // world-units snap radius for point inference
}

// TriadGizmo is the session's triad state.
type TriadGizmo struct {
	spec     wire.TriadSpec
	drag     *triadDrag
	hovered  types.TriadSegment
	hasHover bool
}

// TriadSpec returns the current triad declaration (Visible false when hidden).
func (s *Session) TriadSpec() wire.TriadSpec { return s.triad.spec }

// ShowTriad places (or re-places) the triad. Omitted axes default to identity.
func (s *Session) ShowTriad(spec wire.TriadSpec) error {
	for _, seg := range spec.Allowed {
		if seg > types.TriadZRing {
			return fmt.Errorf("app: triad segment %d out of range (0..9)", seg)
		}
	}
	s.triad.spec = spec
	return nil
}

// HideTriad dismisses the triad (ending any in-flight drag silently).
func (s *Session) HideTriad() {
	s.triad.spec.Visible = false
	s.triad.drag = nil
}

// TriadAllows reports whether a segment is grabbable under the spec's mask.
func (s *Session) TriadAllows(seg types.TriadSegment) bool {
	if len(s.triad.spec.Allowed) == 0 {
		return true
	}
	for _, allowed := range s.triad.spec.Allowed {
		if allowed == seg {
			return true
		}
	}
	return false
}

// HoverTriadSegment reports the hovered segment (the head's hit-test); transitions
// emit the segment-selection event.
func (s *Session) HoverTriadSegment(seg types.TriadSegment, hovered bool) {
	if s.triad.hasHover == hovered && (!hovered || s.triad.hovered == seg) {
		return
	}
	s.triad.hasHover, s.triad.hovered = hovered, seg
	event.Emit(s.bus, event.After, TriadSegmentChanged{Segment: seg, Hovered: hovered})
}

// triadAxes resolves the spec's orientation (identity when omitted).
func (s *Session) triadAxes() (x, y, z math.Vector3) {
	x, y, z = math.V3(1, 0, 0), math.V3(0, 1, 0), math.V3(0, 0, 1)
	if a := s.triad.spec.AxisX; a != nil {
		x = math.V3(a.X, a.Y, a.Z)
	}
	if a := s.triad.spec.AxisY; a != nil {
		y = math.V3(a.X, a.Y, a.Z)
	}
	if a := s.triad.spec.AxisZ; a != nil {
		z = math.V3(a.X, a.Y, a.Z)
	}
	return x, y, z
}

// TriadPosition returns the triad's position as a point.
func (s *Session) TriadPosition() math.Point3 {
	p := s.triad.spec.Position
	return math.P3(p.X, p.Y, p.Z)
}

// segmentConstraint resolves a segment to its motion constraint: the move type,
// the constraint axis (axial/ring moves), and the working-plane normal.
func (s *Session) segmentConstraint(seg types.TriadSegment, rayDir math.Vector3) (types.TriadMoveType, math.Vector3, math.Vector3, error) {
	axisOf, normalOf := s.segmentVectors()
	switch {
	case seg >= types.TriadXAxis && seg <= types.TriadZAxis:
		return types.TriadTranslate, axisOf[seg], math.Vector3{}, nil
	case seg >= types.TriadXYPlane && seg <= types.TriadXZPlane:
		return types.TriadTranslatePlanar, math.Vector3{}, normalOf[seg], nil
	case seg >= types.TriadXRing && seg <= types.TriadZRing:
		return types.TriadRotate, axisOf[seg], axisOf[seg], nil
	case seg == types.TriadOrigin:
		return types.TriadFree, math.Vector3{}, rayDir.Scale(-1), nil
	default:
		return 0, math.Vector3{}, math.Vector3{}, fmt.Errorf("app: unknown triad segment %d", seg)
	}
}

// segmentVectors maps segments to their constraint axis / working-plane normal.
func (s *Session) segmentVectors() (map[types.TriadSegment]math.Vector3, map[types.TriadSegment]math.Vector3) {
	x, y, z := s.triadAxes()
	axisOf := map[types.TriadSegment]math.Vector3{
		types.TriadXAxis: x, types.TriadYAxis: y, types.TriadZAxis: z,
		types.TriadXRing: x, types.TriadYRing: y, types.TriadZRing: z,
	}
	normalOf := map[types.TriadSegment]math.Vector3{
		types.TriadXYPlane: z, types.TriadYZPlane: x, types.TriadXZPlane: y,
	}
	return axisOf, normalOf
}

// BeginTriadDrag starts a gesture on a segment from the given pointer ray. snapTol
// is the world-units radius for point inference (0 disables snapping).
func (s *Session) BeginTriadDrag(seg types.TriadSegment, rayO math.Point3, rayD math.Vector3, snapTol float64, shift, ctrl bool) error {
	if !s.triad.spec.Visible {
		return fmt.Errorf("app: the triad is not visible")
	}
	if !s.TriadAllows(seg) {
		return fmt.Errorf("app: triad segment %v is not in the allowed set", seg)
	}
	moveType, axis, normal, err := s.segmentConstraint(seg, rayD)
	if err != nil {
		return err
	}
	drag := &triadDrag{
		segment: seg, moveType: moveType, origin: s.TriadPosition(),
		axis: unitOrZero(axis), normal: unitOrZero(normal),
		delta: identity4(), snapTol: snapTol,
	}
	drag.grab = drag.grabPoint(rayO, rayD)
	s.triad.drag = drag
	event.Emit(s.bus, event.After, TriadDragged{
		Phase: DragStart, Segment: seg, MoveType: moveType, Delta: drag.delta,
		Context: dragContext(drag.grab, rayO, rayD, shift, ctrl, types.InferenceNone),
	})
	return nil
}

// grabPoint resolves where on the constraint the pointer grabbed.
func (d *triadDrag) grabPoint(rayO math.Point3, rayD math.Vector3) math.Point3 {
	switch d.moveType {
	case types.TriadTranslate:
		t := closestParamOnAxis(d.origin, d.axis, rayO, rayD)
		return d.origin.TranslateBy(d.axis.Scale(t))
	default: // planar, free and ring grabs all live on the working plane
		if hit, ok := rayPlane(rayO, rayD, d.origin, d.normal); ok {
			return hit
		}
		return d.origin
	}
}

// DragTriad advances the gesture to the current pointer ray, emitting the
// accumulated delta. Translation moves snap to nearby work points (inference).
func (s *Session) DragTriad(rayO math.Point3, rayD math.Vector3, shift, ctrl bool) error {
	d := s.triad.drag
	if d == nil {
		return fmt.Errorf("app: no triad drag in flight")
	}
	inference := types.InferenceNone
	switch d.moveType {
	case types.TriadTranslate, types.TriadTranslatePlanar, types.TriadFree:
		move := d.translationTo(rayO, rayD)
		move, inference = s.snapTranslation(d, move)
		d.delta = translation4(move)
	case types.TriadRotate:
		d.delta = d.rotationTo(rayO, rayD)
	}
	event.Emit(s.bus, event.After, TriadDragged{
		Phase: DragMove, Segment: d.segment, MoveType: d.moveType, Delta: d.delta,
		Context: dragContext(d.grab, rayO, rayD, shift, ctrl, inference),
	})
	return nil
}

// translationTo resolves the constrained translation from grab to the current ray.
func (d *triadDrag) translationTo(rayO math.Point3, rayD math.Vector3) math.Vector3 {
	if d.moveType == types.TriadTranslate {
		t0 := closestParamOnAxis(d.origin, d.axis, rayO, rayD)
		return d.axis.Scale(t0 - d.origin.VectorTo(d.grab).Dot(d.axis))
	}
	hit, ok := rayPlane(rayO, rayD, d.origin, d.normal)
	if !ok {
		return math.V3(0, 0, 0)
	}
	return d.grab.VectorTo(hit)
}

// rotationTo resolves the ring rotation (about the triad position) to the ray.
func (d *triadDrag) rotationTo(rayO math.Point3, rayD math.Vector3) [16]float64 {
	hit, ok := rayPlane(rayO, rayD, d.origin, d.normal)
	if !ok {
		return d.delta
	}
	v0, v1 := d.origin.VectorTo(d.grab), d.origin.VectorTo(hit)
	if v0.Length() == 0 || v1.Length() == 0 {
		return d.delta
	}
	angle := stdmath.Atan2(float64(v0.Cross(v1).Dot(d.axis)), float64(v0.Dot(v1)))
	return rotationAbout4(d.origin, d.axis, angle)
}

// snapTranslation snaps the translated triad position to a nearby work point when
// one sits within the drag's snap radius (the point-inference hook).
func (s *Session) snapTranslation(d *triadDrag, move math.Vector3) (math.Vector3, types.PointInferenceKind) {
	if d.snapTol <= 0 {
		return move, types.InferenceNone
	}
	landed := d.origin.TranslateBy(move)
	part, err := activePart(s)
	if err != nil {
		return move, types.InferenceNone
	}
	points := part.WorkPoints()
	for i := 0; i < points.Count(); i++ {
		wp := points.Item(i)
		if float64(landed.DistanceTo(wp.Point())) <= d.snapTol {
			return d.origin.VectorTo(wp.Point()), types.InferencePoint
		}
	}
	return move, types.InferenceNone
}

// EndTriadDrag finishes the gesture, baking the final delta into the triad's
// position (translations) so it stays where the user left it.
func (s *Session) EndTriadDrag(rayO math.Point3, rayD math.Vector3, shift, ctrl bool) error {
	d := s.triad.drag
	if d == nil {
		return fmt.Errorf("app: no triad drag in flight")
	}
	s.triad.drag = nil
	if d.moveType != types.TriadRotate {
		landed := d.origin.TranslateBy(math.V3(d.delta[3], d.delta[7], d.delta[11]))
		s.triad.spec.Position = types.Point{X: float64(landed.X), Y: float64(landed.Y), Z: float64(landed.Z)}
	}
	event.Emit(s.bus, event.After, TriadDragged{
		Phase: DragEnd, Segment: d.segment, MoveType: d.moveType, Delta: d.delta,
		Context: dragContext(d.grab, rayO, rayD, shift, ctrl, types.InferenceNone),
	})
	return nil
}

// TriadDragging reports whether a gesture is in flight (the head suppresses
// navigation while it is).
func (s *Session) TriadDragging() bool { return s.triad.drag != nil }

// dragContext assembles the wire context for an event.
func dragContext(start math.Point3, rayO math.Point3, rayD math.Vector3, shift, ctrl bool, inf types.PointInferenceKind) wire.DragContext {
	return wire.DragContext{
		Start:     types.Point{X: float64(start.X), Y: float64(start.Y), Z: float64(start.Z)},
		RayOrigin: types.Point{X: float64(rayO.X), Y: float64(rayO.Y), Z: float64(rayO.Z)},
		RayDir:    types.Vector{X: float64(rayD.X), Y: float64(rayD.Y), Z: float64(rayD.Z)},
		Shift:     shift, Ctrl: ctrl, Inference: inf,
	}
}

// closestParamOnAxis returns the parameter t (world units along the unit axis from
// p0) of the point on the axis closest to the ray — the standard two-line closest
// approach, degenerate (parallel) cases collapsing to 0.
func closestParamOnAxis(p0 math.Point3, axis math.Vector3, rayO math.Point3, rayD math.Vector3) float64 {
	w := rayO.VectorTo(p0) // rayO → p0 (the textbook w0 = p0 − ro)
	b := float64(axis.Dot(rayD))
	denom := 1 - b*b
	if stdmath.Abs(denom) < 1e-9 {
		return 0
	}
	d := float64(axis.Dot(w))
	e := float64(rayD.Dot(w))
	return (b*e - d) / denom
}

// rayPlane intersects a ray with the plane (origin, normal).
func rayPlane(rayO math.Point3, rayD math.Vector3, origin math.Point3, normal math.Vector3) (math.Point3, bool) {
	denom := float64(rayD.Dot(normal))
	if stdmath.Abs(denom) < 1e-9 {
		return math.Point3{}, false
	}
	t := float64(rayO.VectorTo(origin).Dot(normal)) / denom
	if t < 0 {
		return math.Point3{}, false
	}
	return rayO.TranslateBy(rayD.Scale(t)), true
}

// identity4 / translation4 / rotationAbout4 build row-major 4×4 transforms.
func identity4() [16]float64 {
	return [16]float64{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func translation4(v math.Vector3) [16]float64 {
	m := identity4()
	m[3], m[7], m[11] = float64(v.X), float64(v.Y), float64(v.Z)
	return m
}

// rotationAbout4 rotates by angle around the unit axis through p (T·R·T⁻¹).
func rotationAbout4(p math.Point3, axis math.Vector3, angle float64) [16]float64 {
	c, s := stdmath.Cos(angle), stdmath.Sin(angle)
	x, y, z := float64(axis.X), float64(axis.Y), float64(axis.Z)
	t := 1 - c
	r := [9]float64{
		t*x*x + c, t*x*y - s*z, t*x*z + s*y,
		t*x*y + s*z, t*y*y + c, t*y*z - s*x,
		t*x*z - s*y, t*y*z + s*x, t*z*z + c,
	}
	px, py, pz := float64(p.X), float64(p.Y), float64(p.Z)
	// translation column = p - R·p
	tx := px - (r[0]*px + r[1]*py + r[2]*pz)
	ty := py - (r[3]*px + r[4]*py + r[5]*pz)
	tz := pz - (r[6]*px + r[7]*py + r[8]*pz)
	return [16]float64{
		r[0], r[1], r[2], tx,
		r[3], r[4], r[5], ty,
		r[6], r[7], r[8], tz,
		0, 0, 0, 1,
	}
}

// unitOrZero normalizes v, keeping the zero vector zero.
func unitOrZero(v math.Vector3) math.Vector3 {
	l := float64(v.Length())
	if l == 0 {
		return v
	}
	return v.Scale(1 / l)
}
