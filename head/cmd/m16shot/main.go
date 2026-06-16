//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command m16shot is a throwaway live-capture driver for the M16 DoD tail: it opens the real
// native window, builds a session, applies a setup (e.g. switch the active color scheme), runs
// the real DrawChrome frame loop for a few frames so the renderer settles, then saves the 3D
// viewport to a PNG. Reuses the production render path, so the PNG is exactly what the app draws.
//
//	go run ./head/cmd/m16shot -scheme "High Contrast" -out /tmp/m16-f06.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/clientgraphics"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
)

// orientByName maps a CLI orientation name to a standard view orientation.
var orientByName = map[string]types.ViewOrientationTypeEnum{
	"front": types.FrontViewOrientation, "back": types.BackViewOrientation,
	"top": types.TopViewOrientation, "bottom": types.BottomViewOrientation,
	"left": types.LeftViewOrientation, "right": types.RightViewOrientation,
	"iso": types.IsoTopRightViewOrientation,
}

func main() {
	o := parseOpts()
	if err := run(o); err != nil {
		fmt.Fprintln(os.Stderr, "m16shot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", o.out)
}

// parseOpts defines and parses the command-line flags into the capture configuration.
func parseOpts() opts {
	var o opts
	flag.StringVar(&o.scheme, "scheme", "", "color scheme to activate before capture")
	flag.StringVar(&o.out, "out", "/tmp/m16shot.png", "viewport PNG output path")
	flag.IntVar(&o.frames, "frames", 8, "frames to render before capture")
	flag.BoolVar(&o.noEnv, "no-env", false, "turn off the environment skybox so the scheme background shows")
	flag.BoolVar(&o.box, "box", false, "build a demo box so geometry is visible")
	flag.StringVar(&o.orient, "orient", "", "jump to a standard orientation (front/top/iso…)")
	flag.StringVar(&o.edge, "edge", "", "override the display-settings edge color as R,G,B (0-255)")
	flag.StringVar(&o.ground, "ground", "", "set the display-settings ground color as R,G,B and enable ground shadows")
	flag.BoolVar(&o.overlay, "overlay", false, "add a red surface overlay highlighting the demo box (M16-F05)")
	flag.StringVar(&o.style, "style", "", "assign a color style (e.g. Brass) to the demo box (M16-F02)")
	flag.BoolVar(&o.named, "named", false, "save a couple named views and open the Named Views panel (M16-F03)")
	flag.BoolVar(&o.styles, "styles", false, "select the demo box and open the Color Styles panel (M16-F02)")
	flag.BoolVar(&o.window, "window", false, "capture the WHOLE window (chrome + panels), not just the 3D viewport")
	flag.BoolVar(&o.dialog, "dialog", false, "open the Display Settings dialog (M16-F07)")
	flag.BoolVar(&o.image, "image", false, "add an image billboard overlay at the origin (M16-F05)")
	flag.BoolVar(&o.boxselect, "boxselect", false, "drag a window box-select across the demo box and capture the rubber-band (#916)")
	flag.BoolVar(&o.sketch, "sketch", false, "enter a sketch with a dimensioned rectangle over the grid and capture (depth layering, #909)")
	flag.Parse()
	return o
}

// opts is the capture configuration parsed from the command line.
type opts struct {
	scheme, out           string
	frames                int
	noEnv, box            bool
	orient                string
	edge                  string
	ground                string
	overlay               bool
	style                 string
	named, styles, window bool
	dialog, image         bool
	boxselect, sketch     bool
}

func run(o opts) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if o.box {
		buildBox(s) // AddPart makes the box the active document
	} else if _, err := s.NewPart(); err != nil {
		return err
	}
	if err := applySetup(s, o); err != nil {
		return err
	}
	if o.boxselect {
		return captureBoxSelect(s, o)
	}
	if o.sketch {
		return captureSketch(s, o)
	}
	return renderAndCapture(s, o)
}

