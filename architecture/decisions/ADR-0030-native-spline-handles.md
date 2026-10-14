# ADR-0030: Native fit-spline handles through the owned sketch pipeline

Date: 2026-10-15. Status: Accepted for the Rust target design.
Boundary: planar fit-spline handles and `sketch.setSplineHandle`.
Tracking: Oblikovati#626.

## Decision

An active handle is owned sketch geometry. It has a stable entity identity, one
fit-parent identity, one fit-point index, and one owned end point. Its anchor is
the parent defining point. The end point uses the existing coordinate variables,
constraints, selection, deletion, history, naming, and recipe path.

The released native spline constructor evaluates the parent recipe and applies
active handles as tangent direction and weight. It does not sample a curve or
construct a second spline engine.

## Consequences

This removes the Rust `sketch.setSplineHandle` missing-route decline. It deletes
the assumption that parent spline points are the only spline-owned geometry. It
adds an explicit recipe version for handles. Activation, edit, deactivation,
history, persistence, API inspection, and native pixels are acceptance criteria.
