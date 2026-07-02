//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command filletgateshot is a throwaway live-capture driver for the sick-config commit gate:
// it opens the real native window, starts the Fillet tool on a base cube with a radius the block
// cannot admit (a sick configuration), runs the production DrawChrome frame loop, and saves the
// whole window — so the PNG shows the OK button DISABLED with the amber "why" line exactly as the
// app draws it. It then repeats with a buildable radius so the second PNG shows OK ENABLED.
//
//	go run ./head/cmd/filletgateshot -out /tmp/gate      # writes /tmp/gate-sick.png and /tmp/gate-ok.png
package main

import (
	"flag"
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/cmd/internal/shotscene"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/topo"
)

const gateDocName = "filletgateshot.opd"

func main() {
	out := flag.String("out", "/tmp/gate", "output PNG prefix (writes <prefix>-sick.png and <prefix>-ok.png)")
	frames := flag.Int("frames", 8, "frames to render before each capture")
	flag.Parse()
	if err := run(*out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "filletgateshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out+"-sick.png", "and", *out+"-ok.png")
}

func run(out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	def := shotscene.BuildBox(s, gateDocName, 6, 6)
	body := def.SurfaceBodies().Item(0)
	edge := shotscene.VerticalEdge(body.Edges())
	shotscene.AimCameraAtEdge(s, body, edge)

	win, err := native.CreateWindow(1280, 800, "filletgateshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()

	// The Fillet panel seeds its radius buffer once per tool instance and writes it back to the
	// tool every frame, so each capture needs a FRESH tool started at its own radius.
	// Sick: a 10-unit rolling ball cannot round a 6×6×6 cube's edge — the gate must disable OK.
	if err := capture(win, s, edge, 10, true, frames, out+"-sick.png"); err != nil {
		return err
	}
	s.CancelTool()
	// Buildable: a small radius previews healthy — OK must be enabled, no warning line.
	return capture(win, s, edge, 1.2, false, frames, out+"-ok.png")
}

// capture starts a fresh Fillet at radius on edge, asserts the gate's blocked-state matches
// wantBlocked, then renders and saves the window (panel + viewport) to path.
func capture(win *native.Window, s *app.Session, edge *topo.Edge, radius float64, wantBlocked bool, frames int, path string) error {
	fillet := app.NewFilletTool()
	s.StartTool(fillet)
	fillet.Pick(s, app.EdgeHandle{Edge: edge})
	fillet.SetRadius(radius)
	if r := s.CommitBlockedReason(); (r != "") != wantBlocked {
		return fmt.Errorf("radius %g: blocked=%q, want blocked=%v", radius, r, wantBlocked)
	}
	for i := 0; i < frames; i++ {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(path)
}
