---
milestone: M21
feature: F11
name: Profiles & Regions API
status: planned
---

# M21 · F11 — Profiles & Regions API

Expose the sketch's derived `Profile`/`Profiles`/`ProfilePath` and region detection
through `/api`, so the regions a sketch yields (and the closed loops a fill/feature
consumes) are enumerable and selectable from add-ins and the UI — the seam between the
sketcher and downstream features.

## In scope

- Enumerate sketch regions (closed areas) and their bounding loops.
- `Profile`/`Profiles`/`ProfilePath` DTOs; pick a region/profile for a feature.
- Region highlight/selection feeding extrude/revolve (already partly wired).

## Out of scope

- Consuming a profile into a solid (M08); 3D profiles (`Profile3D`).

## Key API contracts delivered

- `contract.Profile`; `MethodSketchProfiles`; `wire.ProfileInfo/RegionInfo`
- `client.Sketch.Profiles`

## Depends on

F01; existing `model/sketch/profile.go`, `regions.go`.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-221](PBI-221-profiles-regions-api.md) | Profiles & regions over /api |
