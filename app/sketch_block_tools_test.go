// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestCreateBlockToolEndToEnd: the ribbon command starts the tool; selection +
// name commit into a part-level definition with the replacing instance
// (M06-F07, #622).
func TestCreateBlockToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	sk.Lines().AddByTwoPoints(math.P2(1, 0), math.P2(1, 1))

	if err := s.Execute("Sketch.CreateBlock"); err != nil {
		t.Fatalf("Execute Sketch.CreateBlock: %v", err)
	}
	tool, ok := s.ActiveTool().Tool().(*SketchCreateBlockTool)
	if !ok {
		t.Fatalf("active tool = %T, want *SketchCreateBlockTool", s.ActiveTool().Tool())
	}
	tool.PickSnap(sk.Lines().Item(0), SnapResult{})
	tool.PickSnap(sk.Lines().Item(1), SnapResult{})
	if tool.CanCommit() {
		t.Fatal("the tool must not commit without a block name")
	}
	tool.SetBlockName("corner")
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	blockDef, ok := def.SketchBlocks().ByName("corner")
	if !ok || blockDef.EntityCount() != 2 {
		t.Fatalf("definition lookup = %v, want corner with 2 entities", ok)
	}
	if sk.Blocks().InstanceCount() != 1 {
		t.Errorf("instances = %d, want the replacing one", sk.Blocks().InstanceCount())
	}
}

// TestPlaceBlockToolEndToEnd: the placement tool stamps an instance at each
// clicked insertion point with the dialog's rotation/scale.
func TestPlaceBlockToolEndToEnd(t *testing.T) {
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	l := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if _, _, err := sk.Blocks().CreateFromSelection(def.SketchBlocks(), "rib", []sketch.Entity{l}); err != nil {
		t.Fatalf("CreateFromSelection: %v", err)
	}

	tool := NewSketchPlaceBlockTool("rib")
	if tool.CanCommit() {
		t.Fatal("the tool must wait for an insertion click")
	}
	tool.ClickAt(s, 5, 3)
	if !tool.CanCommit() {
		t.Fatal("the tool must commit after the insertion click")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if got := sk.Blocks().InstanceCount(); got != 2 {
		t.Fatalf("instances = %d, want the replacing + the placed one", got)
	}
	placed := sk.Blocks().Item(1)
	if origin := placed.Transform().TransformPoint(math.P2(0, 0)); !origin.IsEqualTo(math.P2(5, 3), 1e-9) {
		t.Errorf("placed origin = %v, want the (5, 3) click", origin)
	}

	missing := NewSketchPlaceBlockTool("ghost")
	missing.ClickAt(s, 0, 0)
	if err := missing.Commit(s); err == nil {
		t.Error("placing an unknown definition must fail")
	}
}
