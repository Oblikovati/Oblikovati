// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/math"
)

// Add-in manipulator handles (M05-F13, #620): world-space hotspots an add-in
// declares over its client graphics; dragging one streams manipulator.drag events
// with the handle's view-plane position (snapped to work points within the snap
// radius). The hit-testing lives in the head (it owns pixels); the drag math here.

// ManipulatorDragged fires (After) on every handle drag phase.
type ManipulatorDragged struct {
	Gizmo    string
	Handle   string
	Phase    string
	Position [3]float64
	Context  wire.DragContext
}

// EventID implements event.Event.
func (ManipulatorDragged) EventID() event.TypeID { return tidManipulatorDrag }

// manipulatorGizmo is one declared handle set.
type manipulatorGizmo struct {
	handles []wire.ManipulatorHandleSpec
	command string
}

// manipulatorDrag is one in-flight handle gesture.
type manipulatorDrag struct {
	gizmo, handle string
	start         math.Point3
	normal        math.Vector3 // the view plane the handle slides in
	snapTol       float64
}

// ManipulatorBoard holds the declared gizmos in creation order.
type ManipulatorBoard struct {
	order  []string
	gizmos map[string]manipulatorGizmo
	drag   *manipulatorDrag
}

// NewManipulatorBoard returns an empty board.
func NewManipulatorBoard() *ManipulatorBoard {
	return &ManipulatorBoard{gizmos: map[string]manipulatorGizmo{}}
}

// Handles returns every declared gizmo's handles in creation order, flattened for
// the head's hit-test (each spec keyed by its gizmo).
func (b *ManipulatorBoard) Handles() map[string][]wire.ManipulatorHandleSpec {
	out := map[string][]wire.ManipulatorHandleSpec{}
	for _, id := range b.order {
		out[id] = b.gizmos[id].handles
	}
	return out
}

// Manipulators returns the session's manipulator board.
func (s *Session) Manipulators() *ManipulatorBoard { return s.manipulators }

// SetManipulators replaces one gizmo's handle set.
func (s *Session) SetManipulators(id string, handles []wire.ManipulatorHandleSpec, command string) error {
	if id == "" {
		return fmt.Errorf("app: manipulator gizmo needs an id")
	}
	for _, h := range handles {
		if h.ID == "" {
			return fmt.Errorf("app: manipulator handle of gizmo %q needs an id", id)
		}
	}
	if _, exists := s.manipulators.gizmos[id]; !exists {
		s.manipulators.order = append(s.manipulators.order, id)
	}
	s.manipulators.gizmos[id] = manipulatorGizmo{handles: handles, command: command}
	return nil
}

// RemoveManipulators dismisses a gizmo's handles.
func (s *Session) RemoveManipulators(id string) error {
	if _, ok := s.manipulators.gizmos[id]; !ok {
		return fmt.Errorf("app: no manipulator gizmo %q", id)
	}
	delete(s.manipulators.gizmos, id)
	for i, x := range s.manipulators.order {
		if x == id {
			s.manipulators.order = append(s.manipulators.order[:i], s.manipulators.order[i+1:]...)
			break
		}
	}
	return nil
}

// BeginManipulatorDrag starts a handle gesture: the handle slides in the view
// plane through its position (viewNormal = the camera's forward at grab).
func (s *Session) BeginManipulatorDrag(gizmo, handle string, viewNormal math.Vector3, rayO math.Point3, rayD math.Vector3, snapTol float64, shift, ctrl bool) error {
	spec, err := s.manipulatorHandle(gizmo, handle)
	if err != nil {
		return err
	}
	start := math.P3(spec.Position[0], spec.Position[1], spec.Position[2])
	s.manipulators.drag = &manipulatorDrag{
		gizmo: gizmo, handle: handle, start: start,
		normal: unitOrZero(viewNormal), snapTol: snapTol,
	}
	event.Emit(s.bus, event.After, ManipulatorDragged{
		Gizmo: gizmo, Handle: handle, Phase: DragStart, Position: spec.Position,
		Context: dragContext(start, rayO, rayD, shift, ctrl, types.InferenceNone),
	})
	return nil
}

