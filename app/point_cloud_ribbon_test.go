// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"reflect"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
	"oblikovati.org/model/sketch"
)

// pointCloudButton returns the named button from the 3D Model ▸ Point Cloud panel and whether it
// is currently enabled.
func pointCloudButton(t *testing.T, s *Session, name string) (RibbonButton, bool) {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
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

	modeTab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	modePanel, ok := modeTab.Panel(PanelPointCloud)
	if !ok || modePanel.Selector == nil {
		t.Fatal("Point Cloud panel is missing the folded-in display-mode selector")
	}
	if got := len(modePanel.Selector.Options); got != len(types.AllPointCloudDisplayModes()) {
		t.Fatalf("Point Cloud display-mode selector has %d options, want %d", got, len(types.AllPointCloudDisplayModes()))
	}
	if modePanel.Slider == nil {
		t.Fatal("Point Cloud controls are missing the Render Density slider")
	}
	if modePanel.Slider.Value != 100 || modePanel.Slider.Min != 0 || modePanel.Slider.Max != 100 {
		t.Fatalf("Render Density slider = %+v, want 0..100 at 100", modePanel.Slider)
	}
	if !modePanel.Slider.Percent {
		t.Fatal("Render Density slider should render as a percentage")
	}
	if modePanel.PointSizeSlider == nil {
		t.Fatal("Point Cloud controls are missing the Point Size slider")
	}
	if modePanel.PointSizeSlider.Value != 1 || modePanel.PointSizeSlider.Min != 1 || modePanel.PointSizeSlider.Max != 10 {
		t.Fatalf("Point Size slider = %+v, want 1..10 at 1", modePanel.PointSizeSlider)
	}
	if modePanel.IntensityRamp != nil {
		t.Fatal("intensity ramp should be hidden until the selected cloud is in intensity mode")
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
	if err := s.Execute("PointCloud.DisplayMode.RGB"); err != nil {
		t.Errorf("Execute(PointCloud.DisplayMode.RGB): %v", err)
	}
	if pc.DisplayMode() != types.PointCloudDisplayModeRGB {
		t.Errorf("display mode = %q, want rgb", pc.DisplayMode())
	}
	if err := s.Execute("PointCloud.DisplayMode.Intensity"); err != nil {
		t.Errorf("Execute(PointCloud.DisplayMode.Intensity): %v", err)
	}
	modeTab, ok = BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab after intensity mode")
	}
	displayPanel, ok := modeTab.Panel(PanelPointCloud)
	if !ok || displayPanel.IntensityRamp == nil {
		t.Fatal("Point Cloud panel should show the intensity ramp in intensity mode")
	}
	if displayPanel.IntensityRamp.Low.Value != [4]float32{1, 0, 0, 1} || displayPanel.IntensityRamp.High.Value != [4]float32{1, 1, 0, 1} {
		t.Fatalf("intensity ramp = %+v, want red/yellow", displayPanel.IntensityRamp)
	}
	if displayPanel.IntensityRamp.Histogram != nil {
		t.Fatalf("cloud without intensity data should expose an empty histogram, got %v", displayPanel.IntensityRamp.Histogram)
	}
}

func TestPointCloudIntensityHistogramBins(t *testing.T) {
	pc := pointcloud.NewWithSamples("Scan", "s.xyz", "rid", []pointcloud.PointSample{
		{Point: math.P3(0, 0, 0), HasIntensity: true, Intensity: 0},
		{Point: math.P3(1, 0, 0), HasIntensity: true, Intensity: 0},
		{Point: math.P3(2, 0, 0), HasIntensity: true, Intensity: 50},
		{Point: math.P3(3, 0, 0), HasIntensity: true, Intensity: 100},
	})
	got := pointCloudIntensityHistogram(pc, 4)
	want := []float32{1, 0, 0.5, 0.5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("histogram = %v, want %v", got, want)
	}
	pc = pointcloud.NewWithSamples("Flat", "s.xyz", "rid", []pointcloud.PointSample{
		{Point: math.P3(0, 0, 0), HasIntensity: true, Intensity: 3},
		{Point: math.P3(1, 0, 0), HasIntensity: true, Intensity: 3},
	})
	if got := pointCloudIntensityHistogram(pc, 4); got != nil {
		t.Fatalf("flat histogram = %v, want nil", got)
	}
}

// TestCachedIntensityHistogramMemoized proves the ribbon's per-frame histogram recomputes only when
// the displayed set changes: an unchanged set returns the cached slice, and a display-budget change
// that rebuilds the displayed set invalidates the memo (#645 perf).
func TestCachedIntensityHistogramMemoized(t *testing.T) {
	s, def := emptyPartSession(t)
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, intensityCloudSamples())
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	first := s.cachedIntensityHistogram(pc)
	again := s.cachedIntensityHistogram(pc)
	if len(first) == 0 || &first[0] != &again[0] {
		t.Fatal("unchanged displayed set should return the memoized histogram, not recompute")
	}
	pc.SetMaximumPointCount(2) // rebuilds the displayed set → new backing array → memo miss
	rebuilt := s.cachedIntensityHistogram(pc)
	if len(rebuilt) == 0 || &rebuilt[0] == &first[0] {
		t.Fatal("changing the display budget should invalidate the histogram memo")
	}
}

