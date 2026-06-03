---
milestone: M21
feature: F09
name: Editing Operations
status: done
---

# M21 · F09 — Editing Operations

The sketch edit verbs that transform existing geometry: move, rotate, copy
(`MoveSketchObjects`/`RotateSketchObjects`/`CopyEntitiesTo`/`CopyContentsTo`),
trim/extend/split, and mirror about a line — each preserving or re-binding the affected
constraints.

## In scope

- Move / rotate / copy a selection (optionally producing copies).
- Trim a curve to its nearest intersections; extend to the nearest boundary; split at a point.
- Mirror a selection about a line (with symmetry constraints).
- Delete entities (cascading constraint cleanup).

## Out of scope

- Pattern (F10).

## Key API contracts delivered

- `MethodSketchTransform`, `MethodSketchTrim`, `MethodSketchExtend`, `MethodSketchSplit`,
  `MethodSketchMirror`
- `client.Sketch.{Move,Rotate,Copy,Trim,Extend,Split,Mirror}`

## Depends on

F02 entities, F06 constraints.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-216](PBI-216-move-rotate-copy.md) | Move / rotate / copy |
| [PBI-217](PBI-217-trim-extend-split.md) | Trim / extend / split |
| [PBI-218](PBI-218-mirror-delete.md) | Mirror & delete |
