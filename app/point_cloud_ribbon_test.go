// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// pointCloudButton returns the named button from the 3D Model ▸ Point Cloud panel and whether it
// is currently enabled.
func pointCloudButton(t *testing.T, s *Session, name string) (RibbonButton, bool) {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab(tab3DModel)
	if !ok {
		t.Fatal("ribbon has no 3D Model tab")
	}
	panel, ok := tab.Panel("Point Cloud")
	if !ok {
		t.Fatal("3D Model tab has no Point Cloud panel")
	}
	for _, b := range panel.Buttons {
		if b.Command.DisplayName() == name {
			return b, true
		}
	}
	return RibbonButton{}, false
}

// TestPointCloudRibbonButtonsWiredAndEnable verifies the Point Cloud panel's buttons exist with
// icons, run their commands, and enable only in the right context — so clicking them in the ribbon
// actually drives the feature (#645).
func TestPointCloudRibbonButtonsWiredAndEnable(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}

	for _, name := range []string{"Import Point Cloud", "Fit Work Plane", "Work Point", "Crop Box", "Move"} {
		b, ok := pointCloudButton(t, s, name)
		if !ok {
			t.Fatalf("Point Cloud panel is missing the %q button", name)
		}
		if b.Command.Icon() == "" {
			t.Errorf("%q has no icon (renders as a blank/text button)", name)
		}
	}

	// Context-sensitive buttons start disabled (nothing selected) except Import (just needs a part).
	mustEnabled(t, s, "Import Point Cloud", true)
	mustEnabled(t, s, "Fit Work Plane", false)
	mustEnabled(t, s, "Crop Box", false)
	mustEnabled(t, s, "Work Point", false)
	mustEnabled(t, s, "Move", false)

	// Attaching and selecting a cloud enables the cloud-level commands.
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(0, 0, 5), math.P3(2, 0, 5), math.P3(0, 2, 5)})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	mustEnabled(t, s, "Fit Work Plane", true)
	mustEnabled(t, s, "Crop Box", true)
	mustEnabled(t, s, "Move", true)
	mustEnabled(t, s, "Work Point", false) // needs a snapped scan point, not the cloud

	// Selecting a snapped scan point enables Work Point.
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(0, 0, 5)})
	mustEnabled(t, s, "Work Point", true)

	// Executing through the command registry (the ribbon's click path) runs the feature.
	if err := s.Execute("PointCloud.WorkPoint"); err != nil {
		t.Errorf("Execute(PointCloud.WorkPoint): %v", err)
	}
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})
	if err := s.Execute("PointCloud.FitPlane"); err != nil {
		t.Errorf("Execute(PointCloud.FitPlane): %v", err)
	}
	if err := s.Execute("PointCloud.CropBox"); err != nil { // starts the crop tool
		t.Errorf("Execute(PointCloud.CropBox): %v", err)
	}
}

// TestSketchProjectScanPointButtonWired verifies the Sketch ▸ Create ▸ Project Scan Point button is
// wired and enables only in a sketch with a scan point selected (#645).
func TestSketchProjectScanPointButtonWired(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, _ := def.PointClouds().Add("Scan", "s.xyz", rid, []math.Point3{math.P3(1, 2, 3)})

	cmd, ok := s.Commands().ByID("Sketch.ProjectScanPoint")
	if !ok {
		t.Fatal("Sketch.ProjectScanPoint command is not registered")
	}
	if cmd.Icon() == "" {
		t.Error("Project Scan Point has no icon")
	}
	if cmd.IsEnabled(s) {
		t.Error("Project Scan Point should be disabled outside a sketch")
	}
	if _, err := s.CreateSketch(sketch.XYPlane()); err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	s.Select(PointCloudPointHandle{Cloud: pc, Point: math.P3(1, 2, 3)})
	if !cmd.IsEnabled(s) {
		t.Error("Project Scan Point should be enabled in a sketch with a scan point selected")
	}
	if err := s.Execute("Sketch.ProjectScanPoint"); err != nil {
		t.Errorf("Execute(Sketch.ProjectScanPoint): %v", err)
	}
}

func mustEnabled(t *testing.T, s *Session, name string, want bool) {
	t.Helper()
	b, ok := pointCloudButton(t, s, name)
	if !ok {
		t.Fatalf("no %q button", name)
	}
	if b.Enabled != want {
		t.Errorf("%q enabled = %v, want %v", name, b.Enabled, want)
	}
}
