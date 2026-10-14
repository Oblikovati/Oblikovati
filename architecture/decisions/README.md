# Rust architecture decision records

The Rust application starts with a clean ADR set, as requested on 2026-10-10.
The migration of all 66 legacy records is recorded in the
[legacy map](legacy-map.md). The consolidated Rust records are accepted
as the target design. Application implementation and runtime proof are pending.

## Authority and numbering

Preexisting Go ADRs are **legacy references**. Their accepted status, numbering,
and supersession chains do not carry into the Rust application. Read them for
intent, requirements, alternatives, and lessons from the Go implementation.
Keep the original records in the Go repository, with references to the fixed
[source revision](../rust-port/source-map.md).

The new sequence starts at `ADR-0001`. Use filenames such as
`ADR-0001-<decision-title>.md`; assign numbers when records are created.
The next unused number is `ADR-0021`.
Qualify old references with the repository, commit, and legacy ADR number so
they cannot be confused with a new Rust ADR that has the same number.

Only decisions reviewed and accepted for Rust become current Rust ADRs.
Applicable legacy decisions must be rewritten for Rust ownership, lifetimes,
threading, errors, crate dependencies, native interfaces, and Martensite.
Carry forward relevant product requirements and tested behavior. Reassess Go
implementation choices and unsupported claims against the current source.
Design acceptance is separate from implementation acceptance. Each record
states its pending milestone evidence; keep runtime receipts and capability
status in the implementation ledgers. Do not infer that all legacy roadmap
requirements were implemented in the Go baseline.

## One current ADR per architectural boundary

Consolidate legacy ADRs that describe the same architectural boundary into
one coherent Rust ADR. Organize by responsibility and interface, rather than
by old ADR number, bug, implementation wave, or feature option.

Resolve conflicting legacy decisions in the new record. State the chosen Rust
design, reasons, trade-offs, and acceptance checks together. Readers must not
need to follow a chain of legacy amendments to understand the current design.

A legacy ADR can cover more than one boundary. Map its relevant sections to
the appropriate new records. Record each decision once, then link to it from
other boundaries. Keep implementation details, incident reports, and long test
catalogues in supporting documents and issues so each ADR stays under 500 lines.

The index below names exactly one current accepted ADR per boundary.
Drafts can change during review. If an accepted Rust decision later needs
replacement, use an explicit superseding record and update the index; retain
the historical decision. There must still be only one current authority for
that boundary.

## Migration process

The [legacy-to-Rust map](legacy-map.md) completes the ADR portion of R0.
Each row identifies a legacy ADR or section, its source revision, its boundary,
and one disposition:

- **Adapt and consolidate:** applicable to Rust; name the target Rust record
  and the evidence needed to accept it.
- **Reference only:** useful history, with a reason it supplies no current
  Rust decision.
- **Not applicable:** specific to the old implementation; record why it is
  omitted and where any remaining product requirement is covered.

An unresolved disposition remains pending work. Do not treat a decision as
accepted because its source ADR was accepted. Removing a Go-specific design
choice does not remove an application capability from the port requirements.

Each new ADR includes its boundary and repository owner, status, context,
decision, alternatives, consequences, interfaces and invariants, validation
evidence, and legacy references. Explain what was retained, changed, combined,
or omitted. Acceptance depends on the new Rust design and evidence.

Application decisions live here. Kernel-internal decisions belong to
Martensite; API decisions belong to Oblikovati.API. A Rust application ADR
records the integration contract and links to the owning repository's record
instead of restating its internal design as a second authority. If an owning
record does not exist yet, record it as an implementation prerequisite, as
ADR-0007 does for required Martensite extensions. Do not invent its approval.

## Vulkan consolidation

[ADR-0008: Vulkan renderer](ADR-0008-vulkan-renderer.md) consolidates the
renderer interface, presentation, resources, synchronization, platform
capabilities, display modes, lighting, instancing, capture, and validation.
Its legacy inputs are assigned by section in the migration map.

