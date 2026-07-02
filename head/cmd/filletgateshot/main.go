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
	stdmath "math"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/scene"
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
	def := buildBaseBox(s)
	body := def.SurfaceBodies().Item(0)
	edge := verticalEdge(body.Edges())
	aimCameraAtEdge(s, body, edge)

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

func buildBaseBox(s *app.Session) *compdef.PartComponentDefinition {
	pd, err := compdef.AddPart(s.Workspace(), gateDocName, true)
	if err != nil {
		panic(err)
	}
	_ = s.Workspace().SetActiveDocument(pd)
	def := pd.Content().(*compdef.PartComponentDefinition)
	sk := addSquare(def, 0, 0, 6)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 6 })
	def.Recompute()
	return def
}

func addSquare(def *compdef.PartComponentDefinition, ox, oy, side float64) *sketch.Sketch {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(ox, oy))
	c1 := sk.Points().Add(math.P2(ox+side, oy))
	c2 := sk.Points().Add(math.P2(ox+side, oy+side))
	c3 := sk.Points().Add(math.P2(ox, oy+side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return sk
}

// aimCameraAtEdge frames the body from outside the given edge so the filleted corner faces us.
func aimCameraAtEdge(s *app.Session, body *topo.Body, e *topo.Edge) {
	pts := ops.TessellateEdge(e, ops.DefaultQuality())
	mid := pts[len(pts)/2]
	rb := body.RangeBox()
	cx, cy := (rb.Min.X+rb.Max.X)/2, (rb.Min.Y+rb.Max.Y)/2
	ox, oy := mid.X-cx, mid.Y-cy
	n := stdmath.Hypot(ox, oy)
	if n == 0 {
		n = 1
	}
	dist := (rb.Max.X - rb.Min.X) * 2.4
	cam := scene.NewCamera(1280, 800)
	cam.Target = mid
	cam.Eye = math.P3(mid.X+ox/n*dist, mid.Y+oy/n*dist, mid.Z+dist*0.5)
	cam.Up = math.V3(0, 0, 1)
	s.SetCamera(cam)
}

// verticalEdge returns the first edge running mostly along Z (a box's vertical corner).
func verticalEdge(edges []*topo.Edge) *topo.Edge {
	for _, e := range edges {
		pts := ops.TessellateEdge(e, ops.DefaultQuality())
		if len(pts) < 2 {
			continue
		}
		a, b := pts[0], pts[len(pts)-1]
		dz := abs(a.Z - b.Z)
		if dz > abs(a.X-b.X) && dz > abs(a.Y-b.Y) {
			return e
		}
	}
	return edges[0]
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
