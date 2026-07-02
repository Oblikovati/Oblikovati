// SPDX-License-Identifier: GPL-2.0-only

// Package exchange owns the kernel-side seam for foreign-format import/export. It
// declares the interfaces a concrete translator satisfies — [BodyImporter] and
// [BodyExporter], tuned by a [TranslationOptions] — so the rest of the kernel/model
// depends on the abstraction, not on a parser; concrete codecs are wired in by the
// model layer (model/exchange). See ADR-0018 and the M17 plan.
//
// Concrete format implementations live in the subpackages: step (STEP B-rep),
// dwg and dxf (2D drawing exchange), pdf (vector plot import), meshio (STL/OBJ/3MF
// meshes), plyfmt/e57fmt/lasfmt (point clouds), and drawing (drawing-sheet export).
//
// The one unit invariant: translators scale through
// [TranslationOptions.TargetUnitMM], anchored by [DBUnitMM] — the kernel's
// geometry is centimetres, so one database unit is 10 mm (#146); this package is
// the single place the model↔file unit scale is anchored.
package exchange
