---
milestone: M17
feature: F02
pbi: PBI-160
title: ACIS SAT & Parasolid exchange
status: planned
estimate: L
---

# PBI-160 — ACIS SAT & Parasolid exchange

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F02 Neutral CAD Formats

## Goal

Implement SAT and Parasolid (x_t/x_b) import/export for kernel-level B-rep interchange.

## Scope / work

- SAT read/write.
- Parasolid read/write.
- Body/precision mapping.

## API contracts (interfaces / enums / collections)

- SAT/Parasolid translators

## Acceptance criteria

- SAT and Parasolid solids round-trip as valid bodies.

## Depends on

_See feature dependencies._
