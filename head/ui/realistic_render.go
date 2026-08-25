//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"errors"
	"fmt"
	"hash/fnv"
	stdmath "math"
	"math/rand"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/shading/openpbr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/persistence/userprefs"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// realisticSession is the narrow view of *app.Session the Realistic-mode path tracer
// needs (audit I5, the arrowSession/cloudBounder pattern — head/ui's session-coupling
// ratchet, viewport_instancing.go's cloudBounder) — *app.Session satisfies it
// implicitly, so this file's functions never take the whole session.
type realisticSession interface {
	VisibleInstances() []app.InstanceGroup
	SurfaceLookup() renderer.SurfaceLookup
	SceneLighting() renderer.SceneLighting
	ViewCubePrefs() userprefs.Prefs
	SetViewCubePrefs(userprefs.Prefs)
}

// Realistic-mode live path tracer (M45-F05 PBI-350, ADR-0053): replaces mesh.frag's
// raster pbr() pass for renderer.Realistic with the F04 path-tracing pipeline
// (PBI-345..349), rendering a whole image per call and accumulating samples across
// frames while idle. Material fidelity is intentionally a single scene-representative
// value (the first visible body's base color/roughness/metalness), not full per-body
// OpenPBR binding — that belongs to the appearance-editor/catalog PBIs (PBI-351/352),
// which is where materials become widely author-able; this PBI's job is getting the
// rendering PIPELINE correctly wired end to end. Likewise, geometry is baked to
// world-space triangles per instance (RTScene.AddMesh has no separate per-instance TLAS
// transform yet — its own doc comment: "identity-transform instances, in add-order"),
// so repeated components each get their own BLAS rather than sharing one — a documented
// performance follow-up, not a correctness gap.

// realisticCacheKey scopes persistent GPU state to one document's one viewport tile —
// a split-view layout renders multiple tiles from the same *app.Session, each needing
// its own accumulation buffer and texture.
type realisticCacheKey struct {
	s    realisticSession
	slot int
}

// realisticState is the persistent per-tile GPU state a progressive render needs across
// frames: the built RT/SW scenes, the accumulation buffer, and the presentation
// texture. Keyed globally by realisticCacheKey. Unlike viewport_instancing.go's
// sourceMeshCache (CPU-side data only, cheap to leak), this holds real Vulkan objects
// (descriptor pools, pipelines, command pools) — DestroyRealisticState below must be
// called when a session's window goes away (a live app hooks this to document/window
// close; tests must defer it explicitly — an early attempt at this file skipped that
// and leaked GPU resources across the whole head/ui test binary, compounding into
// hangs deep in a full-suite run once enough leaked resources piled up).
type realisticState struct {
	rt       *native.RTScene
	sw       *native.SWScene
	sceneKey uint64
	accum    *renderer.Accumulator
	tex      uint64
	texW     int
	texH     int
	rng      *rand.Rand
}

var realisticCache = map[realisticCacheKey]*realisticState{}

// DestroyRealisticState frees every GPU resource s's Realistic-mode render state holds
// on win (every tile/slot) — call when s's window/document closes, before win itself is
// destroyed (RTScene/SWScene/the presentation texture all reference win's device). A
// no-op if s never rendered in Realistic mode.
func DestroyRealisticState(win *native.Window, s realisticSession) {
	for key, st := range realisticCache {
		if key.s != s {
			continue
		}
		if st.rt != nil {
			st.rt.Destroy()
		}
		if st.sw != nil {
			st.sw.Destroy()
		}
		if st.tex != 0 {
			win.DestroyTexture(st.tex)
		}
		delete(realisticCache, key)
	}
}

// realisticMesh is one world-space triangle mesh ready for RTScene.AddMesh /
// renderer.BuildBVH+SWScene.Build.
type realisticMesh struct {
	verts      []float32
	indices    []uint32
	instanceID uint32
}

