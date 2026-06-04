---
milestone: M05
feature: F05
pbi: PBI-066
title: Persist saveWithDocument client graphics in .obk
status: planned
estimate: M
---

# PBI-066 — Persist saveWithDocument client graphics in .obk

**Milestone:** M05 Application UI, Commands, Interaction & Add-in Platform  ·  **Feature:** F05 Client Graphics

## Goal

Round-trip document-owned client graphics through the `.obk` YAML so an add-in's
persistent overlay geometry survives save/reload — closing the remaining half of
PBI-064's acceptance ("saves/reloads"). Follows the transient-first split: PBI-064
delivered the full in-session object model + render; this adds durability.

## Scope / work

- A `saveWithDocument` flag on the graphics group (wire + `clientgraphics.Group`).
- A YAML schema for groups/nodes/primitives + datasets (coordinates/indices/colors/
  normals/scalars/mapper) under `yamlcodec`; serialize only flagged groups.
- Load path: rehydrate flagged groups into the session `clientgraphics.Store` on open.

## Acceptance criteria

- An add-in draws persistent overlay geometry; after Save + reopen it is still shown.

## Depends on

PBI-064.
