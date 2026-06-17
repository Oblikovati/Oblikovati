// SPDX-License-Identifier: GPL-2.0-only

// Package param is the parametric variable layer: dimensioned quantities in
// canonical database units, a unit-aware expression engine, the Parameter model
// (the Expression→Value→ModelValue triad), and the dependency graph that drives
// recompute. It is pure Go, cgo-free, and depends on neither the kernel nor the
// renderer (architecture/core/04-parameters-expressions.md).
//
// Everything lives in one package on purpose: Parameter holds an [Expr], Expr
// evaluates against parameter values, and the [Graph] binds expression
// references to stable parameter ids — splitting these across packages would
// create an import cycle. Files are organized by responsibility instead:
// unit/quantity, the expr_* engine, parameter, and graph.
//
// Two rules from the architecture shape the whole design:
//   - every numeric value is a [Quantity] stored in database units (cm, radians);
//     conversion and formatting happen only at the parse/display boundary
//     ([UnitsOfMeasure]).
//   - expression references bind by stable [ID], not display text, so
//     renaming a parameter is a relabel — never a global string rewrite.
//
// Unit system, end to end (Oblikovati/Oblikovati#146): the database length unit
// is the CENTIMETRE — the same unit the geometry kernel's coordinates use, so a
// Quantity value, a sketch coordinate, and a topo vertex all share one scale (no
// hidden conversion between the parameter and geometry layers). Display units are
// a presentation choice on top ([UnitsOfMeasure] turns 6 cm into "60 mm" or
// "2.36 in"). Foreign-format exchange is the only place lengths leave this unit:
// kernel/exchange scales database centimetres to/from a file's declared unit via
// exchange.TranslationOptions (TargetUnitMM = exchange.DBUnitMM = 10 mm), and an
// export writes the document's preferred unit.
package param
