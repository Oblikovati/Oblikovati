# ADR-0012 — Exact (analytic) hidden-line removal for drawing views

**Status:** accepted · **Context:** drawing views (plan M14-F02, PBI-139, flagged
XL) need visible/hidden edges from a 3D model. *How* we compute them is a genuine
fork that shapes the whole drawing subsystem.

## Decision

Compute drawing-view edges by **exact (analytic) hidden-line removal** from the
B-rep — producing **vector** visible/hidden curves with true geometry and model
associativity — not by reading back a rendered depth/ID image (raster HLR).

## Why exact, not raster

- **Drawings are vector deliverables.** Dimensions snap to true edges/arcs; export
  to PDF/DWG/DXF must be crisp vector lines at any scale. Raster HLR gives pixels,
  not curves.
- **Associativity needs real edges.** A dimension references a model edge by
  reference key (core/05); the drawing curve must *be* that edge's projection, so it
  updates when the model changes. Raster pixels have no identity.
- **Hidden lines are first-class.** Mechanical drawings draw hidden edges as dashed
  lines — we must classify each edge segment visible/hidden exactly, not discard
  occluded pixels.
- **Section/detail correctness.** Section hatching and detail boundaries need
  analytic intersection of the cutting plane with the B-rep, not image sampling.

## How (the engine)

```
view(model, camera, scale):
  silh   := silhouette edges (where face normal·view flips) + sharp model edges
  segs   := project candidate edges to the view plane
  occlude:= for each segment, test visibility against the projected faces
            (analytic: ray/face occlusion using the kernel's topology + evaluators, core/03)
  curves := classify each segment visible|hidden, tagged with the source edge's RefKey
```

- Reuses the **kernel** (core/03): face/edge evaluators, ray-occlusion, silhouette
  extraction — no new geometry engine, an *application* of the kernel.
- Output `DrawingCurve`s carry the **model-edge reference key** → associative; the
  whole view recomputes **async** (ADR-0007) when the model changes, like any
  dependent.
- Pure Go (ADR-0002); cross-compiles and tests headless via the null renderer
  (core/08) for thumbnails/CI.

## Costs / mitigations

- **Exact HLR is computationally heavy and intricate** (curved-surface silhouettes,
  tangent edges, self-occlusion) → phase it: start with **polyhedral/tessellated HLR**
  (occlusion against the triangle mesh) for correct-looking results early, upgrade to
  fully analytic curved silhouettes later behind the same `DrawingView` API. (Same
  de-risking shape as the kernel and solver: a correct simpler version first.)
- **Large assemblies** → cull by view frustum + occlusion BVH; compute per-component
  on the worker pool (core/00); cache per view, invalidate on model/camera change.
- **Performance** → views recompute off the frame loop and cache vector results;
  panning/zooming the sheet is pure 2D redraw, no HLR re-run.

## Consequences

See [apps/00](../apps/00-drawing-documentation.md). The drawing view is a DAG
dependent of the model (core/04 dirty-propagation) whose recompute is the HLR pass;
its output is reference-keyed 2D curves that dimensions and annotations attach to.
