---
milestone: M17
feature: F04
name: Mesh & Modern Exchange
status: planned
---

# M17 · F04 — Mesh & Modern Exchange

Tessellation- and lightweight-exchange formats for 3D printing, visualization, and downstream consumption, plus shrinkwrap/derived simplified-export for IP protection and performance.

## In scope

- STL/OBJ/3MF (mesh).
- glTF/JT/3D-PDF (visualization).
- Shrinkwrap/derived simplified export.

## Out of scope

_None._

## Key API contracts delivered

- mesh/visualization translators over `TranslatorAddIn`,`SurfaceBody` tessellation (M07)

## Depends on

F01,M07.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-162](PBI-162-mesh-formats.md) | STL/OBJ/3MF & glTF/JT/3D-PDF export |
| [PBI-163](PBI-163-shrinkwrap.md) | Shrinkwrap / derived simplified export |
