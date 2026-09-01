// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// Moving the top face of a block outward grows its volume.
func TestMoveFaceTool(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	def := activePartDef(t, s)
	vol0 := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume

	tool := NewMoveFaceTool()
	tool.dz = 2
	s.StartTool(tool)
	s.Click(100, 100) // pick the top face
	if err := s.OK(); err != nil {
		t.Fatalf("move face OK: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("move-face tool should deactivate after OK")
	}
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid {
		t.Fatalf("moved body invalid: %+v", r)
	}
	vol1 := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if vol1 <= vol0 {
		t.Fatalf("moving top face +Z should grow volume: %g → %g", vol0, vol1)
	}
}

// Combining two bodies (Join) yields a single body.
func TestCombineTool(t *testing.T) {
	t.Parallel()
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool() // default 2×1 → a second body
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("setup wanted 2 bodies, got %d", def.SurfaceBodies().Count())
	}

	tool := NewCombineTool() // Join
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(1)})
	if err := s.OK(); err != nil {
		t.Fatalf("combine OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("after join = %d bodies, want 1", def.SurfaceBodies().Count())
	}
}

// TestCombineToolCombinesMultipleTools is the #2069 regression: the tool must boolean the target
// against EVERY picked tool body in one feature, not just the first — before this it consumed only
// the second pick and ignored the rest.
func TestCombineToolCombinesMultipleTools(t *testing.T) {
	t.Parallel()
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool()
	pat.countX = 3 // target + two tool copies → 3 bodies in a row
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 3 {
		t.Fatalf("setup wanted 3 bodies, got %d", def.SurfaceBodies().Count())
	}

	tool := NewCombineTool() // Join
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(1)})
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(2)})
	if err := s.OK(); err != nil {
		t.Fatalf("combine OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Fatalf("joining a target + two tools = %d bodies, want 1", def.SurfaceBodies().Count())
	}
}

// TestCombineToolKeepsToolBodies is the #2069 regression for keep-tool-bodies: with the checkbox on,
// the tool body must survive the boolean instead of being consumed. It also asserts the checkbox is
// exposed as a BoolParam so the dialog can drive it.
func TestCombineToolKeepsToolBodies(t *testing.T) {
	t.Parallel()
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool() // 2×1 → target + one tool
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("setup wanted 2 bodies, got %d", def.SurfaceBodies().Count())
	}

	tool := NewCombineTool()
	keep, ok := boolParam(tool.Params(), "Keep tool bodies")
	if !ok {
		t.Fatal("Combine has no 'Keep tool bodies' checkbox — the dialog cannot reach KeepToolBodies (#2069)")
	}
	keep.Set(true) // drive it through the dialog's BoolParam, not the setter directly
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(1)})
	if err := s.OK(); err != nil {
		t.Fatalf("combine OK: %v", err)
	}
	// Keep-tool-bodies drops only the target and appends the result, so the kept tool remains: 2
	// bodies, not the 1 a consuming join leaves.
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("keep-tool-bodies join = %d bodies, want 2 (result + kept tool)", def.SurfaceBodies().Count())
	}
}

// boolParam finds a BoolParam by label in a tool's Params.
func boolParam(p ToolParams, label string) (BoolParam, bool) {
	for _, b := range p.Bools {
		if b.Label == label {
			return b, true
		}
	}
	return BoolParam{}, false
}

