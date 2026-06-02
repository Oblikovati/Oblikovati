---
milestone: M08
feature: F04
pbi: PBI-097
title: Derived part/component features
status: planned
estimate: L
---

# PBI-097 — Derived part/component features

**Milestone:** M08 Part Modeling: Sketched & Work Features  ·  **Feature:** F04 Derived & Reference Features

## Goal

Implement associative derived parts/components that pull geometry from a source document with scale/mirror/body & parameter selection and update on source change.

## Scope / work

- `DerivedPartComponent`/`DerivedAssemblyComponent` definitions.
- Scale/mirror; body/work/parameter inclusion.
- Associative update on source edit.

## API contracts (interfaces / enums / collections)

- `DerivedPartComponent(s)`,`DerivedAssemblyComponent(s)`

## Acceptance criteria

- A derived part updates when its source changes; mirror/scale apply.

## Depends on

_See feature dependencies._
