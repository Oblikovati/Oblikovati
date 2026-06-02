# ADR-0005 — Vulkan 1.3 renderer

**Status:** accepted · **Context:** the original Inventor stack owns its own
renderer; we build one. (realtime-3d skill §4–6)

## Decision

Render with **Vulkan 1.3**, using its modern core features, behind a layered
backend abstraction (`renderer/*_vulkan.go`) so app code never sees a `Vk*` type.

Lean on the 1.3 baseline:

- **Dynamic rendering** (`VK_KHR_dynamic_rendering`, core in 1.3) — no
  `VkRenderPass`/`VkFramebuffer` objects; begin/end rendering inline. Simpler
  pass management for our viewport + ID-buffer + overlay passes.
- **Bindless** descriptors (`descriptor_indexing`) — one large descriptor set of
  textures/buffers indexed by integer, set per-frame. Suits CAD's many materials
  and the draw-call-as-data model (the per-draw struct carries indices, not sets).
- **Timeline semaphores** + **synchronization2** — clean async between the recompute/
  tessellation jobs (ADR-0007) and the frame's upload/render.
- **Buffer device address** — pointer-based access to vertex/instance buffers,
  feeding typed per-instance "shader data" structs (realtime-3d §6).

## Platform note (important)

- **Linux/Windows:** native Vulkan 1.3 drivers.
- **macOS:** Vulkan via **MoltenVK** (Vulkan-on-Metal). MoltenVK exposes Vulkan
  1.2 + many 1.3 features as extensions; **dynamic rendering and timeline
  semaphores are available**, but a few 1.3 corners are emulated or absent. We
  therefore (a) **feature-gate** optional paths via `VkPhysicalDeviceVulkan13Features`
  queries, and (b) keep a portable fallback for anything not universally present.
  Do **not** assume the full 1.3 feature set on Mac. Validate via the device-feature
  query at startup, log the resolved feature level, and select code paths from it.

## Why these, not older Vulkan

The 1.3 feature set removes the most boilerplate-heavy, error-prone Vulkan code
(render-pass/framebuffer objects, per-object descriptor sets, binary semaphore
juggling). Less ceremony → fewer bugs → faster to a moving, picked, shaded model
on screen. All three target platforms support enough of 1.3 (natively or via
MoltenVK) to make it the baseline.

## Costs / mitigations

- **MoltenVK gaps** → device-feature gating + fallbacks (above).
- **Vulkan verbosity** even in 1.3 → wrap behind `Device`/`SwapChain`/`Painter`
  (realtime-3d §4); validation layers on in `build.Debug`.

## Consequences

See [core/08](../core/08-renderer-vulkan.md). Windowing/surface + the Vulkan
loader are the cgo (or purego) edge — ADR-0008.
