---
milestone: M09
feature: F01
pbi: PBI-101
title: Shell, draft & thread
status: planned
estimate: L
---

# PBI-101 — Shell, draft & thread

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F01 Dress-up Features

## Goal

Implement shell (uniform/varied thickness, removed faces), face draft (pull-direction taper), and thread (cosmetic/modeled with thread tables).

## Scope / work

- `ShellFeature` thickness & face removal.
- `FaceDraftFeature` neutral plane & angle.
- `ThreadFeature` thread-table-driven specs.

## API contracts (interfaces / enums / collections)

- `ShellFeature(s)`,`FaceDraftFeature(s)`,`ThreadFeature(s)`,`ThreadInfo`

## Acceptance criteria

- Shell hollows a solid; draft tapers faces; thread carries spec data for drawings.

## Depends on

_See feature dependencies._
