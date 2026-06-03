---
milestone: M20
feature: F11
pbi: PBI-189
title: Decal, Reference, Client, Mark & Finish
status: planned
estimate: M
---

# PBI-189 — Decal, Reference, Client, Mark & Finish

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F11 Cosmetic & Reference Features

## Goal

The features that annotate or reference rather than cut material, for full API/persistence parity.

## Scope / work

`DecalFeature` (image on a face via sketch); `ReferenceFeature` (frozen reference body from elsewhere); `ClientFeature` (add-in-owned feature with attribute payload); `MarkFeature` (laser/etch mark); `FinishFeature` (surface finish spec). Pass-through recompute; full triangle + .obk round-trip.

## API contracts (interfaces / enums / collections)

- `DecalFeature(s)`/`ReferenceFeature(s)`/`ClientFeature(s)`/`MarkFeature(s)`/`FinishFeature(s)` + `*Definition`.

## Acceptance criteria

- Each adds via its collection, round-trips through .obk, passes the running body through unchanged, and carries its payload (image ref/attributes/finish spec).

## Depends on

M08

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
