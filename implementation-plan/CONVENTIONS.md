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

## Naming (mirror the contract library)

`XFeature` / `XFeatures` / `XDefinition` / `XProxy` / `XEnumerator` / `XEvents` /
`XEventsSink_Event`. `_X : X` is the dual-interface convention for additive
versioning; `_member` marks provisional/internal-but-exposed surface. `kXxx`
enum members carry **stable, explicit numeric ids** — never renumber.

## Kernel boundary

The public API is interfaces over a native geometry kernel (Parasolid/ACIS/
OpenCASCADE-class). Keep wrappers thin; logic lives in the kernel. All marshaling
ugliness is centralized in the interop layer (M00).
