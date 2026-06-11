//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"os"
	"testing"

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
	case "fillet":
		s.StartTool(app.NewFilletTool())
	default:
		ext := app.NewExtrudeTool()
		s.StartTool(ext)
		if pick {
			ext.Pick(s, profile)
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
