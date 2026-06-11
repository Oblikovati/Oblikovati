//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// TestInWindowExtrudePanelVisualHold is a manual visual check: it opens the real window
// with the Extrude tool active over a picked square profile and holds it so a human (or
// a screenshot tool) can inspect the Extrusion property panel. Skipped unless
// OBK_VISUAL_HOLD=1 — it renders nothing assertable and exists only for layout work.
func TestInWindowExtrudePanelVisualHold(t *testing.T) {
	if os.Getenv("OBK_VISUAL_HOLD") == "" {
		t.Skip("manual visual check only (set OBK_VISUAL_HOLD=1)")
	}
	win, err := native.CreateWindow(1500, 900, "obk-extrude-panel-visual")
	if err != nil {
		t.Skipf("no window/Vulkan available: %v", err)
	}
	defer win.Destroy()
	win.InitViewport()
	dockLaidOut = false
	icons = nil
	s, profile := extrudeReadySession(t)
	startVisualFeatureTool(s, profile)
	applyVisualOverrides(t, s)
	for i := 0; i < 6000; i++ { // long enough to screenshot even uncapped (~30s+)
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.1)
	}
}

// startVisualFeatureTool starts the tool OBK_VISUAL_TOOL names (extrude by default;
// revolve / sweep / hole) and seeds its picks unless OBK_VISUAL_EMPTY=1, which leaves
// every selector in its required/empty state instead.
func startVisualFeatureTool(s *app.Session, profile app.ProfileHandle) {
	pick := os.Getenv("OBK_VISUAL_EMPTY") == ""
	switch os.Getenv("OBK_VISUAL_TOOL") {
	case "revolve":
		rv := app.NewRevolveTool()
		s.StartTool(rv)
		if pick {
			rv.Pick(s, profile)
		}
	case "sweep":
		sw := app.NewSweepTool()
		s.StartTool(sw)
		if pick {
			sw.Pick(s, profile) // the path chip stays empty: its required state shows
		}
	case "hole":
		h := app.NewHoleTool()
		s.StartTool(h)
		h.SetCounterbore(true) // show the seat dimension rows
	case "coil":
		co := app.NewCoilTool()
		s.StartTool(co)
		if pick {
			co.Pick(s, profile)
		}
	case "loft":
		lf := app.NewLoftTool()
		s.StartTool(lf)
		if pick {
			lf.Pick(s, profile)
		}
	case "split":
		s.StartTool(app.NewSplitTool())
	case "sketch-text":
		s.StartTool(app.NewSketchTextTool()) // a parameterized sketch tool: text + choices
	case "sketch-polygon":
		s.StartTool(app.NewPolygonTool(6)) // a parameterized sketch tool: int + float
	case "fillet":
		s.StartTool(app.NewFilletTool())
	case "edit-extrude": // commit an extrude, then re-open it for editing (the unified flow)
		ext := app.NewExtrudeTool()
		s.StartTool(ext)
		ext.Pick(s, profile)
		ext.SetDistance(3)
		if err := s.OK(); err == nil {
			s.BeginEditFeature(app.FeatureHandle{Feature: ext.AddedFeature()})
		}
	case "solid": // a committed solid with no tool — for renderer (lighting/style) checks
		ext := app.NewExtrudeTool()
		s.StartTool(ext)
		ext.Pick(s, profile)
		ext.SetDistance(3)
		_ = s.OK()
	case "addin-ui": // M05-F03 surfaces: browser pane tab, dockable window, popup button
		seedAddInSurfaces(s)
	case "preferences": // the Preferences window (General/Sketch Grid/Modeling/Theme tabs)
		showPreferences = true
	case "messaging": // M05-F09 surfaces: progress bar, balloon toast, prompt, messages
		seedMessagingSurfaces(s)
	default:
		ext := app.NewExtrudeTool()
		s.StartTool(ext)
		if pick {
			ext.Pick(s, profile)
		}
	}
}

