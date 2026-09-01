// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// newPartWithProfileAndPath sets up a part with a 2×2 square on XY (the profile) and a
// straight line up Z on the XZ plane (the path), returning the section and path handles.
func newPartWithProfileAndPath(t *testing.T) (*Session, ProfileHandle, PathHandle) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "part.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	prof := centeredSquareSketch(def, sketch.XYPlane(), 1) // 2×2 square, normal +Z
	pathSketch := def.Sketches().Add(sketch.XZPlane())     // (u,v) → (u,0,v)
	a := pathSketch.Points().Add(math.P2(0, 0))            // model (0,0,0)
	b := pathSketch.Points().Add(math.P2(0, 5))            // model (0,0,5)
	pathSketch.Lines().Add(a, b)
	return s, ProfileHandle{Sketch: prof, ProfileIndex: 0}, PathHandle{Sketch: pathSketch, PathIndex: 0}
}

// TestSweepToolEndToEnd drives the Sweep UI: start the tool, click the profile and the
// path, OK — and asserts a valid swept solid (2×2 profile along a length-5 path ⇒ V=20).
func TestSweepToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})

	sw := NewSweepTool()
	s.StartTool(sw)  // ribbon: click "Sweep"
	s.Click(10, 10)  // viewport: click the profile
	s.Click(10, 200) // viewport: click the path
	if !sw.CanCommit() {
		t.Fatalf("sweep not ready: profile=%v path=%v", sw.profile != nil, sw.path != nil)
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies after sweep, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, 20) > 0.02 {
		t.Errorf("swept volume = %g, want ≈20 (area 4 × length 5)", got)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

func TestSweepViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Sweep"); err != nil {
		t.Fatalf("execute Create.Sweep: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*SweepTool); !ok {
		t.Fatal("Sweep command did not start the sweep tool")
	}
	s.Click(0, 0)
	s.Click(0, 0)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Error("ribbon-launched sweep produced no body")
	}
}

// sweepDefOf returns the committed sweep's definition for inspecting how the tool wired Inventor's
// behaviours (orientation, taper, guide rail, scaling) into the model.
func sweepDefOf(t *testing.T, sw *SweepTool) *feature.SweepDefinition {
	t.Helper()
	if sw.AddedFeature() == nil {
		t.Fatal("sweep has not been committed")
	}
	return sw.AddedFeature().Definition().(*feature.SweepFeature).Definition()
}

// sweepVolume validates the part's single body and returns its volume.
func sweepVolume(t *testing.T, s *Session) float64 {
	t.Helper()
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("part has %d bodies, want 1", def.SurfaceBodies().Count())
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("swept body is not a valid solid: %+v", r)
	}
	return ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
}

// TestSweepParallelOrientation checks the Parallel ("Fixed") orientation is wired into the
// definition and still sweeps a valid solid (a straight path makes it geometrically identical to
// Follow Path, so the wiring — not the shape — is what this guards).
func TestSweepParallelOrientation(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	sw := NewSweepTool()
	sw.Pick(s, profile)
	sw.Pick(s, path)
	sw.SetOrientation(types.ParallelToOriginalProfile)
	if sw.Orientation() != types.ParallelToOriginalProfile {
		t.Fatalf("orientation = %v, want Parallel", sw.Orientation())
	}
	if err := sw.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := sweepDefOf(t, sw).Orientation; got != types.ParallelToOriginalProfile {
		t.Errorf("definition orientation = %v, want Parallel", got)
	}
	if v := sweepVolume(t, s); relErrApp(v, 20) > 0.02 {
		t.Errorf("parallel sweep volume = %g, want ≈20", v)
	}
}

