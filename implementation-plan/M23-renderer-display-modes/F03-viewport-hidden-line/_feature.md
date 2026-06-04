---
milestone: M23
feature: F03
name: Viewport Hidden-Line Visibility
status: planned
---

# M23 · F03 — Viewport Hidden-Line Visibility

The three hidden-edge modes: `kWireframeNoHiddenEdges` (8711, visible edges only),
`kWireframeWithHiddenEdgesRendering` (8712, visible solid + hidden dashed), and
`kShadedWithHiddenEdgesRendering` / `kHiddenEdgeRendering` (8707, shaded faces + hidden
dashed). All three need **real-time** hidden-line removal in the viewport: per-frame
classification of each model edge as visible or occluded.

This is the **tessellated, real-time** cousin of [ADR-0012](../../../architecture/decisions/ADR-0012-exact-hidden-line.md)'s
exact analytic drawing HLR — deliberately distinct (ADR-0023 §3). It classifies edges by
testing them against the tessellated mesh depth buffer, which is fast enough per frame
and upgradeable to curved silhouettes later.

## In scope

- A per-frame edge-visibility classifier (depth-prepass tessellated occlusion).
- A dashed hidden-edge line pass.
- Wiring modes 8707 / 8711 / 8712 through the F01 resolver.

## Out of scope

- Exact analytic B-rep HLR (that is ADR-0012 / M14 drawing views).
- Curved-silhouette extraction (future upgrade noted in ADR-0023 §3).

## Key API contracts delivered

- (internal) `renderer` edge-visibility classifier; dashed hidden-edge pass
- No new public surface.

## Depends on

M07 (model edges + tessellation, [renderer/drawlist.go](../../../renderer/drawlist.go)
`ops.TessellateBody`), the depth pass
([architecture/core/08-renderer-vulkan.md](../../../architecture/core/08-renderer-vulkan.md)),
F01 (resolver + the three styles).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-306](PBI-306-edge-visibility.md) | Per-frame tessellated edge-occlusion classification + analytic oracle |
| [PBI-307](PBI-307-hidden-edge-modes.md) | Dashed hidden-edge pass + the three HLR modes wired |
