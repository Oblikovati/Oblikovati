// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
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

// mkCenterlineSketch builds a sketch holding n vertical centerlines (no profile geometry).
func mkCenterlineSketch(n int) *sketch.Sketch {
	sk := sketch.NewSketches().Add(sketch.XYPlane())
	for i := range n {
		cl := sk.Lines().AddByTwoPoints(math.P2(math.Scalar(i), 0), math.P2(math.Scalar(i), 1))
		cl.SetCenterline(true)
	}
	return sk
}

// TestPreselectCenterlineRules covers Inventor's pre-selection rules.
func TestPreselectCenterlineRules(t *testing.T) {
	t.Parallel()
	one := mkCenterlineSketch(1) // one in the profile's sketch → pre-select it
	if _, l, ok := preselectCenterline(one, []*sketch.Sketch{one}); !ok || l == nil {
		t.Error("single in-sketch centerline should pre-select")
	}
	two := mkCenterlineSketch(2) // several in the profile's sketch → user must choose
	if _, _, ok := preselectCenterline(two, []*sketch.Sketch{two}); ok {
		t.Error("multiple in-sketch centerlines should NOT pre-select")
	}
	empty, elsewhere := mkCenterlineSketch(0), mkCenterlineSketch(1) // one visible elsewhere
	if sk, _, ok := preselectCenterline(empty, []*sketch.Sketch{empty, elsewhere}); !ok || sk != elsewhere {
		t.Error("single centerline elsewhere should pre-select")
	}
	same, other := mkCenterlineSketch(1), mkCenterlineSketch(1) // same-sketch one wins (rule 2)
	if sk, _, ok := preselectCenterline(same, []*sketch.Sketch{same, other}); !ok || sk != same {
		t.Error("the profile-sketch centerline should win over one elsewhere")
	}
	none, a, b := mkCenterlineSketch(0), mkCenterlineSketch(1), mkCenterlineSketch(1) // several elsewhere
	if _, _, ok := preselectCenterline(none, []*sketch.Sketch{none, a, b}); ok {
		t.Error("several elsewhere, none in-sketch → no pre-select")
	}
}

// TestRevolveToolPreselectsCenterline: clicking the profile auto-advances and pre-selects the
// sketch's single centerline as the axis (no axis pick, no toggle needed).
func TestRevolveToolPreselectsCenterline(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	cl := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1) // pick the profile → auto-advance + pre-select the centerline
	if line, ok := rv.Centerline(); !ok || line != cl {
		t.Fatal("profile pick should pre-select the sketch's centerline")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	want := stdmath.Pi * (4*4 - 2*2) * 2
	if v := query.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, want) > 0.01 {
		t.Errorf("pre-selected centerline washer = %g, want ≈%g", v, want)
	}
}

// TestRevolveToolMultipleCenterlinesNeedsPick: two centerlines ⇒ no pre-select; the user clicks
// the one to use and the revolve follows it.
func TestRevolveToolMultipleCenterlinesNeedsPick(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	vert := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	vert.SetCenterline(true)
	horiz := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(2, 0))
	horiz.SetCenterline(true)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(1, 1) // profile picked, but two centerlines ⇒ none pre-selected
	if _, ok := rv.Centerline(); ok {
		t.Fatal("two centerlines must not pre-select")
	}
	rv.Pick(s, SketchEntityHandle{Entity: vert}) // user picks the vertical one
	if line, ok := rv.Centerline(); !ok || line != vert {
		t.Fatal("picking a centerline should set it as the axis")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	want := stdmath.Pi * (4*4 - 2*2) * 2
	if v := query.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; relErrApp(v, want) > 0.01 {
		t.Errorf("about the vertical centerline = %g, want ≈%g (24π)", v, want)
	}
}

// TestRevolveToolAboutCenterline drives the Revolve UI with the "about centerline" option: the
// profile sketch carries a vertical centerline (the Y axis), so revolving about it (no axis
// pick) produces the same washer.
func TestRevolveToolAboutCenterline(t *testing.T) {
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	cl := profile.Sketch.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	s.SetPicker(stubPicker{sel: profile})

	rv := NewRevolveTool()
	s.StartTool(rv)
	s.Click(120, 90)
	rv.SetUseCenterline(true)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("centerline-revolved body not a valid solid: %+v", r)
	}
	want := stdmath.Pi * (4*4 - 2*2) * 2
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.01 {
		t.Errorf("centerline-revolved washer = %g, want ≈%g (24π)", got, want)
	}
}

// TestRevolveToolEndToEnd drives the Revolve UI with synthetic input — start the tool,
// click the profile, accept the default full revolution about Y, OK — and asserts a
// validated washer solid (inner r=2, outer r=4, height 2 ⇒ 24π) lands in the part.
func TestRevolveToolEndToEnd(t *testing.T) {
	t.Parallel()
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
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.01 {
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
	t.Parallel()
	s, profile := newPartWithOffsetSquare(t, 2, 2)
	s.SetPicker(stubPicker{sel: profile})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Revolve"); err != nil { // run the Revolve command
		t.Fatalf("execute: %v", err)
	}
	rv, ok := s.ActiveTool().Tool().(*RevolveTool)
	if !ok {
		t.Fatal("Revolve command did not start the revolve tool")
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
	if got := query.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 0.02 {
		t.Errorf("quarter-washer volume = %g, want ≈%g", got, want)
	}
}

func TestRevolveToolNeedsProfile(t *testing.T) {
	t.Parallel()
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
