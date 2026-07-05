//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"path/filepath"
	"testing"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
	"oblikovati.org/update"
)

// The #1759 sweep: every feature dialog now sizes itself through dialogSizeOnce/dialogSize (scale by the
// UI font scale, clamp to the host window) instead of a hard-coded pixel size, so a raised font scale
// grows the box with its text and never pushes the OK/Cancel row off-screen (the #1753 clip, generalised).
// Each swapped call lives inside a cgo draw function no prior test exercised, so this file arms every
// dialog and renders it through real DrawChrome frames at a raised scale — proving the scaled path runs
// (and giving the swapped lines coverage). Skips cleanly where no display/Vulkan is available.

// dialogArm arms one feature dialog on a fresh session and reports whether it stayed open through a
// DrawChrome frame (armed ⇒ the draw reached its size call rather than early-returning).
type dialogArm struct {
	name  string
	arm   func(t *testing.T) *app.Session
	armed func(*app.Session) bool
	reset func() // clears any package-global open flag the arm set; nil for session-scoped dialogs
}

// chromeDialogCases is the roster of dialogs DrawChrome fans out to (chrome.go). A package-level var,
// not a function, so the long list is not subject to the function-length lint. Overlays and dockable/
// add-in panels are intentionally excluded (they are not modal feature dialogs); the four Parameters
// editors are covered by TestInWindowParameterEditorsScaleAndClamp, which drives them directly.
var chromeDialogCases = []dialogArm{
	// Solid-feature tool dialogs — armed by starting their tool (each self-gates on s.ActiveXxx()).
	toolDialogCase("extrude", func(s *app.Session) { s.StartTool(app.NewExtrudeTool()) }, func(s *app.Session) bool { return s.ActiveExtrude() != nil }),
	toolDialogCase("revolve", func(s *app.Session) { s.StartTool(app.NewRevolveTool()) }, func(s *app.Session) bool { return s.ActiveRevolve() != nil }),
	toolDialogCase("coil", func(s *app.Session) { s.StartTool(app.NewCoilTool()) }, func(s *app.Session) bool { return s.ActiveCoil() != nil }),
	toolDialogCase("loft", func(s *app.Session) { s.StartTool(app.NewLoftTool()) }, func(s *app.Session) bool { return s.ActiveLoft() != nil }),
	toolDialogCase("sweep", func(s *app.Session) { s.StartTool(app.NewSweepTool()) }, func(s *app.Session) bool { return s.ActiveSweep() != nil }),
	toolDialogCase("hole", func(s *app.Session) { s.StartTool(app.NewHoleTool()) }, func(s *app.Session) bool { return s.ActiveHole() != nil }),
	toolDialogCase("chamfer", func(s *app.Session) { s.StartTool(app.NewChamferTool()) }, func(s *app.Session) bool { return s.ActiveChamfer() != nil }),
	toolDialogCase("thread", func(s *app.Session) { s.StartTool(app.NewThreadTool()) }, func(s *app.Session) bool { return s.ActiveThread() != nil }),
	toolDialogCase("fillet", func(s *app.Session) { s.StartTool(app.NewFilletTool()) }, func(s *app.Session) bool { return s.ActiveFillet() != nil }),
	toolDialogCase("face-fillet", func(s *app.Session) { s.StartTool(app.NewFaceFilletTool()) }, func(s *app.Session) bool { return s.ActiveFaceFillet() != nil }),
	toolDialogCase("full-round-fillet", func(s *app.Session) { s.StartTool(app.NewFullRoundFilletTool()) }, func(s *app.Session) bool { return s.ActiveFullRoundFillet() != nil }),
	toolDialogCase("shell", func(s *app.Session) { s.StartTool(app.NewShellTool()) }, func(s *app.Session) bool { return s.ActiveShell() != nil }),
	toolDialogCase("split", func(s *app.Session) { s.StartTool(app.NewSplitTool()) }, func(s *app.Session) bool { return s.ActiveSplit() != nil }),
	toolDialogCase("grip-snap", func(s *app.Session) { s.StartTool(app.NewGripSnapTool()) }, func(s *app.Session) bool { return s.ActiveGripSnap() != nil }),
	toolDialogCase("measure", func(s *app.Session) { s.StartTool(app.NewMeasureTool()) }, func(s *app.Session) bool { return s.ActiveMeasure() != nil }),
	toolDialogCase("offset-plane", func(s *app.Session) { s.StartTool(app.NewOffsetWorkPlaneTool()) }, func(s *app.Session) bool { return s.ActiveOffsetPlane() != nil }),
	// Surface-edit tool dialogs.
	toolDialogCase("face-offset", func(s *app.Session) { s.StartTool(app.NewFaceOffsetTool()) }, func(s *app.Session) bool { return s.ActiveFaceOffset() != nil }),
	toolDialogCase("draft", func(s *app.Session) { s.StartTool(app.NewDraftTool()) }, func(s *app.Session) bool { return s.ActiveDraft() != nil }),
	toolDialogCase("delete-face", func(s *app.Session) { s.StartTool(app.NewDeleteFaceTool()) }, func(s *app.Session) bool { return s.ActiveDeleteFace() != nil }),
	toolDialogCase("replace-face", func(s *app.Session) { s.StartTool(app.NewReplaceFaceTool()) }, func(s *app.Session) bool { return s.ActiveReplaceFace() != nil }),
	toolDialogCase("thicken", func(s *app.Session) { s.StartTool(app.NewThickenTool()) }, func(s *app.Session) bool { return s.ActiveThicken() != nil }),
	// One router serves all seventeen sheet-metal tools; one representative arms its single size call.
	toolDialogCase("sheet-metal", func(s *app.Session) { s.StartTool(app.NewSheetMetalFaceTool()) }, func(s *app.Session) bool { return s.ActiveSheetMetalFace() != nil }),
	// The generic parameter panel serves every ParameterizedTool (here: a rectangular pattern).
	toolDialogCase("tool-params", func(s *app.Session) { s.StartTool(app.NewFeatureRectPatternTool()) }, func(s *app.Session) bool { _, _, ok := activeToolParams(s); return ok }),

	// Non-tool windows gated on session state.
	sessionDialogCase("keymap", func(s *app.Session) { s.OpenKeymapEditor() }, func(s *app.Session) bool { return s.KeymapEditorOpen() }),
	sessionDialogCase("script-console", func(s *app.Session) { s.OpenScriptConsole() }, func(s *app.Session) bool { return s.ScriptConsoleOpen() }),
	sessionDialogCase("update", func(s *app.Session) {
		s.ShowUpdateResult(update.Result{Current: "1.0.0", Skipped: true, SkipReason: "test"})
	}, func(s *app.Session) bool { return s.PendingUpdate() != nil }),
	sessionDialogCase("web-view", armWebView, func(s *app.Session) bool { return len(s.WebViews()) > 0 }),
	sessionDialogCase("sketch3d-settings", armSketch3D, func(s *app.Session) bool { return s.ActiveSketch3D() != nil }),

	// Non-tool windows gated on a package-global open flag — reset after each subtest.
	globalDialogCase("addin-catalogue", func() { addInCatalogueUI.open = true }, func() bool { return addInCatalogueUI.open }, func() { addInCatalogueUI.open = false }),
	globalDialogCase("report-bug", func() { reportBugUI.open = true }, func() bool { return reportBugUI.open }, func() { reportBugUI.open = false }),

	// In-place editors — real committed features, routed to the generic scalar/reference editors.
	{name: "feature-edit", arm: patternEditSession, armed: func(s *app.Session) bool { return s.IsEditingFeature() }},
	{name: "work-plane-edit", arm: workPlaneEditSession, armed: func(s *app.Session) bool { return s.IsEditingWorkPlane() }},
}

