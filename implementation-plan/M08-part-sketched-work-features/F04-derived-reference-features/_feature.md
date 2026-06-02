---
milestone: M08
feature: F04
name: Derived & Reference Features
status: planned
---

# M08 · F04 — Derived & Reference Features

Features that bring external geometry into a part: derived parts/components (associative copy of another part/assembly with scale/mirror/body-selection) and reference/non-parametric base bodies from import.

## In scope

- Derived part/component (associative).
- `ReferenceComponent`/reference features.
- `NonParametricBaseFeature`; `ImportedComponent`.

## Out of scope

_None._

## Key API contracts delivered

- `DerivedPartComponent`,`DerivedPartComponents`,`DerivedAssemblyComponent`,`ReferenceFeature`,`ReferenceFeatures`
- `NonParametricBaseFeature(s)`,`ImportedComponent(s)`,`ImportedComponentDefinition`

## Depends on

F01,F03.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-097](PBI-097-derived-components.md) | Derived part/component features |
| [PBI-098](PBI-098-base-imported.md) | Non-parametric base & imported geometry features |
