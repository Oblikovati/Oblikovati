// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/app/cmdline"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// The 2D-sketch dynamic-input HUD (#790): Inventor's heads-up entry of the next point's
// coordinates, length, and angle while a geometry tool is drawing. It is the in-canvas twin
// of the command line — the same CommandDriven tools that accept a typed "10,0" accept the
// HUD's typed point — so the HUD resolves its fields to one absolute sketch coordinate and
// submits it through the shared command engine (SubmitResolvedCoord), getting auto-commit and
// continuous-chaining for free. The two fields are X/Y for a shape's first point and
// Length/Angle (polar, relative to the pending reference point) for every point after it,
// matching Inventor's precise-input behaviour. Fields not yet typed track the live cursor.

// HUDMode selects which pair of fields the HUD shows.
type HUDMode uint8

const (
	// HUDCartesian shows absolute X / Y — the first point of a shape (no reference yet).
	HUDCartesian HUDMode = iota
	// HUDPolar shows Length / Angle relative to the pending reference point.
	HUDPolar
)

// sketchHUD is the live typing state of the dynamic-input HUD. fields holds each field's
// typed override ("" ⇒ the field tracks the cursor); active is the field receiving
// keystrokes; engaged becomes true once the user types or Tabs, switching the HUD from a
// passive readout to active entry.
type sketchHUD struct {
	fields  [2]string
	active  int
	engaged bool
}

// SketchHUDView is the per-frame HUD snapshot the head draws: the field labels and the value
// each shows (typed text, or the live cursor measurement), the highlighted field, and the
// length unit. Visible is false when the HUD should not draw at all.
type SketchHUDView struct {
	Visible bool
	Mode    HUDMode
	Labels  [2]string
	Values  [2]string
	Active  int
	Engaged bool
	Unit    string
}

// HUDEnabled reports whether the dynamic-input HUD is switched on (Application Options ▸
// Sketch ▸ Heads-Up Display, persisted). The head also gates drawing on being in a sketch.
func (s *Session) HUDEnabled() bool { return s.hudEnabled }

// SetHUDEnabled turns the dynamic-input HUD on or off and persists the choice.
func (s *Session) SetHUDEnabled(on bool) {
	if s.hudEnabled == on {
		return
	}
	s.hudEnabled = on
	s.appOptions.Sketch.HeadsUpDisplay = on
	if !on {
		s.sketchHUD = sketchHUD{}
	}
	_ = s.saveOptions()
}

// HUDEngaged reports whether the user has begun typing into the HUD this point — the head
// uses it to claim keystrokes (so Enter commits the HUD, Esc clears it) before the viewport.
func (s *Session) HUDEngaged() bool {
	return s.hudEnabled && s.InSketch() && s.sketchHUD.engaged
}

// SketchHUDView builds the HUD snapshot for the cursor at (px,py). It is invisible unless the
// HUD is enabled, a sketch is being edited, the active tool accepts coordinate input, and the
// cursor maps into the sketch plane.
func (s *Session) SketchHUDView(px, py float64) SketchHUDView {
	tool := s.hudTool()
	if tool == nil {
		return SketchHUDView{}
	}
	cur, ok := s.CursorSketchPoint(px, py)
	if !ok {
		return SketchHUDView{}
	}
	u := s.DocumentUnits()
	v := SketchHUDView{Visible: true, Active: s.sketchHUD.active, Engaged: s.sketchHUD.engaged, Unit: u.PreferredName(param.Length)}
	if ref, hasRef := hudReference(tool); hasRef {
		v.Mode = HUDPolar
		v.Labels = [2]string{"Length", "Angle"}
		v.Values = s.hudPolarValues(ref, cur, u)
		return v
	}
	v.Mode = HUDCartesian
	v.Labels = [2]string{"X", "Y"}
	v.Values = s.hudCartesianValues(cur, u)
	return v
}

// hudTool returns the active tool when the HUD applies to it: a coordinate-driven tool while a
// 2D sketch is being edited. Feature tools (extrude/revolve) never match because the HUD is
// gated on InSketch.
func (s *Session) hudTool() CommandDriven {
	if !s.hudEnabled || !s.InSketch() || s.tool == nil {
		return nil
	}
	driven, ok := s.tool.tool.(CommandDriven)
	if !ok {
		return nil
	}
	return driven
}

// hudCartesianValues formats the X/Y fields: the typed override, or the cursor coordinate in
// the document's length unit.
func (s *Session) hudCartesianValues(cur math.Point2, u param.UnitsOfMeasure) [2]string {
	return [2]string{
		s.hudFieldText(0, u.ToPreferred(param.Q(cur.X, param.Length))),
		s.hudFieldText(1, u.ToPreferred(param.Q(cur.Y, param.Length))),
	}
}

// hudPolarValues formats the Length/Angle fields relative to ref: the typed override, or the
// live distance (document unit) and direction (degrees, CCW from +X) from ref to the cursor.
func (s *Session) hudPolarValues(ref, cur math.Point2, u param.UnitsOfMeasure) [2]string {
	d := ref.DistanceTo(cur)
	ang := stdmath.Atan2(cur.Y-ref.Y, cur.X-ref.X) * 180 / stdmath.Pi
	return [2]string{
		s.hudFieldText(0, u.ToPreferred(param.Q(d, param.Length))),
		s.hudFieldText(1, ang),
	}
}

// hudFieldText returns field i's typed override, or the live value formatted for display.
func (s *Session) hudFieldText(i int, live float64) string {
	if t := s.sketchHUD.fields[i]; t != "" {
		return t
	}
	return formatHUDNumber(live)
}