// toolDialogCase builds a case for a tool dialog: start the tool on a fresh part session, then verify
// the tool stayed active (so its self-gated dialog drew its size call).
func toolDialogCase(name string, start func(*app.Session), armed func(*app.Session) bool) dialogArm {
	return dialogArm{name: name, arm: func(t *testing.T) *app.Session { s := dialogPartSession(t); start(s); return s }, armed: armed}
}

// sessionDialogCase builds a case whose arm mutates session state (no package global to reset).
func sessionDialogCase(name string, open func(*app.Session), armed func(*app.Session) bool) dialogArm {
	return dialogArm{name: name, arm: func(t *testing.T) *app.Session { s := dialogPartSession(t); open(s); return s }, armed: armed}
}

// globalDialogCase builds a case whose arm sets a package-global open flag, restored by reset.
func globalDialogCase(name string, open func(), armed func() bool, reset func()) dialogArm {
	return dialogArm{
		name:  name,
		arm:   func(t *testing.T) *app.Session { s := dialogPartSession(t); open(); return s },
		armed: func(*app.Session) bool { return armed() },
		reset: reset,
	}
}

// TestInWindowAllFeatureDialogsScaleAndClamp is the #1759 acceptance gate: at a raised font scale every
// feature dialog opens through the scale-and-clamp helper without clipping or panicking. It renders each
// armed dialog through real DrawChrome frames (which also clamps the tall Hole dialog to the host window),
// and saves one PNG to eyeball the whole set.
func TestInWindowAllFeatureDialogsScaleAndClamp(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	for _, d := range chromeDialogCases {
		t.Run(d.name, func(t *testing.T) {
			if d.reset != nil {
				defer d.reset()
			}
			s := d.arm(t)
			if err := s.SetUIFontScale(1.6); err != nil { // the hi-dpi cohort the clip bit hardest
				t.Fatalf("SetUIFontScale: %v", err)
			}
			renderChromeFrames(win, s, 3)
			if !d.armed(s) {
				t.Fatalf("the %s dialog did not stay open through DrawChrome — its scaled size call never ran", d.name)
			}
			// Eyeball only two representatives, not 33 artifacts: the tall Hole (which the clamp pulls
			// back into the host window, keeping OK/Cancel in view) and a normal-height Extrude.
			if d.name == "hole" || d.name == "extrude" {
				if err := win.SaveWindowPNG(filepath.Join(outDir(), "dialog-scale-"+d.name+".png")); err != nil {
					t.Logf("SaveWindowPNG: %v", err)
				}
			}
		})
	}
}

