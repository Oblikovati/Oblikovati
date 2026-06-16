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
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// orientByName maps a CLI orientation name to a standard view orientation.
var orientByName = map[string]types.ViewOrientationTypeEnum{
	"front": types.FrontViewOrientation, "back": types.BackViewOrientation,
	"top": types.TopViewOrientation, "bottom": types.BottomViewOrientation,
	"left": types.LeftViewOrientation, "right": types.RightViewOrientation,
	"iso": types.IsoTopRightViewOrientation,
}

func main() {
	scheme := flag.String("scheme", "", "color scheme to activate before capture")
	out := flag.String("out", "/tmp/m16shot.png", "viewport PNG output path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	noEnv := flag.Bool("no-env", false, "turn off the environment skybox so the scheme background shows")
	box := flag.Bool("box", false, "build a demo box so geometry is visible")
	orient := flag.String("orient", "", "jump to a standard orientation (front/top/iso…)")
	flag.Parse()

	if err := run(opts{*scheme, *out, *frames, *noEnv, *box, *orient}); err != nil {
		fmt.Fprintln(os.Stderr, "m16shot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

// opts is the capture configuration parsed from the command line.
type opts struct {
	scheme, out string
	frames      int
	noEnv, box  bool
	orient      string
}

func run(o opts) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if _, err := s.NewPart(); err != nil {
		return err
	}
	if o.box {
		buildBox(s)
	}
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
	if o.noEnv {
		e := s.Environment()
		e.ShowImage = false
		s.SetEnvironment(e)
	}
	return renderAndCapture(s, o.frames, o.out)
}

// renderAndCapture opens the window, runs the real DrawChrome loop for n frames so the
// renderer settles, then saves the 3D viewport to out.
func renderAndCapture(s *app.Session, frames int, out string) error {
	win, err := native.CreateWindow(1280, 800, "m16shot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for i := 0; i < frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveViewportPNG(out)
}

// buildBox seeds the active part with a 4x3x5 extruded box and frames it, so the captured
// viewport shows real geometry (for confirming orientation/overlay features).
func buildBox(s *app.Session) {
	pd, err := compdef.AddPart(s.Workspace(), "m16box.opd", true)
	if err != nil {
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
