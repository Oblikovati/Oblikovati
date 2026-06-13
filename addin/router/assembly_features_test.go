// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// assemblySessionWithBoxes makes an assembly the active document, places one unit-box
// part per X translation, and returns the router, session, assembly, and occurrences.
func assemblySessionWithBoxes(t *testing.T, xs ...float64) (*Router, *app.Session, *compdef.AssemblyComponentDefinition, []*occurrence.Occurrence) {
	t.Helper()
	s := app.NewSession()
	d, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate: %v", err)
	}
	asm := d.Content().(*compdef.AssemblyComponentDefinition)
	occs := make([]*occurrence.Occurrence, len(xs))
	for i, x := range xs {
		occs[i] = asm.Place(fmt.Sprintf("box:%d", i+1), blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Translation4(math.V3(x, 0, 0)))
	}
	return New(opregistry.Default()), s, asm, occs
}

// featureResultVolume sums an occurrence's machined assembly-feature result volume.
func featureResultVolume(asm *compdef.AssemblyComponentDefinition, o *occurrence.Occurrence) float64 {
	v := 0.0
	for _, b := range asm.Features().Result(o) {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// topHalfCut is the add request for a tool removing the top half (z>0.5) of unit boxes
// across a wide X range.
func topHalfCut() string {
	return `{"toolMin":[-1,-1,0.5],"toolMax":[100,2,2],"operation":"difference"}`
}

// TestAssemblyFeaturesAddAndListOverWire adds a cut over the wire and checks it lists
// with default participation, machining each participant to half volume.
func TestAssemblyFeaturesAddAndListOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0, 5)

	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &added)
	if added.Feature.Kind != "assemblyCut" || len(added.Feature.Participants) != 2 {
		t.Fatalf("added feature = %+v, want assemblyCut with 2 participants", added.Feature)
	}

	var list wire.AssemblyFeaturesResult
	call(t, r, s, "assemblyFeatures.list", `{}`, &list)
	if len(list.Features) != 1 || list.Features[0].ID != added.Feature.ID {
		t.Fatalf("list = %+v, want the one added feature", list.Features)
	}
	for _, o := range occs {
		if got := featureResultVolume(asm, o); stdmath.Abs(got-0.5) > 1e-6 {
			t.Errorf("participant machined volume = %g, want 0.5", got)
		}
	}
}

// TestAssemblyExtrudeOverWire drives the assembly sketching subsystem over the wire:
// create a sketch on the active assembly, rectangle a profile, and extrude-cut it into
// the participant — gated against the analytic value (unit box minus a 0.5×1×0.6
// pocket = 0.7; the database unit is the centimetre, so a 1 cm profile is 1 unit).
func TestAssemblyExtrudeOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)

	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", fmt.Sprintf(`{"sketchIndex":%d,"width":"0.5 cm","height":"1 cm"}`, sk.SketchIndex), &rect)
	if rect.Profiles != 1 {
		t.Fatalf("rectangle profiles = %d, want 1", rect.Profiles)
	}

	var added wire.AssemblyFeatureResult
	args := fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"distance":0.6,"operation":"difference"}`, sk.SketchIndex)
	call(t, r, s, "assemblyFeatures.addExtrude", args, &added)
	if added.Feature.Kind != "assemblyExtrude" {
		t.Fatalf("feature kind = %q, want assemblyExtrude", added.Feature.Kind)
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.7) > 1e-6 {
		t.Errorf("extrude-cut volume = %g, want 0.7 (unit box minus a 0.5×1×0.6 pocket)", got)
	}
}

// TestAssemblyRevolveOverWire drives the assembly revolve over the wire: a rectangle
// profile authored on the assembly's XY sketch (x∈[0,0.5], y∈[0,1]) revolved a full turn
// about the world Y axis is a cylinder (r=0.5) whose first quadrant cuts a quarter
// cylinder (≈π/16) from the unit-box participant — so it removes material without
// consuming the body. (The tight analytic gate is the model unit test; the boolean
// re-facets the small-radius cylinder coarsely here.)
func TestAssemblyRevolveOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", fmt.Sprintf(`{"sketchIndex":%d,"width":"0.5 cm","height":"1 cm"}`, sk.SketchIndex), &rect)

	var added wire.AssemblyFeatureResult
	args := fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"origin":[0,0,0],"axis":[0,1,0],"angle":%g,"operation":"difference"}`, sk.SketchIndex, 2*stdmath.Pi)
	call(t, r, s, "assemblyFeatures.addRevolve", args, &added)
	if added.Feature.Kind != "assemblyRevolve" {
		t.Fatalf("feature kind = %q, want assemblyRevolve", added.Feature.Kind)
	}
	if got := featureResultVolume(asm, occs[0]); got >= 1.0 || got < 0.7 {
		t.Errorf("revolve-cut volume = %g, want a unit box minus a quarter cylinder (≈0.8)", got)
	}

	if _, err := r.Handle(s, "assemblyFeatures.addRevolve", []byte(fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"origin":[0,0,0],"axis":[0,0,0],"angle":1,"operation":"difference"}`, sk.SketchIndex))); err == nil {
		t.Error("addRevolve with a zero axis should fail")
	}
	if _, err := r.Handle(s, "assemblyFeatures.addRevolve", []byte(fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"origin":[0,0,0],"axis":[0,1,0],"angle":0,"operation":"difference"}`, sk.SketchIndex))); err == nil {
		t.Error("addRevolve with a zero angle should fail")
	}
}

