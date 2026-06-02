---
milestone: M17
feature: F04
pbi: PBI-162
title: STL/OBJ/3MF & glTF/JT/3D-PDF export
status: planned
estimate: M
---

# PBI-162 — STL/OBJ/3MF & glTF/JT/3D-PDF export

**Milestone:** M17 Interoperability & Translation  ·  **Feature:** F04 Mesh & Modern Exchange

## Goal

Implement tessellation-based export (STL/OBJ/3MF) and visualization exchange (glTF/JT/3D-PDF) using the kernel faceting with appearance data.

## Scope / work

- Mesh export with tolerance/units.
- glTF/JT/3D-PDF with appearances (M16).
- Import of mesh (to M10 mesh features).

## API contracts (interfaces / enums / collections)

- mesh/glTF/JT translators,`SurfaceBody` tessellation

## Acceptance criteria

- A part exports a watertight STL and a textured glTF.

## Depends on

_See feature dependencies._
