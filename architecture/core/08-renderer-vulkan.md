# Core 08 — Vulkan 1.3 renderer & viewport scene graph

*New subsystem (Inventor owned its own renderer). Implements ADR-0005, ADR-0008,
**ADR-0014 (testability)**. Applies realtime-3d skill §3 (scene graph), §4 (backend
abstraction), §5 (caches), §6 (draw-call-as-data).*

> **The renderer is the hardest part of this codebase to test and the hardest for an
> LLM to debug** (bugs are visual, not exceptions; the model can't see the output).
> Testability is therefore a *first-class structural requirement* here, not an
> afterthought — see the [Testability](#testability-adr-0014) section, which is as
> important as the rendering design itself, and [testing/00](../testing/00-renderer-oracle-pipeline.md).

## Layered backend with three implementations (realtime-3d §4, ADR-0014)

```
renderer/
  Renderer            # high-level: register drawings, run passes, manage caches
  backend/
    Backend (iface)   # Device/SwapChain/Painter surface — the only seam GPU code crosses
    *_vulkan.go       # 1) windowed Vulkan — production
    offscreen_vulkan.go# 2) headless Vulkan → image buffer; runs on SOFTWARE Vulkan
                      #    (Mesa lavapipe/llvmpipe) in CI — deterministic, no GPU (ADR-0014)
    null.go           # 3) records the command stream, renders nothing — CPU unit asserts
```

App/UI code depends on the `Renderer` interface and never sees a descriptor set or
image handle (realtime-3d §4). The `platform.Window` surface comes from the cgo/
purego edge (ADR-0008). **The three backends are the backbone of renderer testing**
(ADR-0014): `null` for unit tests on the draw stream, `offscreen` for deterministic
image capture in CI, `vulkan` for production.

### Vulkan 1.3 usage (ADR-0005)

- **Dynamic rendering** — viewport, ID/picking buffer, and ImGui overlay are
  inline `begin/endRendering` scopes; no render-pass/framebuffer objects.
- **Bindless** — one big descriptor array of textures/material buffers indexed by
  integer; per-draw data carries indices (fits draw-call-as-data below).
- **Timeline semaphores + sync2** — order the async tessellation upload (ADR-0007)
  against the frame cleanly.
- **Feature-gated on macOS/MoltenVK** — query `VkPhysicalDeviceVulkan13Features` at
  startup, log the resolved level, branch via build/runtime fallbacks (ADR-0005).

## Viewport scene graph (realtime-3d §3)

The viewport (distinct from the *model*) is an **entity + transform hierarchy with
dirty flags**. CAD entities: a body's tessellation, edges, work features,
manipulators, ground plane, occurrence instances.

```go
package scene
type Entity struct { Transform; Parent *Entity; Children []*Entity; Data any }
type Transform struct { local, world math.Mat4; pos, rot, scl; dirty bool }
```

- A setter flags dirty and **cascades to children**; matrices are **not** recomputed
  immediately.
- Once per frame (phase 6, core/00), the **worker pool recomputes all dirty world
  matrices in parallel** — independent subtrees concurrently. This is the same
  dirty-DAG discipline as parameters (core/04) and feature history.
- Assembly instancing maps perfectly: one tessellated `Body` mesh (shared, like the
  shared `ComponentDefinition`), many scene entities with per-occurrence transforms
  (the flyweight, parametric-cad §5) — GPU instancing draws N occurrences of one
  mesh.

## Resource caches + instancing (realtime-3d §5)

Meshes (from kernel tessellation), pipelines, textures, and materials go through
**caches keyed by stable id**; loading is idempotent.

```go
mesh := r.Meshes.Get(bodyRefKey)     // tessellation cache; re-tessellate ⇒ swap in place
pipe := r.Pipelines.Get("surface")   // hot-reloadable in build.Debug
mat  := r.Materials.Get("steel").Instance(...)  // share pipeline, vary bindings
```

When async recompute finishes (ADR-0007), the tessellation cache entry for a changed
body is swapped at a frame boundary — every scene entity referencing it updates with
zero per-entity sync (the cache is the single swap point).

## Draw-call-as-data + render queue (realtime-3d §6)

A renderable is **plain data** handed to a queue — not an object with a `Draw()`:

```go
type Drawing struct {
    Material  MaterialInstance   // pipeline + bindless indices
    Mesh      MeshHandle         // tessellated body / edges
    Instance  ShaderData         // per-instance uniforms mirroring the shader struct
    Transform *scene.Transform   // ← POINTER to the live entity transform
    Culler    CameraID
}
r.Add(drawing)
```

The **transform is referenced, not copied** — moving an occurrence (its transform)
automatically moves what's drawn, no per-frame sync (the key wiring, §6). Phase 7
culls per camera and batches by pipeline/material to minimize state changes.

> **The GPU line (ADR-0014).** Everything up to and including building the batched
> draw list — culling, sorting, batching, `ShaderData` packing, transform resolution
> — is a **pure function of data** and is unit-tested on the CPU with the `null`
> backend, *no device*. `Build(queue, camera) → []Batch` is deterministic and
> assertable. Only the final "record these batches into a command buffer + submit" step
> needs a GPU. Keep this line bright: the more logic sits above it, the more of the
> renderer is testable as ordinary Go.

## CAD-specific passes — each independently capturable (ADR-0014)

Every pass writes a named target that the `offscreen` backend can **capture as an
AOV image in isolation**, so a failure localizes to one pass instead of "the picture
is wrong" (ADR-0014):

- **Depth pass** — analytic oracle (expected depth of known geometry is computable).
- **Normal pass** — world/view normals; CPU-reference + analytic comparable.
- **Surface (lit) pass** — shaded triangles (per-face appearance, M16); CPU-reference
  oracle (software shading) + Blender for full PBR.
- **Edge pass** — model edges as anti-aliased lines; silhouette comparable analytically.
- **ID/picking pass** — entity IDs to an offscreen target; hit-testing reads the pixel
  under the cursor (replaces COM `HitTestManager`). **Exactly oracle-able** — expected
  IDs per region are known, so picking is bit-comparable (no tolerance).
- **Overlay (client graphics)** — previews/manipulators + add-in `ClientGfx` (core/07).
- **ImGui pass** — the shell (core/09).

Capturable AOVs are the backbone of the oracle hierarchy: geometric passes (depth/
normal/ID/edge) compare **exactly** against analytic oracles; the lit pass compares
against the CPU reference (tight) and Blender (perceptual). See
[testing/00](../testing/00-renderer-oracle-pipeline.md).

## Determinism mode (ADR-0014)

A build/runtime flag puts the renderer in a **reproducible** state for tests: fixed
viewport, fixed camera, **no temporal jitter/TAA**, fixed RNG seeds, no frame-time
dependence, software Vulkan (llvmpipe). Output is stable enough to diff against
goldens within tolerance. Production rendering leaves it off.

## Camera

Primary perspective/ortho 3D camera + a screen-space pass for UI (realtime-3d §8's
two-camera idea). Orbit/pan/zoom, named views, view cube. Camera changes are events
(core/06) the renderer consumes.

## Headless & thumbnails

The `offscreen` backend renders without a window — reused by CI image-diff tests and
by drawing thumbnails / view generation (M14). Because the domain doesn't depend on
the renderer (core/01), most tests need no GPU at all.

## Testability (ADR-0014)

This section is load-bearing: the renderer is the project's hardest test surface and
the hardest for an LLM to debug, so its structure is shaped to be testable. The full
treatment is [testing/00](../testing/00-renderer-oracle-pipeline.md); the structural
commitments the *renderer design* must honor are:

1. **Three backends** (above) — `null` (unit asserts on the draw stream), `offscreen`
   (deterministic image capture on software Vulkan in CI), `vulkan` (production).
2. **The GPU line** (above) — all pre-submission logic is pure & CPU-tested; the GPU
   layer is thin.
3. **Per-pass AOV capture** (above) — each pass diffable in isolation.
4. **A CPU reference shading path** — a small pure-Go software implementation of the
   shading math (reusing `math/`) is the oracle for the lit pass: GPU output must
   match it within tolerance, with **no external dependency** (the LLM inner loop).
5. **Determinism mode** (above) for reproducible goldens.
6. **Validation layers as a hard test gate** — all renderer tests run with Vulkan
   validation on; any validation message fails the test. This catches the API-misuse
   class of bugs LLMs commonly introduce, at the source rather than as silent corruption.
7. **Structured numeric diff reports** — comparisons emit per-pass scores, the failing
   tier, and the error-region bounding box (not just an image), so Claude gets a
   *localized numeric signal* it can act on instead of a screenshot it cannot see.

The oracle hierarchy that consumes these (analytic → CPU-reference → metamorphic →
Blender) lives in [testing/00](../testing/00-renderer-oracle-pipeline.md).
