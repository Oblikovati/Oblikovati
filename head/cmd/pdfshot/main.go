//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command pdfshot is a throwaway live-capture driver for the vector-PDF importer: it imports
// a CAD plot-to-PDF onto the XY plane of a fresh part, frames it top-down, and saves a PNG
// from the real DrawChrome viewport — so the image is exactly what the app renders, proving
// the imported sketch reproduces the source drawing.
//
//	go run ./head/cmd/pdfshot -in /path/plan.pdf -out /tmp/pdfshot.png
package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/pdf"
	gmath "oblikovati.org/math"
)

func main() {
	in := flag.String("in", "", "PDF file to import")
	out := flag.String("out", "/tmp/pdfshot.png", "output PNG path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "pdfshot: -in is required")
		os.Exit(2)
	}
	if err := run(*in, *out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "pdfshot:", err)
		os.Exit(1)
	}
}

func run(in, out string, frames int) error {
	s := app.NewSession()
	if err := app.RegisterStandardCommands(s); err != nil {
		return err
	}
	if _, err := s.NewPart(); err != nil {
		return err
	}
	res, err := s.ImportPDFOnPlane(in, "XY Plane")
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "imported %d entities (3D=%v, %d warnings) from %s\n", res.EntityCount, res.Is3D, len(res.Warnings), in)
	// The app's Zoom-All frames body bounds, which a sketch-only part lacks; frame the
	// drawing's own extent (computed from the decoded geometry) so the capture shows it all.
	frameDrawing(s, in)
	s.TickCameraAnimation(100)
	return capture(s, out, frames)
}

// frameDrawing points a top-down camera at the imported drawing's bounding box (in the
// model's cm unit), so the whole sketch is visible in the capture.
func frameDrawing(s *app.Session, in string) {
	box, ok := drawingBox(in)
	if !ok {
		return
	}
	cam := s.Camera()
	cam.Up = gmath.V3(0, 1, 0)
	cam.Target = box.Center()
	cam.Eye = box.Center().TranslateBy(gmath.V3(0, 0, 1).Scale(float64(box.Diagonal().Length())))
	s.SetCamera(cam.Fit(box))
}

// drawingBox decodes the PDF and returns the bounding box of all page geometry, converted
// from the decoder's millimetres to the model's centimetres (1 unit = 10 mm).
func drawingBox(in string) (gmath.Box, bool) {
	data, err := os.ReadFile(in)
	if err != nil {
		return gmath.Box{}, false
	}
	pages, _, err := pdf.DecodePages(data)
	if err != nil {
		return gmath.Box{}, false
	}
	lo := gmath.P3(math.Inf(1), math.Inf(1), -1)
	hi := gmath.P3(math.Inf(-1), math.Inf(-1), 1)
	for _, dr := range pages {
		lo, hi = accumulate(dr, lo, hi)
	}
	if math.IsInf(float64(lo.X), 1) {
		return gmath.Box{}, false
	}
	return gmath.NewBox(lo, hi), true
}

// accumulate expands lo/hi by every polyline vertex and spline control point of a page.
func accumulate(dr *drawing.Drawing, lo, hi gmath.Point3) (gmath.Point3, gmath.Point3) {
	for _, e := range dr.Entities {
		switch g := e.(type) {
		case *drawing.LwPolyline:
			for _, p := range g.Points {
				lo, hi = expand(lo, hi, p[0], p[1])
			}
		case *drawing.Spline:
			for _, p := range g.ControlPoints {
				lo, hi = expand(lo, hi, p[0], p[1])
			}
		}
	}
	return lo, hi
}

// expand grows the lo/hi corners to include (x, y) — converting the decoder's millimetres
// to the model's centimetres (1 unit = 10 mm) — while keeping the z margin.
func expand(lo, hi gmath.Point3, xmm, ymm float64) (gmath.Point3, gmath.Point3) {
	x, y := xmm*0.1, ymm*0.1
	return gmath.P3(math.Min(float64(lo.X), x), math.Min(float64(lo.Y), y), float64(lo.Z)),
		gmath.P3(math.Max(float64(hi.X), x), math.Max(float64(hi.Y), y), float64(hi.Z))
}

// capture opens the live window, renders frames of the real chrome, and writes the viewport.
func capture(s *app.Session, out string, frames int) error {
	win, err := native.CreateWindow(1400, 1000, "pdfshot")
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
	if err := win.SaveViewportPNG(out); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "wrote", out)
	return nil
}
