# Testing 00 — Renderer oracle pipeline

*The deep treatment of renderer testing. Implements ADR-0014; consumes the
structural commitments in [core/08](../core/08-renderer-vulkan.md) (three backends,
the GPU line, per-pass AOV capture, CPU reference, determinism mode, validation
gate). Written for an LLM author who **cannot see the output** — every technique
here produces a number and a location, not a picture to judge.*

## The problem, stated precisely

Renderer bugs are: a wrong matrix multiply, a flipped winding, a descriptor bound to
the wrong slot, a sync hazard, a shader off-by-one, an sRGB/linear mixup. They
manifest as *images*, often subtly, sometimes only on some drivers. An LLM cannot
look at the framebuffer and say "the specular is too tight." So we never ask it to.
Instead: **render to capturable buffers, compare to an oracle, and report a localized
numeric diff.** The art is choosing the *cheapest oracle that can judge each thing*.

## The oracle hierarchy (cheapest & most exact first)

### Tier 1 — Analytic oracles (exact, no deps, fully explainable)

For anything whose correct output is **computable in closed form**, compute it and
compare exactly (bit-exact or within ULPs). These need no GPU readback semantics
beyond the captured AOV, no external tool, and give the clearest possible signal.

| What | Analytic expectation |
|---|---|
| **Camera projection** | project known 3D points; assert their pixel coordinates |
| **Coverage** | a triangle/quad at known NDC → expected covered pixel mask |
| **Depth pass** | depth of a known plane/sphere per pixel = closed form |
| **Object-ID / picking** | which entity covers each region is known → **bit-exact** ID buffer |
| **Edge/silhouette** | a cube/sphere silhouette is a known polygon/circle |
| **Normals** | a known plane's normal is constant; a sphere's is `normalize(p-c)` |

Picking is the highlight: the ID pass is **exactly oracle-able**, so selection — a
correctness-critical, interaction-driving feature — is tested with zero tolerance and
no Blender. Most *geometric* renderer bugs (transforms, projection, winding, viewport,
clipping) are caught here, deterministically, by the cheapest tier.

### Tier 2 — CPU reference oracle (tight tolerance, no deps)

For the **lit pass**, a small pure-Go software implementation of the *same shading
math* (reusing `math/`, the configurable-scalar lib) renders a reference image on the
CPU. The GPU output must match it within a tight linear-space RMSE. This is a
**self-contained differential test** (GPU vs CPU implementation of one spec) requiring
no external tool — the ideal LLM inner loop:

```
 refImg  := cpuShade(scene, camera)          // pure Go, deterministic, debuggable
gpuImg   := offscreen.RenderPass(scene, camera, PassLit)
report   := compareLinear(gpuImg, refImg, RMSE<0.01)   // localized numeric diff
```

When they diverge, **both sides are inspectable in Go** — the model can add prints to
the CPU path, narrow the failing pixel, and reason about which term (diffuse/specular/
normal/light) is wrong. A GPU-only bug becomes an ordinary value mismatch.

### Tier 3 — Metamorphic oracles (no reference image at all)

The most powerful and most LLM-authorable tests assert **invariances** — properties
that must hold *between* renders, needing no ground truth. They catch huge classes of
bugs cheaply and are trivial for a model to write:

| Metamorphic relation | Catches |
|---|---|
| translate scene by Δ ⇒ image shifts by `project(Δ)` | view/projection/transform errors |
| rotate camera 360° ⇒ identical image | accumulation/precision/state leaks |
| render at 2× resolution ⇒ matches supersampled 1× | sampling/AA/coverage errors |
| draw N instances vs N separate draws ⇒ identical | instancing/per-draw data packing |
| reorder draw submission (opaque) ⇒ identical | hidden ordering deps, sync hazards |
| mirror scene + mirror camera ⇒ mirrored image | winding/culling/handedness |
| empty scene ⇒ exact clear color; one triangle ⇒ exact coverage | clear/blend/state |

Metamorphic + analytic together cover most of the renderer with **zero external
dependencies and zero golden images** — they are the backbone of the inner loop.

