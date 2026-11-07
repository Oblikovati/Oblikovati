# ADR-0039: One value editor for placed sketch dimensions

Status: Proposed for the Rust target. Date: 2026-11-07.

## Context

Go `app/sketch_dimension_tool.go` commits the measured constraint when the user
places its label. It then opens that constraint in the value editor. Cancelling
the value proposal retains the placed constraint and its measured expression.

The Rust host currently keeps an uncommitted `GuiDimensionDraft` until Apply.
Its separate `GuiDimensionEditor` drops the whole proposal on Escape. Existing
labels use `GuiDimensionValueEditor` instead. This conflicts with the benchmark's
`features/sketch/ellipse/scenario.feature` scenarios 13 through 18, particularly
scenario 15: Escape cancels the proposed minor-radius value but retains the new
dimension and its original measured expression.

The earlier [#3823](https://github.com/Oblikovati/Oblikovati/issues/3823)
acceptance required deferred creation until expression acceptance and cancellation
without creating a dimension. This record supersedes that timing policy to follow
the current user-selected BDD benchmark; the historical requirement is not treated
as the final product behavior.

## Decision

An explicit placement click commits one measured constraint through the existing
typed dimension API and its normal validity and independent-equation gates.
The existing value editor then proposes changes to that owned constraint. A new
label and an existing label use the same value-edit operation and UI component.
Escape discards the proposed value; it does not remove the placed constraint.

After accepting or cancelling a value proposal, General Dimension is ready for
another selection and placement. Escape from that selection state exits the
command. Owned constraint identity and the current sketch generation must remain
valid at each transition.

Creation and value acceptance remain separate history operations. Undo after an
accepted value restores the measured value and its annotation; the next Undo
removes the placement. Redo replays those operations in order. Failed value edits
retain the owned constraint, its expression and its geometry. They display the
named refusal and allow correction or cancellation.

A refused initial driving placement must not publish an unhealthy constraint.
The same editor retains the proposed operands and measured expression, presents
the refusal, and permits an explicit Reference correction through the existing
API role. This preserves ADR-0033's independent-equation acceptance condition.

Completion deletes the separate `GuiDimensionEditor` and its duplicate buffer.
It also deletes the successful-placement path that waits for a value acceptance
before creating any constraint. It adds no solver, geometry algorithm, tolerance,
or private geometry fallback. Canonical ellipse axis kinds remain API-owned.

## Verification

Owner input tests cover selection before placement, measured placement, value
acceptance and Escape, failure preservation, explicit reference correction, and
ordered Undo/Redo. Existing circle, arc, line and two-point dimension journeys
must retain their measured quantities and validity.

Native mouse and keyboard runs cover the external ellipse dimension scenarios,
including selected initial expressions, retained annotations after Escape,
reopened expressions, zero-radius refusal, and Finish/Edit Sketch visibility.
Fixtures are independent where the source describes a fresh fixture. Tests must
not introduce undocumented center motion to reproduce a recorded value.
