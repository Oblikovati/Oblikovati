---
milestone: M22
feature: F08
name: Editing & Reference Ops (3D)
status: partial (PBI-240 edit-ops done; PBI-241 include/refkey TODO)
---

# M22 · F08 — Editing & Reference Ops (3D)

Editing operations on 3D sketch geometry — move/rotate/copy/delete (a 3D affine
transform applied to selected entities) — plus `Include` (project part edges/vertices/
work geometry into the active 3D sketch) and reference-key surfacing
(`GetReferenceKey`) so picks survive recompute.

## Depends on
F02 (entities), M07 (B-rep topology for Include), M03 (reference keys).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-240](PBI-240-edit-ops.md) | Move/rotate/copy/delete 3D entities + API + tools |
| [PBI-241](PBI-241-include-refkey.md) | Include part geometry + GetReferenceKey (3D) |