func TestPointCloudIntensityRampPanelCarriesHistogram(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasIntensity: true, Intensity: 0},
		{Point: math.P3(1, 0, 5), HasIntensity: true, Intensity: 50},
		{Point: math.P3(2, 0, 5), HasIntensity: true, Intensity: 100},
	})
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	pc.SetDisplayMode(types.PointCloudDisplayModeIntensity)
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: pc})

	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel(PanelPointCloud)
	if !ok || panel.IntensityRamp == nil {
		t.Fatal("Point Cloud panel should show the intensity ramp in intensity mode")
	}
	if got := len(panel.IntensityRamp.Histogram); got != pointCloudIntensityHistogramBins {
		t.Fatalf("histogram bins = %d, want %d", got, pointCloudIntensityHistogramBins)
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

// pointCloudDisplayPanel returns the consolidated Surfaces & Mesh ▸ Point Cloud panel.
func pointCloudDisplayPanel(t *testing.T, s *Session) RibbonPanel {
	t.Helper()
	tab, ok := BuildRibbon(s).Tab(tabSurfacesMesh)
	if !ok {
		t.Fatal("ribbon has no Surfaces & Mesh tab")
	}
	panel, ok := tab.Panel(PanelPointCloud)
	if !ok {
		t.Fatal("no Point Cloud panel")
	}
	return panel
}

func intensityCloudSamples() []pointcloud.PointSample {
	return []pointcloud.PointSample{
		{Point: math.P3(0, 0, 5), HasIntensity: true, Intensity: 0},
		{Point: math.P3(1, 0, 5), HasIntensity: true, Intensity: 50},
		{Point: math.P3(2, 0, 5), HasIntensity: true, Intensity: 100},
	}
}

// TestPointCloudLoneCloudIsAutoTarget verifies a single attached scan drives the Point Cloud
// display controls without any browser selection: the size/density sliders are enabled and the
// display mode is settable, so the intensity ramp appears once switched to intensity mode (#645).
func TestPointCloudLoneCloudIsAutoTarget(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	pc, err := def.PointClouds().AddWithSamples("Scan", "s.xyz", rid, intensityCloudSamples())
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	// Deliberately no s.Select: the lone scan must be the target on its own.

	panel := pointCloudDisplayPanel(t, s)
	if panel.PointSizeSlider.Disabled || panel.Slider.Disabled {
		t.Fatal("a lone attached scan should enable the size/density sliders without a selection")
	}
	if err := s.Execute("PointCloud.DisplayMode.Intensity"); err != nil {
		t.Fatalf("Execute(PointCloud.DisplayMode.Intensity) on lone scan: %v", err)
	}
	if pc.DisplayMode() != types.PointCloudDisplayModeIntensity {
		t.Fatalf("display mode = %q, want intensity", pc.DisplayMode())
	}
	if pointCloudDisplayPanel(t, s).IntensityRamp == nil {
		t.Fatal("intensity ramp should show for the lone scan in intensity mode")
	}
}

// TestPointCloudMultipleCloudsRequireSelection verifies that with several scans attached the
// display controls stay disabled until the user selects one, then target the selected scan (#645).
func TestPointCloudMultipleCloudsRequireSelection(t *testing.T) {
	s, def := emptyPartSession(t)
	if err := RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	rid := def.AddResource(doc.Resource{Encoding: doc.EncodingUTF8, Value: []byte("x")})
	if _, err := def.PointClouds().AddWithSamples("ScanA", "a.xyz", rid, intensityCloudSamples()); err != nil {
		t.Fatalf("attach A: %v", err)
	}
	if _, err := def.PointClouds().AddWithSamples("ScanB", "b.xyz", rid, intensityCloudSamples()); err != nil {
		t.Fatalf("attach B: %v", err)
	}

	panel := pointCloudDisplayPanel(t, s)
	if !panel.PointSizeSlider.Disabled || !panel.Slider.Disabled {
		t.Fatal("with several scans and none selected, the size/density sliders should be disabled")
	}
	if err := s.Execute("PointCloud.DisplayMode.RGB"); err == nil {
		t.Fatal("changing display mode should require selecting a scan when several are attached")
	}

	a, _ := def.PointClouds().ByName("ScanA")
	s.Select(PointCloudHandle{Clouds: def.PointClouds(), Cloud: a})
	if pointCloudDisplayPanel(t, s).Slider.Disabled {
		t.Fatal("selecting a scan should enable the display controls")
	}
	if err := s.Execute("PointCloud.DisplayMode.RGB"); err != nil {
		t.Fatalf("Execute(PointCloud.DisplayMode.RGB) after selecting: %v", err)
	}
	if a.DisplayMode() != types.PointCloudDisplayModeRGB {
		t.Fatalf("selected scan display mode = %q, want rgb", a.DisplayMode())
	}
}