// dialogPartSession returns a session with an empty part document — the environment every feature tool
// needs to run and every dialog needs to render its chrome.
func dialogPartSession(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "dialog-scale.opd", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	return s
}

// armWebView shows a visible web dialog so drawWebViews renders it (M05-F08).
func armWebView(s *app.Session) {
	if err := s.ShowWebDialog(wire.WebDialogSpec{ID: "docs", Title: "Help", URL: "https://example.org", Visible: true}); err != nil {
		panic("ShowWebDialog: " + err.Error())
	}
}

// armSketch3D enters a fresh 3D sketch so drawSketch3DSettings renders its settings panel.
func armSketch3D(s *app.Session) {
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	s.EnterSketch3D(def.Sketches3D().Add())
}

// workPlaneEditSession commits a redefinable offset work plane and opens it for editing — the generic
// work-plane editor whose height scales with its scalar/reference count.
func workPlaneEditSession(t *testing.T) *app.Session {
	t.Helper()
	s := dialogPartSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	wp := def.WorkPlanes().AddByPlaneAndOffset(feature.OriginXYPlane, func() float64 { return 2 })
	s.BeginEditWorkPlane(app.WorkPlaneHandle{Plane: wp})
	return s
}

// patternEditSession builds a part with an extrude, patterns it, then opens the pattern for editing —
// a feature that routes to the generic parameter/reference editor (not a full creation panel), so it
// arms drawFeatureEditDialog whose height scales with its parameter/reference count.
func patternEditSession(t *testing.T) *app.Session {
	t.Helper()
	s := dialogPartSession(t)
	def := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	seedExtrudeFeature(t, s, def)
	commitRectPattern(t, s, def)
	last := def.Features().Item(def.Features().Count() - 1)
	s.BeginEditFeature(app.FeatureHandle{Feature: last})
	return s
}

// seedExtrudeFeature extrudes a 2×2 square 5 cm tall via the real tool flow, leaving one committed
// solid feature to pattern.
func seedExtrudeFeature(t *testing.T, s *app.Session, def *compdef.PartComponentDefinition) {
	t.Helper()
	sk := def.Sketches().Add(sketch.XYPlane())
	addClosedSquare(sk, 2)
	s.SetPicker(fixedPicker{sel: app.ProfileHandle{Sketch: sk, ProfileIndex: 0}})
	ext := app.NewExtrudeTool()
	s.StartTool(ext)
	s.Click(100, 100)
	ext.SetDistance(5)
	if err := s.OK(); err != nil {
		t.Fatalf("seed extrude: %v", err)
	}
}

