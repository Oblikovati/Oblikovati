//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command asmparamshot is a throwaway live-capture driver confirming the Assembly Parameters
// dialog renders: it opens a part with a user parameter, an assembly (made active) with its
// own parameter that links the part's parameter, opens Manage ▸ Parameters, and saves a PNG
// of the real DrawChrome viewport. Used to verify M39-F04 (#1560) end to end.
//
//	go run ./head/cmd/asmparamshot -out /tmp/asmparamshot.png
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
	out := flag.String("out", "/tmp/asmparamshot.png", "output PNG path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	flag.Parse()
	if err := run(*out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "asmparamshot:", err)
		os.Exit(1)
	}
}

func run(out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	gears, err := s.NewPart()
	if err != nil {
		return err
	}
	if err := s.AddNumericUserParameter("module", "2 mm"); err != nil {
		return err
	}
	if _, err := s.NewAssembly(); err != nil {
		return err
	}
	if err := s.AddNumericUserParameter("plateWidth", "40 mm"); err != nil {
		return err
	}
	if err := s.AddBooleanUserParameter("machined", true); err != nil {
		return err
	}
	if _, err := s.AddDerivedParameterTable(gears.FullDocumentName(), []string{"module"}); err != nil {
		return err
	}
	s.OpenParameters()
	fmt.Fprintf(os.Stdout, "assembly active=%v, %d derived tables\n", s.ParametersOpen(), len(s.DerivedTableRows()))
	return capture(s, out, frames)
}

func capture(s *app.Session, out string, frames int) error {
	win, err := native.CreateWindow(1400, 1000, "asmparamshot")
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
	// SaveWindowPNG (not SaveViewportPNG): the Parameters dialog is ImGui chrome over the
	// swapchain, not 3D viewport geometry.
	if err := win.SaveWindowPNG(out); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "wrote", out)
	return nil
}
