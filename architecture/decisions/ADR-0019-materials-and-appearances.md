# ADR-0019: Materials and appearances

Date: 2026-10-12. Status: Accepted for the Rust target design.
Supersedes [ADR-0010](ADR-0010-materials-and-appearances.md) in full.
Implementation evidence: partial typed assets, catalog, and API reads; R5/R7 remain open.

Boundary: physical asset state, visual asset state, and assignment resolution.
Owner: application model; public values are owned by Oblikovati.API.

## Context

A physical material and a rendered appearance are different properties.
Documents must retain both when moved away from a project library. Legacy
records also contain a mesh-based mass calculation that conflicts with the
current analytic-kernel requirements. The Go assignment regression for issue #1103
also contradicts ADR-0010's stated order: an explicit part appearance must win
over the appearance implied by an assigned material. Keep that verified user
choice in Rust; assigning a physical material must not disable visual editing.

## Decision

Keep typed material and appearance records with stable asset IDs. Import
public groups and enum values from the API. Physical properties include density,
mechanical, thermal, and electrical data with their units and provenance.
An appearance controls the rendered result and does not alter density or mass.

Resolve asset definitions in document, project, then built-in scope. Embed
non-built-in definitions used by a document so it does not silently acquire
different properties when a project library changes. Resolve assignments by
persistent reference and occurrence context. Preserve the baseline precedence:
face appearance override, body appearance override, explicit part appearance,
then the effective material's appearance, then the defined neutral default.
The effective material is the body assignment if present, otherwise the part
material. Absence of a material appearance permits the neutral default.
An explicit missing asset is an error, not an invitation to try lower priority
assignments. The renderer receives the resolved value or the named failure.

Body and face assignments are keyed by persistent reference plus occurrence
path, within one owning document. Equal reference text in separate occurrences
must not share an override. A face query must use its body's occurrence context;
reference ownership and face/body incidence are verified at the host's topology
boundary before storing or rendering assignments. Lost references are reported.
Only this model resolver applies precedence. Inventory additional assembly
occurrence policies before their implementation.

Ship built-in catalogs as embedded, reviewable asset files in a fixed order.
Preserve IDs, source citations, units, and any stated derivation of values.
Validate catalog shape and required assets during tests/build. A malformed
catalog must not become a runtime panic or silently substituted property.
Changes to numeric material data require evidence, not guessed nonzero values.

Retain isotropic, orthotropic, and transversely isotropic property forms and
their declared principal axes. Keep the nine independent orthotropic elastic
constants and directional thermal expansion. Validate consistency, finite
values, and physically valid stiffness. Do not infer directional strength,
plasticity, or complete simulation support from the presence of elastic data.

Retain the existing appearance surface alongside OpenPBR appearance values.
Resolve supported old appearances through a tested conversion that preserves
their intended meaning; do not delete their API or stored form during this
port. OpenPBR parameter names, ranges, defaults, and wire representation stay
API-owned. Rendering compliance and hardware selection belong to
[ADR-0008](ADR-0008-vulkan-renderer.md).

## Interfaces and invariants

- Compute volume, area, centroid, and inertia from Martensite's analytic B-rep
  integration. Combine them with compatible density units. Display tessellation
  is never the source of physical mass properties.
- Assignment references obey [ADR-0005](ADR-0005-identity-and-references.md).
  A lost face/body assignment is reported; it cannot silently move to another.
- Asset changes invalidate the appropriate presentation/analysis revisions,
  without rebuilding unrelated feature geometry.
- Catalog embedding uses the storage codec and resource rules in
  [ADR-0003](ADR-0003-documents-and-storage.md). No private public DTO copies
  or parallel appearance-resolution code enter the renderer.

## Alternatives and consequences

Generic untyped asset bags would weaken validation and duplicate API schema
knowledge. Computing mass from display meshes would make a physical value
depend on visual quality. Reject both. Checking material appearance before an explicit part appearance
would recreate Go #1103, making a user's later visual choice ineffective.
Selecting a lower-priority asset after a missing explicit reference would hide
loss of intent. Refuse that silent substitution and preserve the failed assignment
for diagnosis and repair. Typed assets require migrations when
the contract changes; embedded copies increase file size but preserve intent.

## Legacy disposition

Consolidate Oblikovati ADR-0022, 0024, 0025, and the application asset portion
of 0053 at the [fixed revision](legacy-map.md#source-baseline). Replace
`go:embed`, startup panic, mesh mass properties, and obsolete texture deferral
assumptions with Rust asset validation and current resource/render interfaces.
Keep sourced anisotropy and distinguish measured values from derived ratios.

## Validation required

Use the baseline catalog, material recipes, assignment tests, and API types.
Go `model/material/assignment.go` and `assignment_test.go` at
`aa8297d1b0ac33c024f19e54821536d76e8a447a` prove the #1103 order; Rust tests must
include body material plus explicit part appearance, clear-to-material behavior,
separate occurrence paths, and named failures for missing explicit assets.
Test scope/override precedence, repeated occurrences, missing assets, every
non-default field, anisotropic validity, and old appearance conversion.
Compare analytic mass properties with independent solids at multiple working
scales. Verify saved documents after project assets and source files are absent.
