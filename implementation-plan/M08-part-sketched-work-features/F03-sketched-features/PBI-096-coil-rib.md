---
milestone: M08
feature: F03
pbi: PBI-096
title: Coil & rib features
status: planned
estimate: M
---

# PBI-096 — Coil & rib features

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F03 Sketched Features

## Goal

Implement coil (helical sweep with pitch/revolutions/taper) and rib (thin support from open profile).

## Scope / work

- `CoilFeature`/`Definition` (pitch/height/revolutions/taper).
- `RibFeature` (open profile, thickness, to-next).

## API contracts (interfaces / enums / collections)

- `CoilFeature(s)`,`RibFeature(s)`

## Acceptance criteria

- A coil builds a helix; a rib thickens an open profile to adjacent geometry.

## Depends on

_See feature dependencies._