// HUDInputRune appends a typed character to the active field when it is part of a number
// (digits, sign, decimal point), engaging the HUD. Other runes are ignored.
func (s *Session) HUDInputRune(r rune) {
	if !isHUDNumberRune(r) {
		return
	}
	s.sketchHUD.engaged = true
	s.sketchHUD.fields[s.sketchHUD.active] += string(r)
}

// HUDBackspace deletes the last character of the active field (a no-op when it is empty).
func (s *Session) HUDBackspace() {
	f := &s.sketchHUD.fields[s.sketchHUD.active]
	if *f == "" {
		return
	}
	*f = (*f)[:len(*f)-1]
}

// HUDTab moves keystroke focus to the other field (and engages the HUD), so the user can lock
// one value and type the next — Inventor's Tab field-cycling.
func (s *Session) HUDTab() {
	s.sketchHUD.engaged = true
	s.sketchHUD.active = (s.sketchHUD.active + 1) % len(s.sketchHUD.fields)
}

// HUDCancel clears the HUD's typed state, returning it to cursor-tracking (Esc with text
// typed — the head only routes Esc here while HUDEngaged, so an empty HUD still cancels the
// tool as usual).
func (s *Session) HUDCancel() { s.sketchHUD = sketchHUD{} }

// HUDCommit resolves the typed/live fields to an absolute sketch point and feeds it to the
// active tool through the shared command engine, then resets the HUD for the next point.
// It errors (and leaves the HUD intact for correction) when a field is not a number.
func (s *Session) HUDCommit(px, py float64) error {
	tool := s.hudTool()
	if tool == nil {
		return errors.New("sketch HUD: no coordinate tool is active")
	}
	cur, ok := s.CursorSketchPoint(px, py)
	if !ok {
		return errors.New("sketch HUD: the cursor is not over the sketch plane")
	}
	coord, err := s.hudResolveCoord(tool, cur)
	if err != nil {
		return err
	}
	if err := s.CommandLine().SubmitResolvedCoord(s, coord); err != nil {
		return err
	}
	s.sketchHUD = sketchHUD{}
	return nil
}

// hudResolveCoord resolves the two fields to one absolute sketch coordinate (model units):
// polar fields offset the reference point by length∠angle, Cartesian fields are the point
// directly. A blank field uses the live cursor measurement.
func (s *Session) hudResolveCoord(tool CommandDriven, cur math.Point2) (cmdline.Coord, error) {
	u := s.DocumentUnits()
	if ref, hasRef := hudReference(tool); hasRef {
		length, err := s.hudLengthModel(0, ref.DistanceTo(cur), u)
		if err != nil {
			return cmdline.Coord{}, err
		}
		angle, err := s.hudAngleRad(1, stdmath.Atan2(cur.Y-ref.Y, cur.X-ref.X))
		if err != nil {
			return cmdline.Coord{}, err
		}
		return cmdline.Coord{X: ref.X + length*stdmath.Cos(angle), Y: ref.Y + length*stdmath.Sin(angle)}, nil
	}
	x, err := s.hudLengthModel(0, cur.X, u)
	if err != nil {
		return cmdline.Coord{}, err
	}
	y, err := s.hudLengthModel(1, cur.Y, u)
	if err != nil {
		return cmdline.Coord{}, err
	}
	return cmdline.Coord{X: x, Y: y}, nil
}

// hudLengthModel returns field i as a model-unit length: the typed value (interpreted in the
// document's length unit) converted to model units, or liveModel when the field is blank.
func (s *Session) hudLengthModel(i int, liveModel float64, u param.UnitsOfMeasure) (float64, error) {
	t := s.sketchHUD.fields[i]
	if t == "" {
		return liveModel, nil
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, fmt.Errorf("sketch HUD: %q is not a number", t)
	}
	return u.FromPreferred(v, param.Length).Value, nil
}

// hudAngleRad returns field i as radians: the typed value (degrees) converted to radians, or
// liveRad when the field is blank.
func (s *Session) hudAngleRad(i int, liveRad float64) (float64, error) {
	t := s.sketchHUD.fields[i]
	if t == "" {
		return liveRad, nil
	}
	v, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return 0, fmt.Errorf("sketch HUD: %q is not a number", t)
	}
	return v * stdmath.Pi / 180, nil
}

// hudReferenced is the optional capability a geometry tool implements to expose the point the
// next input is measured from (its last placed point), so the HUD can show polar Length/Angle.
type hudReferenced interface {
	PendingReferencePoint() (math.Point2, bool)
}

// hudReference returns the tool's pending reference point, or false when it has none yet (the
// shape's first point) or the tool does not support polar entry.
func hudReference(tool CommandDriven) (math.Point2, bool) {
	if r, ok := tool.(hudReferenced); ok {
		return r.PendingReferencePoint()
	}
	return math.Point2{}, false
}

// isHUDNumberRune reports whether r is part of a typed number (digit, sign, or decimal point).
func isHUDNumberRune(r rune) bool {
	return (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '+'
}

// formatHUDNumber renders a measurement with up to three decimals and no trailing zeros, so
// the live readout stays compact (e.g. "12.5", "0", "-3.25").
func formatHUDNumber(v float64) string {
	t := strings.TrimRight(strconv.FormatFloat(v, 'f', 3, 64), "0")
	t = strings.TrimRight(t, ".")
	if t == "" || t == "-" {
		return "0"
	}
	return t
}
