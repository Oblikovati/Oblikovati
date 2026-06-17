// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
)

func TestCoGMarkerToolMarksView(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	tool := NewCoGMarkerTool()
	tool.Start(s)
	if tool.Name() != "Center of Gravity" || !tool.CanCommit() {
		t.Fatalf("CoG tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	tool.Pick(s, nil)
	tool.Cancel(s)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.CoGMarkerAnnotation {
		t.Fatalf("CoG marker not added (count=%d)", an.Count())
	}
}

func TestCoGMarkerToolWithoutView(t *testing.T) {
	s := drawingWithModelSession(t) // no base view added
	tool := NewCoGMarkerTool()
	tool.Start(s)
	if tool.CanCommit() {
		t.Error("CoG tool can commit with no view, want it disabled")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no view = ok, want error")
	}
}

func TestRevisionCloudToolDropsCloud(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewRevisionCloudTool()
	tool.Start(s)
	if tool.Name() != "Revision Cloud" || !tool.CanCommit() {
		t.Fatalf("cloud tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	tool.Params().Floats[0].Set(70)
	tool.Pick(s, nil)
	tool.Cancel(s)
	if got := tool.PreviewCurves(s); len(got) != 4 {
		t.Errorf("cloud preview = %d curves, want a 4-edge region outline", len(got))
	}
	tool.SetPlacement(150, 150)
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.RevisionCloudAnnotation {
		t.Fatalf("revision cloud not added (count=%d)", an.Count())
	}
}