// TestCombineToolOperationIsChoice is the #1803 regression: the Combine "Operation" input
// must be a ChoiceParam (a named dropdown), NOT an IntParam whose long self-documenting
// label overflowed the 95px column and collided with the InputInt steppers into garble. It
// asserts a short "Operation" label, the three named options in enum order, and that Set
// routes into the tool's operation.
func TestCombineToolOperationIsChoice(t *testing.T) {
	t.Parallel()
	tool := NewCombineTool()
	p := tool.Params()
	if len(p.Ints) != 0 {
		t.Errorf("Combine still exposes %d IntParam(s) — the long-label overflow (#1803) is back", len(p.Ints))
	}
	if len(p.Choices) != 1 {
		t.Fatalf("Combine Choices = %d, want 1 (the Operation dropdown)", len(p.Choices))
	}
	op := p.Choices[0]
	if op.Label != "Operation" {
		t.Errorf("Operation label = %q, want the short %q", op.Label, "Operation")
	}
	if want := []string{"Join", "Cut", "Intersect"}; !equalStrings(op.Options, want) {
		t.Errorf("Operation options = %v, want %v (enum order Join=0/Cut=1/Intersect=2)", op.Options, want)
	}
	op.Set(int(ops.Intersect))
	if op.Get() != int(ops.Intersect) {
		t.Errorf("after Set(Intersect) Get = %d, want %d", op.Get(), int(ops.Intersect))
	}
}

// Moving a body translates it.
func TestMoveBodyTool(t *testing.T) {
	t.Parallel()
	s, def, _ := extrudedPart(t)
	minX0 := float64(def.SurfaceBodies().Item(0).RangeBox().Min.X)

	tool := NewMoveBodyTool()
	tool.dx = 10
	s.StartTool(tool)
	s.feedPick(BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if err := s.OK(); err != nil {
		t.Fatalf("move body OK: %v", err)
	}
	minX1 := float64(def.SurfaceBodies().Item(0).RangeBox().Min.X)
	if math.Abs(minX1-(minX0+10)) > 1e-6 {
		t.Fatalf("body min.X = %g, want %g (shifted by 10)", minX1, minX0+10)
	}
}

// TestModifyToolsDraftFeature asserts Move Face, Combine and Move Bodies build the draft
// the commit gate inspects (#1626): no draft before the tool is commit-ready, a non-nil
// draft once it is. Combine and Move resolve their body operands from the session at
// draft time, exactly as Commit does.
func TestModifyToolsDraftFeature(t *testing.T) {
	t.Parallel()
	s, def, src := extrudedPart(t)
	pat := NewFeatureRectPatternTool() // default 2×1 → a second body for Combine
	s.StartTool(pat)
	s.feedPick(src)
	if err := s.OK(); err != nil {
		t.Fatalf("pattern setup OK: %v", err)
	}

	moveFace := NewMoveFaceTool()
	moveFace.Pick(s, FaceHandle{Face: topFaceOf(t, def.SurfaceBodies().Item(0)), Body: def.SurfaceBodies().Item(0)})
	if _, ok := moveFace.DraftFeature(s); ok {
		t.Error("move face: draft ready with a zero move vector")
	}
	moveFace.dz = 2
	if draft, ok := moveFace.DraftFeature(s); !ok || draft == nil {
		t.Errorf("move face: no draft once commit-ready (ok=%v)", ok)
	}

	combine := NewCombineTool()
	combine.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if _, ok := combine.DraftFeature(s); ok {
		t.Error("combine: draft ready with one body picked")
	}
	combine.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(1)})
	if draft, ok := combine.DraftFeature(s); !ok || draft == nil {
		t.Errorf("combine: no draft once commit-ready (ok=%v)", ok)
	}

	moveBody := NewMoveBodyTool()
	moveBody.Pick(s, BodyHandle{Body: def.SurfaceBodies().Item(0)})
	if _, ok := moveBody.DraftFeature(s); ok {
		t.Error("move bodies: draft ready with a zero move vector")
	}
	moveBody.dx = 10
	if draft, ok := moveBody.DraftFeature(s); !ok || draft == nil {
		t.Errorf("move bodies: no draft once commit-ready (ok=%v)", ok)
	}
}

func TestDirectEditCommandsRegistered(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	for _, id := range []string{"Modify.Combine", "Modify.MoveFace", "Modify.MoveBodies"} {
		if _, ok := s.Commands().ByID(id); !ok {
			t.Errorf("command %q not registered", id)
		}
	}
}
