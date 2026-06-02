---
milestone: M17
feature: F04
pbi: PBI-163
title: Shrinkwrap / derived simplified export
status: planned
estimate: M
---

# PBI-163 — Shrinkwrap / derived simplified export

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F04 Mesh & Modern Exchange

## Goal

Implement shrinkwrap/derived export that simplifies an assembly/part (remove internals, fill holes) for sharing and performance.

## Scope / work

- Envelope/shrinkwrap generation.
- Internal-component removal; hole patching.
- Derived simplified body output.

## API contracts (interfaces / enums / collections)

- shrinkwrap over `DerivedAssemblyComponent`(M08)

## Acceptance criteria

- A large assembly produces a single simplified body hiding internal IP.

## Depends on

_See feature dependencies._
