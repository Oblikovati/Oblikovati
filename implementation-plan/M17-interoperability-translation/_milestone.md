---
milestone: M17
name: Interoperability & Translation
status: planned
---

# M17 — Interoperability & Translation

Import/export to the broader CAD/manufacturing ecosystem. A translator add-in framework (so formats are pluggable), neutral CAD formats (STEP/IGES/SAT/Parasolid), AutoCAD DWG/DXF interop, and modern visualization/exchange formats (STL/OBJ/3MF/glTF/JT/3D-PDF) plus shrinkwrap/derived export. Built on the kernel's tessellation and B-rep (M07) and healing on import.

## Goals

- A pluggable translator add-in framework with options.
- Neutral B-rep CAD exchange (STEP/IGES/SAT/Parasolid).
- AutoCAD DWG/DXF import/export.
- Mesh/visualization formats and shrinkwrap/derived export.

## In scope

- `TranslatorAddIn`; `TranslationContext`; capabilities; options.
- STEP/IGES/SAT/Parasolid import/export.
- DWG/DXF entities & interop.
- STL/OBJ/3MF/glTF/JT/3D-PDF; shrinkwrap.

## Out of scope (handled elsewhere)

- Drawing PDF/DWG export (M14 uses these translators).
- Native file format (M03).

## Exit criteria

- A STEP file round-trips a solid with healed topology.
- A DWG imports as geometry/blocks; a part exports to DWG/DXF.
- A part exports to STL and glTF for downstream use.

## Depends on

M07

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Translator Add-in Framework](F01-translator-framework/_feature.md) | 1 | Pluggable translators with context and options. |
| **F02** | [Neutral CAD Formats](F02-neutral-cad/_feature.md) | 2 | STEP, IGES, SAT, Parasolid B-rep exchange. |
| **F03** | [AutoCAD DWG/DXF](F03-dwg-dxf/_feature.md) | 1 | DWG/DXF entities, blocks, and interop. |
| **F04** | [Mesh & Modern Exchange](F04-mesh-exchange/_feature.md) | 2 | STL/OBJ/3MF/glTF/JT/3D-PDF and shrinkwrap. |
