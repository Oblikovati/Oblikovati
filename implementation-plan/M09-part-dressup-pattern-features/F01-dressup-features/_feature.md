---
milestone: M09
feature: F01
name: Dress-up Features
status: planned
---

# M09 · F01 — Dress-up Features

Topology-modifying dress-up features that round/bevel edges, hollow bodies, taper faces, and apply threads — each consuming reference-keyed edge/face selections and surviving upstream recompute.

## In scope

- Fillet (constant/variable radius, setbacks).
- Chamfer (distance/distance-angle/two-distance).
- Shell (thickness, removed faces).
- FaceDraft; Thread.

## Out of scope

_None._

## Key API contracts delivered

- `FilletFeature(s)`,`FilletDefinition`,`ChamferFeature(s)`,`ShellFeature(s)`,`FaceDraftFeature(s)`,`ThreadFeature(s)`
- edge/face `*Proxy` inputs

## Depends on

M08.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-099](PBI-099-fillet.md) | Fillet feature (constant/variable/setback) |
| [PBI-100](PBI-100-chamfer.md) | Chamfer feature |
| [PBI-101](PBI-101-shell-draft-thread.md) | Shell, draft & thread |
