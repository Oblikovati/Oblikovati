// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/sketch"
)

// Typing a value into the in-place dimension boxes broke after ONE digit (#2033). A bare
// alphanumeric key means "start typing a command" (#1751 S2), which hands focus to the Command
// Window — so the first digit both filled the box AND focused the command line, and from the next
// frame on the head skipped the HUD router because a text widget owned the keyboard. The second
// digit, and Tab, went to the command line. It affected every geometry tool, since the path is
// shared.

// placingRectangle returns a session with the rectangle tool mid-placement — one corner down, so
// the Width/Height boxes are up and waiting for a value.
func placingRectangle(t *testing.T) *Session {
	t.Helper()
	s, _ := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	if len(s.PlacementFields()) != 2 {
		t.Fatalf("setup: got %d placement fields, want the rectangle's Width and Height", len(s.PlacementFields()))
	}
	return s
}

// TestDigitWhilePlacingDoesNotGrabTheCommandWindow is the regression: the keystroke must not steal
// keyboard focus, or every digit after the first lands in the command line.
func TestDigitWhilePlacingDoesNotGrabTheCommandWindow(t *testing.T) {
	t.Parallel()
	s := placingRectangle(t)

	if err := s.PressKey(KeyEvent{Key: "1"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}

	if s.TakeCommandInputFocus() {
		t.Error("a digit typed while placing a shape focused the Command Window — the next digit would land there")
	}
	if seed, _ := s.TakeCommandTypeSeed(); seed != "" {
		t.Errorf("the digit seeded the command line with %q instead of the dimension box", seed)
	}
}

// TestNoKeyWhilePlacingFocusesTheCommandWindow: the inhibition covers EVERY bare key, not just the
// digits that exposed it. A letter would take the keyboard just as effectively, and the placement's
// entry surfaces would go dead the same way.
func TestNoKeyWhilePlacingFocusesTheCommandWindow(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"1", "L", "7", "x"} {
		s := placingRectangle(t)

		if err := s.PressKey(KeyEvent{Key: key}); err != nil {
			t.Fatalf("PressKey(%q): %v", key, err)
		}

		if s.TakeCommandInputFocus() {
			t.Errorf("%q focused the Command Window while a shape was being placed", key)
		}
		if seed, _ := s.TakeCommandTypeSeed(); seed != "" {
			t.Errorf("%q seeded the command line with %q while placing", key, seed)
		}
	}
}

