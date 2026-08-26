//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command scallopshot is a throwaway live-capture driver for the partial curved-on-planar boolean
// (#1591, ADR-0049): it models a plate and drills an edge-CLIPPING through-hole (an edge scallop,
// CutEdgeScallop) through the real feature pipeline, renders the production frame loop and saves a
// PNG — so the capture visually confirms the analytic scallop wall renders CRACK-FREE (the live
// equivalent of the headless freeEdgeCount==0 tessellation gate).
//
//	go run ./head/cmd/scallopshot -out /tmp/scallop   # writes /tmp/scallop-cut.png
package main

import (
	"flag"
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/cmd/internal/shotscene"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

func main() {
	out := flag.String("out", "/tmp/scallop", "output PNG prefix")
	frames := flag.Int("frames", 8, "frames to render before capture")
	flag.Parse()
	if err := run(*out+"-cut.png", *frames); err != nil {
		fmt.Fprintln(os.Stderr, "scallopshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out+"-cut.png")
}

func run(path string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	def := shotscene.BuildBox(s, "scallopshot-cut.opd", 10, 2) // plate [0,10]²×[0,2]
	sk := def.Sketches().Add(sketch.XYPlane())
	sk.Circles().AddByCenterRadius(math.P2(9, 5), 2) // clips the +x edge (x=10), signed distance d=1
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.Cut, func() float64 { return 2 })
	def.Recompute()

	win, err := native.CreateWindow(1280, 800, "scallopshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	return renderBody(win, s, def, math.P3(10, 5, 1), frames, path)
}

// renderBody aims the camera at the body's feature near target, runs the production frame loop and saves.
func renderBody(win *native.Window, s *app.Session, def *compdef.PartComponentDefinition, target math.Point3, frames int, path string) error {
	body := def.SurfaceBodies().Item(0)
	shotscene.AimCameraAtEdge(s, body, nearestEdge(body, target))
	for range frames {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(path)
}

// nearestEdge returns the body edge whose midpoint is closest to target (so the camera frames the notch).
func nearestEdge(body *topo.Body, target math.Point3) *topo.Edge {
	var best *topo.Edge
	bestD := math.Scalar(1e30)
	for _, e := range body.Edges() {
		mid := e.StartVertex().Point().Midpoint(e.EndVertex().Point())
		if d := mid.DistanceTo(target); d < bestD {
			bestD, best = d, e
		}
	}
	return best
}
