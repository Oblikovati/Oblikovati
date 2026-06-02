---
milestone: M07
feature: F03
name: Boolean & Modeling Operations
status: planned
---

# M07 · F03 — Boolean & Modeling Operations

The core kernel operations features compose: boolean combination of bodies, tessellation for display/export, and geometry healing/validation that keeps topology sound after operations.

## In scope

- Boolean union/subtract/intersect.
- Tessellation (faceting) with tolerance.
- Healing/validation; sew/stitch primitives.

## Out of scope

_None._

## Key API contracts delivered

- (kernel) boolean/tessellate ops
- `PartFeatureOperationEnum`
- `SurfaceBody` tessellation API

## Depends on

F01,F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-082](PBI-082-booleans.md) | Boolean operations (join/cut/intersect/new-body) |
| [PBI-083](PBI-083-tessellation.md) | Tessellation & display faceting |
| [PBI-084](PBI-084-healing-validation.md) | Geometry healing & validation |
