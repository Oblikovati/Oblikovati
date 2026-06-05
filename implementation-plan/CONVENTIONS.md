# Conventions

This document defines the cross-cutting rules every milestone and PBI obeys.
Read it once; it removes repetition from the individual files.

## The core mental model

A model is an **evaluated program**, not a bag of geometry: parameters →
sketches → features (in order) are *replayed* to produce geometry. Recompute,
rollback, identity, and undo all follow from this. Geometry is a cache of the
last evaluation, never the source of truth.

## Public API contract (/api) ↔ implementation (/source)

The public API lives in its own Apache-2.0 module, **`github.com/Oblikovati/api`**
(`/api`); the GPL application (`/source`) implements it (ADR-0018). Every PBI that
touches the public surface ships **the contract and the implementation together**:

1. **/api (contract, Apache-2.0):** enums/value types in `api/types` (defined once
   here; `/source` aliases them with `type X = types.X`), in-proc Go interfaces in
   `api/contract`, method-name constants + JSON DTOs in `api/wire`, and a typed
   method group in `api/client`.
2. **/source (implementation, GPL-2.0):** the behavior in `kernel/model/app/head`,
   a compile-time assertion that the impl satisfies `api/contract`
   (`var _ contract.X = (*impl.X)(nil)`), and the handler wired into `addin/router`
   keyed on the `api/wire` method constant.

Never re-declare a DTO or method string outside `api/wire`; never call the host from
an add-in with raw JSON (use `api/client`). `/api` must never import `/source`.

## The Definition → Add → Feature triangle (every modeling PBI)

Every feature ships **three** things, not one:

1. an `XDefinition` object (the editable, inert recipe of inputs),
2. an `XFeatures.Add(definition)` factory on the owning collection (plus
   `AddByXxx(...)` convenience overloads), and
3. an `XFeature` realized object whose `.Definition` get/set round-trips back to
   the recipe and triggers recompute.

When a PBI says "implement feature X" it always means all three, plus the
`XFeatureProxy` (assembly-context view) and the typed `XFeatures` collection.

## Universal object members

Every model object exposes `Type` (an `ObjectTypeEnum`, see M00) and a
`Parent`/`Application` back-pointer. Navigation up the ownership tree is as
important as navigation down. Construction always happens through the owning
**collection** (`.Add`), never via raw constructors — the collection is the seam
that enforces history, identity, transactions, and undo.

## Units & quantities (M02)

All geometry/values are stored in canonical **database units** (cm, radians).
User-facing units exist only at the display boundary. Every value that can be a
number is a dimensioned quantity with a `UnitsTypeEnum`. Many inputs accept
`object` so the caller may pass either a numeric value (database units) or an
expression string — resolve via the parameter/expression engine.

## Persistent identity (M03)

Topology references (a picked Face/Edge) must survive recompute, which destroys
and recreates the B-rep. Use **reference keys** (`ReferenceKeyManager`), never
pointer identity. Binding may fail ("reference lost") → the consumer goes
`HealthStatusEnum` sick; this is normal, not exceptional.

## Transactions & events (M04)

Every user-visible mutation is wrapped in a named `Transaction` (the undo label).
Events use the **object/sink split** (`XEventsObject` + `XEventsSink_Event`
composed into `XEvents`), fire with **before/after** timing, and let `before`
handlers **veto** via a `HandlingCode` out-param plus a `NameValueMap` context.

## PBI file template

Each `PBI-*.md` carries: **Goal**, **Scope / work**, **API contracts** (the
`api/types|contract|wire|client` additions — see "Public API contract" above),
**Acceptance criteria**, **Depends on**, optional **Notes**. Estimate sizes are
relative: `S` (<1wk), `M` (1-3wk), `L` (3-6wk), `XL` (kernel-deep, multi-month).

## Testing & verification (every PBI)

A PBI's **Acceptance criteria are executable tests**, not prose. The test
approach follows the layer (full strategy: `../architecture/testing/`):

- **Domain layers** (kernel, model, parameters, sketch/assembly solver, feature
  engine) are pure and GPU-free → unit + **property/metamorphic** tests run
  `CGO_ENABLED=0` in CI (e.g. `A ∪ A = A`, recompute idempotence, solver 0-DOF).
- **Identity** PBIs add the three reference-key tests: survives-recompute,
  fails-honestly (→ sick, no crash), survives-reload.
