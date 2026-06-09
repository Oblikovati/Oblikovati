// SPDX-License-Identifier: GPL-2.0-only

// Package exchange owns the kernel-side seam for foreign-format (STEP, IGES, …)
// import/export. It declares the interfaces a concrete translator satisfies so the
// rest of the kernel/model depends on the abstraction, not on a parser. The STEP
// implementation lives under exchange/step and is wired in by the model layer (the
// contract/wire/CLI surface is PBI-D, deferred). See ADR-0018 and the M17 plan.
package exchange

import "oblikovati.org/kernel/topo"

// TranslationOptions tunes a translation independent of the concrete format. The
// zero value is valid: a millimeter target, default chord tolerance, no warnings
// collected beyond the importer's own.
//
// Example:
//
//	opts := TranslationOptions{TargetUnitMM: 1, ChordTolerance: 1e-3}
//	body, warns, err := imp.ImportSolids(data, opts)
type TranslationOptions struct {
	// TargetUnitMM is the database-unit size in millimeters (1 ⇒ the kernel works
	// in mm). Imported lengths are scaled from the file's unit into this unit.
	TargetUnitMM float64
	// ChordTolerance bounds curve/surface faceting used by validation/mass props.
	ChordTolerance float64
}

// BodyImporter turns foreign-format bytes into kernel bodies. Implementations
// MUST NOT panic on malformed input; they return an error whose message names the
// offending token/entity. Non-fatal issues (an unsupported surface that fell back
// to a coarser representation) are returned as warnings, not errors.
type BodyImporter interface {
	// ImportSolids parses data and returns every solid/surface body it found,
	// the collected non-fatal warnings, and a fatal error (nil on success).
	ImportSolids(data []byte, opts TranslationOptions) ([]*topo.Body, []string, error)
}

// BodyExporter serializes kernel bodies into foreign-format bytes.
type BodyExporter interface {
	// ExportSolids writes the given bodies as one file's bytes plus any warnings.
	ExportSolids(bodies []*topo.Body, opts TranslationOptions) ([]byte, []string, error)
}