// TestCommandWindowStaysOpenBeforeTheFirstClick: a started command with no point down yet is
// exactly when a user may want to TYPE the first point's coordinates, so the Command Window must
// still take the keystroke. The inhibition begins at the first MOUSE-placed point, not at the
// tool starting.
func TestCommandWindowStaysOpenBeforeTheFirstClick(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	s.StartTool(NewRectangleTool()) // started, nothing clicked

	if err := s.PressKey(KeyEvent{Key: "5"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}

	if !s.TakeCommandInputFocus() {
		t.Error("with no point placed yet, a keystroke must still reach the Command Window to type coordinates")
	}
	if seed, _ := s.TakeCommandTypeSeed(); seed != "5" {
		t.Errorf("command seed = %q, want %q", seed, "5")
	}
}

// TestTypedFirstPointKeepsTheCommandWindow: placing the first point through the COMMAND LINE must
// not hand the keyboard away — the user is plainly working there, and would otherwise be unable to
// type the second point.
func TestTypedFirstPointKeepsTheCommandWindow(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	s.StartTool(NewLineTool())
	if err := s.CommandLine().Submit(s, "0,0"); err != nil {
		t.Fatalf("submit first point: %v", err)
	}

	if s.PlacingGeometry() {
		t.Error("a point typed into the command line started the mouse-placement inhibition")
	}
	if err := s.PressKey(KeyEvent{Key: "4"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}
	if !s.TakeCommandInputFocus() {
		t.Error("after typing the first point, the Command Window must still take the next one")
	}
}

// TestNextCommandStartsWithTheKeyboardFree: the inhibition must not outlive the placement that
// raised it. Starting a second command after a completed one has to leave its first point typeable
// again — otherwise the first shape drawn in a session would poison the Command Window for the
// rest of it.
func TestNextCommandStartsWithTheKeyboardFree(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	// Switch commands MID-placement — the ribbon button clicked with a point already down. That
	// path cancels the running tool inside StartTool rather than through CancelTool/commit, so it
	// is the one that leaks a stale flag.
	s.StartTool(NewLineTool())
	s.Click(40, 40)

	s.StartTool(NewCircleTool()) // a new command, nothing clicked yet
	if s.PlacingGeometry() {
		t.Fatal("a new command began already inhibited — the previous placement's state leaked")
	}
	if err := s.PressKey(KeyEvent{Key: "2"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}
	if !s.TakeCommandInputFocus() {
		t.Error("the new command's first point is no longer typeable into the Command Window")
	}
}

// TestPlacementInhibitionEndsWithTheTool: the Command Window is only held off DURING a placement.
// Once the tool is gone a bare key starts a command again, or the fix would break command typing
// for the rest of the session.
func TestPlacementInhibitionEndsWithTheTool(t *testing.T) {
	t.Parallel()
	s := placingRectangle(t)
	s.CancelTool()

	if err := s.PressKey(KeyEvent{Key: "L"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}
	if !s.TakeCommandInputFocus() {
		t.Error("with the placement tool gone, a bare key must begin command typing again")
	}
}

// TestDigitStillStartsACommandOutsideASketch: the dimension boxes only own digits while a shape is
// being placed. With no sketch tool running, a digit begins a command as it always did.
func TestDigitStillStartsACommandOutsideASketch(t *testing.T) {
	t.Parallel()
	s, _ := emptyPartSession(t)

	if err := s.PressKey(KeyEvent{Key: "1"}); err != nil {
		t.Fatalf("PressKey: %v", err)
	}

	if !s.TakeCommandInputFocus() {
		t.Error("a digit outside a sketch should still begin command typing")
	}
}

// TestDigitsAccumulateAndTabAdvances is the user-visible behaviour end to end: several digits fill
// one box, Tab locks it and moves to the next.
func TestDigitsAccumulateAndTabAdvances(t *testing.T) {
	t.Parallel()
	s := placingRectangle(t)
	for _, r := range "125" {
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	s.PlacementFieldInput('8')

	fields := s.PlacementFields()
	if got := fields[0]; got.Value != "125" || !got.Locked {
		t.Errorf("Width = %q locked=%v, want \"125\" locked", got.Value, got.Locked)
	}
	if got := fields[1]; got.Value != "8" || !got.Active {
		t.Errorf("Height = %q active=%v, want \"8\" active after Tab", got.Value, got.Active)
	}
}

// TestKeystrokeReachesTheBoxesNotNowhere: holding the Command Window off must HAND the keystroke
// to the in-place boxes, not drop it. The head feeds them from its own character queue, so this is
// what every other caller — the wire's viewport.key, a script — depends on.
func TestKeystrokeReachesTheBoxesNotNowhere(t *testing.T) {
	t.Parallel()
	s := placingRectangle(t)

	for _, key := range []string{"2", "5"} {
		if err := s.PressKey(KeyEvent{Key: key}); err != nil {
			t.Fatalf("PressKey(%q): %v", key, err)
		}
	}

	if got := s.PlacementFields()[0].Value; got != "25" {
		t.Errorf("the active box holds %q, want %q — the keystrokes went nowhere", got, "25")
	}
}

// TestLockedValueShapesTheCommittedGeometry: a value the user locked must DRIVE the shape it
// creates. The commit created the dimension but never re-solved, so the rectangle kept the size the
// cursor happened to be at while its dimension claimed the typed one (#2034).
func TestLockedValueShapesTheCommittedGeometry(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	for _, r := range "50" { // 50 mm = 5 cm
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	s.Click(140, 120)

	dims := sk.DimensionConstraints().All()
	if len(dims) != 1 {
		t.Fatalf("got %d dimensions, want the locked Width", len(dims))
	}
	want := dims[0].Parameter().ModelValue()
	l := sk.Lines().Item(0) // the width edge, built first by the recipe
	if got := float64(l.A.Position().DistanceTo(l.B.Position())); got < want-0.001 || got > want+0.001 {
		t.Errorf("the committed edge is %.4f cm but its dimension says %.4f — the locked value did not drive it", got, want)
	}
}

// TestBothLockedValuesShapeTheCommittedRectangle is the reported flow: type a width, Tab, type a
// height, Tab, then click. Both values must shape the rectangle at once — it committed at the
// cursor's size (23.1 x 24.6 mm) carrying dimensions that said 10 x 30, and only snapped to them
// when the user dragged the geometry and forced a re-solve (#2034).
func TestBothLockedValuesShapeTheCommittedRectangle(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.Click(40, 40)
	for _, r := range "10" { // Width 10 mm
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	for _, r := range "30" { // Height 30 mm
		s.PlacementFieldInput(r)
	}
	s.PlacementFieldTab()
	s.Click(140, 120)

	if got := sk.DimensionConstraints().Count(); got != 2 {
		t.Fatalf("got %d dimensions, want one per locked value", got)
	}
	w, h := edgeLength(sk, 0), edgeLength(sk, 1)
	if w < 0.999 || w > 1.001 { // 10 mm = 1 cm
		t.Errorf("committed width %.4f cm, want the typed 1", w)
	}
	if h < 2.999 || h > 3.001 { // 30 mm = 3 cm
		t.Errorf("committed height %.4f cm, want the typed 3", h)
	}
}

// edgeLength is the length of the sketch's i'th line, in database units.
func edgeLength(sk *sketch.Sketch, i int) float64 {
	l := sk.Lines().Item(i)
	return float64(l.A.Position().DistanceTo(l.B.Position()))
}

// TestLockedValueShapesEveryGeometryKind: the defect was in the shared recipe commit, so it hit
// every tool that routes through it — not only the rectangle it was reported on.
func TestLockedValueShapesEveryGeometryKind(t *testing.T) {
	t.Parallel()
	t.Run("circle diameter", func(t *testing.T) {
		s, sk := sketchSession(t)
		s.StartTool(NewCircleTool())
		s.Click(60, 60)
		for _, r := range "20" { // Diameter 20 mm ⇒ radius 1 cm
			s.PlacementFieldInput(r)
		}
		s.PlacementFieldTab()
		s.Click(160, 60)

		if sk.Circles().Count() != 1 {
			t.Fatalf("got %d circles, want 1", sk.Circles().Count())
		}
		if got := float64(sk.Circles().Item(0).Radius); got < 0.999 || got > 1.001 {
			t.Errorf("committed radius %.4f cm, want the typed 1 (20 mm diameter)", got)
		}
	})
}
