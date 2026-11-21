# ADR-0032: Converged dimension authoring with available targets

Date: 2026-11-06. Status: Accepted for the Rust target design.
Tracking: [#3824](https://github.com/Oblikovati/Oblikovati/issues/3824).

## Decision

A dimension constructor evaluates its isolated candidate through the existing
solver. If all target sources are available, the candidate must pass the existing
`require_converged_sketch` check before its parameters or sketch are adopted.
DOF zero alone does not prove that a dimension agrees with fixed coordinates.
A typed readiness cause reaches the API and the native editor.

This supersedes numerical-success-by-DOF for dimension creation. Delete that
permissive adoption path; retain the incompatible fixed-point input as a refusal
regression. Preserve geometry, parameters and history on refusal. Attempted
permanent names and identities stay retired. No solver nudge, changed tolerance,
second constructor or UI undo-after-adoption path is added.

Placement locks classify their document redundancy policy on the isolated
appended relation before the solve. Delete the later driving-to-reference repair
and its second solve. Explicit API requests retain their requested role. Both
entry points use the same creation pipeline and convergence post-condition.

The existing unavailable-source authoring contract remains separate: its report
names unavailable targets, sketch health stays failed, and later binding can
repair the document. Do not erase this evidence or invent zero targets. Explicit
parameter edits, recompute and persisted sick-document recovery keep their
existing contracts. The native value editor checks its expression with the shared
scalar evaluator before requesting creation.

## Evidence

Model and typed API tests must retain the incompatible fixed-point case and an
exhausted iteration budget. Refusal must leave geometry and parameters unchanged
and must not add undo history. Valid reference creation must still succeed after
a refusal with fresh permanent identities. Existing unavailable-source and repair
checks remain required. The native editor must retain its draft after failure.

The ordinary creation path is the only numerical path. No geometry recognizer,
tolerance constant, fallback or type assertion is added.
