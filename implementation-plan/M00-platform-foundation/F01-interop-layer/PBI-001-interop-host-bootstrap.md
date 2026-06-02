---
milestone: M00
feature: F01
pbi: PBI-001
title: Interop host bootstrap & native runtime init
status: planned
estimate: L
---

# PBI-001 — Interop host bootstrap & native runtime init

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F01 Native/Managed Interop Layer

## Goal

Stand up the managed↔native host that loads the kernel, initializes the runtime, and exposes a typed entry point for creating root objects.

## Scope / work

- Initialize the native kernel and managed host (assembly load context).
- Define the root factory that yields the first `Application` handle.
- Deterministic teardown releasing native resources.

## API contracts (interfaces / enums / collections)

- (internal) InteropHost.Initialize/Shutdown
- Entry factory returning `Application`

## Acceptance criteria

- Host initializes and tears down with no leaked native handles.
- A root object can be obtained and disposed repeatedly within one process.

## Depends on

_See feature dependencies._
