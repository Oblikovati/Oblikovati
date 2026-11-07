# ADR-0037: Own drawing content in document history

Date: 2026-11-07. Status: Accepted for the drawing foundation slice.

## Context

Application BDD scenarios now require populated drawing layouts and saved drawings.
A sheet drawn from GUI constants cannot supply that document content or preserve
it through undo, save and reopen. The Go application is the architecture and
implementation reference; the scenarios define the required behavior.

The Go reference is `aa8297d1b0ac33c024f19e54821536d76e8a447a`:

- `model/drawing/content.go` owns sheets and injects property, body and BOM resolvers.
- `model/drawing/sheet.go` creates the default A3 landscape sheet.
- `model/drawing/border.go` and `titleblock.go` define its border and property bindings.
- `model/drawing/recipe.go` and `recipe_view.go` store sheet and view definitions.
- `app/drawing_environment.go` reconnects workspace-backed resolvers.

## Decision

`DocumentSnapshot` owns an optional `DrawingDocument`. Drawing documents receive
that content during create and open. Other document kinds have none. Ordered
sheet definitions, their border and title-block definitions, the active sheet,
the model reference and base-view definitions participate in history comparison
and the existing snapshot save path.

Reuse canonical API enums. Store their native Go recipe spellings under the
existing `model` section: `modelReference`, `standard`, `activeSheet` and `sheets`.
Keep other model and package sections when capturing a drawing. Store the initial
sheet explicitly on the first save; a reader-specific default is not persistence.
Refuse unsupported sheet or view recipe fields instead of dropping them.

A view definition is authored content. It is not an evaluated projection. A host
must resolve the referenced document and project checked geometry before it can
present a model view. Kernel handles, workspace pointers and resolved property
values do not enter this recipe. Future projection adapters use project-owned
interfaces and exact source references; they cannot use a GUI placeholder or a
renderer mesh as modeling input.

## Scope and remaining work

This slice implements model ownership, default sheet format, supported base-view
definitions, history, and package save/reopen. It does not implement HLR projection,
parts-list generation, populated-layout confirmation, stale-model warnings, cached
view curves or deferred-update behavior. Native BDD acceptance for those behaviors
remains required. The GUI's static sheet and browser branches must be replaced by
owner snapshots as those operations land under the Rust port task, issue #3563.

The Go parts-list resolver only accepts assemblies. The new BDD also asks for a
part source. A later implementation must define a real part row from its document
properties. Go view recipes reproject on open; deferred opening also needs retained
evaluation data and stable source revisions. Neither behavior is satisfied by
copying a filename or storing a dirty flag in the GUI.

Exchange parser ownership remains as defined in ADR-0018: the application owns
normalized drawing content, and Martensite owns the exchange provider. This slice
adds no parser dependency and does not expose parser types.

## Verification

Model tests check default dimensions and property bindings, invalid recipe refusal,
and atomic edits. A native Go-shaped supported recipe is read and round-tripped.
A FileWorkspace integration test creates a drawing, edits a referenced base-view
definition, checks undo/redo, saves, closes, reopens and checks the same owned state.
These checks prove recipe ownership and storage; they do not prove projection or
native drawing UI acceptance.