### Tier 4 — Blender oracle (high-fidelity ground truth, perceptual tolerance)

For **full PBR on integrated scenes** that are impractical to reference analytically
(area lights, multi-bounce, complex materials), Blender (Cycles/Eevee) is the ground
truth — the user's oracle idea, used where the cheaper tiers can't reach, with
**perceptual** tolerance (exact parity with Blender is unattainable and not the goal).

## The Blender oracle pipeline (Tier 4)

```
   scene.json (NEUTRAL description) ──┬──▶ our offscreen renderer ──▶ ours.exr
   (meshes, transforms, explicit      │
    camera intrinsics/extrinsics,     └──▶ blender -b -P oracle.py ─▶ ref.exr
    lights, PBR materials, color space)        (maps scene.json → Blender, renders AOVs)
                                                          │
                              compareLinear(ours, ref, SSIM/FLIP perceptual) ─▶ report + diff heatmap
```

Making it sound (the parts that matter):

- **One neutral scene description** feeds *both* renderers — never hand-build a
  Blender scene that drifts from ours. (It is the same scene format the app exports;
  glTF + an explicit camera/material extension works well.)
- **Calibrate the comparison or it's noise:** render to **EXR in linear space**,
  Blender filmic/tonemap **off** (raw/standard), AA matched or disabled, **identical
  camera model** (sensor/focal ↔ fov, same near/far), same light units. Compare in
  linear space, not sRGB PNG.
- **Match a shading model both can produce** for tight tests (e.g. a metallic-roughness
  BRDF Eevee and ours both implement); reserve Cycles path-traced references for
  *perceptual* checks only.
- **Pin everything:** a containerized, version-locked Blender; without it, references
  drift across Blender releases and "failures" are just version noise.
- **Per-AOV comparison:** export Blender's depth/normal/albedo AOVs and compare each to
  our matching pass — divergence localizes to a stage (geometry vs shading vs lighting).

### Tier 4 in practice: the OpenPBR golden harness (M45-F05 PBI-353)

The first concrete implementation of this pipeline is
`test-utilities/openpbr-goldens/` (Blender driver + params + committed reference
PNGs) and `head/internal/native/openpbr_golden_test.go` (the Go-side comparison),
generating our render via `head/internal/native`'s `RenderGoldenSphere` — the exact
`RTScene`/`SWScene` dispatch `renderer.Realistic` mode uses — through
`head/cmd/openpbrgoldenshot` for standalone reference regeneration. It follows the
pipeline above with three deliberate, documented deviations from the ideal stated
elsewhere in this doc:

- **Reference shader, not Blender's OpenPBR node.** This environment's Blender
  (4.0.2) has no native OpenPBR node yet, and neither MaterialX nor Substance is
  available to script headlessly. `render_reference.py` uses Blender's Principled
  BSDF v2 instead — verified (via a live headless parameter probe) to expose a
  near-1:1 superset of OpenPBR's own parameter groups (Base Color/Metallic/
  Roughness/IOR, Coat Weight/Roughness/IOR, Sheen Weight/Roughness/Tint for
  OpenPBR's Fuzz, Subsurface Weight/Radius, Transmission Weight), so it is a
  faithful perceptual reference for this tier, not an unrelated stand-in. Swap to a
  real OpenPBR node the day Blender ships one.
- **Comparison in display (post-tonemap sRGB PNG) space, not linear EXR.** Both
  sides already own a real tone-mapping pipeline meant to be the thing a user
  actually sees — Blender's Standard view transform on its side,
  `kernel/shading/openpbr.ToDisplay` (PBI-349, ACEScg → ACES-filmic → sRGB) on
  ours — so the golden compares those two *display-referred* outputs directly
  rather than re-deriving a separate linear-EXR path with its own tonemap-off
  configuration to maintain.
- **SSIM only; FLIP deferred.** A pure-Go windowed SSIM
  (`head/internal/native/ssim.go`) is implemented and unit-tested; FLIP's full
  perceptual color-difference model is not, since SSIM alone has proven adequately
  discriminating so far (see the calibration note below). A documented follow-up,
  not a placeholder — add FLIP if a future golden shows SSIM is too permissive.

