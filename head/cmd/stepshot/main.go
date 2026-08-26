//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command stepshot is a throwaway live-capture driver for imported-STEP tessellation review
// (#585): it imports a STEP file into a fresh part, frames it iso, and saves a shaded PNG and a
// Normal-Debug PNG (front-facing green, back-facing red) from the real DrawChrome viewport — so
// the image is exactly what the app renders, proving the freeform faces read smooth and fold-free.
//
//	go run ./head/cmd/stepshot -in /path/EDF.STEP -out /tmp/edf
package main

import (
	"flag"
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
)

func main() {
	in := flag.String("in", "", "STEP file to import")
	out := flag.String("out", "/tmp/stepshot", "output PNG prefix (writes <prefix>-shaded.png and <prefix>-normals.png)")
	frames := flag.Int("frames", 8, "frames to render before each capture")
	view := flag.String("view", "iso", "view orientation: iso|front|back|top|right")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "stepshot: -in is required")
		os.Exit(2)
	}
	if err := run(*in, *out, *frames, *view); err != nil {
		fmt.Fprintln(os.Stderr, "stepshot:", err)
		os.Exit(1)
	}
}

// orientation maps a -view name to its enum (defaults to the iso top-right view).
func orientation(view string) types.ViewOrientationTypeEnum {
	switch view {
	case "front":
		return types.FrontViewOrientation
	case "back":
		return types.BackViewOrientation
	case "top":
		return types.TopViewOrientation
	case "right":
		return types.RightViewOrientation
	default:
		return types.IsoTopRightViewOrientation
	}
}

func run(in, out string, frames int, view string) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if _, err := s.NewPart(); err != nil {
		return err
	}
	res, err := s.ImportFile(in)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "imported %d bodies from %s\n", res.BodyCount, in)
	if err := s.SetViewOrientation(orientation(view), true); err != nil {
		return err
	}
	s.TickCameraAnimation(100)
	return captureShadedAndNormals(s, out, frames)
}

// captureShadedAndNormals opens the live window and writes the shaded then Normal-Debug images.
func captureShadedAndNormals(s *app.Session, out string, frames int) error {
	win, err := native.CreateWindow(1280, 800, "stepshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()
	if err := capture(win, s, out+"-shaded.png", frames); err != nil {
		return err
	}
	s.SetNormalDebug(true)
	return capture(win, s, out+"-normals.png", frames)
}

// capture renders frames of the live chrome and writes the viewport image to path.
func capture(win *native.Window, s *app.Session, path string, frames int) error {
	for range frames {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	if err := win.SaveViewportPNG(path); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "wrote", path)
	return nil
}
