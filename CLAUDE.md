## Code style

- Functions: 4-20 lines. Split if longer.
- Files: under 500 lines. Split by responsibility.
- One thing per function, one responsibility per module (SRP).
- Names: specific and unique. Avoid `data`, `handler`, `Manager`.
  Prefer names that return <5 grep hits in the codebase.
- Types: explicit. No `any`, no `Dict`, no untyped functions.
- No code duplication. Extract shared logic into a function/module.
- Early returns over nested ifs. Max 2 levels of indentation.
- Exception messages must include the offending value and expected shape.

## Comments

- Keep your own comments. Don't strip them on refactor — they carry
  intent and provenance.
- Write WHY, not WHAT. Skip `// increment counter` above `i++`.
- Docstrings on public functions: intent + one usage example.
- Reference issue numbers / commit SHAs when a line exists because
  of a specific bug or upstream constraint.

## Tests

- Tests run with a single command: `eg: go test`.
- Every new function gets a test. Bug fixes get a regression test.
- Mock external I/O (API, DB, filesystem) with named fake classes,
  not inline stubs.
- Tests must be F.I.R.S.T: fast, independent, repeatable,
  self-validating, timely.

## Dependencies

- Inject dependencies through constructor/parameter, not global/import.
- Wrap third-party libs behind a thin interface owned by this project.

## Structure

- Follow the framework's, sdk's language's convention.
- Prefer small focused modules over god files.
- Predictable paths: controller/model/view, src/lib/test, etc.

## Formatting

- Use the language default formatter (`cargo fmt`, `gofmt`, `prettier`,
  `black`, `rubocop -A`). Don't discuss style beyond that.

## Logging

- Structured JSON when logging for debugging / observability.
- Plain text only for user-facing CLI output.

## Directory Structure

**This repo is the GPL-v2 application** (`oblikovati`): the
kernel, UI head, CLI, tests, and release pipeline live at the repo root. The
Apache-2.0 contract was split out into a sibling repo for licensing reasons
(ADR-0018); it is resolved for local dev via the `go.work` workspace over sibling
checkouts (not committed; CI checks out the sibling).

This repo's top-level layout:
- the Go application module at the root — `kernel/`, `model/`, `app/`, `command/`,
  `event/`, `persistence/`, `renderer/`, `scene/`, `addin/` (incl. `addin/router`,
  which serves `api/wire`), and `cmd/` (the `oblikovati` and `oblikovati-cli`
  binaries). It `require`s `oblikovati.org/api` and implements its contracts.
- /head -> the cgo Vulkan + Dear ImGui windowed app, a separate submodule so the
  cgo build never touches the headless-tested core. It vendors the C ABI header at
  `head/internal/addinhost/include/oblikovati_addin.h` (the contract an add-in's
  shared library is compiled against) so it builds standalone.
- /test-utilities -> utilities and artifacts to help us test the code (e.g. blender projects).
- /architecture -> HOW we want to build; Documentation for architecture, ADRs.
- /experiments -> disposable experiments to validate things quickly before implementation (git-ignored).
- WHAT we want to build is tracked on GitHub: milestones (M00–M25 capability blocks)
  and issues (type Feature / Task, `area/*` labels). Conventions for that work live in
  /architecture/implementation-conventions.md; the pre-migration progress log is
  /architecture/history/implementation-log.md.

The Autodesk Inventor C# API reference (Oblikovati.Contracts.CSharp) is intentionally
NOT in this repo — it lives, read-only, in the archived monorepo
(github.com/Oblikovati/Oblikovati.Contracts) for consultation.

Sibling repo (checked out alongside this one; tied together by `go.work`):
- ../Oblikovati.API -> the public API contract, a standalone Go module
  (`oblikovati.org/api`), **Apache-2.0**. Four packages: `types` (enums, ids,
  value structs), `contract` (in-proc Go interfaces), `wire` (method-name constants
  + JSON DTOs), `client` (a `Transport` + typed client for add-ins). The source of
  truth for the API; it must NEVER import this application module (the dependency
  only flows the other way; CI enforces it). See ADR-0018.