// renderRealisticViewportImage renders one Realistic-mode progressive-accumulation
// sample and returns the presentation texture (0, false on failure — the caller falls
// back to the raster pass so the panel is never left blank). Call every frame; the
// accumulation buffer only resets when the camera, scene, or material actually changes
// (renderer.Accumulator.SyncState).
func renderRealisticViewportImage(win *native.Window, s realisticSession, slot int, cam scene.Camera, pw, ph int) (tex uint64, sampleCount int, ok bool) {
	if pw <= 0 || ph <= 0 {
		return 0, 0, false
	}
	st := realisticStateFor(s, slot, pw, ph)

	meshes, material, haveGeometry := realisticGeometry(s, cam)
	if !haveGeometry {
		return 0, 0, false
	}

	state := renderer.AccumulationState{Camera: cameraHash(cam), Scene: realisticSceneHash(meshes), Material: materialHash(material)}
	if reset := st.accum.SyncState(state); reset || st.sceneKey != state.Scene {
		if !rebuildRealisticScene(win, st, meshes) {
			return 0, 0, false
		}
		st.sceneKey = state.Scene
	}

	lighting := s.SceneLighting()
	pixels, err := traceRealisticFrame(st, win, s, cam, pw, ph, lighting, material)
	if err != nil {
		return 0, 0, false
	}
	st.accum.AddFrame(pixels)

	presentRealisticFrame(win, st, pw, ph, float64(lighting.Exposure))
	return st.tex, st.accum.SampleCount(), st.tex != 0
}

// realisticStateFor returns the persistent GPU state for (s,slot), creating it (and/or
// resizing its accumulator to pw×ph) on first use or on a viewport resize.
func realisticStateFor(s realisticSession, slot, pw, ph int) *realisticState {
	key := realisticCacheKey{s: s, slot: slot}
	st := realisticCache[key]
	if st == nil {
		st = &realisticState{rng: rand.New(rand.NewSource(1))}
		realisticCache[key] = st
	}
	if st.accum == nil || st.accum.Width() != pw || st.accum.Height() != ph {
		st.accum = renderer.NewAccumulator(pw, ph)
	}
	return st
}

// presentRealisticFrame tone-maps st's accumulated image to display RGBA8 and uploads
// it to the presentation texture — created once per resolution, updated in place after.
func presentRealisticFrame(win *native.Window, st *realisticState, pw, ph int, exposure float64) {
	rgba := resolveDisplayRGBA(st.accum, exposure)
	if st.tex == 0 || st.texW != pw || st.texH != ph {
		if st.tex != 0 {
			win.DestroyTexture(st.tex)
		}
		st.tex = win.CreateTexture(rgba, pw, ph)
		st.texW, st.texH = pw, ph
		return
	}
	win.UpdateTexture(st.tex, rgba, pw, ph)
}

// traceRealisticFrame dispatches one sample through whichever backend the hardware-RT
// checkbox (persisted, PBI-332/333) resolves to for this device.
func traceRealisticFrame(st *realisticState, win *native.Window, s realisticSession, cam scene.Camera, pw, ph int, lighting renderer.SceneLighting, material renderer.DrawItem) ([]float32, error) {
	basis := nativeCameraBasis(cam)
	params := pickLightParams(lighting, st.rng, material)
	if realisticHardwareEnabled(win, s) && st.rt != nil {
		return st.rt.TraceRealisticImage(pw, ph, basis, params)
	}
	if st.sw != nil {
		return st.sw.TraceRealisticPathtraceImage(pw, ph, basis, params)
	}
	return nil, errNoRealisticBackend
}

var errNoRealisticBackend = errors.New("ui: no ray-tracing backend available (neither hardware nor software scene built)")

// rebuildRealisticScene discards any previously built RT/SW scenes and builds fresh
// ones from meshes — called only when the scene's content hash changes, not every
// frame. Succeeds if AT LEAST ONE backend built (hardware RT is optional; software is
// expected to always work on any compute-capable device).
func rebuildRealisticScene(win *native.Window, st *realisticState, meshes []realisticMesh) bool {
	if st.rt != nil {
		st.rt.Destroy()
		st.rt = nil
	}
	if st.sw != nil {
		st.sw.Destroy()
		st.sw = nil
	}

	if rt, err := win.NewRTScene(); err == nil {
		if buildRTScene(rt, meshes) {
			st.rt = rt
		} else {
			rt.Destroy()
		}
	}

	if sw, err := win.NewSWScene(); err == nil {
		if buildSWScene(sw, meshes) {
			st.sw = sw
		} else {
			sw.Destroy()
		}
	}

	return st.rt != nil || st.sw != nil
}

func buildRTScene(rt *native.RTScene, meshes []realisticMesh) bool {
	for _, m := range meshes {
		if rt.AddMesh(m.verts, m.indices, m.instanceID) != nil {
			return false
		}
	}
	if rt.Build() != nil {
		return false
	}
	rgen, miss, shadowMiss, chit := native.RealisticPipelineShaders()
	return rt.BuildRealisticPipeline(rgen, miss, shadowMiss, chit) == nil
}

