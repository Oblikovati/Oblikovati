# ADR-0025: Typed sketch spline authoring through the existing owner

Date: 2026-10-14. Status: Accepted for the Rust target design.
Boundary: canonical sketch entity and dimension requests to owned sketch recipes.
Tracking: [#3789](https://github.com/Oblikovati/Oblikovati/issues/3789).

## Decision

Use the existing Apache API `sketch.addEntity` and `sketch.addDimension` DTOs,
kind constants, and typed client. No new public shape or raw JSON path is needed.
Delete the current host declines for fit/control/offset spline creation and
`offsetSplineDim` when each typed route passes owner and native acceptance.

Classify a spline constructor once from its kind and variant. Resolve `pointExprs`
before literal points, as for other sketch entities. Retain the API fit method,
closure, authored point order, and construction role in a typed input. A fit
parent owns newly allocated defining points. A control parent has the same point
ownership with control intent. A derived offset owns one parent spline identity
and one signed working length; it creates no defining points.

Append these inputs through the existing primitive command and isolated snapshot
candidate. Validate all values and references before reserving permanent IDs.
Return the existing entity and point ID result. Do not add a second transaction,
history, naming, or persistence path. The released kernel constructor resolves
the accepted recipe later through ADR-0024; no sampled curve is stored.

Bind `offsetSplineDim` through the existing seeded dimension command. Its source
is a length expression, and the model captures the offset's current side before
the solver drives its magnitude. A zero seed still requires explicit side
selection. General dimension edit, delete, reference, annotation, history and
save/reopen behavior applies to this relation.

## Acceptance

Use the pinned Go application at `aa8297d1b0ac33c024f19e54821536d76e8a447a`
for constructor and dimension behavior. Check both part and assembly owners,
negative and positive side, fit methods, closure, unit-bearing input, invalid
references, source edits, undo/redo, save/reopen, copy, and native display.
The full local gate and native pixel check must pass before merging this work.

This decision adds no geometry recognizer, tolerance constant, fallback site or
geometry type assertion. It removes host declines only as their routes pass.
