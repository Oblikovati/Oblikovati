---
milestone: M02
feature: F03
name: Parameter Model
status: planned
---

# M02 · F03 — Parameter Model

The `Parameter` object family — model, user, reference, derived, and table parameters — each carrying expression, value (db units), model value (post-tolerance), units, tolerance, precision, and display format.

## In scope

- `Parameter` base + type-specific behavior.
- `Parameters`/`ModelParameters`/`UserParameters` collections.
- `Tolerance`, `Precision`, `DisplayFormat`, `ExposedAsProperty`.

## Out of scope

_None._

## Key API contracts delivered

- `Parameter`,`ModelParameter`,`UserParameter`,`ReferenceParameter`,`DerivedParameter`,`TableParameter`
- `Parameters`,`ModelParameters`,`UserParameters`
- `ParameterTypeEnum`,`Tolerance`,`ParameterDisplayFormatEnum`

## Depends on

F01,F02.

## Backlog items

| PBI | Title |
|-----|-------|
| [PBI-027](PBI-027-parameter-base.md) | Parameter base: expression/value/model-value/units |
| [PBI-028](PBI-028-parameter-types.md) | Parameter types & collections (model/user/reference/derived/table) |
| [PBI-029](PBI-029-tolerance-precision.md) | Tolerance, precision & display format |
