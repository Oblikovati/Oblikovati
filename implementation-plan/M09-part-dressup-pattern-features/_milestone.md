---
milestone: M09
name: Part Modeling: Dress-up & Pattern Features
status: planned
---

# M09 — Part Modeling: Dress-up & Pattern Features

The features that operate on existing topology rather than sketch profiles: dress-up (fillet, chamfer, shell, draft, thread), holes and bosses, patterns and mirror, and direct-edit/modify features (combine, split, move/delete/replace face, thicken). These lean hardest on persistent topological identity (M03) because they consume picked faces/edges as inputs.

## Goals

- Dress-up features operating on selected edges/faces with variable options.
- Parametric hole and boss features with thread support.
- Rectangular/circular/sketch-driven patterns and mirror with element control.
- Direct-edit & modify features (combine/split/move-face/thicken/etc.).

## In scope

- Fillet/Chamfer/Shell/FaceDraft/Thread.
- HoleFeature (drilled/cbore/csink/tapped); Boss.
- Rectangular/Circular/SketchDriven patterns; Mirror; pattern elements.
- Combine/Split/MoveFace/DeleteFace/ReplaceFace/Thicken/DirectEdit.

## Out of scope (handled elsewhere)

- Surface-specific operations (M10).
- Sheet-metal features (M13).

## Exit criteria

- Fillet/chamfer/shell on selected topology recompute when upstream geometry changes.
- A hole with thread data drives a hole table later (M14).
- A pattern reuses a source feature with per-element suppression.

## Depends on

M08

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Dress-up Features](F01-dressup-features/_feature.md) | 3 | Fillet, chamfer, shell, draft, thread. |
| **F02** | [Hole & Boss Features](F02-hole-boss-features/_feature.md) | 2 | Parametric holes (drilled/cbore/csink/tapped) and bosses. |
| **F03** | [Patterns & Mirror](F03-patterns-mirror/_feature.md) | 2 | Rectangular/circular/sketch-driven patterns and mirror. |
| **F04** | [Modify & Direct-Edit Features](F04-modify-direct-features/_feature.md) | 3 | Combine, split, move/delete/replace face, thicken, direct edit. |
