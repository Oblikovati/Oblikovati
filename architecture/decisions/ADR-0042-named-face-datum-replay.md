# ADR-0042: Resolve named faces in the datum program

Status: proposed; Go document migration remains open.
Date: 2026-11-14.
Tracking: [#3882](https://github.com/Oblikovati/Oblikovati/issues/3882),
[#3884](https://github.com/Oblikovati/Oblikovati/issues/3884).

## Context

An offset work plane can depend on a generated planar face. The common part
program already orders datum and body features and quarantines failed readers.
It previously accepted only origin and datum planes as plane references. The
Go fixture in `architecture/rust-port/receipts/2026-11-14-go-dependent-face/`
stores a face key in a dependent plane and moves the plane after a size edit.

## Decision

Extend the existing plane-reference type with the complete semantic face key.
Validate its scheme and face kind when it is read. Derive its producing feature
from the key and include that feature in the existing dependency footprint.
During part replay, resolve the key against the current, unconsumed body through
the injected named geometry adapter. A missing body, lost name, non-planar face,
or ambiguous match is an error for that feature; dependents are quarantined.
Use the same datum construction operations for face and datum plane references.

Version the authored face-bearing definition as 13 and its enclosing work
program as 18. Preserve earlier versions for definitions without face inputs.
This deletes the blanket face-input refusal in named part replay. Datum-only
replay and adapters without named-face support still return an explicit error.
It adds no second feature graph, geometric proximity lookup, or saved native
handle.

## Limits and verification

The Martensite named prism adapter reads an analytic planar frame from the
unique oriented face occurrence in the solid. Its in-plane axes differ from
those observed in Go, even when origin and normal agree. Legacy migration must
map the face name and any dependent sketch coordinates explicitly; it must not
reinterpret a Go key as a Rust semantic key. The old profile-index mapping
also remains open. Before a public command writes a face reference, it must
resolve the name back to the same current entity. This decision does not
establish that command or import of the Go fixture.

Verify recipe version rejection, feature ordering, missing and consumed bodies,
foreign or malformed names, and local failure. Verify the real adapter against
all six Go cube face normals through edit, undo, redo, and recipe reopen. Verify
session invalidation and replay separately. The full R3 exit, supported public
commands, full gate, and foreground journey still require their own evidence.