// TestAssemblyProxyCutOverWire adds a proxy-input cut whose tool is another
// occurrence's geometry, gated against the analytic value and shown to be associative:
// moving the source component re-resolves the cut on the next recompute.
func TestAssemblyProxyCutOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)
	// A tool occurrence: a unit box straddling the top half of the participant at occs[0].
	tool := asm.Place("tool:1", blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Translation4(math.V3(0, 0, 0.5)))

	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.addProxyCut", fmt.Sprintf(`{"source":%d,"operation":"difference"}`, tool.ID()), &added)
	if added.Feature.Kind != "assemblyProxyCut" {
		t.Fatalf("feature kind = %q, want assemblyProxyCut", added.Feature.Kind)
	}
	// The source tool must not be a participant (a component does not machine itself).
	for _, p := range added.Feature.Participants {
		if p == tool.ID() {
			t.Error("source occurrence should be excluded from participation")
		}
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("proxy-cut participant volume = %g, want 0.5 (top half removed by the tool)", got)
	}

	// Associativity: move the tool clear and recompute — the cut follows.
	tool.SetTransform(math.Translation4(math.V3(0, 0, 5)))
	call(t, r, s, "assembly.setEndOfFeatures", `{"position":-1}`, &wire.AssemblyFeaturesResult{}) // any recompute
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("after moving the tool clear: volume = %g, want 1.0 (associative)", got)
	}
}

// TestAssemblyAddHoleOverWire drills a parametric hole through the participant and
// checks it removed material (a faceted bore) without consuming the whole body.
func TestAssemblyAddHoleOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var added wire.AssemblyFeatureResult
	args := `{"center":[0.5,0.5,0],"axis":[0,0,1],"diameter":0.5,"depth":1.5}`
	call(t, r, s, "assemblyFeatures.addHole", args, &added)
	if added.Feature.Kind != "assemblyHole" {
		t.Fatalf("feature kind = %q, want assemblyHole", added.Feature.Kind)
	}
	got := featureResultVolume(asm, occs[0])
	if got >= 1.0 || got < 0.7 {
		t.Errorf("holed volume = %g, want a unit box minus a thin bore (≈0.8)", got)
	}

	if _, err := r.Handle(s, "assemblyFeatures.addHole", []byte(`{"center":[0,0,0],"axis":[0,0,0],"diameter":1,"depth":1}`)); err == nil {
		t.Error("addHole with a zero axis should fail")
	}
}

// TestAssemblySetParticipantsOverWire narrows participation to one occurrence; the
// dropped one is left whole.
func TestAssemblySetParticipantsOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0, 5)
	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &added)

	args := fmt.Sprintf(`{"id":%d,"participants":[%d]}`, added.Feature.ID, occs[0].ID())
	var setp wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.setParticipants", args, &setp)
	if len(setp.Feature.Participants) != 1 || setp.Feature.Participants[0] != occs[0].ID() {
		t.Fatalf("participants = %v, want just occurrence %d", setp.Feature.Participants, occs[0].ID())
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("kept participant volume = %g, want 0.5", got)
	}
	if got := featureResultVolume(asm, occs[1]); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("dropped participant volume = %g, want 1.0 (untouched)", got)
	}
}

// TestAssemblySetSuppressedOverWire suppresses then unsuppresses a feature in batch.
func TestAssemblySetSuppressedOverWire(t *testing.T) {
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)
	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &added)

	var list wire.AssemblyFeaturesResult
	call(t, r, s, "assemblyFeatures.setSuppressed", fmt.Sprintf(`{"ids":[%d],"suppressed":true}`, added.Feature.ID), &list)
	if !list.Features[0].Suppressed {
		t.Error("feature not reported suppressed")
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("suppressed-feature volume = %g, want 1.0 (passthrough)", got)
	}

	call(t, r, s, "assemblyFeatures.setSuppressed", fmt.Sprintf(`{"ids":[%d],"suppressed":false}`, added.Feature.ID), &list)
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("unsuppressed-feature volume = %g, want 0.5", got)
	}
}

