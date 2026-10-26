# Martensite v0.4.0 migration

The Rust application pins the Martensite v0.4.0 release commit
`8373005675a5f286cdcc0f62aa9a4bf621dac79a`. Its adapter uses only released
`martensite-*` crates. It does not retain RustyCADKernel as a fallback.

## Released adapter path

The active adapter has the kernel support needed for the current vertical slice:
planar sketch construction, constraint solving, closed-profile validation,
prism creation, topology inspection, mass properties, and mesh production.

## Blocked Boolean profile regions

Martensite v0.4.0 does not release the former `TKBO` Boolean crate. The
`profile_regions` integration target therefore remains outside this release
migration. It stays in source as the regression corpus for the future general
Boolean pipeline. Do not restore RustyCADKernel only to run it.

The required upstream capability is tracked by Martensite issues 3, 40, 41, 42,
and 43. Re-enable the target only when its exact operation, provenance,
and per-face acceptance requirements are supplied by a released Martensite
crate.

## Native presentation migration

The workspace viewport still uses the former `martensite-vulkan` and `TKV3d`
crates. Martensite v0.4.0 releases `TKService` but not those renderer crates.
The native presentation migration is tracked by Oblikovati issue #3630. It must
replace the viewport renderer with the released Martensite services and a
separate maintained graphics backend. It must not extend the RustyCADKernel
renderer dependency.
