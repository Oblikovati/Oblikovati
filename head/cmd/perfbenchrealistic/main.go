//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

// Command perfbenchrealistic measures Realistic-mode (ray-traced) orbit performance
// against a real document, mirroring cmd/perfbench's orbit scenario but forcing
// DisplayModeEnum.RealisticRendering — hardware ray tracing by default, -software to
// force the compute-BVH backend instead, for an apples-to-apples comparison. Built while
// diagnosing #2155's live-testing finding that camera-only orbiting was needlessly
// rebuilding the RT/SW scene every frame (since fixed) and that even a rebuild-free
// dispatch is expensive enough per pixel to justify Realistic mode's interactive preview
// resolution (realisticInteractiveDownscale, head/ui/realistic_render.go).
//
//	go run ./cmd/perfbenchrealistic -doc "/path/to/piston-head.opd" -frames 60 -png /tmp/out.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"sort"
	"time"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/head/ui"
	"oblikovati.org/persistence"
)

// printOut writes user-facing CLI text to stdout — a single indirection point so this
// file's own summary/progress lines aren't scattered raw fmt.Print* calls (CLAUDE.md:
// "plain text only for user-facing CLI output," which this genuinely is; the project
// lint rule wants that routed through one place per command, not that it never happen).
func printOut(format string, args ...any) { fmt.Fprintf(os.Stdout, format+"\n", args...) }

func main() {
	docPath := flag.String("doc", "", "document (.opd/.oad) to open")
	frames := flag.Int("frames", 60, "frames in the orbit")
	pngPath := flag.String("png", "", "save a viewport PNG after the orbit")
	software := flag.Bool("software", false, "force the software backend instead of hardware RT")
	flag.Parse()
	if *docPath == "" {
		fmt.Fprintln(os.Stderr, "usage: perfbenchrealistic -doc path/to/file.opd")
		os.Exit(2)
	}
	if err := run(*docPath, *frames, *pngPath, *software); err != nil {
		fmt.Fprintln(os.Stderr, "perfbenchrealistic:", err)
		os.Exit(1)
	}
}

func run(docPath string, frames int, pngPath string, software bool) error {
	s, err := openDocument(docPath)
	if err != nil {
		return err
	}

	win, err := native.CreateWindow(1600, 1000, "perfbenchrealistic")
	if err != nil {
		return err
	}
	defer win.Destroy()
	win.InitViewport()

	if err := configureRealisticScene(s, software); err != nil {
		return err
	}
	s.FitView()
	warmUp(win, s)

	total, times := orbit(win, s, frames)
	report(frames, total, times)
	settle(win, s)

	if pngPath == "" {
		return nil
	}
	return writeResultPNG(win, pngPath)
}

// openDocument loads docPath into a fresh session and makes it the active document.
func openDocument(docPath string) (*app.Session, error) {
	s := app.NewSessionWithStore(persistence.NewPackageStore())
	root, err := s.Workspace().Open(docPath, true)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", docPath, err)
	}
	if err := s.Workspace().SetActiveDocument(root); err != nil {
		return nil, err
	}
	return s, nil
}

// writeResultPNG saves the converged render and reports the path written.
func writeResultPNG(win *native.Window, pngPath string) error {
	if err := saveWindowPNG(win, pngPath); err != nil {
		return fmt.Errorf("save png: %w", err)
	}
	printOut("wrote %s", pngPath)
	return nil
}

// warmUp renders one untimed frame — the pipeline/BLAS/TLAS build happens on the first
// dispatch, so timing it would conflate one-time setup cost with per-frame cost.
func warmUp(win *native.Window, s *app.Session) {
	fmt.Fprintln(os.Stderr, "warm-up frame: starting")
	w0 := time.Now()
	drawFrame(win, s)
	fmt.Fprintf(os.Stderr, "warm-up frame: done in %.1fms\n", float64(time.Since(w0).Microseconds())/1000)
}

// configureRealisticScene puts s into Realistic mode with a three-point light rig and an
// HDR sky environment (the scene this tool's own live-testing history was diagnosed
// under), and the requested ray-tracing backend.
func configureRealisticScene(s *app.Session, software bool) error {
	if err := s.SetDisplayMode(types.RealisticRendering); err != nil {
		return fmt.Errorf("SetDisplayMode: %w", err)
	}
	if err := s.SetLightingStyle("Three Point"); err != nil {
		return fmt.Errorf("SetLightingStyle: %w", err)
	}
	s.SetEnvironment(app.EnvironmentState{Preset: "Sky", Intensity: 1, ShowImage: true})
	hwOn := !software
	prefs := s.ViewCubePrefs()
	prefs.HardwareRayTracing = &hwOn
	s.SetViewCubePrefs(prefs)
	return nil
}

// orbit drives one full 360° camera orbit, one drawFrame per step, timing each frame.
func orbit(win *native.Window, s *app.Session, frames int) (total time.Duration, times []float64) {
	yaw := 2 * math.Pi / float64(frames)
	times = make([]float64, 0, frames)
	t0 := time.Now()
	for range frames {
		s.SetCamera(s.Camera().Orbit(yaw, 0))
		f0 := time.Now()
		drawFrame(win, s)
		times = append(times, float64(time.Since(f0).Microseconds())/1000)
	}
	return time.Since(t0), times
}

// settle holds the camera still and accumulates more samples — orbit ends mid-motion, so
// a PNG saved immediately after would show the reduced-resolution interactive preview
// (realisticInteractiveDownscale), not a converged image.
func settle(win *native.Window, s *app.Session) {
	const settleFrames = 40
	for range settleFrames {
		drawFrame(win, s)
	}
}

// saveWindowPNG reads back the composited swapchain (not SaveViewportPNG's offscreen
// raster target, which Realistic mode's live path bypasses entirely — #2149) so the
// saved image actually shows the live Realistic-mode render.
func saveWindowPNG(win *native.Window, path string) error {
	px, w, h, ok := win.ReadbackWindow()
	if !ok || w == 0 || h == 0 {
		return fmt.Errorf("ReadbackWindow failed or empty (%dx%d)", w, h)
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i+3 < len(px); i += 4 {
		b, g, r, a := px[i], px[i+1], px[i+2], px[i+3]
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = r, g, b, a
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawFrame(win *native.Window, s *app.Session) {
	win.BeginFrame()
	ui.DrawChrome(win, s)
	win.EndFrame(ui.WindowClearColor())
}

func report(frames int, total time.Duration, times []float64) {
	sorted := append([]float64(nil), times...)
	sort.Float64s(sorted)
	median := sorted[len(sorted)/2]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	max := sorted[len(sorted)-1]
	min := sorted[0]
	printOut("Realistic mode, %d frames, %.1fs total", frames, total.Seconds())
	printOut("  frame time: min=%.1fms median=%.1fms p95=%.1fms max=%.1fms", min, median, p95, max)
	printOut("  fps: median=%.1f p95(worst-case)=%.1f", 1000/median, 1000/p95)
}
