//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package native

import (
	"fmt"
	"image"
	"image/color"
	stdmath "math"

	"oblikovati.org/kernel/shading/openpbr"
	"oblikovati.org/renderer"
)

// UVSphereTriangles tessellates a UV sphere of the given radius, longitude/latitude
// segment counts, tagged with instanceID — shared by head/cmd/openpbrgoldenshot (which
// writes standalone reference PNGs) and this package's own golden tests (M45-F05
// PBI-353), so the two never drift into rendering slightly different geometry.
func UVSphereTriangles(radius float32, segments, rings int, instanceID uint32) []renderer.Triangle {
	var tris []renderer.Triangle
	var id uint32
	for ring := 0; ring < rings; ring++ {
		tris, id = appendSphereRing(tris, id, radius, ring, segments, rings, instanceID)
	}
	return tris
}

// appendSphereRing emits the two-triangles-per-quad strip for one latitude ring of
// UVSphereTriangles, split out to keep UVSphereTriangles itself within CLAUDE.md's
// 20-statement function limit.
func appendSphereRing(tris []renderer.Triangle, id uint32, radius float32, ring, segments, rings int, instanceID uint32) ([]renderer.Triangle, uint32) {
	pos := func(u, v float32) [3]float32 {
		theta := v * stdmath.Pi
		phi := u * 2 * stdmath.Pi
		st, ct := stdmath.Sincos(float64(theta))
		sp, cp := stdmath.Sincos(float64(phi))
		return [3]float32{radius * float32(st*cp), radius * float32(st*sp), radius * float32(ct)}
	}
	v0 := float32(ring) / float32(rings)
	v1 := float32(ring+1) / float32(rings)
	for seg := 0; seg < segments; seg++ {
		u0 := float32(seg) / float32(segments)
		u1 := float32(seg+1) / float32(segments)
		p00, p01 := pos(u0, v0), pos(u0, v1)
		p10, p11 := pos(u1, v0), pos(u1, v1)
		if ring > 0 {
			tris = append(tris, renderer.Triangle{V0: p00, V1: p11, V2: p10, InstanceID: instanceID, PrimitiveID: id})
			id++
		}
		if ring < rings-1 {
			tris = append(tris, renderer.Triangle{V0: p00, V1: p01, V2: p11, InstanceID: instanceID, PrimitiveID: id})
			id++
		}
	}
	return tris, id
}

// PinholeCameraBasis builds the eye/forward/right/up basis both ray-tracing backends'
// shaders expect for a camera at eye looking at the origin, world-up = +Z — the
// convention test-utilities/openpbr-goldens/oracle/render_reference.py's Blender camera
// setup mirrors, so a golden test's Go-side render and its Blender reference share
// identical framing.
func PinholeCameraBasis(eye [3]float32, tanHalfFovY float32, width, height int) CameraBasis {
	forward := normalizeVec3(negVec3(eye))
	worldUp := [3]float32{0, 0, 1}
	right := normalizeVec3(crossVec3(forward, worldUp))
	up := crossVec3(right, forward)
	aspect := float32(1)
	if height > 0 {
		aspect = float32(width) / float32(height)
	}
	return CameraBasis{
		Eye: eye, TMin: 0, Forward: forward, TMax: 1e6,
		Right: right, TanHalfFovY: tanHalfFovY, Up: up, Aspect: aspect,
	}
}

// ToneMappedImage runs a width*height*3 linear-radiance buffer (an RTScene/SWScene
// TraceRealisticImage result) through kernel/shading/openpbr.ToDisplay — the exact
// pipeline PBI-349 built for Realistic-mode display — and returns an *image.RGBA ready
// to compare against a golden PNG or write to disk.
func ToneMappedImage(pixels []float32, width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			r, g, b := pixels[i], pixels[i+1], pixels[i+2]
			disp := openpbr.ToDisplay(openpbr.NewColor3(float64(r), float64(g), float64(b)), 1.0)
			img.Set(x, y, color.RGBA{clampToByte(disp.R), clampToByte(disp.G), clampToByte(disp.B), 255})
		}
	}
	return img
}

