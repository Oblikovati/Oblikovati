//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command dwgshot is a throwaway live-capture driver to confirm a DWG import renders: it
// imports a .dwg onto the XY plane of a fresh part, frames it top-down from the decoded
// geometry's extent, and saves a PNG from the real DrawChrome viewport. Used to verify the
// #1549 fix (large object-map location deltas) end to end.
//
//	go run ./head/cmd/dwgshot -in /path/file.dwg -out /tmp/dwgshot.png
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/kernel/exchange/dwg"
	gmath "oblikovati.org/math"
)

func main() {
	in := flag.String("in", "", "DWG file to import")
	out := flag.String("out", "/tmp/dwgshot.png", "output PNG path")
	frames := flag.Int("frames", 8, "frames to render before capture")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "dwgshot: -in is required")
		os.Exit(2)
	}
	if err := run(*in, *out, *frames); err != nil {
		fmt.Fprintln(os.Stderr, "dwgshot:", err)
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
	res, err := s.ImportDWGOnPlane(in, "XY Plane")
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "imported %d entities (3D=%v, %d warnings) from %s\n", res.EntityCount, res.Is3D, len(res.Warnings), in)
	frameDrawing(s, in)
	s.TickCameraAnimation(100)
	return capture(s, out, frames)
}

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

// drawingBox returns an origin-centred framing box for the drawing's dense core. A
// georeferenced survey carries a few stray entities millions of units away that would blow up
// a raw min/max box, so we mirror the importer: subtract the per-axis median (which it shifts
// to the origin) and frame the 95th percentile of |offset|, in the model's cm (1 unit = 10 mm).
func drawingBox(in string) (gmath.Box, bool) {
	data, err := os.ReadFile(in)
	if err != nil {
		return gmath.Box{}, false
	}
	dr, _, err := dwg.Decode(data)
	if err != nil {
		return gmath.Box{}, false
	}
	xs, ys := points(dr)
	if len(xs) == 0 {
		return gmath.Box{}, false
	}
	hx := percentileAbs(xs, gmath.Median(append([]float64{}, xs...)), 0.95)
	hy := percentileAbs(ys, gmath.Median(append([]float64{}, ys...)), 0.95)
	return gmath.NewBox(gmath.P3(-hx*0.1, -hy*0.1, -1), gmath.P3(hx*0.1, hy*0.1, 1)), true
}

// points collects the 2D vertices of every line and polyline (in the decoder's millimetres).
func points(dr *drawing.Drawing) ([]float64, []float64) {
	var xs, ys []float64
	for _, e := range dr.Entities {
		switch g := e.(type) {
		case *drawing.LwPolyline:
			for _, p := range g.Points {
				xs, ys = append(xs, p[0]), append(ys, p[1])
			}
		case *drawing.Line:
			xs, ys = append(xs, g.Start[0], g.End[0]), append(ys, g.Start[1], g.End[1])
		}
	}
	return xs, ys
}

// percentileAbs returns the q-quantile of |v - center| over vs (the robust half-extent).
func percentileAbs(vs []float64, center, q float64) float64 {
	d := make([]float64, len(vs))
	for i, v := range vs {
		d[i] = math.Abs(v - center)
	}
	sort.Float64s(d)
	return d[int(q*float64(len(d)-1))]
}

func capture(s *app.Session, out string, frames int) error {
	win, err := native.CreateWindow(1400, 1000, "dwgshot")
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
