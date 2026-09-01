// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
)

// TestDecalToolEndToEnd drives the Decal UI: pick a face, set an image, OK — and asserts the
// decal feature is recorded (cosmetic: the body is unchanged).
func TestDecalToolEndToEnd(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	top := topFaceOf(t, block)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: top, Body: block}})
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	before := def.SurfaceBodies().Count()

	decal := NewDecalTool()
	s.StartTool(decal)
	s.Click(100, 100)
	decal.SetImage("logo.png")
	if !decal.CanCommit() {
		t.Fatal("decal not ready after face + image")
	}
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if decal.AddedFeature() == nil {
		t.Error("decal feature not created")
	}
	if def.SurfaceBodies().Count() != before {
		t.Errorf("decal changed body count %d→%d; a decal is cosmetic", before, def.SurfaceBodies().Count())
	}
}

// TestDecalToolDraftFeature asserts the decal tool builds the draft the commit gate
// inspects (#1626): no draft until both the face and the image are set, a non-nil draft
// once commit-ready (a decal is cosmetic, so the draft previews trivially healthy).
func TestDecalToolDraftFeature(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	tool := NewDecalTool()
	tool.Pick(s, FaceHandle{Face: topFaceOf(t, block), Body: block})
	if _, ok := tool.DraftFeature(s); ok {
		t.Error("decal: draft ready with no image set")
	}
	tool.SetImage("logo.png")
	if draft, ok := tool.DraftFeature(s); !ok || draft == nil {
		t.Errorf("decal: no draft once commit-ready (ok=%v)", ok)
	}
}

func TestDecalViaRibbonCommand(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	s.SetPicker(stubPicker{sel: FaceHandle{Face: topFaceOf(t, block), Body: block}})
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("register commands: %v", err)
	}
	if err := s.Execute("Create.Decal"); err != nil {
		t.Fatalf("execute Create.Decal: %v", err)
	}
	if _, ok := s.ActiveTool().Tool().(*DecalTool); !ok {
		t.Fatal("Decal command did not start the decal tool")
	}
}
