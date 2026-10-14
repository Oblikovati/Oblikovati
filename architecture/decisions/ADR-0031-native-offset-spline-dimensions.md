# ADR-0031: Native offset-spline distance through the owned sketch pipeline

Date: 2026-10-15. Status: Accepted for the Rust target design.
Boundary: planar offset splines and `offsetSplineDim`.
Tracking: Oblikovati#3789, Oblikovati#1874.

## Decision

An offset spline owns one stable entity identity, references exactly one parent
spline, and stores one finite signed working-distance variable. The sign is the
selected offset side. A distance dimension measures the magnitude of that
variable and retains the selected side while a driving expression changes its
magnitude.

The normal sketch variable, constraint, solver, history, persistence, and
reference-key paths own this state. The native planar curve pipeline resolves
the parent spline once and constructs its analytic offset from the signed
distance. Display tessellation consumes that resolved curve only.

## Consequences

This removes the `offsetSplineDim` decline at the API boundary. It deletes any
separate sampled-polyline offset path. Parent identity, side metadata,
parameters, annotations, undo, veto, save/reopen/copy, and both document
owners are acceptance criteria. A degenerate or unsupported native offset is a
named curve failure; it does not produce substitute geometry.