func buildSWScene(sw *native.SWScene, meshes []realisticMesh) bool {
	var triangles []renderer.Triangle
	for _, m := range meshes {
		for i := 0; i+2 < len(m.indices); i += 3 {
			v := func(j int) [3]float32 {
				k := m.indices[i+j] * 3
				return [3]float32{m.verts[k], m.verts[k+1], m.verts[k+2]}
			}
			triangles = append(triangles, renderer.Triangle{
				V0: v(0), V1: v(1), V2: v(2), InstanceID: m.instanceID, PrimitiveID: uint32(i / 3),
			})
		}
	}
	if len(triangles) == 0 {
		return false
	}
	bvh := renderer.BuildBVH(triangles)
	if sw.Build(native.SWBuildInputFrom(bvh, triangles)) != nil {
		return false
	}
	return sw.BuildRealisticPathtracePipeline(native.RealisticPathtraceShader()) == nil
}

// realisticGeometry extracts world-space triangles for every visible instance (local
// per-component tessellation, transform baked in per placement — see this file's doc
// comment on why not real TLAS instancing yet), plus one representative DrawItem for
// its material (base color/roughness/metalness) — the first non-empty triangle item
// encountered, deterministic for a given scene.
func realisticGeometry(s realisticSession, cam scene.Camera) (meshes []realisticMesh, material renderer.DrawItem, ok bool) {
	groups := s.VisibleInstances()
	lookup := s.SurfaceLookup()
	var nextID uint32 = 1
	for _, g := range groups {
		local := renderer.BuildDrawListStyled([]*topo.Body{g.Source}, cam, ops.DefaultQuality(), lookup, renderer.Realistic)
		for _, item := range local.Items {
			if item.Primitive != renderer.Triangles || len(item.Positions) == 0 {
				continue
			}
			if !ok {
				material, ok = item, true
			}
			for _, t := range g.Transforms {
				meshes = append(meshes, bakeInstanceMesh(item, t, nextID))
				nextID++
			}
		}
	}
	return meshes, material, ok
}

func bakeInstanceMesh(item renderer.DrawItem, t math.Matrix4, instanceID uint32) realisticMesh {
	verts := make([]float32, 0, len(item.Positions)*3)
	for _, p := range item.Positions {
		wp := t.TransformPoint(p)
		verts = append(verts, float32(wp.X), float32(wp.Y), float32(wp.Z))
	}
	indices := make([]uint32, len(item.Indices))
	for i, idx := range item.Indices {
		indices[i] = uint32(idx)
	}
	return realisticMesh{verts: verts, indices: indices, instanceID: instanceID}
}

func realisticSceneHash(meshes []realisticMesh) uint64 {
	h := fnv.New64a()
	for _, m := range meshes {
		hashFloat32s(h, m.verts)
	}
	return h.Sum64()
}

func cameraHash(cam scene.Camera) uint64 {
	h := fnv.New64a()
	hashFloat32s(h, []float32{
		float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z),
		float32(cam.Target.X), float32(cam.Target.Y), float32(cam.Target.Z),
		float32(cam.Up.X), float32(cam.Up.Y), float32(cam.Up.Z),
		float32(cam.FOV),
	})
	return h.Sum64()
}

func materialHash(item renderer.DrawItem) uint64 {
	h := fnv.New64a()
	hashFloat32s(h, []float32{item.Color[0], item.Color[1], item.Color[2], item.Color[3], item.Metallic, item.Roughness})
	return h.Sum64()
}

// nativeCameraBasis converts the app's scene.Camera into the pinhole eye/forward/right/
// up basis pathtrace_realistic.rgen/swpathtrace_realistic.comp expect — the same
// construction as scene.Camera.RayThrough, generalized off a single pixel to the whole
// image (the GPU shader computes each pixel's NDC offset itself).
func nativeCameraBasis(cam scene.Camera) native.CameraBasis {
	forward := cam.Forward()
	right := forward.Cross(cam.Up)
	if l := right.Length(); l > 1e-9 {
		right = right.Scale(1 / l)
	}
	up := right.Cross(forward)
	aspect := 1.0
	if cam.Height > 0 {
		aspect = float64(cam.Width) / float64(cam.Height)
	}
	return native.CameraBasis{
		Eye:         [3]float32{float32(cam.Eye.X), float32(cam.Eye.Y), float32(cam.Eye.Z)},
		TMin:        0,
		Forward:     [3]float32{float32(forward.X), float32(forward.Y), float32(forward.Z)},
		TMax:        1e6,
		Right:       [3]float32{float32(right.X), float32(right.Y), float32(right.Z)},
		TanHalfFovY: float32(stdmath.Tan(cam.FOV / 2)),
		Up:          [3]float32{float32(up.X), float32(up.Y), float32(up.Z)},
		Aspect:      float32(aspect),
	}
}

