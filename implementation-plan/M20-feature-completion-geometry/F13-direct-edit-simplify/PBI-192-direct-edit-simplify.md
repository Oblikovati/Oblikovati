---
milestone: M20
feature: F13
pbi: PBI-192
title: DirectEdit, Simplify, Unwrap & ModelTolerance
status: planned
estimate: M
---

# PBI-192 — DirectEdit, Simplify, Unwrap & ModelTolerance

**Milestone:** M20 Feature Completion & Geometry Parity  ·  **Feature:** F13 Direct-Edit & Simplify Features

## Goal

The whole-model editing features that don't fit a single local op.

## Scope / work

`DirectEditFeature` (move/size/scale/rotate/delete-face wrapper over F04 ops); `SimplifyFeature` (remove faces/fill voids/envelope to reduce a model); `UnwrapFeature` (flatten a curved face to a planar patch); `ModelToleranceFeature` (GD&T tolerance carrier on the model).

## API contracts (interfaces / enums / collections)

- `DirectEditFeature(s)`/`SimplifyFeature(s)`/`UnwrapFeature(s)`/`ModelToleranceFeature(s)` + `*Definition`.

## Acceptance criteria

- A direct-edit move dispatches to the F04 move-face op
- simplify removes a selected feature set producing a validated lighter body
- unwrap flattens a cylindrical face
- round-trip.

## Depends on

M20·F04

## Notes

Follows the canonical feature pattern set by Extrude (M08·PBI-092): full
`Definition → Add → Feature(+Proxy)` triangle, `/api` `wire`+`client`+`contract`,
`addin/router` handler, `.obk` round-trip, and executable tests.