// commitRectPattern patterns the part's first feature into a 2×1 grid, appending a pattern feature.
func commitRectPattern(t *testing.T, s *app.Session, def *compdef.PartComponentDefinition) {
	t.Helper()
	s.SetPicker(fixedPicker{sel: app.FeatureHandle{Feature: def.Features().Item(0)}})
	s.StartTool(app.NewFeatureRectPatternTool())
	s.Click(100, 100)
	if err := s.OK(); err != nil {
		t.Fatalf("commit rectangular pattern: %v", err)
	}
}

// addClosedSquare draws a closed side×side square in the sketch — one bounded profile (index 0).
func addClosedSquare(sk *sketch.Sketch, side float64) {
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(side, 0))
	c2 := sk.Points().Add(math.P2(side, side))
	c3 := sk.Points().Add(math.P2(0, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}

// fixedPicker returns one canned selectable for any pick — feeds a profile to the extrude tool and a
// feature handle to the pattern tool without a live viewport hit.
type fixedPicker struct{ sel app.Selectable }

func (p fixedPicker) Pick(_, _ float64, _ *app.SelectionFilter) (app.Selectable, bool) {
	return p.sel, p.sel != nil
}

// TestInWindowDialogFitClampsToHostWindow pins the scale-and-clamp arithmetic dialogSizeOnce/dialogSize
// share (#1759): a dialog scales with the font, but never opens larger than the host window minus a
// margin — the invariant that keeps a scaled dialog's OK/Cancel row on-screen. Run in-window because the
// clamp reads the live main-viewport size (here the 800×600 test window).
func TestInWindowDialogFitClampsToHostWindow(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	prev := uiFontScale
	defer func() { uiFontScale = prev }()

	win.BeginFrame()
	uiFontScale = 1.5
	// A modest dialog just scales (fits well inside 800×600): 200×100 → 300×150.
	if w, h := dialogFit(200, 100); w != 300 || h != 150 {
		t.Errorf("dialogFit(200,100) at 1.5x = (%v,%v), want (300,150)", w, h)
	}
	// An over-large dialog is clamped to the viewport minus the margins, so it cannot overflow: width to
	// vw-dialogMargin (800-24), height to vh-2*dialogMargin (600-48) — regardless of how far the scale
	// blows it up. A 0 axis (auto-size) is preserved.
	if w, h := dialogFit(4000, 9000); w != inWinW-dialogMargin || h != inWinH-2*dialogMargin {
		t.Errorf("dialogFit(4000,9000) clamp = (%v,%v), want (%v,%v)", w, h, inWinW-dialogMargin, inWinH-2*dialogMargin)
	}
	if w, h := dialogFit(0, 0); w != 0 || h != 0 {
		t.Errorf("dialogFit(0,0) = (%v,%v), want (0,0) (auto-size axes preserved)", w, h)
	}
	win.EndFrame(0.1, 0.1, 0.1)
}

// TestInWindowParameterEditorsScaleAndClamp covers the four Parameters editors (value-list, tolerance,
// add-to-group, link), which the Parameters panel — not DrawChrome — fans out to. Each self-gates on a
// package-global target, so the test sets it, draws the editor directly in a frame at a raised scale, and
// clears it. Icons are warmed by one DrawChrome frame first so a direct draw never hits a nil icon cache.
func TestInWindowParameterEditorsScaleAndClamp(t *testing.T) {
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil
	prev := uiFontScale
	uiFontScale = 1.6 // direct draws (no DrawChrome) read the package global straight
	defer func() { uiFontScale = prev }()

	s := framedSession()
	renderChromeFrames(win, s, 1) // warm the icon cache

	cases := []struct {
		name string
		open func()
		draw func(*app.Session)
		shut func()
	}{
		{"value-list", func() { parametersUI.listFor = param.ID(1) }, drawValueListEditor, func() { parametersUI.listFor = 0 }},
		{"tolerance", func() { parametersUI.tolFor = param.ID(1) }, drawToleranceEditor, func() { parametersUI.tolFor = 0 }},
		{"add-to-group", func() { parametersUI.groupFor = param.ID(1) }, drawAddToGroupDialog, func() { parametersUI.groupFor = 0 }},
		{"link", func() { derivedUI.pickerOpen = true }, drawLinkParametersDialog, func() { derivedUI.pickerOpen = false }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.open()
			defer c.shut()
			win.BeginFrame()
			c.draw(s)
			win.EndFrame(0.1, 0.1, 0.1)
		})
	}
}
