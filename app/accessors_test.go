// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/event"
)

func TestCommandMetadataAccessors(t *testing.T) {
	c := NewCommand("id", "Disp", "Cat", func(*Session) error { return nil }).
		WithAlias("X").WithKind(ToggleControl).WithTooltip("tip")
	if c.ID() != "id" || c.DisplayName() != "Disp" || c.Category() != "Cat" ||
		c.Alias() != "X" || c.Kind() != ToggleControl || c.Tooltip() != "tip" {
		t.Error("command metadata accessors wrong")
	}
	m := NewCommandManager()
	_ = m.Add(c)
	if len(m.All()) != 1 {
		t.Error("All() wrong")
	}
}

func TestSelectionHandleKindsAndItems(t *testing.T) {
	handles := []Selectable{
		FaceHandle{},
		EdgeHandle{},
		VertexHandle{},
		BodyHandle{},
		FeatureHandle{},
		ProfileHandle{},
		SketchEntityHandle{},
	}
	wantKinds := []SelectionKind{SelectFace, SelectEdge, SelectVertex, SelectBody, SelectFeature, SelectProfile, SelectSketchEntity}
	sel := NewSelection()
	for i, h := range handles {
		if h.SelectionKind() != wantKinds[i] {
			t.Errorf("handle %d kind = %v, want %v", i, h.SelectionKind(), wantKinds[i])
		}
		sel.Add(h)
	}
	if len(sel.Items()) != len(handles) {
		t.Errorf("Items len = %d, want %d", len(sel.Items()), len(handles))
	}
	if NewSelection().First() != nil {
		t.Error("empty selection First should be nil")
	}
}

func TestEventIDsStable(t *testing.T) {
	ids := map[event.Event]event.TypeID{
		CommandStarted{}:   0x0501,
		CommandEnded{}:     0x0502,
		SelectionChanged{}: 0x0503,
	}
	for e, want := range ids {
		if e.EventID() != want {
			t.Errorf("%T EventID = %#x, want %#x", e, e.EventID(), want)
		}
	}
}

func TestExtrudeToolAccessors(t *testing.T) {
	ext := NewExtrudeTool()
	if ext.Name() != "Extrude" || ext.AddedFeature() != nil {
		t.Error("extrude tool accessors wrong before commit")
	}
	s := NewSession()
	ext.Cancel(s) // restores default filter; must not panic
	if !s.Selection().Filter().Accepts(SelectFace) {
		t.Error("cancel did not restore an all-accepting filter")
	}
	// ToolInstance.Name
	st := &stubTool{}
	s.StartTool(st)
	if s.ActiveTool().Name() != "Stub" {
		t.Error("ToolInstance.Name wrong")
	}
}
