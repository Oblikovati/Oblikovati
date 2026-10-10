# ADR-0001: Rust application architecture

Date: 2026-10-10. Status: Accepted for the Rust target design.
Implementation evidence: pending R0-R9; this record does not certify a port.

Boundary: application composition and repository ownership.
Owner: `oblikovati-rust`; API and kernel internals have separate owners.

## Context

The user chose a new Rust application, the shared Rust API, and Martensite.
Copying the Go package graph would also copy its large session object and
kernel implementation. The new application needs explicit ownership and small
interfaces that can be tested without the desktop.

## Decision

Build a Cargo workspace with the responsibilities in the
[port plan](../rust-port/plan.md#architecture). Create crates when needed.
The model owns recipes and evaluation interfaces. Concrete storage and kernel
adapters implement those interfaces. The binaries construct and inject them.
Session, automation, add-in host, viewport, shell, platform, script, and solver
remain separate responsibilities.

Use Rust ownership, typed IDs, enums, traits, and `Result` for domain behavior.
Keep mutable collections with one owner; connect model relationships with IDs.
Do not reproduce COM variants, Go reflection-based dispatch, package `init`
registration, or application singletons. Register built-in capabilities
explicitly, in a deterministic order, and reject duplicate identifiers.
Use generics where they remove real duplication; do not convert every domain
interface into a generic framework.

Consume `oblikovati-api` from Oblikovati.API. The application is GPL-2.0-only;
the public API is Apache-2.0. Add-ins depend on the API. Neither the API nor
Martensite imports the application. The API owns public value types, traits,
wire declarations, clients, and C ABI declarations.

Martensite in RustyCADKernel is the only production modeling kernel. Required
Go-kernel behavior outside OCCT must be ported and adapted there. Missing OCCT
behavior must also be completed there. The application adapter owns conversion
and evaluation orchestration, with no private replacement geometry engine.

Only the kernel adapter and viewport import Martensite toolkits. The model
and solver have no window, GPU, filesystem, or loader dependencies. Keep native
UI and OS bindings in the platform layer, C add-in calls in the host layer,
and Vulkan resource access at the viewport interface defined in
[ADR-0008](ADR-0008-vulkan-renderer.md). Each unsafe boundary must document
ownership, validity, thread, and shutdown requirements.

## Interfaces and invariants

- Application interfaces carry immutable recipe inputs, typed results,
  diagnostics, and revision-scoped references. No public trait exposes a
  Martensite or window-library type.
- Expected modeling failure is a typed error or health result. A stub, panic,
  or unconditional success does not implement a capability.
- No Go host, Go kernel FFI bridge, or native OCCT fallback enters production.
  OCCT and SolveSpace can supply independent test evidence.
- Rust 1.98.1 / edition 2024 is the initial toolchain candidate from Martensite.
  R0 must prove it with the pinned API and kernel before locking the toolchain.

## Alternatives and consequences

A Rust wrapper around the Go application would retain two implementations and
their ownership problems. Copying the kernel into this repository would divide
the numerical authority. Both are rejected. A narrow native UI binding remains
possible because it does not own model behavior. The cost is early work on
interfaces and kernel readiness before a complete application can run.

## Legacy disposition

Legacy inputs are Oblikovati ADR-0001, 0002, 0006, 0008, and 0054 at the
[fixed revision](legacy-map.md#source-baseline). Retain typed behavior, isolated
platform code, and explicit identity. Replace Go, cgo build tags, and the
in-repository kernel with Rust ownership and Martensite. Omit the old native
OCCT escape route and Go generic-method conversion policy. The
[map](legacy-map.md) records section ownership.

## Validation required

R0 must build the first workspace from clean pinned dependencies and check
crate imports, explicit registration, API isolation, and license notices.
R1 must prove shape ownership through the real adapter. R1V must prove the
native UI interface. Until those checks pass, the crate list is a design,
not evidence that a dependency or thread model works.
