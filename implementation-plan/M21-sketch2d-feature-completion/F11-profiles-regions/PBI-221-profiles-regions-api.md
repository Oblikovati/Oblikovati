---
milestone: M21
feature: F11
pbi: PBI-221
title: Profiles & regions over /api
status: planned
estimate: M
---

# PBI-221 — Profiles & regions over /api

**Milestone:** M21  ·  **Feature:** F11 Profiles & Regions API

## Goal

Make the sketch's regions and profiles enumerable and selectable through `/api`, closing
the loop between the sketcher and the features that consume profiles.

## Scope / work

- **/api:** `contract.Profile`; `wire.ProfileInfo` (loops, area, closed) +
  `wire.RegionInfo`; `MethodSketchProfiles`; `client.Sketch.Profiles`.
- **/source:** `addin/router/sketch_profiles.go` surfacing `model/sketch/profile.go` +
  `regions.go` (region detection already exists); `var _ contract.Profile` assertion.
- **UI:** region highlight on hover already exists in `sketch_select.go`; ensure a picked
  region resolves to a profile id usable by feature tools; e2e.

## Acceptance criteria

- Dogfood: a sketch with two nested circles enumerates the expected regions (annulus +
  inner) with correct areas; picking a region returns a stable profile id consumable by
  `features.add`. `make ci` green.

## Depends on

PBI-200, existing region detection.