- **Persistence** PBIs add save→load **round-trip** equality + golden model files.
- **Renderer** PBIs are verified through the **oracle hierarchy** (analytic →
  CPU-reference → metamorphic → Blender) on the offscreen/null backends, never by
  eyeballing — see `../architecture/testing/00-renderer-oracle-pipeline.md`.
- **Public-API** PBIs are exercised through the `/api` contract — the `api/client`
  typed client over the host's `api/wire` methods (the **dogfood** suite), which
  doubles as the add-in conformance test. (A gRPC transport behind the same wire
  surface is a deferred future, ADR-0003/0016/0018.)

A PBI is "done" when its acceptance criteria are green in CI, not when it renders
something that looks right.

## Definition of Done — every user-facing feature ships its UI (MANDATORY)

A modeling feature is **not done when its model layer + serialization + unit tests
are green**. It is done only when a user can invoke it in the application and an
**end-to-end test drives that UI and validates the resulting geometry**. Every
user-facing feature ships **all four** of these, mirroring Extrude (the reference):

1. **Model** (`model/feature`): the `Definition → Add → Feature` triangle, `.obk`
   serialize round-trip, and unit tests.
2. **Command & interaction** (`app/`): a `CommandDefinition` ribbon button registered
   in `app/commands_standard.go`. Its **tab and panel placement MUST match the canonical
   ribbon tree** in
   [`architecture/mapping/inventor-ribbon-structure.md`](../architecture/mapping/inventor-ribbon-structure.md)
   — do not invent tabs/panels. (`NewCommand(id,name,panel,…).WithTab("3D Model")
   .WithIcon(…).WithTooltip(…)`), plus an interactive `Tool` (`app/<feat>_tool.go` —
   `Start`/`Pick`/`Set*`/`Commit`) and/or a settings window that edits the feature's
   inputs and writes them through to the definition.
3. **Property window** (`head/ui/<feat>_dialog.go`): the rendered Dear ImGui dialog —
   value fields, option combos, OK/Cancel, and in-canvas preview/highlight.
4. **End-to-end tests** (`app/<feat>_tool_test.go`): drive the command (by alias or
   synthetic click) → set the tool/dialog fields → commit → assert the feature lands
   in the model with the **expected geometry** (validated manifold, volume, etc.).
   The reference tests are `TestExtrudeToolEndToEnd`, `TestExtrudeViaCommandAlias`,
   `TestExtrudeDialogPathBuildsSolid`.

A feature whose only entry point is a Go API call, with no ribbon button, no property
window, and no UI-driven test, is **incomplete** regardless of model-layer coverage.

## Status model — three axes, not one ✅ (MANDATORY for trackers)

A single ✅ has historically hidden two failure modes: a feature with a complete model
layer but no UI, and a feature whose `Recompute` resolves its inputs then returns the
body **unchanged** (`ErrDeferred`/`NotYetImplemented`) — i.e. produces no geometry. To
keep PROGRESS.md and milestone tables honest, every PBI is graded on **three
independent axes**:

- **M (Model):** the `Definition → Add → Feature` triangle + `.obk` serialize
  round-trip + model unit tests are green.
- **G (Geometry):** the feature produces **real geometry**. An `ErrDeferred` or
  `NotYetImplemented` code path on the feature's primary case means **G ⬜**, no matter
  how complete M is. (Passthrough-with-Warning is *not* G-done.)
- **U (UI + e2e):** ribbon command + dialog/property window + preferences (where
  Inventor exposes them) + an end-to-end test driving the UI to validated geometry,
  per the Definition of Done above.

Write the grade as `M✅ G✅ U⬜`. A PBI is **Done** only when **all three** are ✅;
anything else is 🟦 and must show the per-axis flags so the gap is visible. A purely
infrastructural PBI (no geometry, no UI) is exempt from G/U and says so explicitly.

## Naming (mirror the contract library)

`XFeature` / `XFeatures` / `XDefinition` / `XProxy` / `XEnumerator` / `XEvents` /
`XEventsSink_Event`. `_X : X` is the dual-interface convention for additive
versioning; `_member` marks provisional/internal-but-exposed surface. `kXxx`
enum members carry **stable, explicit numeric ids** — never renumber.

## Kernel boundary

The public API is interfaces over a native geometry kernel (Parasolid/ACIS/
OpenCASCADE-class). Keep wrappers thin; logic lives in the kernel. All marshaling
ugliness is centralized in the interop layer (M00).
