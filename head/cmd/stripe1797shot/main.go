//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command stripe1797shot is a throwaway live-capture driver for the #1797 tangent-stripe fillet
// (ADR-0050 P4b): it builds a 4×4×4 box, rounds the 4 vertical edges (r=0.5 → 4 quarter-cylinders),
// then fillets the WHOLE top perimeter (r=0.25) — a closed tangent chain of 4 straight + 4 arc edges
// that now builds as one continuous blend stripe — and renders the real body so the PNG shows the
// rounded top rim flowing smoothly through the corners (no facets, no gaps).
//
//	DISPLAY=:1 go run ./head/cmd/stripe1797shot -out /tmp/stripe1797.png
package main

import (
	"flag"
	"fmt"
	stdmath "math"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/cmd/internal/shotscene"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/scene"
)

func main() {
	out := flag.String("out", "/tmp/stripe1797.png", "output PNG path")
	frames := flag.Int("frames", 12, "frames to render before capture")
	flag.BoolVar(&closeUp, "close", false, "close-up on a top corner")
	flag.BoolVar(&openRun, "open", false, "fillet only a straight–arc–straight OPEN sub-run (flat caps)")
	flag.Parse()
	if err := run(*out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "stripe1797shot:", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "wrote", *out)
}

func run(out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	def := shotscene.BuildBox(s, "stripe1797.opd", 4, 4)
	if err := filletVerticals(def); err != nil {
		return err
	}
	rim := filletTopRim
	if openRun {
		rim = filletTopRimOpen
	}
	if err := rim(def); err != nil {
		return err
	}
	body := def.SurfaceBodies().Item(0)
	if rep := ops.Validate(body); !rep.Valid || !body.IsSolid() {
		return fmt.Errorf("result not a valid solid: %+v", rep.Issues)
	}
	fmt.Fprintf(os.Stdout, "result: valid solid, %d faces, chi=%d\n", len(body.Faces()), ops.Validate(body).EulerCharacteristic)
	aimTop(s, body)

	win, err := native.CreateWindow(1280, 800, "stripe1797")
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
	return win.SaveWindowPNG(out)
}

// filletVerticals rounds the four vertical edges of the box at r=0.5.
func filletVerticals(def *compdef.PartComponentDefinition) error {
	body := def.SurfaceBodies().Item(0)
	var keys [][]byte
	for _, e := range body.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	feature.NewDressUpFeatures(def.Features()).AddFillet(keys, func() float64 { return 0.5 })
	def.Recompute()
	return nil
}

// filletTopRim rounds the whole top perimeter at r=0.25 — the tangent-stripe path.
func filletTopRim(def *compdef.PartComponentDefinition) error {
	body := def.SurfaceBodies().Item(0)
	maxZ := 0.0
	for _, v := range body.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	var keys [][]byte
	for _, e := range body.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 8 {
		return fmt.Errorf("expected 8 top-rim edges, found %d", len(keys))
	}
	feature.NewDressUpFeatures(def.Features()).AddFillet(keys, func() float64 { return 0.25 })
	def.Recompute()
	return nil
}

// filletTopRimOpen rounds only a contiguous straight–arc–straight OPEN sub-run of the top perimeter
// (r=0.25) — the ADR-0050 P6 open-stripe path, which terminates each end in a flat setback cap.
func filletTopRimOpen(def *compdef.PartComponentDefinition) error {
	body := def.SurfaceBodies().Item(0)
	seed, err := straightTopSeed(body)
	if err != nil {
		return err
	}
	chain, _, err := ops.TangentEdgeChain(body, seed, ops.DefaultTangentChainAngle)
	if err != nil {
		return err
	}
	if len(chain) < 3 {
		return fmt.Errorf("top rim chain too short: %d edges", len(chain))
	}
	feature.NewDressUpFeatures(def.Features()).AddFillet(chain[:3], func() float64 { return 0.25 })
	def.Recompute()
	return nil
}

// straightTopSeed returns a top-rim straight (non-arc) edge key to anchor the tangent chain.
func straightTopSeed(body *topo.Body) ([]byte, error) {
	maxZ := 0.0
	for _, v := range body.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	for _, e := range body.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			if _, isArc := e.Geometry().(geom.Arc3d); !isArc {
				return e.ReferenceKey(), nil
			}
		}
	}
	return nil, fmt.Errorf("no straight top-rim edge found")
}

// aimTop frames the box from an above-front three-quarter angle so the rounded top rim and its
// corners face the camera.
func aimTop(s *app.Session, body *topo.Body) {
	rb := body.RangeBox()
	span := stdmath.Max(float64(rb.Max.X-rb.Min.X), float64(rb.Max.Y-rb.Min.Y))
	if closeUp {
		// Frame the +X/−Y top corner, where a straight cylinder-blend meets an arc torus-blend.
		corner := math.P3(rb.Max.X, rb.Min.Y, rb.Max.Z)
		cam := scene.NewCamera(1280, 800)
		cam.Target = corner
		cam.Eye = math.P3(float64(corner.X)+span*0.5, float64(corner.Y)-span*0.6, float64(corner.Z)+span*0.45)
		cam.Up = math.V3(0, 0, 1)
		s.SetCamera(cam)
		return
	}
	cx, cy := (rb.Min.X+rb.Max.X)/2, (rb.Min.Y+rb.Max.Y)/2
	cam := scene.NewCamera(1280, 800)
	cam.Target = math.P3(cx, cy, rb.Max.Z-0.3)
	cam.Eye = math.P3(cx+span*1.4, cy-span*1.6, float64(rb.Max.Z)+span*1.3)
	cam.Up = math.V3(0, 0, 1)
	s.SetCamera(cam)
}

var (
	closeUp bool
	openRun bool
)
