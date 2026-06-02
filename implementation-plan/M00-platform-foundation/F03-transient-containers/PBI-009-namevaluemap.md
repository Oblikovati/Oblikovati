---
milestone: M00
feature: F03
pbi: PBI-009
title: NameValueMap (string→variant options bag)
status: planned
estimate: S
---

# PBI-009 — NameValueMap (string→variant options bag)

**Milestone:** M00 Platform Foundation & Interop  ·  **Feature:** F03 Transient Object Collections

## Goal

Implement the universal options/context bag used for optional/extensible method arguments and event context.

## Scope / work

- Add/Insert/Remove/Clear/Count, name & index access, variant values.

## API contracts (interfaces / enums / collections)

- `NameValueMap`

## Acceptance criteria

- Used as `Options` for `Documents.OpenWithOptions`.
- Insert-before/after ordering works.

## Depends on

_See feature dependencies._
