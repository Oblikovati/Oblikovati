// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/doc"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// newPartWithOffsetSquare sets up a session whose active part has a sketch with one
// closed square at x∈[x0,x0+side], y∈[0,side] — a profile offset from the Y axis so a
// revolve about Y produces a washer. Returns the session and that profile handle.
func newPartWithOffsetSquare(t *testing.T, x0, side float64) (*Session, ProfileHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(x0, 0))
	c1 := sk.Points().Add(math.P2(x0+side, 0))
	c2 := sk.Points().Add(math.P2(x0+side, side))
	c3 := sk.Points().Add(math.P2(x0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return s, ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

// TestRevolveToolEndToEnd drives the Revolve UI with synthetic input — start the tool,
// click the profile, accept the default full revolution about Y, OK — and asserts a
// validated washer solid (inner r=2, outer r=4, height 2 ⇒ 24π) lands in the part.
func TestRevolveToolEndToEnd(t *testing.T) {
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)  // ribbon: click "Revolve"
	s.Click(120, 90) // viewport: click the profile
	// default axis = Y origin, default angle = full revolution
	if err := s.OK(); err != nil { // dialog: OK
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after revolve, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("revolved body not a valid solid: %+v", r)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2 // 24π
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.01 {
		t.Errorf("washer volume = %g, want ≈%g (24π)", got, want)
	}
	if def.Features().Count() != 1 || !def.Features().Item(0).Health().OK() {
		t.Error("revolve feature missing or unhealthy")
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestRevolveViaCommandAlias shows the ribbon command launching the tool by its alias,
// then a partial-angle revolve through the property-window setters.
func TestRevolveViaCommandAlias(t *testing.T) {
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.PressKey(KeyEvent{Key: "R"}); err != nil { // type the "R" alias
		t.Fatalf("alias: %v", err)
	}
	rv, ok := s.ActiveTool().Tool().(*RevolveTool)
	if !ok {
		t.Fatal("Revolve alias did not start the revolve tool")
	}
	s.Click(1, 1)                   // pick the profile
	rv.SetAxis(feature.OriginYAxis) // property window: axis = Y
	rv.SetAngle(stdmath.Pi / 2)     // property window: 90°
	if !rv.CanCommit() {
		t.Fatal("tool not ready after picking a profile")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	want := stdmath.Pi * (4*4 - 2*2) * 2 / 4 // quarter washer
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.02 {
		t.Errorf("quarter-washer volume = %g, want ≈%g", got, want)
	}
}

func TestRevolveToolNeedsProfile(t *testing.T) {
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})
	rv := NewRevolveTool()
	s.StartTool(rv)
	if rv.CanCommit() {
		t.Error("tool ready with no profile picked")
	}
	s.Click(0, 0)
	if !rv.CanCommit() {
		t.Error("tool not ready after picking a profile")
	}
}

// relErrApp is the relative error helper for app-layer geometry assertions.
func relErrApp(got, want float64) float64 {
	if want == 0 {
		return stdmath.Abs(got)
	}
	return stdmath.Abs(got-want) / stdmath.Abs(want)
}
