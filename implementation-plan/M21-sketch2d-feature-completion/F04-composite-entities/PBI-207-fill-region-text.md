---
milestone: M21
feature: F04
pbi: PBI-207
title: Fill regions & text
status: planned
estimate: M
---

# PBI-207 — Fill regions & text

**Milestone:** M21  ·  **Feature:** F04 Composite & Parametric Entities

## Goal

Add the two remaining 2D-sketch annotative entities: `SketchFillRegion` (a hatched closed
region) and `TextBox` (sketch text with anchor + formatting).

## Scope / work

- **/source:** `model/sketch/fill_region.go` (closed-loop reference + fill style) and
  `model/sketch/text_box.go` (anchor point, string, height, justification, rotation);
  serialize round-trip. Region reuses existing region detection.
- **/api:** `addEntity` kinds `fillRegion`/`text`; `client.Sketch.AddFillRegion/AddText`.
- **UI:** fill-region tool (pick a region), text tool + `head/ui/sketch_text_dialog.go`
  (string, height, justification); e2e.

## Acceptance criteria

- Dogfood + UI e2e: fill region binds a closed loop; text box places with the given anchor
  and content. Round-trip preserves both. `make ci` green.

## Depends on

PBI-205, F11 region detection.
