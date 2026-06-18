// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
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

// TestSurfaceTextureToolDropsSymbol: the surface-texture tool drops a checkmark symbol at the
// placed point with the chosen roughness and variant.
func TestSurfaceTextureToolDropsSymbol(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewSurfaceTextureTool()
	tool.Start(s)
	tool.SetPlacement(85, 85)
	tool.Params().Texts[0].Set("3.2") // Roughness
	tool.Params().Choices[0].Set(2)   // Material removal = Prohibited
	if tool.Name() != "Surface Texture" || !tool.CanCommit() {
		t.Fatalf("surface tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.SurfaceTextureAnnotation {
		t.Fatalf("surface texture not added (count=%d)", an.Count())
	}
	if got := an.Item(0).Labels(); len(got) != 1 || got[0].Text != "3.2" {
		t.Errorf("surface texture labels = %v, want roughness 3.2", got)
	}
}

// TestPartsListToolDropsTable: the parts-list tool drops a table on a drawing referencing an
// assembly, with one row per parts-only BOM item.
func TestPartsListToolDropsTable(t *testing.T) {
	s := NewSession()
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("commands: %v", err)
	}
	asmDoc, err := compdef.AddAssembly(s.Workspace(), "asm.obk", true)
	if err != nil {
		t.Fatalf("AddAssembly: %v", err)
	}
	asm := asmDoc.Content().(*compdef.AssemblyComponentDefinition)
	for i, name := range []string{"p1.opd", "p2.opd"} { // two distinct parts → two BOM rows
		p, err := compdef.AddPart(s.Workspace(), name, true)
		if err != nil {
			t.Fatalf("AddPart: %v", err)
		}
		asm.Place(fmt.Sprintf("c:%d", i+1), p.Content().(*compdef.PartComponentDefinition), math.Identity4())
	}
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		t.Fatalf("activate assembly: %v", err)
	}
	if _, err := s.NewDrawing(); err != nil {
		t.Fatalf("NewDrawing: %v", err)
	}
	c, err := ActiveDrawing(s)
	if err != nil {
		t.Fatalf("ActiveDrawing: %v", err)
	}
	c.SetModelReference("asm.obk")

	tool := NewPartsListTool()
	tool.Start(s)
	tool.SetPlacement(40, 260)
	if tool.Name() != "Parts List" || !tool.CanCommit() {
		t.Fatalf("parts-list tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.PartsListAnnotation || an.Item(0).RowCount() != 2 {
		t.Fatalf("parts list not added with 2 rows (count=%d)", an.Count())
	}
}

// TestBalloonToolDropsBalloon: the balloon tool drops a circled item number with a leader.
func TestBalloonToolDropsBalloon(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewBalloonTool()
	tool.Start(s)
	tool.SetPlacement(100, 200)
	tool.Params().Ints[0].Set(5)     // Item
	tool.Params().Floats[0].Set(130) // Leader X
	tool.Params().Floats[1].Set(180) // Leader Y
	if tool.Name() != "Balloon" || !tool.CanCommit() {
		t.Fatalf("balloon tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.BalloonAnnotation || an.Item(0).Tag() != "5" {
		t.Fatalf("balloon not added (count=%d)", an.Count())
	}
}

// TestHoleTableToolDropsTable: the hole-table tool drops a table listing a base view's holes.
func TestHoleTableToolDropsTable(t *testing.T) {
	s := drawingWithCylinderSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.Params().Choices[0].Set(1) // Top, so the cylinder's rim projects as a circle
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}
	tool := NewHoleTableTool()
	tool.Start(s)
	tool.SetPlacement(220, 240)
	if tool.Name() != "Hole Table" || !tool.CanCommit() {
		t.Fatalf("hole-table tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	c, _ := ActiveDrawing(s)
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.HoleTableAnnotation || an.Item(0).RowCount() != 1 {
		t.Fatalf("hole table not added (count=%d, rows=%d)", an.Count(), an.Item(0).RowCount())
	}
}

// TestHoleTableToolWithoutView: the tool is inert and errors with no base view to read holes from.
func TestHoleTableToolWithoutView(t *testing.T) {
	s := drawingWithModelSession(t) // no base view added
	tool := NewHoleTableTool()
	tool.Start(s)
	if tool.CanCommit() {
		t.Error("hole-table tool can commit with no view, want it disabled")
	}
	if err := tool.Commit(s); err == nil {
		t.Error("Commit with no base view should error")
	}
}

// TestRevisionTableToolDropsTable: the revision-table tool drops a one-row table seeded with its
// revision fields.
func TestRevisionTableToolDropsTable(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewRevisionTableTool()
	tool.Start(s)
	tool.SetPlacement(250, 60)
	tool.Params().Texts[0].Set("C")             // Revision
	tool.Params().Texts[2].Set("Tightened fit") // Description
	if tool.Name() != "Revision Table" || !tool.CanCommit() {
		t.Fatalf("revision-table tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.RevisionTableAnnotation || an.Item(0).RowCount() != 1 {
		t.Fatalf("revision table not added (count=%d)", an.Count())
	}
}

// TestRevisionTagToolDropsTag: the revision-tag tool drops a triangle holding the revision letter.
func TestRevisionTagToolDropsTag(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewRevisionTagTool()
	tool.Start(s)
	tool.SetPlacement(120, 90)
	tool.Params().Texts[0].Set("C")
	if tool.Name() != "Revision Tag" || !tool.CanCommit() {
		t.Fatalf("revision-tag tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	an := c.Sheets().Active().Annotations()
	if an.Count() != 1 || an.Item(0).Kind() != types.RevisionTagAnnotation || an.Item(0).Tag() != "C" {
		t.Fatalf("revision tag not added (count=%d)", an.Count())
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
