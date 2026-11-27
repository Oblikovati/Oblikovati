# ADR-0041: One kernel path for sketch profile extrusion

Status: proposed; implementation and qualification pending.
Date: 2026-11-07.
Tracking: [#3857](https://github.com/Oblikovati/Oblikovati/issues/3857).

## Context

The Rust host stores a parameter-driven, directed-line prism recipe and can
replay its body. The public `features.add` Extrude command and the desktop
Extrude tool do not yet use that recipe. The stored recipe requires explicit
boundaries, while the public command selects a bounded sketch region.

The pinned Martensite release exposes `profile_extrusion::build_planar_regions`
and `profile_extrusion::extrude`. The region operation accepts a solved sketch,
caller-owned curve and vertex identities, a plane, and a resolution. Its
result supplies bounded regions and source-to-edge mapping. The extrusion
result supplies a validated solid, source history, findings, and achieved
tolerance. These published interfaces require host qualification.

## Decision

Use the released Martensite region and extrusion operations as the one
production path for the public command and desktop tool. The host supplies the
current solved sketch, placement, identities, and achieved solver precision.
It persists authored parameter sources and region lineage, not a native
handle, tessellation, or an ordinal assumed stable across edits.

The desktop tool calls the same owner command as the typed public API. Preview
uses an isolated result and cancel leaves document history unchanged. Commit,
parameter edit, undo, redo, and reopen all replay through the same evaluator.
An unsupported profile or extent returns a named decline before publication.

This migration deletes the existing production selected-prism face and sweep
construction in `crates/martensite/src/prism` after its full corpus passes
through the released operations. The inspection-only `sketch_region_at` path
does not become a feature classifier. No fallback, rectangle-specific branch,
or second production mesher is added. Existing saved recipes and reference keys
need an explicit versioned migration before the old path is removed.

## Qualification

The gate covers zero-DOF parameter-driven rectangles and general supported
profiles; adjacent and nested regions; construction curves; shared and
coincident identities; invalid and near-degenerate inputs; and typed declines.
Per-face analytic properties, topology, validity, history, and persistent key
round trips are checked against independent OCCT evidence. The cube journey
also runs through keyboard and mouse input, parameter edit, undo/redo, save and
reopen, and named-face resolution. A total-volume match or a passing step
binding alone does not establish acceptance.