**Calibration finding:** measured SSIM between our base-lobes render and the
Blender reference for a representative base material is **~0.85**, and is
essentially unaffected by sphere tessellation density (48×24 vs 96×48
segments/rings scored 0.8522 vs 0.8524) — meaning that ceiling reflects genuine
BRDF-model disagreement (our EON diffuse + single-scatter GGX vs Blender's
multi-scatter-GGX Principled BSDF v2), not a fixable rendering defect. The
committed test's threshold (`openpbrGoldenMinSSIM = 0.70`) sits with real margin
below that measured baseline.

**Scope:** only the base lobes (diffuse + specular dielectric/metal) are wired
into the live GLSL path tracer today, so only that tier has a live comparison
(`TestOpenPBRBaseLobesMatchBlenderReference`). Reference PNGs for
+coat/+fuzz/+subsurface/+transmission are already committed and a skipped test
table names them (`TestOpenPBRExtendedLobesPendingGLSLPort`); porting those lobes
into the live shaders is tracked separately (issue #2148).

## Determinism: software Vulkan (llvmpipe) in CI

GPU output is **not bit-reproducible** across drivers/hardware — so CI renders on
**Mesa lavapipe/llvmpipe** (a software Vulkan 1.x implementation): no GPU required,
identical output on every runner, every run. Combined with determinism mode (core/08:
fixed camera/viewport, no jitter/TAA, fixed seeds), goldens are stable within
tolerance. On a developer machine the same tests can run on the real GPU (looser
tolerance) to catch driver-specific issues. **Never assert bit-equality on shaded
output** — only on integer AOVs (ID, coverage masks).

## Comparison metrics & the LLM-facing report

Tiered, matched to the oracle:

| Tier | Metric | Threshold |
|---|---|---|
| analytic geometry (depth/ID/coverage/edge) | exact / ULP | bit-exact (IDs) or ε |
| CPU-reference lit | linear RMSE / MAE | tight (e.g. <0.01) |
| Blender PBR | **SSIM** + **FLIP** (perceptual) | loose, perceptual |

Every comparison emits a **structured report**, the heart of the LLM workflow:

```json
{ "pass":"normal", "tier":"analytic", "result":"FAIL",
  "rmse":0.214, "threshold":0.02,
  "error_region":{ "bbox":[112,72,40,40], "centroid":[132,92], "frac_pixels":0.06 },
  "passes_ok":["depth","object_id","coverage"],
  "artifacts":["normal_ours.png","normal_ref.png","normal_diff_heatmap.png"] }
```

Claude reads *"depth/ID/coverage pass; the **normal** pass fails by RMSE 0.21 in a
40×40 region at (132,92)"* and reasons about the normal-generation shader/stage —
instead of being handed two images it cannot see. The heatmap artifacts are for the
human reviewer; the **numbers and the failing pass** drive the model.

## Frame/buffer capture for deep debugging

When a report isn't enough, the `offscreen` backend can **dump every intermediate
buffer** (each AOV, the depth buffer, the ID buffer, pre/post-tonemap) as images +
**numeric stats** (min/max/mean/histogram per channel). The model is told the stats,
not shown the images: "ID buffer contains only id=0 → nothing was drawn → the draw
list is empty → test the `null`-backend draw-list build" — a localized chain back to
CPU-testable code. Optionally integrate a RenderDoc capture on CI failure as a
human-facing artifact.

## What this buys an LLM-built renderer

- The **geometric correctness** of the renderer (transforms, projection, culling,
  picking, winding, clipping) is pinned by **exact** analytic + metamorphic tests with
  no external dependency — the bugs most fatal to a CAD tool, caught cheapest.
- The **shading correctness** is pinned by a **CPU reference** (tight, dependency-free)
  with Blender as the **perceptual** backstop for full PBR.
- Every failure is a **number and a location**, with validation layers catching API
  misuse at the source — turning the renderer from the one un-debuggable subsystem
  into one where Claude gets the same actionable signal it gets from the pure-Go core.
