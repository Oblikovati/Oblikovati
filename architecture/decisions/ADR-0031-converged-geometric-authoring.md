# ADR-0031: Converged geometric constraint authoring

Date: 2026-11-05. Status: Accepted for the Rust target design.
Boundary: adoption of a newly created geometric sketch constraint.
Tracking: [#3818](https://github.com/Oblikovati/Oblikovati/issues/3818).

## Decision

A geometric constraint constructor must evaluate its isolated candidate and
reuse `require_converged_sketch` before adoption. DOF measures the remaining
freedom; it does not certify convergence. A numerical result with DOF zero can
still violate a fixed coordinate or another constraint.

This supersedes the numerical-failure adoption policy recorded in the
[constraint creation ledger](../rust-port/implementation-constraint-creation.md).
Delete the acceptance test that treats an unconverged authored candidate as a
successful construction. Retain its input as a refusal corpus case. There is no
second permissive constructor or UI-specific undo-after-failure path.

The typed readiness cause reaches the session, API and UI. A refusal preserves
coordinates, evaluation state, generation and undo history. Attempted identity
reservations remain retired. Free, converged sketches remain valid authoring
results; this decision does not require DOF zero for each intermediate edit.

[ADR-0006](ADR-0006-parameters-and-constraints.md) remains the solver authority.
Do not nudge a seed, alter a tolerance or hide a solver failure to make a request
succeed. A singular orientation seed can receive a named convergence refusal.
General solver improvements must retain those cases and prove their branch
choice. Explicit recompute, constraint deletion and persisted sick-document
recovery keep their separate contracts.

## Evidence required

Test an incompatible relation between fixed points and an exhausted iteration
budget. Assert refusal before adoption through the model and typed API, including
part and assembly owners and monotonic reservations. Keep the earlier parallel
line seed that stalls a perpendicular constructor as a refusal corpus case.
Normal, converged constructors must retain their canonical kinds and DOF.
Native constraint tools must keep their picks and prompt after a failed commit.
The full local gate remains required before publication.

No geometry recognizer, tolerance constant, fallback or type assertion is added.
The single existing readiness implementation remains authoritative.
