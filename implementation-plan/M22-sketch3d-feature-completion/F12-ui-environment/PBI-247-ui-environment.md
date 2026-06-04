---
milestone: M22
feature: F12
pbi: PBI-247
title: 3D Sketch ribbon + environment + Sketch3DSettings + e2e tests
status: partial (app-layer env+tools+ribbon+e2e done; head/ui ImGui dialogs + more tools TODO)
estimate: M
---

# PBI-247 — 3D sketch environment & settings

**Milestone:** M22  ·  **Feature:** F12 UI Environment & Settings

## Goal
Let a user create, enter, and model a 3D sketch in the app, with settings and full
end-to-end coverage.

## Scope / work
- `app/create_sketch3d_tool.go`: "3D Sketch" `CommandDefinition` (ribbon, "3D Model"
  tab) that creates a `Sketch3D` and enters its environment.
- `app/sketch3d_env.go`: 3D sketch environment state (active sketch, model-space pick,
  axis triad); reuse orbit camera.
- `head/ui/sketch3d_overlay.go`: entity property window + `Sketch3DSettings`
  (`AutoBendRadius`) dialog.
- `/api`: `Sketch3DSettings` if exposed (AutoBendRadius getter/setter via setProperty).

## Acceptance criteria
- e2e: `TestSketch3DEnvironmentEndToEnd` — invoke "3D Sketch" command → environment
  active → add a line via the tool → exit → sketch persisted.
- Every 3D tool added in F02–F11 has ≥1 e2e test (consolidated here or in its PBI).
- `make ci` green; overall coverage ≥80%.

## Depends on
PBI-232 … PBI-246.
