# ADR-0033: Independent driving dimension authoring

Status: Accepted for the Rust target. Date: 2026-11-06.
Tracking: [#3824](https://github.com/Oblikovati/Oblikovati/issues/3824).

## Context

Live native angle creation on a fully constrained rectangle exposed a gap in
[ADR-0032](ADR-0032-converged-dimension-authoring.md): numerical convergence
alone can accept a redundant driving equation and publish an unhealthy sketch.

## Decision

This record supersedes ADR-0032's available-target acceptance condition.
Available dimension targets require numerical convergence. The default policy
also requires zero redundant equations before adoption. An explicit document
`Apply Driving` setting keeps the Go contract: a consistent redundant equation
can remain driving, with its redundancy diagnostics retained. Reuse the structure from the current solve
report. Do not compute a second rank decision. A refusal retains the original
sketch and parameters and names the redundant equation count. Permanent
reservations stay retired. The UI retains its draft so the user can choose a
reference dimension explicitly and retry through the same typed API operation.

Explicit API roles remain explicit. Placement's existing reference policy still
selects the role before solving. Reported unavailable-formula authorship and
repair remain unchanged. This deletes the successful adoption path for redundant
available driving targets; it adds no geometry recognizer or fallback.

## Evidence

A fixed-point distance regression reproduces success before this change and
requires atomic refusal afterwards. Typed API tests cover Part and Assembly.
Live mouse input must reject a redundant rectangle angle, retain the editor,
and accept the explicit Reference correction with healthy DOF zero.
