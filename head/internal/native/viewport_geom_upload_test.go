//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestViewportGeomUploadDirtySkip is the live #1422 stress test: it drives the real offscreen
// viewport (under lavapipe + xvfb in CI) and asserts the geometry buffer is re-uploaded ONLY when
// the geometry actually changes — a static scene being orbited touches only the MVP push-constant,
// not the whole-model PCIe transfer. It also pins the #1218 regression: recreating the target
// (a resize / dock-layout transition) must force a re-upload so the new target is never sampled
// stale-or-blank. Skips when no window can be created (a truly headless dev box).
func TestViewportGeomUploadDirtySkip(t *testing.T) {
	w, err := CreateWindow(640, 480, "Oblikovati (#1422 upload test)")
	if err != nil {
		t.Skipf("no window/GPU available: %v", err)
	}
	defer w.Destroy()
	w.InitViewport()
	verts, idx, min, max := smokeScene()
	configureViewportSmokeLighting(w, min, max)
	cam := smokeCamera()
	eye := []float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)}

	// render submits one frame for geomKey at w×h. Orbiting is simulated by nudging the camera each
	// call (a new MVP) WITHOUT changing the geometry — exactly the static-orbit path the skip targets.
	render := func(geomKey uint64, pw, ph int) {
		cam.Eye.X += 0.05 // nudge the camera only (simulated orbit); geometry stays unchanged
		mvp := renderer.ViewProjection(cam, 0.1, 100)
		w.BeginFrame()
		w.RenderViewport(0, pw, ph, mvp[:], eye, verts, len(verts)/16, idx,
			nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil, nil, 0, nil,
			nil, 0, nil, nil, 0, nil, // stroked-line streams (#2015)
			len(idx), 0, nil, nil, nil, geomKey)
		w.EndFrame(0.10, 0.10, 0.12)
	}

	const keyA, keyB uint64 = 0xA11CE, 0xB0B
	render(keyA, 640, 480)
	// Warm up the frames-in-flight ring (#1421): each ring slot has its own offscreen target, and the
	// first use of each creates it — a recreation that (correctly, per #1218) forces a re-upload. So
	// the steady state begins only after every ring slot has rendered keyA at this size.
	for i := 0; i < 6; i++ {
		render(keyA, 640, 480)
	}
	// Static orbit: same geometry (keyA), moving camera, several frames — ZERO further uploads, no
	// matter the ring depth. This is the #1422 property.
	base := w.ViewportGeomUploads()
	for i := 0; i < 5; i++ {
		render(keyA, 640, 480)
	}
	if got := w.ViewportGeomUploads(); got != base {
		t.Errorf("static orbit re-uploaded geometry: uploads went %d→%d, want no change (only the MVP should change) — #1422", base, got)
	}
	// A genuine geometry change (new key) re-uploads exactly once — the resident key is shared across
	// the ring, so once the first tile uploads keyB every other frame skips.
	render(keyB, 640, 480)
	render(keyB, 640, 480)
	if got := w.ViewportGeomUploads(); got != base+1 {
		t.Errorf("geometry change: uploads went %d→%d, want +1 (one re-upload then skip)", base, got)
	}
	// #1218 guard: a resize recreates each ring slot's target on its next use, and a recreated target
	// must re-upload (or it samples stale/blank geometry). Render enough frames to cover the ring; the
	// upload count MUST climb (no stale geometry), which is the regression M34-F4 missed.
	resizeBase := w.ViewportGeomUploads()
	for i := 0; i < 6; i++ {
		render(keyB, 400, 300)
	}
	if got := w.ViewportGeomUploads(); got <= resizeBase {
		t.Errorf("after resize: uploads stayed %d, want an increase (recreated targets must re-upload) — #1218", got)
	}
	// The legacy/unknown path (geomKey 0) never skips, so it always re-uploads — at least once per frame.
	legacyBase := w.ViewportGeomUploads()
	render(0, 400, 300)
	render(0, 400, 300)
	if got := w.ViewportGeomUploads(); got != legacyBase+2 {
		t.Errorf("legacy geomKey 0: uploads went %d→%d, want +2 (geomKey 0 always uploads)", legacyBase, got)
	}

	// Image-parity sanity: after all that, the offscreen image is still a real render, not blank.
	pixels, pw, ph, ok := w.ReadbackViewport(0)
	if !ok || pw <= 0 || ph <= 0 {
		t.Fatalf("ReadbackViewport failed: ok=%v %dx%d", ok, pw, ph)
	}
	if allZero(pixels) {
		t.Error("offscreen image is fully blank after the skip path — geometry was lost (regression vs the always-upload path)")
	}
}

// allZero reports whether every byte is 0 (a fully-black/transparent readback — i.e. nothing drew).
func allZero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}
