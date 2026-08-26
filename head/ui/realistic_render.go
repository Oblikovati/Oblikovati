//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"encoding/binary"
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
	Environment() app.EnvironmentState
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
	rt             *native.RTScene
	sw             *native.SWScene
	sceneKey       uint64
	sceneBuilt     bool // guards the FIRST rebuild independent of sceneKey's zero value — see renderRealisticViewportImage
	lastCameraHash uint64
	haveLastCamera bool // guards the FIRST frame, before there's a previous camera to compare against
	accum          *renderer.Accumulator
	tex            uint64
	texW           int
	texH           int
	rng            *rand.Rand
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

// realisticInteractiveDownscale is the linear resolution divisor Realistic mode traces at
// while the camera is actively moving, rather than the full panel resolution (#2155
// live-testing finding): a full-panel dispatch measured ~330-400ms on this project's own
// dev hardware for a TRIVIAL two-mesh part, scaling with pixel count — and because the
// trace runs synchronously in the main render loop (traceRealisticFrame), that single-
// handedly blocked the UI thread for that long on EVERY orbit frame. Tracing at 1/3 linear
// resolution (1/9 the pixels) while the camera is in motion keeps a visibly-present,
// interactive preview (rather than skipping the frame) while cutting worst-case blocking
// roughly 9x; realisticGeometry's own camera-hash reset makes this free to switch back —
// the accumulator naturally jumps to full pw×ph and restarts the moment the camera stops
// (ensureRealisticAccumSize + Accumulator.SyncState below), so idle convergence quality is
// completely unaffected. The Vulkan-level fix for the remaining worst case — a dispatch
// that never returns at all (a lost/faulted device) — is rtCommandTimeoutNs/
// swCommandTimeoutNs (raytrace.cpp/swtrace.cpp): traceRealisticFrame already treats that
// as an ordinary failed frame (err != nil below), not a hang.
const realisticInteractiveDownscale = 3

// realisticTraceResolution returns the resolution to actually dispatch a trace at: the
// full panel size when idle (cameraMoving false), or a reduced interactive preview size
// otherwise — see realisticInteractiveDownscale's own doc comment.
func realisticTraceResolution(pw, ph int, cameraMoving bool) (tracePW, tracePH int) {
	if !cameraMoving {
		return pw, ph
	}
	tracePW, tracePH = pw/realisticInteractiveDownscale, ph/realisticInteractiveDownscale
	if tracePW < 1 {
		tracePW = 1
	}
	if tracePH < 1 {
		tracePH = 1
	}
	return tracePW, tracePH
}

// renderRealisticViewportImage renders one Realistic-mode progressive-accumulation
// sample and returns the presentation texture (0, false on failure — the caller falls
// back to the raster pass so the panel is never left blank). Call every frame; the
// accumulation buffer resets whenever the camera, scene, or material changes
// (renderer.Accumulator.SyncState), but the built RT/SW scene (BLAS/TLAS/pipeline) is
// rebuilt ONLY when the scene's mesh content actually changes — camera orbiting and
// material edits must never pay that cost (a bug found live during #2155: this used to
// retrigger a full destroy+rebuild of the RT scene on EVERY frame of camera movement).
// The actual trace/accumulate/present resolution drops to a reduced interactive preview
// while the camera is moving — see realisticInteractiveDownscale.
func renderRealisticViewportImage(win *native.Window, s realisticSession, slot int, cam scene.Camera, pw, ph int) (tex uint64, sampleCount int, ok bool) {
	if pw <= 0 || ph <= 0 {
		return 0, 0, false
	}
	st := realisticStateFor(s, slot)

	meshes, material, haveGeometry := realisticGeometry(s, cam)
	if !haveGeometry {
		return 0, 0, false
	}

	env := s.Environment()
	state := realisticAccumulationState(win, cam, meshes, material, env)
	tracePW, tracePH := realisticPrepareFrame(st, state, pw, ph)

	st.accum.SyncState(state) // resets the accumulator on ANY change — a separate concern from rebuilding the RT scene below
	if !ensureRealisticSceneBuilt(win, st, meshes, state.Scene) {
		return 0, 0, false
	}

	shading := realisticShadingInputs{lighting: s.SceneLighting(), material: material, env: env}
	pixels, err := traceRealisticFrame(st, win, s, cam, tracePW, tracePH, shading)
	if err != nil {
		return 0, 0, false
	}
	st.accum.AddFrame(pixels)

	presentRealisticFrame(win, st, tracePW, tracePH, float64(shading.lighting.Exposure))
	return st.tex, st.accum.SampleCount(), st.tex != 0
}

// realisticPrepareFrame records st's camera-moving state from state.Camera and sizes its
// accumulator to this frame's trace resolution (realisticTraceResolution) — split out of
// renderRealisticViewportImage (funlen).
func realisticPrepareFrame(st *realisticState, state renderer.AccumulationState, pw, ph int) (tracePW, tracePH int) {
	cameraMoving := st.haveLastCamera && st.lastCameraHash != state.Camera
	st.lastCameraHash, st.haveLastCamera = state.Camera, true
	tracePW, tracePH = realisticTraceResolution(pw, ph, cameraMoving)
	ensureRealisticAccumSize(st, tracePW, tracePH)
	return tracePW, tracePH
}

