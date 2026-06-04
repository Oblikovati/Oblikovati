---
milestone: M23
feature: F01
pbi: PBI-301
title: Full VisualStyle set + pure style→passes resolver
status: planned
estimate: M
---

# PBI-301 — Full `VisualStyle` set + pure style→passes resolver

**Milestone:** M23 Renderer Display-Mode Parity & Realistic PBR  ·  **Feature:** F01 Display-Mode Contract & Style Plumbing

## Goal

Extend the renderer's `VisualStyle` from three values to the full Inventor set and give
it a **pure resolver** that maps a style to the set of passes it draws — the single
place modes are defined, CPU-testable with no GPU.

## Scope / work

- Extend `renderer.VisualStyle` ([renderer/drawlist.go:106](../../../renderer/drawlist.go))
  to cover all eleven modes; keep `Shaded`/`ShadedWithEdges`/`Wireframe` behaviour
  unchanged. Update `String()` to stable user-facing names.
- Add a pure `func passesFor(style VisualStyle) PassSet` resolver: which passes a mode
  enables — `surface` (lit/PBR/NPR-shaded), `edge` (all edges), `hiddenEdge` (dashed
  occluded edges, F03), `npr` (stylization config, F04). The resolver is **data, not
  device** — it lives above the GPU line (ADR-0014).
- Have `BuildDrawListStyled` consult the resolver instead of its current `if style !=
  Wireframe` / `if style != Shaded` branches, so adding a mode is a resolver-table edit.
- Stub pass sets for modes whose passes arrive in F02/F03/F04 (resolver returns the
  intended set; the not-yet-implemented passes are no-ops with a tracked TODO), so the
  contract is complete and the resolver tests are total now.
- The renderer must **not** import the public `api` package — `VisualStyle` is internal;
  the app maps `DisplayModeEnum` → `VisualStyle` (ADR-0022 §6).

## API contracts (interfaces / enums / collections)

- (internal) `renderer.VisualStyle` full set; `renderer.PassSet`; `passesFor`

## Acceptance criteria

- `null`-backend unit tests assert, per mode, the exact `PassSet` the resolver returns
  (table-driven, total over all members) — pure, no GPU.
- `null`-backend draw-stream tests: each currently-renderable mode (8706/8708/8710)
  produces the same draw items as before this change (no regression); the new modes
  produce the resolver-intended item categories (with stub passes empty until their
  feature lands).
- Adding a mode requires only a resolver-table entry + enum value (proven by a test that
  every `VisualStyle` value has a `passesFor` entry — no default-case fallthrough).

## Depends on

PBI-300 (the `VisualStyle` values the public enum maps onto).