// DragManipulator advances the gesture: the handle's position is the ray's
// intersection with its view plane, snapped to a work point within the radius.
func (s *Session) DragManipulator(rayO math.Point3, rayD math.Vector3, shift, ctrl bool) error {
	d := s.manipulators.drag
	if d == nil {
		return fmt.Errorf("app: no manipulator drag in flight")
	}
	pos, inference := s.manipulatorPosition(d, rayO, rayD)
	event.Emit(s.bus, event.After, ManipulatorDragged{
		Gizmo: d.gizmo, Handle: d.handle, Phase: DragMove, Position: pos,
		Context: dragContext(d.start, rayO, rayD, shift, ctrl, inference),
	})
	return nil
}

// EndManipulatorDrag finishes the gesture, baking the final position into the
// handle spec so a re-list shows where it landed.
func (s *Session) EndManipulatorDrag(rayO math.Point3, rayD math.Vector3, shift, ctrl bool) error {
	d := s.manipulators.drag
	if d == nil {
		return fmt.Errorf("app: no manipulator drag in flight")
	}
	s.manipulators.drag = nil
	pos, inference := s.manipulatorPosition(d, rayO, rayD)
	s.bakeHandlePosition(d.gizmo, d.handle, pos)
	event.Emit(s.bus, event.After, ManipulatorDragged{
		Gizmo: d.gizmo, Handle: d.handle, Phase: DragEnd, Position: pos,
		Context: dragContext(d.start, rayO, rayD, shift, ctrl, inference),
	})
	return nil
}

// ManipulatorDragging reports whether a handle gesture is in flight.
func (s *Session) ManipulatorDragging() bool { return s.manipulators.drag != nil }

// manipulatorPosition resolves the dragged position with point inference.
func (s *Session) manipulatorPosition(d *manipulatorDrag, rayO math.Point3, rayD math.Vector3) ([3]float64, types.PointInferenceKind) {
	hit, ok := rayPlane(rayO, rayD, d.start, d.normal)
	if !ok {
		hit = d.start
	}
	inference := types.InferenceNone
	if d.snapTol > 0 {
		if snapped, ok := s.nearestWorkPoint(hit, d.snapTol); ok {
			hit, inference = snapped, types.InferencePoint
		}
	}
	return [3]float64{float64(hit.X), float64(hit.Y), float64(hit.Z)}, inference
}

// nearestWorkPoint finds a work point within tol of p.
func (s *Session) nearestWorkPoint(p math.Point3, tol float64) (math.Point3, bool) {
	part, err := activePart(s)
	if err != nil {
		return math.Point3{}, false
	}
	points := part.WorkPoints()
	for i := 0; i < points.Count(); i++ {
		wp := points.Item(i)
		if float64(p.DistanceTo(wp.Point())) <= tol {
			return wp.Point(), true
		}
	}
	return math.Point3{}, false
}

// manipulatorHandle finds one declared handle.
func (s *Session) manipulatorHandle(gizmo, handle string) (wire.ManipulatorHandleSpec, error) {
	g, ok := s.manipulators.gizmos[gizmo]
	if !ok {
		return wire.ManipulatorHandleSpec{}, fmt.Errorf("app: no manipulator gizmo %q", gizmo)
	}
	for _, h := range g.handles {
		if h.ID == handle {
			return h, nil
		}
	}
	return wire.ManipulatorHandleSpec{}, fmt.Errorf("app: gizmo %q has no handle %q", gizmo, handle)
}

// bakeHandlePosition writes the dragged position back into the stored spec.
func (s *Session) bakeHandlePosition(gizmo, handle string, pos [3]float64) {
	g := s.manipulators.gizmos[gizmo]
	for i := range g.handles {
		if g.handles[i].ID == handle {
			g.handles[i].Position = pos
		}
	}
	s.manipulators.gizmos[gizmo] = g
}

// dropCommandGizmos removes the triad and manipulator gizmos tied to a command's
// lifetime — the interaction-graphics lifecycle, like mini-toolbars.
func (s *Session) dropCommandGizmos() {
	if s.triad.spec.Command != "" {
		s.HideTriad()
	}
	for _, id := range append([]string(nil), s.manipulators.order...) {
		if s.manipulators.gizmos[id].command != "" {
			_ = s.RemoveManipulators(id)
		}
	}
}
