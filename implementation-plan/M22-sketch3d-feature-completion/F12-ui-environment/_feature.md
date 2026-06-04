---
milestone: M22
feature: F12
name: UI Environment & Settings
status: partial (app-layer env+tools+ribbon+e2e done; head/ui ImGui dialogs + more tools TODO)
---

# M22 · F12 — UI Environment & Settings

The interactive shell that makes the 3D sketch usable: a "3D Sketch" ribbon command that
creates and enters a 3D sketch environment (no host plane; model-space picking with an
axis triad / orbit), the `Sketch3DSettings` object (`AutoBendRadius`), property windows
for the 3D sketch and its entities, and end-to-end UI tests covering every 3D tool.
Mirrors the 2D sketch environment (`app/sketch_env.go`, `head/ui/sketch_overlay.go`).

## Depends on
F01–F11 (the tools to host), M05 (commands/ribbon).

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-247](PBI-247-ui-environment.md) | 3D Sketch ribbon + environment + Sketch3DSettings + e2e tests |
