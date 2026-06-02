---
milestone: M19
name: Materials & Appearances
status: in-progress
---

# M19 — Materials & Appearances

Two linked subsystems: **Appearance** (metallic-roughness PBR — what the renderer shows) and
**Material** (real-world physical properties: density + mechanical/thermal/electrical, linked
to an appearance). Materials and appearances are assignable to bodies, customizable
(duplicate-and-edit), shareable across a project's documents, and embedded per-document for
portability. Excludes 3D image-texture maps (deferred) — solid PBR values only.

See [ADR-0022](../../architecture/decisions/ADR-0022-materials-appearances.md).

## Goals

- Author and assign appearances/materials to bodies, with a live-updating viewport.
- Custom assets project-scoped (shared across documents) and embedded per-document.
- A physical-properties readout (mass/volume/area/centroid) from the assigned material.
- The whole surface on the public API for add-ins.

## In scope

- `model/material`: typed Appearance/Material assets, Library, built-in catalog, assignment
  store (by reference key) with the appearance-source override chain.
- Document embedding + project DesignData library persistence.
- Renderer PBR surface resolution; `ops` mass properties.
- `api` contract + `addin/router` + the head Materials window.

## Out of scope (later)

- Image-based texture maps (albedo/normal/roughness) + UVs.
- Face-level override *rendering* (mesh split per face); GGX/IBL shading.
- Material-driven simulation (M18) beyond the property data it consumes.

## Exit criteria

- Assign a material to a part, see the viewport recolor live, duplicate an appearance and
  alter its albedo, read the mass; reopen the `.obk` and the assignment + custom asset
  persist; another document in the project reuses the project-library asset.

## Depends on

M07 (B-rep bodies), M08 (part features), M05 (UI shell), ADR-0018, ADR-0020, ADR-0021.

## Features

| Feature | Title |
|---------|-------|
| F01 | Appearance & Material object model + built-in catalog |
| F02 | Scope tiers: project library + document embedding + persistence |
| F03 | Assignment & override chain (by reference key) |
| F04 | Renderer PBR surface resolution |
| F05 | Mass / physical properties |
| F06 | Public API: appearances/materials/assign/physical-properties |
| F07 | UI: Materials window (browser, editors, assign, readout) |
