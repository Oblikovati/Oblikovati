// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestShellToolEndToEnd drives the Shell UI: start the tool, click the block's top face
// (the opening), set the wall thickness, OK — and asserts the body is hollowed to a valid
// open solid of the right volume. Block 4×4×2 (vol 32); cavity [0.5,3.5]²×[0.5,2] =
// 3·3·1.5 = 13.5 ⇒ 18.5.
func TestShellToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})

	sh := NewShellTool()
	s.StartTool(sh)   // ribbon: click "Shell"
	s.Click(100, 100) // viewport: click the top face to open it
	sh.SetThickness(0.5)
	if !sh.CanCommit() {
		t.Fatal("shell tool not ready after face + thickness")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}

	def := activePartDef(t, s)
	body := def.SurfaceBodies().Item(0)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("shelled body not a valid solid: %+v", r)
	}
	want := 32.0 - 3*3*1.5
	if got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; relErrApp(got, want) > 1e-6 {
		t.Errorf("shell volume = %g, want %g", got, want)
	}
	if s.ActiveTool() != nil {
		t.Error("tool should have closed after OK")
	}
}

// TestShellViaRibbonCommand drives the Shell from its ribbon command.
func TestShellViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Modify.Shell"); err != nil {
		t.Fatalf("execute Modify.Shell: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*ShellTool); !ok {
		t.Fatal("Shell command did not start the shell tool")
	}
	s.Click(1, 1)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	def := activePartDef(t, s)
	if v := ops.BodyGeometryProperties(def.SurfaceBodies().Item(0), ops.DefaultQuality()).Volume; v >= 32 {
		t.Errorf("shell did not hollow the body: volume %g, want < 32", v)
	}
}

// TestShellToolNeedsFace checks the tool is not committable until a face is picked.
func TestShellToolNeedsFace(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	sh := NewShellTool()
	s.StartTool(sh)
	if sh.CanCommit() {
		t.Error("shell ready with no face picked")
	}
	s.Click(0, 0)
	if !sh.CanCommit() {
		t.Error("shell not ready after picking a face")
	}
}
