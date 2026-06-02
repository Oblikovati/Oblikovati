//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command oblikovati-head is the windowed application: a Vulkan + Dear ImGui head
// driven by the pure-Go application Session. It seeds a demo part and the standard
// commands, opens the window, and runs the frame loop — building the chrome from the
// live model every frame (ADR-0004) and executing whatever command the user clicks.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Oblikovati/oblikovati/addin/modelaccess"
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
	"github.com/Oblikovati/oblikovati/head/ui"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/feature"
	"github.com/Oblikovati/oblikovati/model/sketch"
	"github.com/Oblikovati/oblikovati/persistence"
)

func main() {
	maxFrames := flag.Int("frames", 0, "exit after N frames (0 = run until window closed); for smoke runs")
	flag.Parse()

	session := newDemoSession()
	if err := run(session, *maxFrames); err != nil {
		fmt.Fprintln(os.Stderr, "oblikovati-head:", err)
		os.Exit(1)
	}
}

// run opens the window and pumps the frame loop. maxFrames > 0 bounds it (so a smoke
// invocation cannot hang); 0 runs until the user closes the window.
func run(session *app.Session, maxFrames int) error {
	win, err := native.CreateWindow(1440, 900, "Oblikovati")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()

	// Load shared-library add-ins (e.g. oblikovati-mcp-bridge) and drain their queued
	// calls on this (the session) goroutine each frame, so add-ins touch the model
	// safely while the window stays responsive.
	addins := startAddIns(session)
	defer addins.stop(session)

	for frame := 0; !win.ShouldClose(); frame++ {
		if maxFrames > 0 && frame >= maxFrames {
			break
		}
		win.BeginFrame()
		if id := ui.DrawChrome(win, session); id != "" {
			if execErr := session.Execute(id); execErr != nil {
				fmt.Fprintf(os.Stderr, "command %q: %v\n", id, execErr)
			}
		}
		addins.drain()
		win.EndFrame(0.12, 0.13, 0.16)
	}
	return nil
}

// newDemoSession wires the standard commands and a small part so the chrome shows real
// content: the ribbon lists sketch/create tools, and the browser shows a part with a
// parameter and a sketch. Clicking Extrude starts the interactive tool.
func newDemoSession() *app.Session {
	// A real .obk store so File ▸ Open/Save/Save As hit disk (the headless tests use the
	// nil-store NewSession). The head injects the concrete store; app depends only on the
	// doc.Store interface.
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	registerCommands(s)
	seedPart(s)
	return s
}

func registerCommands(s *app.Session) {
	// The standard Inventor ribbon: 3D Model (Create 2D Sketch, Extrude), the contextual
	// Sketch tab (Line/Rectangle/Circle/Arc/Spline/Ellipse/Polygon/Point + Finish Sketch),
	// and View. Sketch tools enable only inside the sketch environment (app package).
	if err := app.RegisterStandardCommands(s); err != nil {
		panic(err) // demo wiring: a duplicate id here is a programming error
	}
}

func seedPart(s *app.Session) {
	pd, err := compdef.AddPart(s.Workspace(), "demo.obk", true)
	if err != nil {
		panic(err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	_, _ = def.Parameters().AddUserParameter("width", "4 cm")

	sk := def.Sketches().Add(sketch.XYPlane())
	rectangle(sk, 4, 3)
	feature.NewExtrudeFeatures(def.Features()).
		AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()

	// Install the hit-test so viewport clicks select faces and origin work planes.
	// The closures read the ACTIVE document's part, so picking follows whichever
	// document is current (New Part, a tab, an add-in's activate_document) rather than
	// staying bound to this seeded part.
	s.SetPicker(app.NewRayPicker(s.Camera(),
		func() []*topo.Body { return activeBodies(s) }).
		WithPlanes(func() []*feature.WorkPlane { return activeOriginPlanes(s) }))

	// Frame the box (centered ~(2,1.5,2.5)) from a three-quarter view.
	cam := s.Camera()
	cam.Eye = math.P3(12, 11, 16)
	cam.Target = math.P3(2, 1.5, 2.5)
	cam.Up = math.V3(0, 1, 0)
	s.SetCamera(cam)
}

// activeBodies returns the active document's part bodies (nil if no active part), for
// the picker's hit-test so it follows the current document.
func activeBodies(s *app.Session) []*topo.Body {
	if p, err := modelaccess.ActivePart(s); err == nil {
		return p.SurfaceBodies().All()
	}
	return nil
}

// activeOriginPlanes returns the active part's origin work planes (nil if none), so
// the picker can hit-test the current document's planes.
func activeOriginPlanes(s *app.Session) []*feature.WorkPlane {
	if p, err := modelaccess.ActivePart(s); err == nil {
		return p.OriginPlanes()
	}
	return nil
}

// rectangle adds a closed w×h rectangle (one profile) at the sketch origin.
func rectangle(sk *sketch.Sketch, w, h float64) {
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(w, 0))
	c2 := sk.Points().Add(math.P2(w, h))
	c3 := sk.Points().Add(math.P2(0, h))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
}
