---
milestone: M09
feature: F04
name: Modify & Direct-Edit Features
status: planned
---

# M09 · F04 — Modify & Direct-Edit Features

Multi-body and direct-editing operations: combining/splitting bodies, face-level edits (move/offset/delete/replace), thicken/offset of surfaces into solids, and history-free direct edits.

## In scope

- Combine; Split.
- MoveFace/FaceOffset/DeleteFace/ReplaceFace.
- Thicken/Offset; CopyObject.
- DirectEdit features.

## Out of scope

_None._

## Key API contracts delivered

- `CombineFeature(s)`,`SplitFeature(s)`,`MoveFaceFeature(s)`,`FaceOffsetFeature(s)`,`DeleteFaceFeature(s)`,`ReplaceFaceFeature(s)`,`ThickenFeature(s)`,`DirectEditFeature(s)`

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-106](PBI-106-combine-split.md) | Combine & split (multi-body) |
| [PBI-107](PBI-107-face-edits.md) | Move/offset/delete/replace face |
| [PBI-108](PBI-108-thicken-directedit.md) | Thicken/offset & direct-edit |
