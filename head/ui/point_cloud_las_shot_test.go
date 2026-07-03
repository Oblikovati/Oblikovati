//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// synthLASCube builds a minimal valid uncompressed LAS file whose points form the shell of a cube
// (scale 1, offset 0, point data record format 0), so the live capture has a recognisable shape.
// There is no public-domain LAS in the sample set, so this stands in for real LiDAR data.
func synthLASCube() []byte {
	const headerSize, recordLen = 227, 20
	var pts [][3]int32
	for x := 0; x <= 100; x += 10 {
		for y := 0; y <= 100; y += 10 {
			for z := 0; z <= 100; z += 10 {
				if x == 0 || x == 100 || y == 0 || y == 100 || z == 0 || z == 100 {
					pts = append(pts, [3]int32{int32(x), int32(y), int32(z)})
				}
			}
		}
	}
	h := make([]byte, headerSize)
	copy(h, "LASF")
	h[24], h[25] = 1, 2
	binary.LittleEndian.PutUint16(h[94:], headerSize)
	binary.LittleEndian.PutUint32(h[96:], headerSize)
	binary.LittleEndian.PutUint16(h[105:], recordLen)
	binary.LittleEndian.PutUint32(h[107:], uint32(len(pts)))
	for i := 0; i < 3; i++ {
		binary.LittleEndian.PutUint64(h[131+i*8:], math.Float64bits(1.0)) // scale = 1
	}
	out := h
	for _, p := range pts {
		r := make([]byte, recordLen)
		binary.LittleEndian.PutUint32(r[0:], uint32(p[0]))
		binary.LittleEndian.PutUint32(r[4:], uint32(p[1]))
		binary.LittleEndian.PutUint32(r[8:], uint32(p[2]))
		out = append(out, r...)
	}
	return out
}

// TestInWindowSyntheticLASRenders writes a synthetic .las cube to a temp file, imports it through
// the real AttachPointCloud path, and captures the live viewport — the visual confirmation that a
// .las file decodes and renders as a point cloud (#645). Skips without a display.
func TestInWindowSyntheticLASRenders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cube.las")
	if err := os.WriteFile(path, synthLASCube(), 0o644); err != nil {
		t.Fatalf("write synthetic LAS: %v", err)
	}
	win := newViewportWindow(t)
	defer win.Destroy()
	dockLaidOut = false
	icons = nil

	s := framedSession()
	pc, _, err := s.AttachPointCloud("Cube LAS", path)
	if err != nil {
		t.Fatalf("attach synthetic LAS: %v", err)
	}
	t.Logf("loaded %d LAS points", pc.TotalPointCount())

	frameCameraOn(s, pc.RangeBox())
	for i := 0; i < 10; i++ {
		win.BeginFrame()
		DrawChrome(win, s)
		win.EndFrame(0.1, 0.1, 0.12)
	}
	if err := win.SaveWindowPNG(filepath.Join(outDir(), "point-cloud-las.png")); err != nil {
		t.Logf("SaveWindowPNG: %v", err)
	}
}
