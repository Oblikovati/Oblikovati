---
milestone: M05
feature: F05
pbi: PBI-067
title: Native depth-disabled pass for on-top graphics
status: done
estimate: M
---

# PBI-067 — Native depth-disabled pass for on-top graphics

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F05 Client Graphics

# Goal

Render `OnTop` client/interaction graphics (the overlay lane and burn-through
markers/labels — Inventor's BurnThrough) ignoring the depth test, so manipulators and
always-visible annotations draw over the model instead of being occluded by it. The
`renderer.DrawItem.OnTop` flag and the store/build plumbing already exist (PBI-064);
this adds the GPU pass that honors it.

## Scope / work

- Extend `head/viewport` `Flatten` with on-top triangle + line streams routed when
  `DrawItem.OnTop`.
- Extend the cgo binding `head/internal/native/viewport.go` `RenderViewport` and the C++
  Vulkan backend with a depth-test-disabled pipeline for those streams.

## Acceptance criteria

- An overlay-lane marker drawn behind a solid body is still fully visible.

## Delivered

- `head/viewport` `Mesh`/`Flatten` gained two on-top streams (`TopTri*`/`TopLine*`);
  `DrawItem.OnTop` triangles/lines route there instead of the regular streams (unit-
  tested in `flatten_test.go`).
- The cgo binding `RenderViewport` and `obk_viewport_render` carry the two streams; the
  C++ backend creates `topTriPipeline`/`topLinePipeline` with `VK_COMPARE_OP_ALWAYS` +
  depth-write off and draws them last (after hidden edges) so they composite over the
  model. `chrome_viewport.go` passes the streams through.
- Overlay-lane graphics (and any primitive with `onTop`) now render over the model; the
  prior depth-tested degradation is removed.

## Depends on

PBI-064.
