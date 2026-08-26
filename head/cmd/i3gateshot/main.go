//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command i3gateshot is a throwaway live-capture driver for the #1626 activation seam:
// CombineTool skipped the sick-config commit gate entirely before I3 (no DraftFeature, plain
// StartTool). It now activates via StartFeatureTool, so the gate must block a sick combine —
// the same body picked as both target and tool — with OK DISABLED and the amber "why" line,
// and enable OK for a healthy two-body join. Captures the real window both ways.
//
//	go run ./head/cmd/i3gateshot -out /tmp/i3gate   # writes /tmp/i3gate-sick.png and /tmp/i3gate-ok.png
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
	"oblikovati.org/model/feature"
)

func main() {
	out := flag.String("out", "/tmp/i3gate", "output PNG prefix (writes <prefix>-sick.png and <prefix>-ok.png)")
	frames := flag.Int("frames", 8, "frames to render before each capture")
	flag.Parse()
	if err := run(*out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "i3gateshot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out+"-sick.png", "and", *out+"-ok.png")
}

func run(out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	// Two disjoint boxes: a valid join pair, and (picked twice) a sick self-combine.
	def := shotscene.BuildBox(s, "i3gateshot.opd", 4, 4)
	sk2 := shotscene.AddSquare(def, 8, 0, 4)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk2, 0, ops.NewBody, func() float64 { return 4 })
	def.Recompute()
	bodies := def.SurfaceBodies()
	if bodies.Count() < 2 {
		return fmt.Errorf("scene built %d bodies, want 2", bodies.Count())
	}
	b0, b1 := bodies.Item(0), bodies.Item(1)
	shotscene.AimCameraAtEdge(s, b0, shotscene.VerticalEdge(b0.Edges()))

	win, err := native.CreateWindow(1280, 800, "i3gateshot")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()

	// Sick: the same body as target AND tool — the draft previews sick, the gate must hold.
	if err := capture(win, s, []*topo.Body{b0, b0}, true, frames, out+"-sick.png"); err != nil {
		return err
	}
	s.CancelTool()
	// Healthy: two distinct bodies join cleanly — OK must be enabled, no warning line.
	return capture(win, s, []*topo.Body{b0, b1}, false, frames, out+"-ok.png")
}

// capture starts a fresh Combine through the #1626 seam, picks the bodies, asserts the gate's
// blocked-state matches wantBlocked, then renders and saves the window to path.
func capture(win *native.Window, s *app.Session, picks []*topo.Body, wantBlocked bool, frames int, path string) error {
	combine := app.NewCombineTool()
	s.StartFeatureTool(combine)
	for _, b := range picks {
		combine.Pick(s, app.BodyHandle{Body: b})
	}
	if r := s.CommitBlockedReason(); (r != "") != wantBlocked {
		return fmt.Errorf("picks %d: blocked=%q, want blocked=%v", len(picks), r, wantBlocked)
	}
	for range frames {
		win.BeginFrame()
		ui.DrawChrome(win, s)
		win.EndFrame(ui.WindowClearColor())
	}
	return win.SaveWindowPNG(path)
}