// captureSketch enters a sketch holding a dimensioned rectangle over the grid and captures the
// window, so the PNG shows the depth layering: grid behind, entities above it, the dimension above
// the entities (#909).
func captureSketch(s *app.Session, o opts) error {
	if err := enterDimensionedRectangle(s); err != nil {
		return err
	}
	win, err := native.CreateWindow(1280, 800, "m16shot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for i := 0; i < o.frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(o.out)
}

// enterDimensionedRectangle builds a part with a sketch holding a dimensioned rectangle, enters
// the sketch with the grid visible, and aims the camera straight at the plane.
func enterDimensionedRectangle(s *app.Session) error {
	pd, err := compdef.AddPart(s.Workspace(), "m16sketch.opd", true)
	if err != nil {
		return err
	}
	_ = s.Workspace().SetActiveDocument(pd)
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0, c1 := sk.Points().Add(math.P2(0, 0)), sk.Points().Add(math.P2(4, 0))
	c2, c3 := sk.Points().Add(math.P2(4, 3)), sk.Points().Add(math.P2(0, 3))
	bottom := sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	_, _ = sk.DimensionConstraints().AddDistance(bottom.A, bottom.B, "4 mm")
	s.EnterSketch(sk)
	s.Grid().Visible = true
	s.TickCameraAnimation(100) // finish the enter-sketch swing
	cam := scene.NewCamera(1280, 800)
	cam.Eye, cam.Target, cam.Up = math.P3(2, 1.5, 6), math.P3(2, 1.5, 0), math.V3(0, 1, 0)
	s.SetCamera(cam)
	return nil
}

// captureBoxSelect installs the real ray/region picker over the scene, then injects a left
// drag from an empty corner of the viewport across the demo box and captures the whole
// window mid-drag — so the saved PNG shows the window-select rubber-band over the geometry
// (#916). Coordinates target the central viewport dock node of the default layout.
func captureBoxSelect(s *app.Session, o opts) error {
	picker := app.NewRayPicker(s.Camera(), s.VisibleBodies)
	s.SetPicker(picker)
	s.SetRegionPicker(picker)
	win, err := native.CreateWindow(1280, 800, "m16shot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	frame := func() {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	for i := 0; i < o.frames; i++ {
		frame()
	}
	injectBoxDrag(frame)
	err = win.SaveWindowPNG(o.out) // capture mid-drag: the rubber-band is on screen
	native.InjectMouseButton(native.MouseLeft, false)
	frame()
	return err
}

// injectBoxDrag presses on an empty spot inside the viewport content (left of the box,
// below the tab strip) and drags a window down-right across the box, rendering each frame so
// the rubber-band is on screen when the caller captures. It leaves the button held.
func injectBoxDrag(frame func()) {
	native.InjectMousePos(400, 300)
	frame()
	frame()
	native.InjectMouseButton(native.MouseLeft, true)
	frame()
	for i := 1; i <= 6; i++ {
		native.InjectMousePos(400+float32(125*i), 300+float32(63*i))
		frame()
	}
}

// applySetup applies the requested scene mutations (scheme / orientation / edge / ground /
// overlay / environment) to the session before capture.
func applySetup(s *app.Session, o opts) error {
	if o.scheme != "" {
		if err := s.SetColorScheme(o.scheme); err != nil {
			return err
		}
	}
	if o.orient != "" {
		if err := s.SetViewOrientation(orientByName[o.orient], true); err != nil {
			return err
		}
	}
	applyDisplaySetup(s, o)
	return nil
}

// applyDisplaySetup applies the display / overlay / style / named-view / environment mutations.
func applyDisplaySetup(s *app.Session, o opts) {
	if o.edge != "" {
		applyEdgeColor(s, o.edge)
	}
	if o.ground != "" {
		applyGround(s, o.ground)
	}
	if o.overlay {
		applyOverlay(s)
	}
	if o.style != "" {
		applyStyle(s, o.style)
	}
	if o.named {
		applyNamedViews(s)
	}
	if o.styles {
		applyStylesPanel(s)
	}
	if o.dialog {
		s.OpenDisplaySettings()
	}
	if o.image {
		applyImageOverlay(s)
	}
	if o.noEnv {
		e := s.Environment()
		e.ShowImage = false
		s.SetEnvironment(e)
	}
}

// renderAndCapture opens the window, runs the real DrawChrome loop for n frames so the
// renderer settles, then saves the 3D viewport (or the whole window when wantWindow) to out.
func renderAndCapture(s *app.Session, o opts) error {
	win, err := native.CreateWindow(1280, 800, "m16shot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for i := 0; i < o.frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	if o.window {
		return win.SaveWindowPNG(o.out)
	}
	return win.SaveViewportPNG(o.out)
}

// applyNamedViews saves a couple of named views (from two orientations) and opens the Named
// Views panel, so a whole-window capture shows the populated browser (M16-F03 #404).
func applyNamedViews(s *app.Session) {
	_ = s.SetViewOrientation(orientByName["iso"], true)
	_, _ = s.CaptureNamedView("Iso Home")
	_ = s.SetViewOrientation(orientByName["top"], true)
	_, _ = s.CaptureNamedView("Top")
	s.OpenNamedViewsPanel()
}

// applyStylesPanel selects the first visible body and opens the Color Styles panel, so a
// whole-window capture shows the panel with the style list and the selected-body label.
func applyStylesPanel(s *app.Session) {
	if bodies := s.VisibleBodies(); len(bodies) > 0 {
		s.Selection().Add(app.BodyHandle{Body: bodies[0]})
	}
	s.OpenColorStylesPanel()
}

// applyEdgeColor parses "R,G,B" and stores it as the active document's display-settings edge
// color, so the rebuilt draw list draws the model edges in that color.
func applyEdgeColor(s *app.Session, rgb string) {
	var r, g, b uint8
	_, _ = fmt.Sscanf(rgb, "%d,%d,%d", &r, &g, &b)
	set := s.DisplaySettings(0)
	set.EdgeColor = types.NewColor(r, g, b)
	s.SetDisplaySettings(0, set)
}

// applyGround parses "R,G,B" as the active document's ground-plane color and enables ground
// shadows so the colored ground plane is drawn under the model.
func applyGround(s *app.Session, rgb string) {
	var r, g, b uint8
	_, _ = fmt.Sscanf(rgb, "%d,%d,%d", &r, &g, &b)
	set := s.DisplaySettings(0)
	set.GroundPlane.Color = types.NewColor(r, g, b)
	set.GroundPlane.Visible = true
	s.SetDisplaySettings(0, set)
	sh := s.ShadowSettings()
	sh.GroundShadows = true
	s.SetShadowSettings(sh)
}

// applyStyle assigns a named color style to the first visible body, so it renders in the
// style's color (M16-F02 #403/#408).
func applyStyle(s *app.Session, styleName string) {
	bodies := s.VisibleBodies()
	if len(bodies) == 0 {
		return
	}
	if err := s.AssignColorStyleToBody(string(bodies[0].ReferenceKey()), styleName); err != nil {
		panic(err)
	}
}

// applyImageOverlay writes a small checker PNG and adds an image-billboard overlay anchored at
// the model top, so a capture shows the host-loaded image floating over the box (M16-F05 #641).
func applyImageOverlay(s *app.Session) {
	path := "/tmp/m16-overlay.png"
	if err := writeCheckerPNG(path, 64); err != nil {
		panic(err)
	}
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{
		ClientId: "logo",
		Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsImage), ImagePath: path, Anchor: []float64{2, 5, 2.5}, ImageWidth: 4, ImageHeight: 4,
		}}}},
	})
	if err != nil {
		panic(err)
	}
	s.Graphics().Set(g)
}

