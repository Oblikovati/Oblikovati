// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/app/cmdline"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
)

// hudLineSession opens a sketch with the Line tool active and the camera centred so sketch
// point (0,0) is at screen centre — the dynamic-input HUD fixture (#790).
func hudLineSession(t *testing.T) (*Session, *LineTool) {
	t.Helper()
	s, _ := sketchSession(t)
	s.Grid().SnapToGrid = false
	centerOnSketchPoint(s, 0)
	line := NewLineTool()
	s.StartTool(line)
	return s, line
}

func TestHUDEnableTogglePersists(t *testing.T) {
	s := NewSession()
	if !s.HUDEnabled() {
		t.Fatal("the dynamic-input HUD should be on by default")
	}
	s.SetHUDEnabled(false)
	if s.HUDEnabled() {
		t.Error("SetHUDEnabled(false) did not turn it off")
	}
}

// TestHUDHiddenWhenNotApplicable checks the HUD is invisible when disabled, when no sketch is
// being edited, and when the active tool takes no coordinate input.
func TestHUDHiddenWhenNotApplicable(t *testing.T) {
	s, _ := hudLineSession(t)
	s.SetHUDEnabled(false)
	if s.SketchHUDView(100, 100).Visible {
		t.Error("HUD must be hidden when disabled")
	}
	s.SetHUDEnabled(true)

	plain := NewSession() // not in a sketch
	if plain.SketchHUDView(100, 100).Visible {
		t.Error("HUD must be hidden outside a sketch")
	}
}

// TestHUDCartesianTracksCursor checks the first point shows X/Y fields tracking the cursor.
func TestHUDCartesianTracksCursor(t *testing.T) {
	s, _ := hudLineSession(t)
	v := s.SketchHUDView(130, 80)
	if !v.Visible || v.Mode != HUDCartesian {
		t.Fatalf("first-point HUD = %+v, want a visible Cartesian view", v)
	}
	if v.Labels != [2]string{"X", "Y"} {
		t.Errorf("Cartesian labels = %v, want [X Y]", v.Labels)
	}
	// Untyped fields read the live cursor coordinate (in document units).
	cur, _ := s.CursorSketchPoint(130, 80)
	u := s.DocumentUnits()
	wantX := formatHUDNumber(u.ToPreferred(quantityLength(cur.X)))
	if v.Values[0] != wantX {
		t.Errorf("live X value = %q, want %q", v.Values[0], wantX)
	}
}

// TestHUDPolarAfterFirstPoint checks that once the tool has a point, the HUD switches to
// Length/Angle relative to it.
func TestHUDPolarAfterFirstPoint(t *testing.T) {
	s, line := hudLineSession(t)
	line.points = []math.Point2{math.P2(0, 0)} // a placed reference point
	v := s.SketchHUDView(130, 100)
	if v.Mode != HUDPolar || v.Labels != [2]string{"Length", "Angle"} {
		t.Fatalf("after first point HUD = %+v, want polar Length/Angle", v)
	}
}

// TestHUDTypingEngagesAndCycles checks typing fills the active field and engages the HUD, and
// Tab moves entry to the other field.
func TestHUDTypingEngagesAndCycles(t *testing.T) {
	s, _ := hudLineSession(t)
	if s.HUDEngaged() {
		t.Fatal("HUD should not be engaged before typing")
	}
	for _, r := range "50" {
		s.HUDInputRune(r)
	}
	if !s.HUDEngaged() {
		t.Error("typing should engage the HUD")
	}
	if got := s.SketchHUDView(100, 100); got.Values[0] != "50" || got.Active != 0 {
		t.Errorf("after typing 50: values=%v active=%d, want field0=50 active 0", got.Values, got.Active)
	}
	s.HUDTab()
	for _, r := range "30" {
		s.HUDInputRune(r)
	}
	if got := s.SketchHUDView(100, 100); got.Values[1] != "30" || got.Active != 1 {
		t.Errorf("after Tab+30: values=%v active=%d, want field1=30 active 1", got.Values, got.Active)
	}
}

