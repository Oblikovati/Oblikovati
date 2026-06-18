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

// TestCenterMarkToolMarksHoles: the centre-mark tool places a crosshair at a base view's circular
// edge (a cylinder rim → one mark after rim dedup).
func TestCenterMarkToolMarksHoles(t *testing.T) {
	s := drawingWithCylinderSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.Params().Choices[0].Set(1) // Top, so the cylinder's rim projects as a circle
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}
	tool := NewCenterMarkTool()
	tool.Start(s)
	if tool.Name() != "Center Mark" || !tool.CanCommit() {
		t.Fatalf("centre-mark tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.CenterMarkAnnotation {
		t.Fatalf("centre mark not added (count=%d)", an.Count())
	}
}

// TestCenterlineToolAddsCenterlines: the centerline tool adds a dash-dot cross on a base view.
func TestCenterlineToolAddsCenterlines(t *testing.T) {
	s := drawingWithFrontBase(t)
	c, _ := ActiveDrawing(s)
	tool := NewCenterlineTool()
	tool.Start(s)
	if tool.Name() != "Centerline" || !tool.CanCommit() {
		t.Fatalf("centerline tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.CenterlineAnnotation || an.Item(0).CurveCount() < 4 {
		t.Fatalf("centerlines not added (count=%d)", an.Count())
	}
}

// TestFeatureControlFrameToolDropsFrame: the FCF tool drops a frame at the placed point with the
// chosen characteristic, tolerance and datums.
func TestFeatureControlFrameToolDropsFrame(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewFeatureControlFrameTool()
	tool.Start(s)
	tool.SetPlacement(80, 80)
	tool.Params().Texts[0].Set("0.2")  // Tolerance
	tool.Params().Texts[1].Set("A, B") // Datums
	tool.Params().Choices[0].Set(1)    // Characteristic = Flatness
	if !tool.CanCommit() {
		t.Fatal("FCF tool cannot commit with a tolerance set")
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.FeatureControlFrameAnnotation {
		t.Fatalf("FCF not added (count=%d)", an.Count())
	}
	if n := len(an.Item(0).Labels()); n != 3 { // tolerance + 2 datums (flatness symbol is drawn)
		t.Errorf("FCF labels = %d, want 3", n)
	}
}

// TestDatumFeatureToolDropsSymbol: the datum tool drops a datum feature symbol at the placed point.
func TestDatumFeatureToolDropsSymbol(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewDatumFeatureTool()
	tool.Start(s)
	tool.SetPlacement(90, 90)
	tool.Params().Texts[0].Set("B")
	if tool.Name() != "Datum Feature" || !tool.CanCommit() {
		t.Fatalf("datum tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.DatumFeatureAnnotation || an.Item(0).Tag() != "B" {
		t.Fatalf("datum not added (count=%d)", an.Count())
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