// TestSweepTaperExpandsSection checks a positive taper scales the profile up along the path, so the
// swept frustum encloses MORE volume than the prismatic sweep (2×2 along length 5 = 20).
func TestSweepTaperExpandsSection(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	sw := NewSweepTool()
	sw.Pick(s, profile)
	sw.Pick(s, path)
	sw.SetTaper(0.2) // ≈11.5° draft outward
	if err := sw.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if sweepDefOf(t, sw).Taper == nil || sweepDefOf(t, sw).Taper() == 0 {
		t.Fatal("taper was not wired into the definition")
	}
	if v := sweepVolume(t, s); v <= 20 {
		t.Errorf("tapered sweep volume = %g, want > 20 (an outward draft expands the section)", v)
	}
}

// railSketchParallelToPath adds a straight rail offset +2 in X from the path (model (2,0,0)→(2,0,5)),
// staying a constant distance from it — a valid guide rail that steers without scaling.
func railSketchParallelToPath(def *compdef.PartComponentDefinition) PathHandle {
	rail := def.Sketches().Add(sketch.XZPlane())
	a := rail.Points().Add(math.P2(2, 0))
	b := rail.Points().Add(math.P2(2, 5))
	rail.Lines().Add(a, b)
	return PathHandle{Sketch: rail, PathIndex: 0}
}

// TestSweepGuideRailRoutingAndType checks the armed guide-rail selector routes the next path pick to
// the rail (not the path), flips the sweep type, and wires the rail + scaling into the definition.
func TestSweepGuideRailRoutingAndType(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	rail := railSketchParallelToPath(def)

	sw := NewSweepTool()
	sw.Pick(s, profile)
	sw.Pick(s, path)
	if sw.SweepType() != types.PathSweepType {
		t.Fatalf("type before rail = %v, want path", sw.SweepType())
	}
	sw.ArmGuideRailPicking()
	if !sw.GuideRailArmed() {
		t.Fatal("ArmGuideRailPicking did not arm")
	}
	sw.Pick(s, rail) // armed → routes to the rail slot, NOT the path
	if _, ok := sw.PickedGuideRail(); !ok {
		t.Fatal("guide rail was not picked while armed")
	}
	if sw.GuideRailArmed() {
		t.Error("arming should clear once the rail is picked")
	}
	if got, _ := sw.PickedPath(); got.Sketch == rail.Sketch {
		t.Error("armed pick leaked into the path slot")
	}
	if sw.SweepType() != types.PathAndGuideRailSweepType {
		t.Errorf("type after rail = %v, want pathAndGuideRail", sw.SweepType())
	}
	sw.SetScaling(types.XProfileScaling)
	if err := sw.Commit(s); err != nil {
		t.Fatalf("commit: %v", err)
	}
	d := sweepDefOf(t, sw)
	if d.GuideRail == nil {
		t.Error("guide rail was not wired into the definition")
	}
	if d.Scaling != types.XProfileScaling {
		t.Errorf("definition scaling = %v, want X", d.Scaling)
	}
	sweepVolume(t, s) // a constant-offset rail still sweeps a valid solid
}

// TestSweepClearGuideRailDisarms checks clearing the rail empties the slot and disarms picking.
func TestSweepClearGuideRail(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	sw := NewSweepTool()
	sw.Pick(s, profile)
	sw.Pick(s, path)
	sw.ArmGuideRailPicking()
	sw.Pick(s, railSketchParallelToPath(def))
	sw.ClearGuideRail()
	if _, ok := sw.PickedGuideRail(); ok {
		t.Error("guide rail still present after ClearGuideRail")
	}
	if sw.SweepType() != types.PathSweepType {
		t.Error("type should fall back to path after clearing the rail")
	}
}

func TestSweepToolNeedsProfileAndPath(t *testing.T) {
	t.Parallel()
	s, profile, path := newPartWithProfileAndPath(t)
	s.SetPicker(&seqPicker{sels: []Selectable{profile, path}})
	sw := NewSweepTool()
	s.StartTool(sw)
	s.Click(0, 0) // profile only
	if sw.CanCommit() {
		t.Error("sweep ready with no path")
	}
	s.Click(0, 0) // path
	if !sw.CanCommit() {
		t.Error("sweep not ready after profile + path")
	}
}