// writeCheckerPNG writes an n×n magenta/cyan checker so the overlay is unmistakable in a capture.
func writeCheckerPNG(path string, n int) error {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := color.RGBA{R: 255, B: 255, A: 255} // magenta
			if (x/8+y/8)%2 == 0 {
				c = color.RGBA{G: 220, B: 255, A: 255} // cyan
			}
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// applyOverlay adds a red surface-overlay client-graphics group that highlights the first
// visible body by its reference key — the host tessellates it via the injected resolver
// (M16-F05 #641).
func applyOverlay(s *app.Session) {
	bodies := s.VisibleBodies()
	if len(bodies) == 0 {
		return
	}
	key := string(bodies[0].ReferenceKey())
	g, err := clientgraphics.DecodeGroup(wire.SetClientGraphicsArgs{
		ClientId: "highlight",
		Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind: string(types.GraphicsSurface), BodyKey: key, Color: []float32{1, 0.15, 0.15, 1}, OnTop: true,
		}}}},
	})
	if err != nil {
		panic(err)
	}
	s.Graphics().Set(g)
}

// buildBox seeds the active part with a 4x3x5 extruded box and frames it, so the captured
// viewport shows real geometry (for confirming orientation/overlay features).
func buildBox(s *app.Session) {
	pd, err := compdef.AddPart(s.Workspace(), "m16box.opd", true)
	if err != nil {
		panic(err)
	}
	if err := s.Workspace().SetActiveDocument(pd); err != nil {
		panic(err)
	}
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(4, 0))
	c2 := sk.Points().Add(math.P2(4, 3))
	c3 := sk.Points().Add(math.P2(0, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
}
