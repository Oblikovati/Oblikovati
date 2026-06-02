# ADR-0004 — Dear ImGui shell + custom Vulkan viewport

**Status:** accepted (user decision) · **Replaces:** COM ribbon/browser/docking UI
(M05-F03).

## Decision

Build the application **chrome** (ribbon/toolbars, browser tree, parameter table,
property panels, dialogs, BOM/parts tables) with **Dear ImGui** via **cimgui-go**,
using the **docking** branch. The **3D viewport** is *not* ImGui — it is our own
Vulkan scene rendered into a viewport image, with custom interaction (selection,
manipulators, snapping). ImGui draws *over* the same Vulkan frame.

## Why

- **Tooling velocity.** Immediate-mode UI lets us stand up a usable feature tree,
  parameter editor, and dialogs in days, not months — critical while the kernel
  (the real long pole, ADR-0002) is under construction.
- **Trivial Vulkan integration.** ImGui has a first-class Vulkan 1.3 backend; it
  shares our command buffer and renders last in the frame.
- **Reflection-driven panels fit perfectly.** The realtime-3d skill's reflection
  property-editing pattern (§10) maps onto ImGui widgets: reflect over a feature's
  `Definition` struct + field tags → auto-generate the edit panel. One code path
  edits every feature type.
- **Docking** gives the browser/properties/viewport layout users expect, in one
  window (no floating OS popups — realtime-3d §12).

## Trade-off vs. the realtime-3d skill's recommendation

The skill (§8) advocates a *custom retained-mode* UI on the renderer and warns
against immediate-mode for product shells. We consciously diverge for the chrome,
accepting:

- **Less pixel-perfect product polish** than a bespoke retained UI;
- **Immediate-mode re-evaluation each frame** (cost is bounded; CAD UIs are not
  thousands of widgets and the 3D view dominates frame time anyway);
- **State lives in our model**, not the UI — which is actually *more* aligned with
  "logic in compiled code" than HTML-ish markup would be.

We **keep** the skill's patterns where they matter most — in the **viewport**:
the scene graph with dirty-flag transforms (§3), draw-call-as-data (§6), resource
caches (§5), the orthographic-camera idea (the viewport's own camera vs ImGui's
screen-space pass). So the divergence is scoped to the 2D chrome only.

## Boundaries

- **ImGui owns:** menus, panels, tables, trees, text/number inputs, modals,
  docking layout, toasts/overlays.
- **Viewport owns:** 3D rendering, hit-testing/picking (ID-buffer render target),
  drag manipulators, rubber-band selection, snapping previews (client graphics).
- The two communicate through the **runtime mediator** and the **selection** and
  **command** systems — ImGui never touches Vulkan handles, the viewport never
  hardcodes panel layout.

## Consequences

See [core/09](../core/09-ui-imgui.md). If a later product-polish pass wants a
bespoke shell, the model/command separation means the chrome can be reskinned
without touching domain code — but that is explicitly out of scope now.
