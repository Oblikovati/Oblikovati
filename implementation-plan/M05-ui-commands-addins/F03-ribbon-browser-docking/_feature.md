---
milestone: M05
feature: F03
name: Ribbon, Browser & Docking UI
status: planned
---

# M05 · F03 — Ribbon, Browser & Docking UI

The user-interface host surfaces: the ribbon/command-bar system, the model browser tree with custom nodes, dockable windows for add-in panels, and the environment/workspace system that swaps UI per document type/mode.

## In scope

- Ribbon/`CommandBars`/`CommandBarButton`.
- `BrowserPane`/`BrowserNode`/`BrowserFolder` custom nodes.
- `DockableWindows`.
- `Environment`/`EnvironmentManager` workspaces; `UserInterfaceManager`.

## Out of scope

_None._

## Key API contracts delivered

- `CommandBars`,`CommandBar`,`CommandBarButton`,`CommandBarControls`
- `BrowserPane`,`BrowserPanes`,`BrowserNode`,`BrowserNodeDefinition`,`BrowserFolder`
- `DockableWindows`,`DockableWindow`
- `Environment`,`EnvironmentManager`,`UserInterfaceManager`

## Depends on

F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-058](PBI-058-ribbon-commandbars.md) | Ribbon & command bars |
| [PBI-059](PBI-059-browser-pane.md) | Browser pane with custom nodes |
| [PBI-060](PBI-060-docking-environments.md) | Dockable windows & environments/workspaces |
