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

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/app/options"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/internal/sysopen"
	"oblikovati.org/head/internal/windowstate"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/persistence"
	"oblikovati.org/persistence/userprefs"
	"oblikovati.org/persistence/viewstate"
	"oblikovati.org/theme"
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
	win, err := openMainWindow()
	if err != nil {
		return err
	}
	defer win.Destroy()
	defer saveWindowState(win) // runs before Destroy (LIFO): capture the placement on exit
	win.InitViewport()

	// Load shared-library add-ins (e.g. oblikovati-mcp-bridge) and drain their queued
	// calls on this (the session) goroutine each frame, so add-ins touch the model
	// safely while the window stays responsive.
	addins := startAddIns(session)
	defer addins.stop(session)
	ui.SetScriptController(addins.script) // the Manage ▸ Script Console panel drives this runtime

	for frame := 0; ; frame++ {
		if maxFrames > 0 && frame >= maxFrames {
			break
		}
		if pumpFrame(win, session, addins) {
			break
		}
	}
	return nil
}

// pumpFrame renders one frame: chrome (+ executing a clicked command), draining add-in
// calls, and the close prompt. It returns true when the loop should exit — the user closed
// the window, or a rebuilt add-in on disk needs the supervisor to relaunch (a Go c-shared
// cannot be hot-swapped in-process).
func pumpFrame(win *native.Window, session *app.Session, addins *addInHost) bool {
	win.BeginFrame()
	if id := ui.DrawChrome(win, session); id != "" {
		if execErr := session.Execute(id); execErr != nil {
			fmt.Fprintf(os.Stderr, "command %q: %v\n", id, execErr)
		}
	}
	addins.drain()
	exit := ui.HandleClose(win, session)
	win.EndFrame(ui.WindowClearColor())   // themed swapchain background (ADR-0021)
	ui.ServiceWindowCapture(win, session) // whole-window PNG: AFTER the swapchain composited this frame
	return exit || addins.addInChanged()
}

// openMainWindow creates the window, reopening at the last session's size/position/monitor
// (per-user, in the OS config dir) and falling back to a sensible default the first time.
func openMainWindow() (*native.Window, error) {
	width, height := 1440, 900
	saved, hasSaved := windowstate.Load()
	if hasSaved {
		width, height = saved.Width, saved.Height
	}
	win, err := native.CreateWindow(width, height, "Oblikovati")
	if err != nil {
		return nil, err
	}
	if hasSaved {
		win.ApplyWindowState(saved.X, saved.Y, saved.Maximized) // restore position + maximized
	}
	return win, nil
}

// saveWindowState persists the window's current placement so the next session reopens in
// the same spot. Best-effort: a settings-write failure must not crash shutdown.
func saveWindowState(win *native.Window) {
	x, y, w, h, maximized := win.WindowState()
	_ = windowstate.Save(windowstate.State{X: x, Y: y, Width: w, Height: h, Maximized: maximized})
}

// newDemoSession wires the standard commands and a small part so the chrome shows real
// content: the ribbon lists sketch/create tools, and the browser shows a part with a
// parameter and a sketch. Clicking Extrude starts the interactive tool.
func newDemoSession() *app.Session {
	// A real .obk store so File ▸ Open/Save/Save As hit disk (the headless tests use the
	// nil-store NewSession). The head injects the concrete store; app depends only on the
	// doc.Store interface.
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	if path, err := viewstate.DefaultPath(); err == nil {
		// Per-user camera/view layout, stored outside the .obk so it never dirties the
		// document in git; written when the user saves, restored on open.
		s.SetViewStateStore(viewstate.NewFileStore(path))
	}
	if path, err := userprefs.DefaultPath(); err == nil {
		// Global UI preferences (e.g. ViewCube compass visibility), persisted across sessions.
		s.SetUserPrefsStore(userprefs.NewFileStore(path))
	}
	registerCommands(s)
	s.SetURLOpener(sysopen.SystemOpener{}) // web-view fallback (M05-F08)
	loadAppOptions(s)
	// StartupAction (M05-F11): the historical default seeds a demo part; the user can
	// opt into an empty workspace (the Get Started ribbon) in Preferences ▸ General.
	if s.Options().General.StartupAction != types.StartupEmptyWorkspace {
		seedPart(s)
	} else {
		installPicker(s)
	}
	loadThemes(s)
	return s
}

