---
milestone: M17
feature: F02
name: Neutral CAD Formats
status: planned
---

# M17 · F02 — Neutral CAD Formats

Boundary-representation exchange with the wider CAD world: STEP (AP203/214/242), IGES, ACIS SAT, and Parasolid, with healing on import and assembly-structure preservation where supported.

## In scope

- STEP import/export (solids/assemblies/PMI where available).
- IGES; SAT; Parasolid.
- Import healing (M07); unit mapping.

## Out of scope

_None._

## Key API contracts delivered

- STEP/IGES/SAT/Parasolid translators over `TranslatorAddIn`
- `NonParametricBaseFeature`(M08) for imported solids

## Depends on

F01,M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-159](PBI-159-step-iges.md) | STEP & IGES import/export |
| [PBI-160](PBI-160-sat-parasolid.md) | ACIS SAT & Parasolid exchange |