// ensureRealisticSceneBuilt (re)builds st's RT/SW scene when it hasn't been built yet or
// sceneHash (state.Scene) no longer matches the last build — mesh content changes trigger
// this the same way an environment-image regeneration does (realisticSceneHash's own doc
// comment on why envGen folds into sceneHash) — split out of renderRealisticViewportImage
// (funlen).
func ensureRealisticSceneBuilt(win *native.Window, st *realisticState, meshes []realisticMesh, sceneHash uint64) bool {
	if st.sceneBuilt && st.sceneKey == sceneHash {
		return true
	}
	if !rebuildRealisticScene(win, st, meshes) {
		return false
	}
	st.sceneKey = sceneHash
	st.sceneBuilt = true
	return true
}

// realisticAccumulationState fingerprints this frame's camera/scene/material/environment
// for renderer.Accumulator.SyncState — split out of renderRealisticViewportImage (funlen)
// after #2155's IBL follow-up added the environment generation/hash computation.
func realisticAccumulationState(win *native.Window, cam scene.Camera, meshes []realisticMesh, material renderer.DrawItem, env app.EnvironmentState) renderer.AccumulationState {
	return renderer.AccumulationState{
		Camera:      cameraHash(cam),
		Scene:       realisticSceneHash(meshes, win.EnvironmentGeneration()),
		Material:    materialHash(material),
		Environment: environmentHash(env),
	}
}

// realisticStateFor returns the persistent GPU state for (s,slot), creating it on first
// use — call ensureRealisticAccumSize separately once the caller has decided this frame's
// trace resolution (renderRealisticViewportImage: full panel size when idle, a reduced
// interactive preview size while the camera is moving).
func realisticStateFor(s realisticSession, slot int) *realisticState {
	key := realisticCacheKey{s: s, slot: slot}
	st := realisticCache[key]
	if st == nil {
		st = &realisticState{rng: rand.New(rand.NewSource(1))}
		realisticCache[key] = st
	}
	return st
}

// ensureRealisticAccumSize (re)allocates st's accumulator when pw×ph differs from its
// current size — split out of realisticStateFor (#2155) so a caller can look up the
// persistent state before deciding what resolution to size it to this frame.
func ensureRealisticAccumSize(st *realisticState, pw, ph int) {
	if st.accum == nil || st.accum.Width() != pw || st.accum.Height() != ph {
		st.accum = renderer.NewAccumulator(pw, ph)
	}
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

// realisticShadingInputs bundles traceRealisticFrame's per-frame scene-shading data
// (lighting rig, representative material, environment state) into one parameter — split
// out to keep the function's own parameter count within this repo's 7-parameter limit
// (SonarCloud go:S107) after #2135/#2155 added env for IBL selection.
type realisticShadingInputs struct {
	lighting renderer.SceneLighting
	material renderer.DrawItem
	env      app.EnvironmentState
}

// traceRealisticFrame dispatches one sample through whichever backend the hardware-RT
// checkbox (persisted, PBI-332/333) resolves to for this device.
func traceRealisticFrame(st *realisticState, win *native.Window, s realisticSession, cam scene.Camera, pw, ph int, in realisticShadingInputs) ([]float32, error) {
	basis := nativeCameraBasis(cam)
	params := pickLightParams(in.lighting, st.rng, in.material, in.env, currentEnvironmentDistribution())
	applyEnvironmentParams(&params, in.env)
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

// realisticSceneHash fingerprints meshes' geometry plus envGen (win.EnvironmentGeneration,
// #2155's IBL follow-up) — envGen changes whenever the environment image is regenerated,
// which stale-invalidates the RT/SW pipelines' env descriptor bindings (env_binding_for's
// own doc comment in raytrace.cpp/swtrace.cpp) and so must trigger the same rebuild path a
// mesh edit does, even though the BLAS/TLAS geometry itself is unaffected.
func realisticSceneHash(meshes []realisticMesh, envGen uint64) uint64 {
	h := fnv.New64a()
	for _, m := range meshes {
		hashFloat32s(h, m.verts)
	}
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], envGen)
	_, _ = h.Write(b[:])
	return h.Sum64()
}

