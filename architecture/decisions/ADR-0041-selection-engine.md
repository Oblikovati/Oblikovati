# ADR-0041 — Tools declare their selection; one engine owns filtering, picking and highlight

**Status:** Accepted (2026-06-22) · **Touches:** the `app` selection subsystem
(`app/selecting.go` — the `Selecting` and `Picking` contracts and `Session.installToolFilter` /
`Session.ToolPicks`; `app/selection.go`, `app/selection_priority.go`, `app/selection_filter_state.go`
— the existing filter/ambient state from #1222), the tool lifecycle (`app/session_input.go`), every
interactive tool under `app/*tool*.go` / `*_tools.go`, and the head highlight layer
(`head/ui/tool_highlight.go`).

## Context

Selection was **bespoke per tool**. Each of ~49 tools imperatively called
`s.Selection().SetFilter(NewSelectionFilter(...))` in `Start` and restored it in `Cancel`/`Commit`,
re-stating its accepted kinds inline. The head highlighted a tool's picks by **duck-typing** five
magic accessor names (`Faces()`, `Edges()`, `PickedProfile()`, `PickedProfiles()`, `PickedFace()`);
a tool that named its accessor anything else silently got no highlight, and work planes/axes were
highlighted by separate ad-hoc queries.

This produced real defects: Create 2D Sketch declared a work-plane-only filter, so hovering a solid's
planar face highlighted nothing and a click resolved to the origin plane hidden behind the body —
you could not sketch on a face. The same shape of bug could recur in any tool, and "any tool that can
select a planar face highlights it" was not guaranteed (e.g. Extrude had no termination-face step at
all).

## Decision

**Tools declare what they select; the host owns filtering, picking and highlight.** Two small
contracts in `app/selecting.go`:

- `Selecting { AcceptedKinds() []SelectionKind }` — the kinds pickable at the tool's *current step*.
  Re-read after every pick, so a multi-step tool (Extrude: profile → termination face; Revolve:
  profile → centerline; the redefine tools: per armed slot) changes its accepted kinds reactively.
  Empty ⇒ no restriction (the ambient Selection Filter, #1222, applies).
- `Picking { Picks() []Selectable }` — the tool's current picks, as one uniform list.

The `Session` derives the active filter from the running tool's declaration
(`installToolFilter`, called on tool start and after each pick) and restores the ambient filter when
the tool ends (`restoreSelectionFilter` in `OK`/`CancelTool`). The head highlights preselection (the
hover candidate the filter would pick) and the tool's `ToolPicks()` through **one per-kind renderer**
(`drawSelectable`, keyed on `Selectable.SelectionKind()`): a face is outlined **and** tinted with a
translucent on-top fill (an outline alone z-fights with the model's own edges into invisibility),
edges/profiles are outlined, work planes/axes use their overlays.

**Rules (enforced by convention + review):**
- A tool **never** calls `Selection().SetFilter` and **never** exposes bespoke highlight accessors.
  It declares `AcceptedKinds` and reports `Picks`.
- A selection kind's highlight appearance is defined **once**, in the per-kind renderer.

## Consequences

- The Create-Sketch face-host bug class is gone: a tool's pickable kinds are declared in one place,
  and every tool that accepts a kind highlights it identically. Extrude **To Face** was added as the
  first new consumer (kernel/model already supported `ToFaceExtent`) and got face selection +
  highlight for free.
- The ~49 imperative `SetFilter` call-sites and the head's five duck-typed accessors are removed.
- Selection state still lives in `app` (the head is a thin renderer); no public-API change, matching
  how `SelectionPriority`/`SelectionFilterState` (#1222) shipped.

## Alternatives considered

- **Keep per-tool `SetFilter` + a permanent duck-typed pick adapter.** Rejected: it leaves the
  bespoke coupling and silent-no-highlight failure mode the engine exists to remove.
- **Intersect a tool's filter with the ambient user filter.** Deferred: a tool's declared kinds win
  while it runs (v1), as before; intersurfacing them is a possible later refinement.

## API / add-in exposure

The engine is **not** exposed over the wire, and deliberately so: add-ins drive the model by
*reference*, not by interactive picking. A feature that takes a geometric input takes a reference
key — and the host resolves it the same way a viewport pick would. So **Extrude "To Face" is
add-in-/MCP-drivable through the existing `features.add` path** (no new wire method): the extrude
kind's schema gains `extent: "to-face"` + a `toFace` reference (a planar face key from
`model.referenceKeys`, a work plane `"plane/N"`, or an origin plane `"origin/plane/xy"`), resolved
by the shared `feature.WorkGeometry.PlaneTargetFromRef` (also used by the router's plane re-pick, via
`feature.ParseWorkRef`, so the two never drift). The selection-*set* is already
read/mutable over the wire (`model.select` / `deselect` / `clearSelection` / `selection` +
`selection.changed`). The interactive engine (`AcceptedKinds`/`Picks`, the ambient filter) stays a
host concern — there is no live cursor for a headless add-in — so wiring it would add surface with
no consumer. If a future add-in genuinely needs to constrain the *user's* ambient selection, that is
a small get/set on the `SelectionFilterState` (#1222), added then, not speculatively.

## Out of scope (follow-ups)

- Cross-restart persistence of the ambient filter; a get/set wire surface for it only if an add-in
  needs to steer the user's selection.
- The remaining reference extents over the API (from-to, distance-from-face) — the same
  `PlaneTargetFromRef` seam, wired when needed.
- Geometric sub-type filters (planar vs cylindrical faces, linear vs circular edges) beyond the
  coarse `SelectionKind` taxonomy.
- Highlighting body/path/sketch-entity picks that the per-kind renderer currently draws as no-ops.
