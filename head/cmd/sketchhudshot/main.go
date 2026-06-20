//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command sketchhudshot is a throwaway live-capture driver for the 2D-sketch dynamic-input HUD
// (#790) and Relax Mode (#791): it opens the real native window, creates a part, enters a
// sketch, starts the Line tool, places a first point, injects a cursor over the canvas, types a
// precise length into the HUD, runs the production DrawChrome loop, and saves the window PNG —
// so the image is exactly what the app draws (the heads-up panel near the cursor, the Relax
// Mode toggle on the command-window control row).
//
//	go run ./head/cmd/sketchhudshot -out /tmp/sketchhud.png
package main

import (
	"flag"
	"fmt"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
)

func main() {
	out := flag.String("out", "/tmp/sketchhud.png", "window PNG output path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	relax := flag.Bool("relax", false, "turn on Relax Mode for the capture")
	flag.Parse()
	if err := run(*out, *frames, *relax); err != nil {
		fmt.Fprintln(os.Stderr, "sketchhudshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(out string, frames int, relax bool) error {
	s, err := buildSketchScene(relax)
	if err != nil {
		return err
	}
	win, err := native.CreateWindow(1280, 800, "sketchhudshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	for _, r := range "75" { // type a precise length so the HUD shows an engaged field
		s.HUDInputRune(r)
	}
	for i := 0; i < frames; i++ {
		native.InjectMousePos(760, 470) // park the cursor over the canvas so the HUD draws there
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(out)
}

// buildSketchScene returns a session in a fresh part's 2D sketch with the Line tool active and
// its first point placed, so the HUD shows Length/Angle relative to it.
func buildSketchScene(relax bool) (*app.Session, error) {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return nil, err
	}
	if _, err := s.NewPart(); err != nil {
		return nil, err
	}
	if _, err := s.CreateSketchOnOrigin(app.OriginXY); err != nil {
		return nil, err
	}
	s.SetRelaxMode(relax)
	s.TickCameraAnimation(100) // settle the camera facing the sketch plane so picks/HUD map
	s.StartTool(app.NewLineTool())
	s.Click(560, 360) // place the line's first point → the HUD switches to Length/Angle
	return s, nil
}
