# Authored offset-plane expressions

Tracking: [#3745](https://github.com/Oblikovati/Oblikovati/issues/3745).
This work implements the scalar boundary of ADR-0005 and ADR-0006. It uses the
[feature ownership](implementation-scalar-ownership.md) and
[document ownership](implementation-document-scalars.md) checks.

## Source and units

Each created offset owns one Model parameter with role `offset`. Its source text
and drivers remain in the document parameter graph. A plane definition reads
that parameter and its base plane. The API uses its existing typed contracts;
there is no new wire DTO, callback, evaluator, or parameter graph.

A typed scalar captures its numeric category and bare-input unit in version 1
expression context. A bare number uses the display unit at authoring time.
Formula operands remain unitless. Explicit units override the captured unit.
Changing display preferences does not change an existing source's physical value.
An explicit scalar edit captures the current input unit. Reopen and User copies
retain the original context. Working-scale conversion stays in UnitsOfMeasure.
Context metadata requires document schema 3 and rejects unknown fields, versions,
categories, incompatible units, and nonnumeric or read-only records on restore.
Ordinary parameters omit this field and retain their existing recipe layout.

## Editing and transaction ownership

Offset creation and redefine use the file-owned command transaction. Redefine
keeps the feature key and existing owned parameter ID. Scalar and base-reference
edits are applied together. A rejected candidate cannot publish its reply or
leave partial source changes. Origins are immutable; placed-plane indices include
construction planes even when a list hides them. Duplicate or invalid slots,
wrong reference kinds, self/forward dependencies, and unsupported display edits
produce explicit errors. An empty edit reads the current result without a new
document generation.

The first scalar edit of a legacy constant or unowned parameter creates an owned
Model parameter. It does not overwrite the old external driver. Later driver
failures retain authored source, report local health, and expose no stale plane
coordinates. Undo, redo, grouped abort, commit veto, source copy, and reopen use
the existing history and storage boundaries. Aborted creation retires both IDs.

## Validation scope

Focused Linux tests cover live formulas, captured units, rename/copy, strict
restore, failed-source health, atomic repick, ownership promotion, permanent IDs,
undo/redo, group abort, veto, and file copy/reopen. The existing Go regression
`TestModelLengthClosureStaysLiveForParamExpr` passes at the source baseline and
confirms live driver changes and the centimetre value of an explicit millimetre
literal. The Rust tests also cover persistent expression edits, which the Go
redefine implementation previously froze to evaluated quantities.

The local Linux `make gate` passes with each crate above 80% line coverage and
0.31% duplication. The gate derives the workspace inventory from Cargo metadata
and includes formatting, Clippy, build, tests/coverage, doctests, documentation,
workflow lint, and SPDX checks. No native viewport or B-rep acceptance is
claimed. Fixed-plane scalar edits and display rectangle/associativity flags remain
explicitly unsupported. Kernel release pins and Windows/macOS workflows are
unchanged.
