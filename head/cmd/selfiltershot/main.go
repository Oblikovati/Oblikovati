//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// selfiltershot is a throwaway live-confirmation driver for the Selection Filter & Priority
// window (#1222): it builds a box part, opens the window through the real Session, draws several
// real DrawChrome frames, and saves a full-window PNG so the window can be verified visually.
package main

import (
	"flag"
	"log"

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

func main() {
	out := flag.String("out", "/tmp/selfiltershot.png", "output PNG path")
	frames := flag.Int("frames", 30, "frames to render before capture")
	reorder := flag.Bool("reorder", false, "move Faces to the top and disable Vertices to show an edited state")
	flag.Parse()
	if err := run(*out, *frames, *reorder); err != nil {
		log.Fatal(err)
	}
}

func run(out string, frames int, reorder bool) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	buildBox(s)
	if err := s.SetViewOrientation(types.IsoTopRightViewOrientation, true); err != nil {
		return err
	}
	s.TickCameraAnimation(100)

	s.OpenSelectionFilterWindow()
	if reorder {
		st := s.SelectionFilterState()
		st.Move(st.Rank(app.SelectFace), 0)
		st.SetEnabled(app.SelectVertex, false)
	}

	win, err := native.CreateWindow(1280, 800, "selfiltershot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for range frames {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(out)
}

func buildBox(s *app.Session) {
	pd, err := compdef.AddPart(s.Workspace(), "selfilter.opd", true)
	if err != nil {
		panic(err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(0, 0))
	c1 := sk.Points().Add(math.P2(6, 0))
	c2 := sk.Points().Add(math.P2(6, 6))
	c3 := sk.Points().Add(math.P2(0, 6))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 6 })
	def.Recompute()
}
