// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/display"
	"oblikovati.org/model/style"
)

// TestColorSchemeAdapters exercises the color-scheme contract adapters and session methods
// (the getters are otherwise only reached through the router package, where coverage is not
// attributed to this package).
func TestColorSchemeAdapters(t *testing.T) {
	t.Parallel()
	s := NewSession()
	cs := s.ColorSchemes()
	if cs.Count() < 2 || cs.Item(-1) != nil {
		t.Fatalf("Count=%d / Item(-1) should be nil", cs.Count())
	}
	first := cs.Item(0)
	_ = first.Name()
	_, _, _ = first.ScreenColor(), first.TopScreenColor(), first.BottomScreenColor()
	_, _, _ = first.HighlightColor(), first.PrimarySelectColor(), first.SecondarySelectColor()
	if cs.Active().Name() != "Default" {
		t.Errorf("active = %q, want Default", cs.Active().Name())
	}
	rev := s.ColorSchemeRevision()
	if err := cs.SetActive("High Contrast"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if s.ColorSchemeRevision() == rev {
		t.Error("revision should bump on scheme change")
	}
	if s.ActiveColorScheme().Name != "High Contrast" {
		t.Errorf("ActiveColorScheme = %q, want High Contrast", s.ActiveColorScheme().Name)
	}
	if err := cs.SetBackgroundType(types.ImageBackground); err != nil || cs.BackgroundType() != types.ImageBackground {
		t.Errorf("SetBackgroundType: err=%v bg=%v", err, cs.BackgroundType())
	}
	if err := s.SetColorScheme("nope"); err == nil {
		t.Error("unknown scheme should error")
	}
}

// TestStyleAdapters exercises the style-manager contract adapters and session methods.
func TestStyleAdapters(t *testing.T) {
	t.Parallel()
	s := NewSession()
	sm := s.StyleManager()
	cs := sm.ColorStyles()
	if cs.Count() < 3 || cs.Item(-1) != nil || cs.ByName("nope") != nil {
		t.Fatalf("ColorStyles Count=%d / bad lookups not nil", cs.Count())
	}
	steel := cs.ByName("Steel")
	_ = steel.Name()
	_, _, _, _ = steel.DiffuseColor(), steel.AmbientColor(), steel.SpecularColor(), steel.EmissiveColor()
	_, _, _ = steel.Shininess(), steel.Opacity(), steel.Location()
	_ = cs.Item(0)
	if len(sm.LightingStyles()) == 0 {
		t.Error("expected lighting styles")
	}

	if err := s.SetColorStyle(style.ColorStyle{Name: "Glass", Opacity: 0.3, Location: types.LocalStyleLocation}); err != nil {
		t.Fatalf("SetColorStyle: %v", err)
	}
	if g, ok := s.ColorStyle("Glass"); !ok || g.Opacity != 0.3 {
		t.Errorf("ColorStyle(Glass) = %+v, %v", g, ok)
	}
	if err := s.DeleteColorStyle("Glass"); err != nil {
		t.Fatalf("DeleteColorStyle: %v", err)
	}
	_ = s.ColorStyles()

	// Library import through the adapter (writes a JSON fixture).
	path := filepath.Join(t.TempDir(), "lib.json")
	if err := os.WriteFile(path, []byte(`{"name":"Lib","styles":[{"name":"Ti","opacity":1}]}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := sm.ImportLibrary(path); err != nil {
		t.Fatalf("ImportLibrary: %v", err)
	}
	if names := sm.LibraryNames(); len(names) != 1 || names[0] != "Lib" {
		t.Errorf("LibraryNames = %v, want [Lib]", names)
	}
	if len(s.StyleLibraries()) != 1 {
		t.Errorf("StyleLibraries = %d, want 1", len(s.StyleLibraries()))
	}
	if err := sm.ImportLibrary("/no/such.json"); err == nil {
		t.Error("importing a missing file should error")
	}
}

// TestNamedViewSessionMethods exercises the named-view capture/restore/delete and the
// standard-orientation jump on a part document.
func TestNamedViewSessionMethods(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if _, err := s.CaptureNamedView("iso"); err != nil {
		t.Fatalf("CaptureNamedView: %v", err)
	}
	if len(s.NamedViews()) != 1 {
		t.Fatalf("NamedViews = %d, want 1", len(s.NamedViews()))
	}
	if err := s.RestoreNamedView("iso"); err != nil {
		t.Fatalf("RestoreNamedView: %v", err)
	}
	if err := s.RestoreNamedView("ghost"); err == nil {
		t.Error("restoring an absent named view should error")
	}
	for _, o := range []types.ViewOrientationTypeEnum{
		types.FrontViewOrientation, types.TopViewOrientation, types.IsoTopRightViewOrientation,
		types.CurrentViewOrientation, // no fixed direction → no-op
	} {
		if err := s.SetViewOrientation(o, true); err != nil {
			t.Errorf("SetViewOrientation(%s): %v", o, err)
		}
	}
	if err := s.DeleteNamedView("iso"); err != nil {
		t.Fatalf("DeleteNamedView: %v", err)
	}
	if err := s.DeleteNamedView("iso"); err == nil {
		t.Error("deleting an absent named view should error")
	}
}

// TestDisplayAdapters exercises the display-options and per-document display-settings adapters.
func TestDisplayAdapters(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	o := s.DisplayOptions()
	_ = o.DisplayQuality()
	_, _, _ = o.ViewTransitionTime(), o.MinimumFrameRate(), o.HiddenLineDimmingPercent()
	_, _, _ = o.EdgeColor(), o.NewWindowDisplayMode(), o.NewWindowProjection()
	_, _, _ = o.BackFaceCulling(), o.UseRayTracing(), o.RayTracingQuality()
	sh := o.Shaded()
	_, _, _, _ = sh.EdgeDisplay(), sh.EdgeColor(), sh.Silhouettes(), sh.TransparencyType()
	wf := o.Wireframe()
	_, _, _ = wf.DepthDimming(), wf.Silhouettes(), wf.DimmedHiddenEdges()

	edited := s.DisplayOptionsData()
	edited.UseRayTracing = true
	s.SetDisplayOptions(edited)
	if !s.DisplayOptions().UseRayTracing() {
		t.Error("SetDisplayOptions did not stick")
	}

	// Per-document settings via the active document (document 0 ⇒ active, exercises resolveDocID).
	ds := s.DocumentDisplaySettings(0)
	_, _, _ = ds.BackgroundType(), ds.EdgeColor(), ds.DepthDimming()
	_, _, _ = ds.DisplaySilhouettes(), ds.HiddenLineDimmingPercent(), ds.NewWindowDisplayMode()
	_, _ = ds.DisplayModeSource(), ds.NewWindowProjection()
	gp := ds.GroundPlane()
	_, _, _, _ = gp.Visible(), gp.Color(), gp.HeightOffset(), gp.DisplayGridLines()
	_, _, _, _ = gp.MinorGridLineSpacing(), gp.MinorLinesPerMajorGridLine(), gp.Opacity(), gp.Reflectivity()
	_, _, _ = ds.GroundShadow(), ds.ShadowDirection(), ds.ShowGroundReflections()
	_, _, _ = ds.ShowObjectShadows(), ds.ShowAmbientShadows(), ds.TexturesOn()

	set := display.DefaultSettings()
	set.BackgroundType = types.OneColorBackground
	s.SetDisplaySettings(0, set)
	if s.DisplaySettings(0).BackgroundType != types.OneColorBackground {
		t.Error("SetDisplaySettings did not stick on the active document")
	}
}
