// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
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
		v += query.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
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
	t.Parallel()
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
	t.Parallel()
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

// TestAssemblyFeatureEditOverWire authors a profiled extrude-cut, then edits its depth in
// place over the wire — the assembly-context Edit Feature (#725). The deeper pocket
// removes more material (gated against the analytic value), while the box cut exposes no
// editable scalars (its tool is fixed at construction) so editing it is rejected.
func TestAssemblyFeatureEditOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", fmt.Sprintf(`{"sketchIndex":%d,"width":"0.5 cm","height":"1 cm"}`, sk.SketchIndex), &rect)

	var added wire.AssemblyFeatureResult
	addArgs := fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"distance":0.6,"operation":"difference"}`, sk.SketchIndex)
	call(t, r, s, "assemblyFeatures.addExtrude", addArgs, &added)
	if len(added.Feature.Scalars) != 1 || added.Feature.Scalars[0].Label != "Distance" {
		t.Fatalf("addExtrude scalars = %+v, want one editable Distance scalar", added.Feature.Scalars)
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.7) > 1e-6 {
		t.Fatalf("pre-edit volume = %g, want 0.7 (0.5×1×0.6 pocket)", got)
	}

	// Deepen the pocket from 0.6 to 0.8: the participant loses more material.
	var edited wire.AssemblyFeatureResult
	editArgs := fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"0.8 cm"}]}`, added.Feature.ID)
	call(t, r, s, "assemblyFeatures.edit", editArgs, &edited)
	if edited.Feature.ID != added.Feature.ID {
		t.Fatalf("edit returned feature %d, want %d", edited.Feature.ID, added.Feature.ID)
	}
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.6) > 1e-6 {
		t.Errorf("post-edit volume = %g, want 0.6 (deeper 0.5×1×0.8 pocket)", got)
	}

	// The box cut bakes its tool at construction, so it exposes nothing editable.
	var cut wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.add", topHalfCut(), &cut)
	if len(cut.Feature.Scalars) != 0 {
		t.Errorf("box cut scalars = %+v, want none", cut.Feature.Scalars)
	}
	if _, err := r.Handle(s, "assemblyFeatures.edit", []byte(fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"1 cm"}]}`, cut.Feature.ID))); err == nil {
		t.Error("editing a box cut (no editable scalars) should fail")
	}
	if _, err := r.Handle(s, "assemblyFeatures.edit", []byte(fmt.Sprintf(`{"id":%d,"scalars":[{"index":5,"value":"1 cm"}]}`, added.Feature.ID))); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("out-of-range scalar index err = %v, want 'out of range'", err)
	}
}

// TestAssemblyRevolveOverWire drives the assembly revolve over the wire: a rectangle
// profile authored on the assembly's XY sketch (x∈[0,0.5], y∈[0,1]) revolved a full turn
// about the world Y axis is a cylinder (r=0.5) whose first quadrant cuts a quarter
// cylinder (≈π/16) from the unit-box participant — so it removes material without
// consuming the body. (The tight analytic gate is the model unit test; the boolean
// re-facets the small-radius cylinder coarsely here.)
func TestAssemblyRevolveOverWire(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
		v += query.BodyGeometryProperties(b, ops.DefaultQuality()).Volume
	}
	return v
}

// TestAssemblySetParticipantPathsOverWire restricts a feature to one placement of a
// sub-assembly placed twice; only that placement is machined.
func TestAssemblySetParticipantPathsOverWire(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestAssemblyHoleEditOverWire checks a placed assembly hole advertises editable
// Diameter/Depth scalars and that assemblyFeatures.edit re-dimensions it, re-drilling the
// participant — a wider bore removes more material (#752).
func TestAssemblyHoleEditOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var added wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.addHole", `{"center":[0.5,0.5,0],"axis":[0,0,1],"diameter":0.3,"depth":1.5}`, &added)
	if len(added.Feature.Scalars) != 2 || added.Feature.Scalars[0].Label != "Diameter" || added.Feature.Scalars[1].Label != "Depth" {
		t.Fatalf("hole scalars = %+v, want editable Diameter and Depth", added.Feature.Scalars)
	}
	narrow := featureResultVolume(asm, occs[0])

	// Widen the bore (scalar 0 = Diameter) and confirm more material is removed.
	var edited wire.AssemblyFeatureResult
	call(t, r, s, "assemblyFeatures.edit", fmt.Sprintf(`{"id":%d,"scalars":[{"index":0,"value":"0.6 cm"}]}`, added.Feature.ID), &edited)
	if wide := featureResultVolume(asm, occs[0]); wide >= narrow {
		t.Errorf("widening the bore should remove more material: narrow=%g wide=%g", narrow, wide)
	}
}

// verticalBoxEdgeKey returns the reference key of a vertical edge (z-running) of the
// occurrence's box component — a component-local key the assembly dress-up resolves per
// placement.
func verticalBoxEdgeKey(t *testing.T, occ *occurrence.Occurrence) string {
	t.Helper()
	def, ok := occ.Definition().(interface{ SurfaceBodies() *topo.SurfaceBodies })
	if !ok || len(def.SurfaceBodies().All()) == 0 {
		t.Fatal("occurrence component has no body")
	}
	for _, e := range def.SurfaceBodies().All()[0].Edges() {
		if a, b := e.StartVertex().Point(), e.EndVertex().Point(); a.X == b.X && a.Y == b.Y {
			return string(e.ReferenceKey())
		}
	}
	t.Fatal("no vertical edge on the box component")
	return ""
}

// TestAssemblyChamferOverWire chamfers a component edge and confirms EVERY placed instance
// of that component is machined from the one feature — the occurrence-relative resolution
// (#735). Two unit boxes share the component lineage, so both lose the chamfer wedge.
func TestAssemblyChamferOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0, 5)
	edge := verticalBoxEdgeKey(t, occs[0])

	var added wire.AssemblyFeatureResult
	args := mustJSON(t, wire.AddAssemblyChamferArgs{Edges: []wire.AssemblyEdgeRef{{Occurrence: occs[0].ID(), Edge: edge}}, Distance: 0.2})
	call(t, r, s, "assemblyFeatures.addChamfer", args, &added)
	if added.Feature.Kind != "assemblyChamfer" {
		t.Fatalf("feature kind = %q, want assemblyChamfer", added.Feature.Kind)
	}
	// A 45° flat chamfer of setback 0.2 on a unit-length edge removes a 0.2²/2 prism ⇒ 0.98.
	for i, occ := range occs {
		if got := featureResultVolume(asm, occ); stdmath.Abs(got-0.98) > 1e-6 {
			t.Errorf("participant %d chamfered volume = %g, want 0.98 (box minus a 0.2 chamfer wedge)", i, got)
		}
	}
	// The chamfer advertises an editable Distance scalar.
	if len(added.Feature.Scalars) != 1 || added.Feature.Scalars[0].Label != "Distance" {
		t.Errorf("chamfer scalars = %+v, want one Distance scalar", added.Feature.Scalars)
	}
}

// TestAssemblyFilletOverWire rounds a component edge on every participant, gated below the
// box volume (a fillet removes the convex corner material).
func TestAssemblyFilletOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)
	edge := verticalBoxEdgeKey(t, occs[0])

	var added wire.AssemblyFeatureResult
	args := mustJSON(t, wire.AddAssemblyFilletArgs{Edges: []wire.AssemblyEdgeRef{{Occurrence: occs[0].ID(), Edge: edge}}, Radius: 0.2})
	call(t, r, s, "assemblyFeatures.addFillet", args, &added)
	if added.Feature.Kind != "assemblyFillet" {
		t.Fatalf("feature kind = %q, want assemblyFillet", added.Feature.Kind)
	}
	got := featureResultVolume(asm, occs[0])
	if got >= 1.0 || got < 0.9 {
		t.Errorf("filleted volume = %g, want a unit box minus a small rounded corner (≈0.99)", got)
	}

	// An unknown edge key is rejected.
	if _, err := r.Handle(s, "assemblyFeatures.addChamfer", []byte(fmt.Sprintf(`{"edges":[{"occurrence":%d,"edge":"bogus"}],"distance":0.1}`, occs[0].ID()))); err == nil {
		t.Error("addChamfer with an unknown edge key should fail")
	}
}

// topBoxFaceKey returns the reference key of the box component's top face (highest z),
// whose +z normal makes a +z move-face grow the box.
func topBoxFaceKey(t *testing.T, occ *occurrence.Occurrence) string {
	t.Helper()
	def := occ.Definition().(interface{ SurfaceBodies() *topo.SurfaceBodies })
	var top *topo.Face
	for _, f := range def.SurfaceBodies().All()[0].Faces() {
		if top == nil || f.RangeBox().Center().Z > top.RangeBox().Center().Z {
			top = f
		}
	}
	return string(top.ReferenceKey())
}

// TestAssemblyMoveFaceOverWire translates a component's top face on every participant and
// gates the grown volume (#735): pushing the unit box's top face +0.5z makes a 1.5 box, on
// both placed instances from one feature.
func TestAssemblyMoveFaceOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0, 5)
	face := topBoxFaceKey(t, occs[0])

	var added wire.AssemblyFeatureResult
	args := mustJSON(t, wire.AddAssemblyMoveFaceArgs{Faces: []wire.AssemblyFaceRef{{Occurrence: occs[0].ID(), Face: face}}, Translation: [3]float64{0, 0, 0.5}})
	call(t, r, s, "assemblyFeatures.addMoveFace", args, &added)
	if added.Feature.Kind != "assemblyMoveFace" {
		t.Fatalf("feature kind = %q, want assemblyMoveFace", added.Feature.Kind)
	}
	for i, occ := range occs {
		if got := featureResultVolume(asm, occ); stdmath.Abs(got-1.5) > 1e-6 {
			t.Errorf("participant %d moved-face volume = %g, want 1.5 (top face pushed +0.5z)", i, got)
		}
	}
	if len(added.Feature.Scalars) != 1 || added.Feature.Scalars[0].Label != "Distance" {
		t.Errorf("move-face scalars = %+v, want one editable Distance scalar", added.Feature.Scalars)
	}
}

// TestAssemblySweepOverWire sweeps an assembly sketch profile along an explicit polyline
// path into the participant (#735). A difference-sweep removes a channel (volume drops
// below the box and stays a valid solid); the assembly-sketch profile + path wiring is what
// this gates (the swept-solid geometry itself is covered by the part sweep tests).
func TestAssemblySweepOverWire(t *testing.T) {
	t.Parallel()
	r, s, asm, occs := assemblySessionWithBoxes(t, 0)

	var sk wire.CreateSketchResult
	call(t, r, s, "sketch.create", `{"plane":"XY"}`, &sk)
	var rect wire.SketchRectangleResult
	call(t, r, s, "sketch.rectangle", fmt.Sprintf(`{"sketchIndex":%d,"width":"0.5 cm","height":"1 cm"}`, sk.SketchIndex), &rect)

	var added wire.AssemblyFeatureResult
	args := fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"path":[[0.25,0.5,0],[0.25,0.5,0.6]],"operation":"difference"}`, sk.SketchIndex)
	call(t, r, s, "assemblyFeatures.addSweep", args, &added)
	if added.Feature.Kind != "assemblySweep" {
		t.Fatalf("feature kind = %q, want assemblySweep", added.Feature.Kind)
	}
	// The path starts at the profile centroid (0.25,0.5,0) and runs +0.6z, so the swept
	// channel is the 0.5×1 profile dragged straight down — the same 0.3 cut an extrude makes.
	if got := featureResultVolume(asm, occs[0]); stdmath.Abs(got-0.7) > 1e-6 {
		t.Errorf("swept-cut volume = %g, want 0.7 (unit box minus a 0.5×1×0.6 channel)", got)
	}

	// A single-point path is rejected.
	if _, err := r.Handle(s, "assemblyFeatures.addSweep", []byte(fmt.Sprintf(`{"sketchIndex":%d,"profileIndex":0,"path":[[0,0,0]],"operation":"difference"}`, sk.SketchIndex))); err == nil {
		t.Error("addSweep with a single-point path should fail")
	}
}
