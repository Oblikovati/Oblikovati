# ADR-0038 — Assembly render instancing (one mesh per unique component)

**Status:** accepted · **Context:** an assembly places the *same* part document many
times (a fastener used 10 000×). Today each occurrence is rendered as an independent
world-space body: `worldAssemblyBodies` calls `ops.TransformBody` per occurrence, then
the viewport **tessellates, flattens and uploads each copy separately** (ADR-0004 draw-
list-as-data, ADR-0014). For N identical copies that is N× the tessellation, N× the
`viewport.Flatten`, and N× the GPU vertex upload — 10 000 ellipse-Möbius strips is ~138 M
triangles of duplicated data that does not fit in one buffer. The geometry is identical;
only the placement transform differs. This is the dominant cost of large assemblies and
must be solved by **sharing one mesh across all placements of a component**.

This builds on the per-frame caches already landed: the tessellation cache
(`bodyGeometryCache`) and the flatten/bounds cache (orbit is camera-only). Those remove
*per-frame redundant* work; instancing removes *per-occurrence duplicated* work.

## Decision

Render repeated occurrences by **instancing**: tessellate/flatten/upload each unique
component mesh **once**, in component-local space, and draw it once per occurrence with a
**per-instance model matrix**. The transform moves from baked-into-vertices to a
per-instance GPU vertex attribute.

### Model / collection (pure Go, testable)

- Keep `VisibleBodies()` (world-space bodies) unchanged — picking, reference keys,
  selection, and mass properties stay world-space. Instancing is a **render-only** view.
- Add `VisibleInstances()` → `[]InstanceGroup{ Source *topo.Body; Transforms []Matrix4 }`.
  A part yields one group per body (a single identity transform). An assembly groups
  `PlacedBodies()` by their **shared source body pointer** (`pb.Body`) — the same part
  placed K times is one group with K transforms. Suppression/visibility already filter
  `PlacedBodies`.
- The renderer tessellates + styles each **Source** once (component-local) into a
  `DrawList`; the existing `bodyGeometryCache` keys on the source geometry version, so a
  component is meshed once regardless of placement count.

### GPU (Vulkan, `head/internal/native`)

- The interleaved vertex format (`kVertexFloats=16`, binding 0) is unchanged and holds
  **local-space** positions/normals. Add **binding 1, input rate `INSTANCE`**, stride 64:
  the 4×`vec4` model matrix (attribute locations 7–10).
- `mesh.vert`: `world = inModel * vec4(inPos,1); gl_Position = mvp * world;
  vWorldPos = world.xyz; vNormal = mat3(inModel) * inNormal` (rigid + uniform-scale
  placements; non-uniform scale is a later refinement). `pc.mvp` becomes the view-
  projection (no per-occurrence matrix baked in). Every geometry pipeline and the shadow
  pipeline share `mesh.vert`, so they all gain instancing; overlays/skybox are drawn as a
  single identity instance.
- `obk_viewport_render` is reorganised around **groups**: per unique source, ensure its
  vertex/index buffer is resident (persistent, keyed by a generation id so an unchanged
  source is uploaded once), upload that group's instance matrices, and issue one
  `vkCmdDrawIndexed(..., instanceCount=K, ...)` per stream. The single-part / overlay case
  is the K=1 identity group, so there is one code path.

### Staging

1. **Foundation (this ADR's first commit):** `VisibleInstances`, the instanced scene
   representation, tessellate-once dedup, and unit tests proving N copies build one mesh +
   N transforms. Pure Go, no GPU.
2. **GPU instanced draw:** the binding-1 attribute, `mesh.vert`, pipeline vertex-input,
   and the grouped `obk_viewport_render`. Verified live (a multi-placement assembly renders
   correctly; the unique mesh is uploaded once).

## Consequences

- 10 000 identical strips become 1 mesh + 10 000 × 64-byte matrices (~640 KB) instead of
  ~138 M duplicated triangles; tessellation and flatten run once.
- Per-occurrence picking/selection are unaffected (still world-space bodies).
- Mixed assemblies (several distinct components, each repeated) are several groups — still
  one mesh per *unique* component.
- Non-uniform per-occurrence scale needs the normal matrix (inverse-transpose); deferred.
- True 10 000-instance *frame rate* is then GPU vertex-throughput bound (the real geometry
  is still drawn) — LOD / impostors are an orthogonal later concern, out of scope here.
