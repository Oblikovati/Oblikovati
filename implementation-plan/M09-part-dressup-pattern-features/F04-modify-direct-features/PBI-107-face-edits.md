---
milestone: M09
feature: F04
pbi: PBI-107
title: Move/offset/delete/replace face
status: planned
estimate: L
---

# PBI-107 — Move/offset/delete/replace face

**Milestone:** M09 Part Modeling: Dress-up & Pattern Features  ·  **Feature:** F04 Modify & Direct-Edit Features

## Goal

Implement direct face-level edits as parametric features over reference-keyed faces.

## Scope / work

- MoveFace (translate/rotate).
- FaceOffset; DeleteFace (heal); ReplaceFace.

## API contracts (interfaces / enums / collections)

- `MoveFaceFeature(s)`,`FaceOffsetFeature(s)`,`DeleteFaceFeature(s)`,`ReplaceFaceFeature(s)`

## Acceptance criteria

- Each face edit recomputes and heals topology correctly.

## Depends on

_See feature dependencies._
