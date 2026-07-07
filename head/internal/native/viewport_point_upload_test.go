//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"
	"time"

	"oblikovati.org/renderer"
)

// TestViewportPointUploadRetained is the live #645 retained-buffer test: it drives the real
// offscreen viewport and asserts the point-cloud buffer is uploaded ONCE and then reused as the
// camera orbits — the CloudCompare-style static buffer. Orbiting a loaded scan must touch only the
// MVP push-constant, never the per-point PCIe transfer (the pathology the old per-frame cross-marker
// batch had). A genuine content change (new key) re-uploads exactly once. Skips on a headless box.
func TestViewportPointUploadRetained(t *testing.T) {
	w, err := CreateWindow(640, 480, "Oblikovati (#645 point upload test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()
	w.InitViewport()
	cam := smokeCamera()
	configureViewportSmokeLighting(w, [3]float32{-1, -1, -1}, [3]float32{1, 1, 1})
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}
	pts := gridPointVerts() // 9 cyan points, 7 floats each
	const nPts = 9

	// render uploads the points under key (skipped by the native side when unchanged) and draws a
	// point-only frame (no mesh geometry), nudging the camera each call to simulate an orbit.
	render := func(key uint64) {
		cam.Eye.X += 0.05 // simulated orbit: new MVP, same points
		mvp := renderer.ViewProjection(cam, 0.1, 100)
		w.BeginFrame()
		w.UploadPoints(pts, nPts, key, 3.0)
		w.RenderViewport(0, 640, 480, mvp[:], eye,
			nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil,
			0, 0, nil, nil, nil, 0)
		w.EndFrame(0.10, 0.10, 0.12)
	}

	const keyA, keyB uint64 = 0x5CA2, 0xC10D
	// Warm the frames-in-flight ring: the first use of each ring slot creates its target; the point
	// buffer itself is shared, so it uploads once for keyA and every later keyA frame skips.
	for i := 0; i < 7; i++ {
		render(keyA)
	}
	base := w.ViewportPointUploads()
	if base == 0 {
		t.Fatal("point buffer never uploaded — points would not render")
	}
	// Static orbit: same points (keyA), moving camera — ZERO further uploads. This is the #645 property.
	for i := 0; i < 5; i++ {
		render(keyA)
	}
	if got := w.ViewportPointUploads(); got != base {
		t.Errorf("static orbit re-uploaded points: uploads went %d→%d, want no change (only the MVP should change) — #645", base, got)
	}
	w.UploadPoints(pts, nPts, keyA, 7.0)
	if got := w.ViewportPointUploads(); got != base {
		t.Errorf("point-size-only update re-uploaded points: uploads went %d→%d, want no change", base, got)
	}
	// A genuine content change (new key) re-uploads exactly once, then skips.
	render(keyB)
	render(keyB)
	if got := w.ViewportPointUploads(); got != base+1 {
		t.Errorf("content change: uploads went %d→%d, want +1 (one re-upload then skip)", base, got)
	}
	// The points actually drew: the offscreen image is not blank.
	pixels, pw, ph, ok := w.ReadbackViewport(0)
	if !ok || pw <= 0 || ph <= 0 {
		t.Fatalf("ReadbackViewport failed: ok=%v %dx%d", ok, pw, ph)
	}
	if allZero(pixels) {
		t.Error("offscreen image is blank — the retained point buffer did not render")
	}
}

// TestViewportPointUploadScale drives the native path with a large cloud (2M points) to show the
// retained buffer replaces the old per-frame CPU cross-marker cost: the 2M points upload ONCE, then
// an orbit of many frames adds zero further uploads and each frame is a pure VRAM redraw. This is
// the "no longer dreadful" evidence for #645 — the old path rebuilt 2M×6 line verts every frame on
// the CPU. Logs the initial-upload and steady-orbit frame times. Skips on a headless box.
func TestViewportPointUploadScale(t *testing.T) {
	w, err := CreateWindow(640, 480, "Oblikovati (#645 point scale test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()
	w.InitViewport()
	cam := smokeCamera()
	configureViewportSmokeLighting(w, [3]float32{-1, -1, -1}, [3]float32{1, 1, 1})
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}

	const nPts = 2_000_000
	pts := bigCloudVerts(nPts)
	const key uint64 = 0xB16C10D

	frame := func() {
		cam.Eye.X += 0.03 // simulated orbit: new MVP, same points
		mvp := renderer.ViewProjection(cam, 0.1, 100)
		w.BeginFrame()
		w.UploadPoints(pts, nPts, key, 2.0)
		w.RenderViewport(0, 640, 480, mvp[:], eye,
			nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil,
			0, 0, nil, nil, nil, 0)
		w.EndFrame(0.10, 0.10, 0.12)
	}

	t0 := time.Now()
	frame() // first frame: uploads 2M points once
	t.Logf("2M-point first frame (incl. upload): %v", time.Since(t0))

	for i := 0; i < 6; i++ { // warm the frames-in-flight ring
		frame()
	}
	base := w.ViewportPointUploads()

	const orbit = 30
	torb := time.Now()
	for i := 0; i < orbit; i++ {
		frame()
	}
	per := time.Since(torb) / orbit
	t.Logf("2M-point steady orbit: %v/frame (retained buffer, MVP-only)", per)

	if got := w.ViewportPointUploads(); got != base {
		t.Errorf("2M-point orbit re-uploaded: uploads %d→%d, want no change — the buffer must be retained (#645)", base, got)
	}
	pixels, pw, ph, ok := w.ReadbackViewport(0)
	if !ok || pw <= 0 || ph <= 0 {
		t.Fatalf("ReadbackViewport failed: ok=%v %dx%d", ok, pw, ph)
	}
	if allZero(pixels) {
		t.Error("offscreen image is blank — 2M-point buffer did not render")
	}
}

// bigCloudVerts builds n cyan points in a deterministic filled cube around the origin, interleaved
// [pos.xyz, rgba]. No RNG (unavailable/undesired here) — a cheap integer hash spreads them.
func bigCloudVerts(n int) []float32 {
	v := make([]float32, 0, n*7)
	for i := 0; i < n; i++ {
		x := float32((i*2654435761)%2000)/1000 - 1 // ~[-1,1)
		y := float32((i*40503)%2000)/1000 - 1
		z := float32((i*97)%2000)/1000 - 1
		v = append(v, x, y, z, 0.25, 0.85, 1.0, 1)
	}
	return v
}

// gridPointVerts returns a 3×3 grid of cyan points around the origin, interleaved [pos.xyz, rgba]
// (kPointFloats), in view of smokeCamera.
func gridPointVerts() []float32 {
	var v []float32
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			v = append(v, float32(i), float32(j), 0, 0.25, 0.85, 1.0, 1)
		}
	}
	return v
}
