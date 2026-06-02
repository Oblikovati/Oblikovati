# ADR-0014 — Renderer testability via a differential oracle hierarchy

**Status:** accepted · **Context:** the codebase is overwhelmingly authored and
debugged by an LLM (Claude). The pure-Go core (ADR-0002/0008) is already excellent
for that — failures are wrong *values* with stack traces. The **renderer is the
opposite**: bugs are *visual artifacts*, GPU state is opaque, Vulkan errors surface
far from their cause, and **an LLM cannot see the output to judge it.** This is the
single biggest testability risk in the project and needs a deliberate decision.

## Decision

Design the renderer to be **tested by construction** through five commitments:

1. **Three backends behind the `renderer` interface** (extends core/08, ADR-0008):
   - `vulkan` (windowed) — production;
   - `offscreen` (headless Vulkan → image buffer) — deterministic CI capture, runs on
     **software Vulkan (Mesa lavapipe/llvmpipe)** so it needs no GPU and is
     hardware-independent;
   - `null` (records the command stream, renders nothing) — unit assertions on what
     *would* be drawn.

2. **Push all logic below the GPU line.** Culling, draw-list building, batching/
   sorting, shader-data struct packing, and transform resolution are **pure functions
   over data** (the draw-call-as-data model already mandates this — core/08,
   realtime-3d §6). They are unit-tested on the CPU with the `null` backend, **no
   device required.** Only raw GPU submission needs a device.

3. **Per-pass AOV capture.** Every render pass (depth, normal, albedo, object-ID, lit)
   is **independently capturable** as an image. A failure localizes to one pass
   instead of "the picture is wrong."

4. **A pure-Go CPU reference shading path.** A slow, simple software implementation of
   the shading math (reusing `math/`) produces a reference image for the lit pass →
   the GPU output must match the CPU reference within tolerance. A **self-contained
   oracle with no external dependency** — ideal for an LLM inner loop.

5. **A tiered oracle hierarchy** (cheapest/most-deterministic first; Blender last):
   - **Analytic** — exact math for geometric passes (projection, coverage, depth,
     silhouette, picking IDs). No deps, exact, fully explainable.
   - **CPU-reference** — software shading (commitment 4) for the lit pass. No deps,
     tight tolerance.
   - **Metamorphic** — invariances needing **no reference image** at all (translate/
     rotate/scale/resolution/instancing equivalences). Cheap, powerful, LLM-authorable.
   - **Blender** — Cycles/Eevee as high-fidelity **ground truth** for full PBR on
     integrated scenes that are impractical to reference analytically (the user's
     oracle idea, situated as the top tier).

Plus: **Vulkan validation layers are a hard test gate** (any validation error fails
the test), and every image comparison emits a **structured numeric diff report**
(per-pass score, failing tier, error-region bounding box) designed for an LLM to
reason about — not a screenshot it cannot see.

## Why a hierarchy, not "just Blender"

Blender alone is the wrong primary oracle: it is slow, a heavy CI dependency, and
**exact pixel parity with it is unattainable** (different BRDFs, tonemapping, AA). A
senior test strategy makes the **fast, deterministic, dependency-free** oracles
(analytic / CPU-reference / metamorphic) carry the bulk of coverage as the inner
loop, and reserves Blender for full-PBR ground truth in CI with **perceptual**
tolerance. This honors the Blender-as-oracle insight while putting most of the
LLM-feedback load on oracles that are exact and need no GPU or external tool.

## Why this is the right shape for LLM-authored rendering

It converts the un-judgeable ("looks right?") into the actionable ("normal pass RMSE
0.21 > 0.02 in a 40×40px region centered (120,80); depth & ID pass") — a *localized
numeric signal* an LLM can act on. The cheap tiers run on every change with no GPU;
validation layers catch the API misuse LLMs commonly introduce; small single-purpose
shaders mean a failing golden points at one shader.

## Costs / mitigations

- **CPU reference path is extra code** → it implements only the *shading math* (small,
  reuses `math/`), not a full rasterizer; analytic oracles cover geometry.
- **GPU is not bit-reproducible across drivers** → software Vulkan (llvmpipe) in CI +
  tolerance-based comparison everywhere; never assert bit-equality on shaded output.
- **Blender CI dependency is heavy** → containerized, version-pinned, **outer-loop
  only**; cheap tiers gate the inner loop.

## Consequences

`core/08` is restructured around capturable passes and a sub-GPU testable layer (see
its revised Testability section). A **neutral scene-description format** is shared by
our renderer and the Blender harness. CI gains llvmpipe + a Blender container + a
golden-image **bless** workflow. Full detail: [testing/00](../testing/00-renderer-oracle-pipeline.md).