// loadThemes folds the user's saved custom themes and selected theme into the session
// from the OS config dir, so the chrome opens in the theme last used. A locate/load error
// is non-fatal — the built-in Dark/Light are always available — so a bad theme file never
// blocks startup; it is reported to stderr.
func loadThemes(s *app.Session) {
	root, err := theme.DefaultRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "themes: %v\n", err)
		return
	}
	if err := s.LoadThemes(theme.NewStore(root, theme.OSFileSystem{})); err != nil {
		fmt.Fprintf(os.Stderr, "themes: %v\n", err)
	}
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
	pd, err := compdef.AddPart(s.Workspace(), "demo.opd", true)
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
	installPicker(s)

	// Frame the box (centered ~(2,1.5,2.5)) from a three-quarter view.
	cam := s.Camera()
	cam.Eye = math.P3(12, 11, 16)
	cam.Target = math.P3(2, 1.5, 2.5)
	cam.Up = math.V3(0, 1, 0)
	s.SetCamera(cam)
}

// installPicker installs the hit-test so viewport clicks select faces and origin work
// planes. The closures read the ACTIVE document's part, so picking follows whichever
// document is current (New Part, a tab, an add-in's activate_document) — and works
// even when startup opened an empty workspace (StartupEmptyWorkspace, M05-F11).
func installPicker(s *app.Session) {
	s.SetPicker(app.NewRayPicker(s.Camera(),
		func() []*topo.Body { return activeBodies(s) }).
		WithPlanes(func() []*feature.WorkPlane { return s.PickableWorkPlanes() }).
		WithPoints(func() []*feature.WorkPoint { return s.PickableWorkPoints() }).
		WithAxes(func() []*feature.WorkAxis { return s.PickableWorkAxes() }).
		WithSketches(func() []*sketch.Sketch {
			if !s.ObjectVisibility().Sketches {
				return nil
			}
			return activeSketches(s)
		}).
		WithSketches3D(func() []*sketch.Sketch3D { return activeSketches3D(s) }).
		WithOccurrenceLookup(s.OccurrenceOfBody))
}

// loadAppOptions wires the per-user options file (M05-F11) and applies the stored
// groups; a store failure costs only persistence (defaults still apply).
func loadAppOptions(s *app.Session) {
	path, err := options.DefaultPath()
	if err == nil {
		err = s.UseOptionsStore(options.NewFileStore(path))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "options: %v\n", err)
	}
}

// activeBodies returns the active document's part bodies (nil if no active part), for
// the picker's hit-test so it follows the current document.
func activeBodies(s *app.Session) []*topo.Body {
	return s.VisibleBodies()
}

// activeSketches returns the active part's visible sketches (nil if none), so the picker
// can resolve a click inside a finished sketch's profile region (for extrude/revolve).
// Scope-hidden sketches are excluded like the overlays exclude them.
func activeSketches(s *app.Session) []*sketch.Sketch {
	p, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil
	}
	var out []*sketch.Sketch
	for i := 0; i < p.Sketches().Count(); i++ {
		if sk := p.Sketches().Item(i); sk.Visible() && !s.EditScopeHides(sk.Seq()) {
			out = append(out, sk)
		}
	}
	return out
}

// activeSketches3D returns the active part's visible 3D sketches (nil if none), so the
// picker can resolve a click on a 3D-sketch curve or point for the 3D constraint
// tools (issue #142).
func activeSketches3D(s *app.Session) []*sketch.Sketch3D {
	p, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil
	}
	var out []*sketch.Sketch3D
	for i := 0; i < p.Sketches3D().Count(); i++ {
		if sk := p.Sketches3D().Item(i); sk.Visible() {
			out = append(out, sk)
		}
	}
	return out
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
