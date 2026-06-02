---
milestone: M05
name: Application UI, Commands, Interaction & Add-in Platform
status: planned
---

# M05 — Application UI, Commands, Interaction & Add-in Platform

The interactive application shell and extensibility platform. Delivers the add-in framework (so the product is open from day one), the command/control framework, the ribbon/browser/docking UI, the selection & interaction pipeline, and client (overlay) graphics. This is a *platform* milestone: the framework lands here, while command/ribbon population for each modeling capability is delivered alongside M06–M13.

## Goals

- A registered add-in can load, add commands, and participate in the session.
- A command/control framework with definitions, categories, and lifecycle events.
- Ribbon, browser pane, and dockable-window UI hosting.
- A selection/interaction pipeline (pick, hit-test, filters) and overlay graphics.

## In scope

- Add-in server/site/registration & automation.
- `CommandManager`, `ControlDefinitions`, command lifecycle.
- Ribbon/`CommandBars`, `BrowserPane`/`BrowserNode`, `DockableWindows`, environments.
- `SelectSet`, `InteractionEvents`, mouse/select events, hit-test, filters.
- `ClientGraphics` overlay/preview graphics.

## Out of scope (handled elsewhere)

- Per-feature command implementations (delivered with each modeling milestone).
- Drawing/assembly-specific browser content (their milestones).

## Exit criteria

- A sample add-in adds a ribbon button that runs an interactive command using selection and preview graphics.
- The browser reflects document structure; dockable panels host custom UI.

## Depends on

M04

## Features

| ID | Feature | PBIs | Summary |
|----|---------|:----:|---------|
| **F01** | [Add-in Framework](F01-addin-framework/_feature.md) | 2 | Load, register, and host third-party add-ins. |
| **F02** | [Command & Control Framework](F02-command-framework/_feature.md) | 2 | Command definitions, categories, and lifecycle. |
| **F03** | [Ribbon, Browser & Docking UI](F03-ribbon-browser-docking/_feature.md) | 3 | Ribbon/command bars, browser pane, dockable windows, environments. |
| **F04** | [Interaction & Selection](F04-interaction-selection/_feature.md) | 3 | Selection set, pick/hit-test, interaction events, filters. |
| **F05** | [Client Graphics](F05-client-graphics/_feature.md) | 2 | Add-in overlay & preview graphics in the model space. |