UI behavior is owned by ADR-0009, asset state by ADR-0019, packaging by
ADR-0013, and public declarations by Oblikovati.API. These records reference
the shared rendering interface. There is no separate current renderer ADR
for a display mode, material model, or platform exception.

## Current Rust ADR index

All current records below are **Accepted for the Rust target design**.
Each record states its decision date. No application runtime acceptance is claimed.

| Current record | Boundary | Implementation proof |
| --- | --- | --- |
| [ADR-0001](ADR-0001-rust-application-architecture.md) | Rust application composition and repository ownership | R0/R1 |
| [ADR-0002](ADR-0002-api-and-addin-host.md) | Public automation and native add-in lifecycle | R2/R4 |
| [ADR-0003](ADR-0003-documents-and-storage.md) (superseded by ADR-0014) | Document recipes, resources, projects, configuration | R2/R3/R6/R8 |
| [ADR-0004](ADR-0004-session-and-feature-evaluation.md) | Session transactions, invalidation, feature evaluation | R2-R6 |
| [ADR-0005](ADR-0005-identity-and-references.md) | Persistent identity and external references | R1/R3/R5/R6/R8 |
| [ADR-0006](ADR-0006-parameters-and-constraints.md) (superseded by ADR-0015) | Parameters, units, shared constraint solver | R2/R3/R6 |
| [ADR-0015](ADR-0015-parameters-and-constraints.md) | Parameters, units, solver, analytic ellipse variables | R2/R3/R6 |
| [ADR-0016](ADR-0016-derived-sketch-curves.md) | Derived spline ownership and analytic offset evaluation | R3; #3789 |
| [ADR-0017](ADR-0017-sketch-point-ownership.md) | Standalone entities, defining points, deletion closure | R3; #3789 |
| [ADR-0007](ADR-0007-martensite-integration.md) | Application requirements on Martensite operations | R1-R8; K01-K11 |
| [ADR-0008](ADR-0008-vulkan-renderer.md) | Vulkan presentation and rendering | R1V/R4/R7 |
| [ADR-0009](ADR-0009-ui-and-selection.md) | Desktop UI, tools, selection, themes, editor | R1V/R4/R7 |
| [ADR-0019](ADR-0019-materials-and-appearances.md) | Materials, appearances, asset assignment; supersedes ADR-0010 | R5/R7 |
| [ADR-0011](ADR-0011-scripting.md) | Lua runtime contract and rule execution | R7 |
| [ADR-0012](ADR-0012-build-and-validation.md) | Build and integration evidence | R0 onward |
| [ADR-0013](ADR-0013-packaging-and-release.md) | Packages, updates, release transition | R9 |
| [ADR-0020](ADR-0020-documents-and-storage.md) | Documents, storage, schema-3 links and occurrence assets; supersedes ADR-0014 | R2/R3/R6/R8 |
| [ADR-0018](ADR-0018-exchange-format-ownership.md) | Kernel-owned exchange formats and application workflows | K10/R8 |
| [ADR-0021](ADR-0021-semantic-topology-reference-keys.md) | Semantic topology names and scheme-2 identity | R1/R3/R5 |
| [ADR-0022](ADR-0022-analytic-body-properties.md) | Analytic body integrals, units, and principal inertia | R1/R7 |
| [ADR-0023](ADR-0023-analytic-part-aggregation.md) | Analytic part aggregation and empty-system semantics | R1/R7 |
| [ADR-0024](ADR-0024-released-sketch-spline-construction.md) | Released construction of authored sketch splines | R2/R3/R4 |
| [ADR-0025](ADR-0025-typed-sketch-spline-authoring.md) | Typed sketch spline authoring through the existing owner | R3/R4 |
| [ADR-0026](ADR-0026-owned-native-keyboard-and-text.md) | Owned native keyboard and text input | R1V/R4 |

Use the [domain glossary](../../CONTEXT.md) for shared model terms. The
[port plan](../rust-port/plan.md#architecture-decisions-and-implementation-evidence)
assigns remaining proofs. Dependency selection and runtime evidence do not
change an accepted decision silently; a changed decision needs a superseding
record and an updated boundary index.