// environmentHash fingerprints the display-affecting parts of env (IBL background
// visibility, rotation, intensity) so a change resets the accumulator (renderer.
// Accumulator.SyncState) — separate from realisticSceneHash's envGen, which only changes
// when the underlying image is actually regenerated, not on every rotation/intensity edit.
func environmentHash(env app.EnvironmentState) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(env.Preset))
	_, _ = h.Write([]byte(env.FilePath))
	hashFloat32s(h, []float32{env.Rotation, env.Intensity})
	if env.ShowImage {
		_, _ = h.Write([]byte{1})
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

// pickLightParams picks EITHER one light from lighting's active rig, power-weighted
// (renderer.LightDistribution, PBI-348), OR the environment itself, importance-sampled
// (renderer.EnvironmentDistribution, #2135/#2155's illumination-contribution follow-up)
// — a single Monte Carlo selection per dispatch between the two sampling strategies,
// weighted by their relative total weight (pEnv below) so accumulating many dispatches,
// each possibly picking a different light OR the environment, converges to the correct
// combined-lighting image (the same unbiased-estimator property PBI-348's CPU test
// proved analytically for discrete lights alone, extended here to a second strategy).
// envDist is nil when no environment is active/loaded (currentEnvironmentDistribution).
// A scene with neither active lights nor an active environment returns a zero-intensity
// light, which shades to black rather than crashing.
func pickLightParams(lighting renderer.SceneLighting, rng *rand.Rand, material renderer.DrawItem, env app.EnvironmentState, envDist *renderer.EnvironmentDistribution) native.RealisticLightParams {
	params := native.RealisticLightParams{
		BaseColor: linearBaseColor(material.Color), BaseWeight: 1,
		SpecularRoughness: material.Roughness, SpecularIOR: 1.5, BaseMetalness: material.Metallic,
	}

	lights := lighting.ActiveLights()
	var dist *renderer.LightDistribution
	var lightsWeight float64
	if len(lights) > 0 {
		dist = renderer.NewLightDistribution(lights)
		lightsWeight = dist.TotalWeight()
	}
	envActive := env.IsActive() && envDist != nil
	envWeight := 0.0
	if envActive {
		envWeight = envDist.TotalWeight() * float64(env.Intensity)
	}
	totalWeight := lightsWeight + envWeight
	if totalWeight <= 0 {
		return params // no discrete lights, no active environment: unlit
	}

	pEnv := envWeight / totalWeight
	if envActive && rng.Float64() < pEnv { // NOSONAR: Monte Carlo selection, not a security context — see pickDiscreteLight's own NOSONAR doc comment
		pickEnvironmentLight(&params, rng, envDist, env.Rotation, pEnv)
	} else {
		pickDiscreteLight(&params, rng, dist, 1-pEnv)
	}
	return params
}

// pickEnvironmentLight importance-samples envDist for a direction (renderer.
// EnvironmentDistribution.Sample), rotates it into world space (renderer.RotateAroundZ,
// matching env_sample.glsl's own rotation convention), and scales by 1/(pEnv*pdf) —
// pEnv accounts for this dispatch having chosen the environment strategy at all, pdf for
// which direction within it. A degenerate (all-black) environment leaves params' light
// fields zero rather than dividing by a zero pdf.
func pickEnvironmentLight(params *native.RealisticLightParams, rng *rand.Rand, envDist *renderer.EnvironmentDistribution, rotation float32, pEnv float64) {
	dir, pdf := envDist.Sample(rng.Float64(), rng.Float64()) // NOSONAR: Monte Carlo direction sampling, not a security context
	if pdf <= 0 {
		return
	}
	params.LightDirection = renderer.RotateAroundZ(dir, rotation)
	params.LightIsEnvironment = 1
	params.LightIntensity = float32(1 / (pEnv * pdf))
}

// pickDiscreteLight mirrors this file's own prior single-strategy pickLightParams body:
// power-weighted light-importance sampling (PBI-348), now scaled by 1/(pLight*pdf) —
// pLight accounts for this dispatch having chosen the discrete-light strategy at all
// (1 when no environment competes for selection).
func pickDiscreteLight(params *native.RealisticLightParams, rng *rand.Rand, dist *renderer.LightDistribution, pLight float64) {
	if dist == nil || pLight <= 0 {
		return
	}
	// Monte Carlo light-importance sampling, not a security context: math/rand is the
	// correct, intentional choice here (fast, and seedable via rand.NewSource for
	// deterministic backend-parity tests — crypto/rand offers neither). See
	// realisticState's own rng field/seeding comment.
	_, light, pdf := dist.Sample(rng.Float64()) // NOSONAR
	if pdf <= 0 {
		return
	}
	scale := float32(1 / (pLight * pdf))
	params.LightDirection = light.Direction
	params.LightColor = light.Color
	params.LightIntensity = light.Intensity * scale
}

// applyEnvironmentParams sets params' EnvRotation/EnvIntensity (needed for BOTH background
// visibility and, as of #2135/#2155's illumination-contribution follow-up, light-sampled
// environment color — pickLightParams's own env branch reads them too) whenever an
// environment is active, and EnvEnabled (background visibility ONLY — the miss shaders'
// own gate) additionally requires env.ShowImage: an environment can light the scene while
// hidden as a backdrop, matching how DCC tools separate "used for lighting" from "shown as
// background" (extended_lobes.glsl's OPENPBR_REALISTIC_PARAMS_FIELDS).
func applyEnvironmentParams(params *native.RealisticLightParams, env app.EnvironmentState) {
	if !env.IsActive() {
		return
	}
	params.EnvRotation = env.Rotation
	params.EnvIntensity = env.Intensity
	if env.ShowImage {
		params.EnvEnabled = 1
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
	for y := range h {
		for x := range w {
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
