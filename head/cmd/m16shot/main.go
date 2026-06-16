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

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
)

func main() {
	scheme := flag.String("scheme", "", "color scheme to activate before capture")
	out := flag.String("out", "/tmp/m16shot.png", "viewport PNG output path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	noEnv := flag.Bool("no-env", false, "turn off the environment skybox so the scheme background shows")
	flag.Parse()

	if err := run(*scheme, *out, *frames, *noEnv); err != nil {
		fmt.Fprintln(os.Stderr, "m16shot:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", *out)
}

func run(scheme, out string, frames int, noEnv bool) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if _, err := s.NewPart(); err != nil {
		return err
	}
	if scheme != "" {
		if err := s.SetColorScheme(scheme); err != nil {
			return err
		}
	}
	if noEnv {
		e := s.Environment()
		e.ShowImage = false
		s.SetEnvironment(e)
	}
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
