---
milestone: M00
feature: F01
name: Native/Managed Interop Layer
status: planned
---

# M00 · F01 — Native/Managed Interop Layer

All boundary-crossing concerns live here so the rest of the codebase sees clean typed contracts: host bootstrap, object handle lifetime/identity, value-type marshaling, variant and ref/out conversion, and error propagation.

## In scope

- Managed host init over the native kernel (Coral-style).
- Handle wrappers preserving native object identity & equality.
- Value-type (struct) marshaling and array/`ref byte[]` plumbing.
- Variant boxing/unboxing and exception/HRESULT propagation.

## Out of scope

_None._

## Key API contracts delivered

- (internal) InteropHost, NativeHandle
- `GlobalUsings` (Coral.Managed.Interop)
- Structs: `_FILETIME`, `_LARGE_INTEGER`, `tagSTATSTG`

## Depends on

None.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-001](PBI-001-interop-host-bootstrap.md) | Interop host bootstrap & native runtime init |
| [PBI-002](PBI-002-handle-identity.md) | Object handle wrappers with preserved identity & equality |
| [PBI-003](PBI-003-value-marshaling.md) | Value-type & array marshaling (structs, ref byte[], variants) |
| [PBI-004](PBI-004-error-propagation.md) | Error & exception propagation across the boundary |