// pickLightParams picks one light from lighting's active rig, power-weighted
// (renderer.LightDistribution, PBI-348), and scales its intensity by 1/PDF so
// accumulating many dispatches — each possibly picking a different light — converges
// to the correct multi-light-weighted image (the same unbiased-estimator property
// PBI-348's CPU test proved analytically, applied here per-dispatch instead of
// per-sample). An unlit scene (no active lights) returns a zero-intensity light, which
// shades to black rather than crashing.
func pickLightParams(lighting renderer.SceneLighting, rng *rand.Rand, material renderer.DrawItem) native.RealisticLightParams {
	base := linearBaseColor(material.Color)
	lights := lighting.ActiveLights()
	if len(lights) == 0 {
		return native.RealisticLightParams{BaseColor: base, BaseWeight: 1, SpecularIOR: 1.5}
	}
	dist := renderer.NewLightDistribution(lights)
	// Monte Carlo light-importance sampling, not a security context: math/rand is the
	// correct, intentional choice here (fast, and seedable via rand.NewSource for
	// deterministic backend-parity tests — crypto/rand offers neither). See
	// realisticState's own rng field/seeding comment.
	_, light, pdf := dist.Sample(rng.Float64()) // NOSONAR
	if pdf <= 0 {
		return native.RealisticLightParams{}
	}
	scale := float32(1 / pdf)
	return native.RealisticLightParams{
		LightDirection:    light.Direction,
		LightIntensity:    light.Intensity * scale,
		LightColor:        light.Color,
		BaseColor:         base,
		BaseWeight:        1,
		SpecularRoughness: material.Roughness,
		SpecularIOR:       1.5,
		BaseMetalness:     material.Metallic,
	}
}

// linearBaseColor gamma-decodes a renderer.DrawItem's sRGB-encoded Color the same way
// mesh.frag's toLinear(c) = pow(c, 2.2) does for the raster pipeline — base_lobes.glsl's
// BRDF math (pathtrace_realistic.rchit/swpathtrace_realistic.comp) expects a LINEAR
// reflectance input and applies no decode of its own (#2150), unlike mesh.frag which
// decodes in-shader.
func linearBaseColor(c [4]float32) [3]float32 {
	dec := func(v float32) float32 {
		if v < 0 {
			return 0
		}
		return float32(stdmath.Pow(float64(v), 2.2))
	}
	return [3]float32{dec(c[0]), dec(c[1]), dec(c[2])}
}

// realisticHardwareEnabled resolves the persisted hardware-RT checkbox override
// (PBI-332) against this device's actual capability (PBI-333/ADR-0053's "unchecking it
// only ever costs convergence time, never correctness" rule).
func realisticHardwareEnabled(win *native.Window, s realisticSession) bool {
	accel, pipeline, query := win.RayTracingExtensionSupport()
	supported := renderer.SupportsHardwareRayTracing(renderer.RTDeviceFeatures{
		AccelerationStructure: accel, RayTracingPipeline: pipeline, RayQuery: query,
	})
	return renderer.ResolveHardwareRayTracingEnabled(s.ViewCubePrefs().HardwareRayTracing, supported)
}

// resolveDisplayRGBA converts the accumulator's averaged linear ACEScg radiance to a
// display-ready RGBA8 image (kernel/shading/openpbr.ToDisplay, PBI-349), matching the
// raster pipeline's own exposure/tone-map/gamma chain exactly.
func resolveDisplayRGBA(accum *renderer.Accumulator, exposure float64) []byte {
	w, h := accum.Width(), accum.Height()
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b := accum.At(x, y)
			disp := openpbr.ToDisplay(openpbr.NewColor3(float64(r), float64(g), float64(b)), exposure)
			i := (y*w + x) * 4
			out[i] = toByte(float32(disp.R))
			out[i+1] = toByte(float32(disp.G))
			out[i+2] = toByte(float32(disp.B))
			out[i+3] = 255
		}
	}
	return out
}

// drawRealisticConvergenceIndicator overlays the accumulated sample count in the
// viewport panel's top-left corner — the "converging..." feedback PBI-350's scope
// calls for, in the same on-image-text style as the existing HUD overlays
// (chrome_viewport.go's drawViewportOverlays).
func drawRealisticConvergenceIndicator(samples int, cx, cy float32) {
	native.DrawText(cx+6, cy+4, fmt.Sprintf("Realistic — %d samples", samples), [4]float32{1, 1, 1, 0.85})
}