Add-ins are separate shared libraries (their own repos, not this one). They link
**only** ../Oblikovati.API (Apache-2.0) — never this GPL module in their shipped
build — and reach the host over the C ABI (ADR-0016). Because the contract is
Apache-2.0, an add-in may be licensed however its author chooses, including
closed-source.

## Public API & implementation split (MANDATORY for all new work)

Adding to the public surface is always **two parts, in this order** (ADR-0018):

1. **Contract first, in ../Oblikovati.API** (Apache-2.0):
   - put enums / value types in `api/types` (define them ONCE there; this module
     aliases them with `type X = types.X` so existing call sites don't change);
   - put scalar Go interfaces in `api/contract`;
   - put method-name constants + JSON request/response DTOs in `api/wire`;
   - add a typed method group in `api/client` for any new wire method.
2. **Implementation here** (GPL-v2): build the behavior in `kernel/`, `model/`,
   `app/`, or `head/`; satisfy the `api/contract` interfaces with a compile-time
   assertion (`var _ contract.X = (*impl.X)(nil)`); wire the method into
   `addin/router` keyed on the `api/wire` constant.

Rules:
- Never re-declare a DTO or method-name string in this module or in an add-in —
  import it from `api/wire`.
- Never reach the host from an add-in with raw JSON — use `api/client`.
- Every new exported `.go` file carries an `SPDX-License-Identifier` header
  (Apache-2.0 in ../Oblikovati.API, GPL-2.0-only here); run
  `scripts/add-spdx-headers.py`.

  ## Kernel & host: ground rules for extension and refactoring

The bar is Parasolid/ShapeManager class: one general, exact pipeline per operation, equal to or better
than Inventor and SolidWorks. These rules bind every change under `kernel/`, `model/feature`, and the
host seams that consume them. Evidence: `architecture/audits/`, ADR-0042..0056, PR history 2026-07/08.

### Generality over special cases

- One general pipeline per operation: intersect → imprint/arrange → classify → build/stitch. Every input goes through it.
- A surface-pair closed form is a fast path INSIDE the general intersector, never a top-level handler. Intersectors are bucketed by representation (implicit / parametric), not by N² type pairs.
- A fast path is gated on conditioning, not on type: when the closed form is ill-conditioned, demote to the general path.
- A fast path must return the same topology, naming, and tolerances the general path would return. It is a shortcut, not a second algorithm.
- Before you add a branch for a failing input, find the invariant the pipeline breaks and fix it there. The input becomes a corpus test.
- An unsupported configuration is refused at classification with a named decline, before any geometry is built. Nothing ships as a runtime panic or a "not implemented" branch.
- Dispatch is a classification that selects exactly one path. Never add to an ordered try-list, a first-fit ladder, or a "load-bearing" order.
- A generalization is complete only when the special cases it replaces are deleted. A new engine shipped beside the old one as a fallback is not complete.
- A strangler migration carries the ticket that deletes the old system and the corpus gate that unlocks it. Both systems alive for more than one milestone is a defect.
- Behaviour many operations need (SSI, offset, closest point, seam/periodicity, curvature bounds) is a method on `geom.Surface`/`geom.Curve`. Type switches on geometry kinds live only in `kernel/geom`.
- Every kernel PR reports the net change in recognizers, tolerance constants, fallback sites, and type assertions. The net is ≤ 0 unless an ADR says why.
- A formula generalized to a new sign, side, or orientation must reproduce the original case bit-for-bit. Prove it; do not test it green.
- Certify a root or branch choice at runtime against the geometry (position, second-order test). Never hard-code which root is physical.
- Fillet, chamfer, and draft are one blend engine: spine → section functional → marcher → corner fill (ADR-0050/0051). New blend work lands in `kernel/blend`, not in a new `fillet_*.go`.

### Exactness and tolerance

- Every topological decision (side, inside/outside, coincidence, orientation, convexity) uses an exact or filtered predicate. Epsilon compares metric quantities only.
- One predicate package (`kernel/predicates`). Delete duplicates.
- Decide each incidence once and reuse the result. Two call sites must never recompute the same predicate with a chance of a different answer.
- No bare epsilon literal. A tolerance reads from `geom.Resolution` (ADR-0042) or carries a `// tol:<kind>` annotation, and `TestNoUnjustifiedAbsoluteEpsilons` covers every kernel file.
- Classify a comparison by the origin of its operands: same computation → `Weld()`, independent sources → `Sew()`, angular → arc length through `|dP/du|`. Never compare across classes.
- Achieved tolerance is a measured output of an operation, stored on the entity, capped by a maximum, and refused when a repair would inflate it past a ratio bound.
- Tolerances only tighten as operations compose. Coincidence is transitive; a body where a≈b, b≈c, a≉c is invalid.
- Keep modelling tolerance and approximation tolerance (chord, fit) separate. The sketch solver's achieved precision feeds the kernel's tolerance, not the reverse.
- Never move, nudge, or perturb geometry to make an operation succeed. Robustness lives in the combinatorial layer: choose a consistent topology; output coordinates stay exact.
- Output is byte-identical across runs and platforms: explicit total orders for every tie-break, no map-iteration or pointer/hash ordering, FMA-safe arithmetic in predicates.

### Tessellation is downstream

- No modelling or topological decision reads tessellated data: not classification, containment, mass properties, validity, section curves, or pick → reference key. Tessellation is a derived, regenerable view of the B-rep.
- One tessellator per problem: one constrained Delaunay, one polygon triangulator, one curve discretizer, one facet-count policy derived from tolerance.
- Mass properties integrate the analytic B-rep. An oracle that gates a result must be more exact than the result it gates.

### Validity and failure

- `Validate` is a post-condition of every public kernel operation. An invalid body is an error, never a return value.
- Validity has one implementation with ordered levels: topology/Euler → geometry-topology consistency → self-intersection → tolerance consistency. Each level gates the next.
- Never degrade silently. A fallback, approximation, or dropped element is a `diag.Defect` that reaches feature health, the API, and the UI.
- Failure is local: an operation returns a partial result naming the faulty entity; a sick feature quarantines its dependents and recompute continues.
- Result gates are per-face (area, surface type, loop count) against the oracle. A whole-body volume or area match is a smoke test, never a proof.
- Healing is a separate, explicit operation on a copy, and it is a naming operation: remap identity through it.
- A retry with widened or different inputs is a distinct, named strategy with its own corpus, not a loop around the same engine.

### Identity and history

- Every operation returns provenance: generated, modified, and deleted entities with their parent set (ADR-0043). Names derive procedurally from parents, never from build counters or geometric proximity.
- One naming mechanism, one namespace per feature instance. Ambiguous key resolution is an error; unguarded first-match lookups are forbidden.
- Entities history does not name get their identity from named neighbours, in both directions (lower from higher, higher from lower).
- Verify a name on write: store it, re-resolve it, reject it when it does not round-trip to the same entity.
- Re-lineaging inside an operation preserves every key that existed on input. A pick must survive the operation that consumed it.
- Naming recursion is bounded: depth limit, cycle guard, source cap. Orderings used by naming are versioned and migrated in place.

### Structure, API, and process

- Package by operation, not by case: split `kernel/ops` into boolean, blend, tessellate, validate, heal, query. Dependency direction `geom → topo → brep → ops → model` is enforced by `archguard`.
- Kernel operations are pure functions over immutable inputs, own no I/O, and take one versioned options struct with stable defaults.
- A kernel change that adds a capability starts with an ADR that names what it deletes. An ADR is superseded by a new ADR, never edited in place.
- Three test layers for every kernel change: invariant tests (closed, manifold, Euler, per-face oracle), OCCT/solvespace parity as hard assertions, and generated degenerate/near-degenerate inputs.
- A bug fix adds the failing input to the operation's corpus and makes the general pipeline pass it. A fix that passes only the corpus entry is a special case.
- Delete-first refactor: remove the duplicate, the dead engine, or the unused seam before adding the replacement's second call site.
- "Pre-existing" is proven only by bisecting against the wave base in a clean worktree.
- Size a cluster of failures by driving one representative case to a valid solid first; capabilities are layered and the next layer is hidden until then.