// seedAddInSurfaces declares the M05-F03 add-in UI surfaces the way an add-in would
// over the wire: a popup ribbon control, a browser pane, and a visible dockable
// window — so a screenshot shows all three rendered by the head.
func seedAddInSurfaces(s *app.Session) {
	noop := func(*app.Session) error { return nil }
	_ = s.Commands().Add(app.NewCommand("demo.alpha", "Alpha", "Add-In Demo", noop))
	_ = s.Commands().Add(app.NewCommand("demo.beta", "Beta", "Add-In Demo", noop))
	_ = s.Commands().Add(app.NewCommand("demo.menu", "Demo Menu", "Add-In Demo", noop).
		WithTab("Tools").WithPopupItems("demo.alpha", "demo.beta"))
	_ = s.BrowserPanes().Set(wire.BrowserPaneSpec{
		ID: "sim", Title: "Simulation",
		Nodes: []wire.BrowserNodeSpec{{
			ID: "loads", Label: "Loads", Expanded: true,
			Children: []wire.BrowserNodeSpec{{ID: "f1", Label: "Force 10N"}, {ID: "f2", Label: "Pressure 2bar"}},
		}, {ID: "mesh", Label: "Mesh"}},
	})
	_ = s.SetDockableWindow(wire.DockableWindowSpec{
		ID: "sim.panel", Title: "Simulation", Dock: types.DockRight, Visible: true,
		Controls: []wire.PanelControlSpec{
			{Kind: types.PanelLabel, Text: "Mesh: 12,480 elements"},
			{Kind: types.PanelSeparator},
			{Kind: types.PanelButton, Text: "Run Study", CommandID: "demo.alpha"},
		},
	})
}

// seedMessagingSurfaces stages every M05-F09 feedback surface the way an add-in
// would over the wire: a half-done progress bar, a status text, a balloon toast, a
// pending prompt, and message-center content (so the badge shows).
func seedMessagingSurfaces(s *app.Session) {
	s.SetStatusText("Solving load case 2 of 5")
	if id, err := s.Progress().Begin(10, "Meshing bracket…"); err == nil {
		_, _ = s.Progress().Update(id, 6, "")
	}
	_ = s.BalloonTips().Register(app.BalloonTipSpec{
		ID: "sim.done", Title: "Simulation finished", Text: "Study 'Bracket' converged in 42 s.",
	})
	_, _ = s.ShowBalloonTip("sim.done")
	_, _, _ = s.ShowPrompt(app.PromptSpec{
		ID: "sim.replace", Message: "Replace the existing results?",
		Buttons: []string{"Replace", "Keep both"}, Restriction: types.PromptAllowRemember,
	})
	sec := s.Messages().BeginSection("Meshing")
	s.Messages().AddMessage("12 thin faces refined", types.SeverityInfo)
	s.Messages().AddMessage("degenerate face at fillet F7", types.SeverityWarning)
	_ = s.Messages().EndSection(sec)
	s.SetMessageCenterOpen(true)
}

// applyVisualOverrides applies the OBK_VISUAL_* renderer knobs through the View-tab
// commands, so a screenshot can compare looks: OBK_VISUAL_STYLE (a View.<Style> command
// suffix, e.g. Shaded, Realistic, Monochrome), OBK_VISUAL_LIGHTING (a Lighting Style
// gallery name, e.g. Sun), OBK_VISUAL_ENV (an Environment gallery name, e.g. Studio).
func applyVisualOverrides(t *testing.T, s *app.Session) {
	t.Helper()
	if style := os.Getenv("OBK_VISUAL_STYLE"); style != "" {
		if err := s.Execute("View." + style); err != nil {
			t.Fatalf("set visual style %q: %v", style, err)
		}
	}
	if name := os.Getenv("OBK_VISUAL_LIGHTING"); name != "" {
		if err := s.Execute("View.Lighting." + name); err != nil {
			t.Fatalf("set lighting style %q: %v", name, err)
		}
	}
	if name := os.Getenv("OBK_VISUAL_ENV"); name != "" {
		if err := s.Execute("View.Environment." + name); err != nil {
			t.Fatalf("set environment %q: %v", name, err)
		}
	}
}

// extrudeReadySession builds a part session holding a 2×2 closed square sketch — the
// profile the Extrude panel extrudes — with the standard commands registered.
func extrudeReadySession(t *testing.T) (*app.Session, app.ProfileHandle) {
	t.Helper()
	s := app.NewSession()
	pd, err := compdef.AddPart(s.Workspace(), "extrude-panel-visual.obk", true)
	if err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := app.RegisterStandardCommands(s); err != nil {
		t.Fatalf("RegisterStandardCommands: %v", err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(2, 0))
	c2 := sk.Points().Add(math.P2(2, 2))
	c3 := sk.Points().Add(math.P2(0, 2))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return s, app.ProfileHandle{Sketch: sk, ProfileIndex: 0}
}