// TestHUDCommitPlacesTypedCartesianPoint checks committing typed X/Y feeds the tool a point at
// the resolved sketch coordinate (document units → model units).
func TestHUDCommitPlacesTypedCartesianPoint(t *testing.T) {
	s, line := hudLineSession(t)
	for _, r := range "50" { // X = 50 mm
		s.HUDInputRune(r)
	}
	s.HUDTab()
	for _, r := range "30" { // Y = 30 mm
		s.HUDInputRune(r)
	}
	if err := s.HUDCommit(100, 100); err != nil {
		t.Fatalf("HUDCommit: %v", err)
	}
	// 50 mm / 30 mm in a metric document = 5 cm / 3 cm model units.
	ref, ok := line.PendingReferencePoint()
	if !ok || !ref.IsEqualTo(math.P2(5, 3), 1e-6) {
		t.Errorf("placed point = %v (ok %v), want (5,3) model units", ref, ok)
	}
	if s.HUDEngaged() {
		t.Error("HUDCommit should reset the typing state")
	}
}

// TestHUDCommitRejectsBadNumber checks a non-numeric field surfaces an error and keeps the HUD
// open for correction.
func TestHUDCommitRejectsBadNumber(t *testing.T) {
	s, _ := hudLineSession(t)
	s.HUDInputRune('-')
	s.HUDInputRune('-') // "--" is not a number
	if err := s.HUDCommit(100, 100); err == nil {
		t.Error("committing a non-numeric field should error")
	}
	if !s.HUDEngaged() {
		t.Error("a failed commit should keep the HUD engaged for correction")
	}
}

// TestHUDBackspaceAndCancel checks Backspace edits the active field and Cancel/non-number
// runes clear or are ignored.
func TestHUDBackspaceAndCancel(t *testing.T) {
	s, _ := hudLineSession(t)
	s.HUDBackspace() // no-op on an empty field (must not panic)
	for _, r := range "12" {
		s.HUDInputRune(r)
	}
	s.HUDInputRune('x') // not a number rune → ignored
	s.HUDBackspace()
	if got := s.SketchHUDView(100, 100).Values[0]; got != "1" {
		t.Errorf("after 12 + ignored x + backspace, field0 = %q, want %q", got, "1")
	}
	s.HUDCancel()
	if s.HUDEngaged() {
		t.Error("HUDCancel should disengage the HUD")
	}
}

// TestHUDCommitPolarPoint checks committing a typed Length/Angle resolves to the reference
// point offset by length∠angle (the polar branch + degree→radian conversion).
func TestHUDCommitPolarPoint(t *testing.T) {
	s, line := hudLineSession(t)
	line.points = []math.Point2{math.P2(1, 0)} // a placed reference point → polar mode
	for _, r := range "100" {                  // Length = 100 mm = 10 cm model
		s.HUDInputRune(r)
	}
	s.HUDTab()
	for _, r := range "90" { // Angle = 90° → straight up
		s.HUDInputRune(r)
	}
	if err := s.HUDCommit(100, 100); err != nil {
		t.Fatalf("HUDCommit: %v", err)
	}
	// ref (1,0) + 10 cm at 90° = (1, 10).
	got := line.points[len(line.points)-1]
	if !got.IsEqualTo(math.P2(1, 10), 1e-6) {
		t.Errorf("polar commit landed at %v, want (1,10)", got)
	}
}

// TestHUDCommitWithoutToolErrors checks committing with no coordinate tool active errors (the
// hudTool nil guard) and a non-numeric polar angle is rejected (hudResolveCoord polar branch).
func TestHUDCommitWithoutToolErrors(t *testing.T) {
	s, _ := sketchSession(t) // in a sketch, but no tool started
	s.HUDInputRune('5')
	if err := s.HUDCommit(100, 100); err == nil {
		t.Error("HUDCommit with no coordinate tool active should error")
	}

	sl, lt := hudLineSession(t)
	lt.points = []math.Point2{math.P2(0, 0)} // polar mode
	sl.HUDTab()                              // focus the Angle field
	sl.HUDInputRune('-')
	sl.HUDInputRune('-') // "--" — not a number
	if err := sl.HUDCommit(100, 100); err == nil {
		t.Error("a non-numeric angle should fail the polar commit")
	}
}

// TestSubmitResolvedCoordGuards checks the shared command-engine entry the HUD uses rejects a
// missing or non-coordinate tool.
func TestSubmitResolvedCoordGuards(t *testing.T) {
	s := NewSession()
	if err := s.CommandLine().SubmitResolvedCoord(s, cmdline.Coord{X: 1, Y: 2}); err == nil {
		t.Error("SubmitResolvedCoord with no active tool should error")
	}
}

// quantityLength wraps a model-unit length so the test can convert it for display comparison.
func quantityLength(v float64) param.Quantity { return param.Q(v, param.Length) }