// pathResultVolume sums one placement's machined assembly-feature result volume.
func pathResultVolume(asm *compdef.AssemblyComponentDefinition, path occurrence.OccurrencePath) float64 {
	v := 0.0
	for _, b := range asm.Features().ResultPath(path) {
		v += ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// TestAssemblySetParticipantPathsOverWire restricts a feature to one placement of a
// sub-assembly placed twice; only that placement is machined.
func TestAssemblySetParticipantPathsOverWire(t *testing.T) {
	s := app.NewSession()
	d, err := compdef.AddAssembly(s.Workspace(), "top.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	if err := s.Workspace().SetActiveDocument(d); err != nil {
		t.Fatalf("activate: %v", err)
	}
	top := d.Content().(*compdef.AssemblyComponentDefinition)
	sub := compdef.NewAssemblyComponentDefinition()
	sub.Place("part:1", blockPart(t, math.P3(0, 0, 0), math.P3(1, 1, 1)), math.Identity4())
	top.Place("subA:1", sub, math.Identity4())
	top.Place("subB:1", sub, math.Translation4(math.V3(10, 0, 0)))
	r := New(opregistry.Default())

	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &added)

	args := fmt.Sprintf(`{"id":%d,"paths":[["subA:1","part:1"]]}`, added.Feature.ID)
	var setp wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.setParticipantPaths", args, &setp)
	if len(setp.Feature.ParticipantPaths) != 1 || setp.Feature.ParticipantPaths[0][0] != "subA:1" {
		t.Fatalf("participant paths = %v, want one path through subA:1", setp.Feature.ParticipantPaths)
	}
	if got := pathResultVolume(top, occurrence.OccurrencePath{"subA:1", "part:1"}); stdmath.Abs(got-0.5) > 1e-6 {
		t.Errorf("subA placement volume = %g, want 0.5 (machined)", got)
	}
	if got := pathResultVolume(top, occurrence.OccurrencePath{"subB:1", "part:1"}); stdmath.Abs(got-1.0) > 1e-6 {
		t.Errorf("subB placement volume = %g, want 1.0 (excluded by path)", got)
	}

	if _, err := r.Handle(s, "assemblyFeatures.setParticipantPaths", []byte(fmt.Sprintf(`{"id":%d,"paths":[["nope:1"]]}`, added.Feature.ID))); err == nil {
		t.Error("setParticipantPaths with an unresolvable path should fail")
	}
}

// TestAssemblyEndOfFeaturesOverWire rolls the program back and reads the marker.
func TestAssemblyEndOfFeaturesOverWire(t *testing.T) {
	r, s, _, _ := assemblySessionWithBoxes(t, 0)
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &wire.AssemblyFeatureResult{})
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &wire.AssemblyFeatureResult{})

	var eof wire.EndOfFeaturesResult
	call(t, r, s, "assembly.getEndOfFeatures", `{}`, &eof)
	if eof.Position != -1 || eof.RolledBack {
		t.Fatalf("initial marker = %+v, want position -1 not rolled back", eof)
	}

	var list wire.AssemblyFeaturesResult
	call(t, r, s, "assembly.setEndOfFeatures", `{"position":1}`, &list)
	if !list.RolledBack || list.EndOfFeatures != 1 {
		t.Errorf("after rollback: result = %+v, want position 1 rolled back", list)
	}
	call(t, r, s, "assembly.getEndOfFeatures", `{}`, &eof)
	if eof.Position != 1 || !eof.RolledBack {
		t.Errorf("marker after rollback = %+v, want position 1 rolled back", eof)
	}
}

// TestAssemblyFeaturesRejectsBadInput pins the error paths: a bad operation, an unknown
// participant, and a non-assembly active document.
func TestAssemblyFeaturesRejectsBadInput(t *testing.T) {
	r, s, _, occs := assemblySessionWithBoxes(t, 0)
	if _, err := r.Handle(s, "assemblyFeatures.add", []byte(`{"toolMin":[0,0,0],"toolMax":[1,1,1],"operation":"bogus"}`)); err == nil {
		t.Error("add with a bad operation should fail")
	}
	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &added)
	bad := fmt.Sprintf(`{"id":%d,"participants":[99999]}`, added.Feature.ID)
	if _, err := r.Handle(s, "assemblyFeatures.setParticipants", []byte(bad)); err == nil {
		t.Error("setParticipants with an unknown occurrence should fail")
	}
	_ = occs

	rp, sp := emptyPartSession(t)
	if _, err := rp.Handle(sp, "assemblyFeatures.list", []byte(`{}`)); err == nil {
		t.Error("assemblyFeatures.list on a part document should fail")
	}
}
