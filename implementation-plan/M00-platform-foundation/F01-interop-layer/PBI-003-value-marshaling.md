---
milestone: M00
feature: F01
pbi: PBI-003
title: Value-type & array marshaling (structs, ref byte[], variants)
status: planned
estimate: M
---

# PBI-003 — Value-type & array marshaling (structs, ref byte[], variants)

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F01 Native/Managed Interop Layer

## Goal

Centralize marshaling of by-value structs, byte-array buffers (e.g. reference keys), and variant (`object`) parameters used pervasively in the API.

## Scope / work

- Blittable struct marshaling.
- `ref byte[]` buffer in/out (used by reference keys, storage).
- Variant conversion for `object` params/returns.

## API contracts (interfaces / enums / collections)

- Structs marshaling
- Variant conversion helpers

## Acceptance criteria

- `ref byte[]` round-trips a buffer native→managed→native unchanged.
- A boxed numeric/string/handle survives variant round-trip with correct `Type`.

## Depends on

_See feature dependencies._
