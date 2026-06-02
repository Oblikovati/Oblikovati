---
milestone: M14
feature: F04
pbi: PBI-143
title: Parts list & balloons from BOM
status: planned
estimate: L
---

# PBI-143 — Parts list & balloons from BOM

**Milestone:** M14 Drawing & Documentation  ·  **Feature:** F04 Tables, Balloons & Sketched Symbols

## Goal

Implement parts lists sourced from the assembly BOM with configurable columns and balloons that auto-reference list items, both updating with the assembly.

## Scope / work

- `PartsList` from `BOMView`; column config.
- `Balloon` placement & auto item number.
- Balloon styles.

## API contracts (interfaces / enums / collections)

- `PartsList(s)`,`Balloon(s)`,`BalloonStyle`,`DrawingBOM(s)`

## Acceptance criteria

- A parts list and balloons reflect BOM item numbers and update on assembly change.

## Depends on

_See feature dependencies._