// RenderGoldenSphere dispatches one full-image render of triangles under a directional
// light through whichever ray-tracing backend is available/requested — hardware RT
// first unless forceSoftware, falling back to the software backend. Shared by
// head/cmd/openpbrgoldenshot (standalone reference-PNG generation) and this package's
// golden tests (M45-F05 PBI-353) so both dispatch the SAME two backends the exact same
// way. A single dispatch is already the converged result: the realistic pipeline
// shaders shade a one-light scene deterministically, with no per-sample randomness.
func RenderGoldenSphere(win *Window, triangles []renderer.Triangle, cam CameraBasis, light RealisticLightParams, width, height int, forceSoftware bool) ([]float32, error) {
	if !forceSoftware {
		if pixels, ok := renderGoldenHardware(win, triangles, cam, light, width, height); ok {
			return pixels, nil
		}
	}
	return renderGoldenSoftware(win, triangles, cam, light, width, height)
}

func renderGoldenHardware(win *Window, triangles []renderer.Triangle, cam CameraBasis, light RealisticLightParams, width, height int) ([]float32, bool) {
	rt, err := win.NewRTScene()
	if err != nil {
		return nil, false
	}
	defer rt.Destroy()
	if !buildGoldenRTMesh(rt, triangles) {
		return nil, false
	}
	rgen, miss, shadowMiss, chit := RealisticPipelineShaders()
	if rt.BuildRealisticPipeline(rgen, miss, shadowMiss, chit) != nil {
		return nil, false
	}
	pixels, err := rt.TraceRealisticImage(width, height, cam, light)
	if err != nil {
		return nil, false
	}
	return pixels, true
}

func buildGoldenRTMesh(rt *RTScene, triangles []renderer.Triangle) bool {
	verts := make([]float32, 0, len(triangles)*9)
	indices := make([]uint32, 0, len(triangles)*3)
	for i, t := range triangles {
		verts = append(verts, t.V0[0], t.V0[1], t.V0[2], t.V1[0], t.V1[1], t.V1[2], t.V2[0], t.V2[1], t.V2[2])
		indices = append(indices, uint32(i*3), uint32(i*3+1), uint32(i*3+2))
	}
	if rt.AddMesh(verts, indices, 1) != nil {
		return false
	}
	return rt.Build() == nil
}

func renderGoldenSoftware(win *Window, triangles []renderer.Triangle, cam CameraBasis, light RealisticLightParams, width, height int) ([]float32, error) {
	sw, err := win.NewSWScene()
	if err != nil {
		return nil, fmt.Errorf("no ray-tracing backend available (hardware or software): %w", err)
	}
	defer sw.Destroy()
	bvh := renderer.BuildBVH(triangles)
	if err := sw.Build(SWBuildInputFrom(bvh, triangles)); err != nil {
		return nil, fmt.Errorf("SWScene.Build: %w", err)
	}
	if err := sw.BuildRealisticPathtracePipeline(RealisticPathtraceShader()); err != nil {
		return nil, fmt.Errorf("SWScene.BuildRealisticPathtracePipeline: %w", err)
	}
	return sw.TraceRealisticPathtraceImage(width, height, cam, light)
}

func clampToByte(v float64) uint8 {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	return uint8(v*255 + 0.5)
}

func normalizeVec3(v [3]float32) [3]float32 {
	n := float32(stdmath.Sqrt(float64(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])))
	if n == 0 {
		return v
	}
	return [3]float32{v[0] / n, v[1] / n, v[2] / n}
}

func negVec3(v [3]float32) [3]float32 { return [3]float32{-v[0], -v[1], -v[2]} }

func crossVec3(a, b [3]float32) [3]float32 {
	return [3]float32{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
