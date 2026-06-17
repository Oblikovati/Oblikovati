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
	// in mm; 10 ⇒ centimetres). Imported lengths are scaled from the file's unit
	// into this unit; exported lengths are scaled from it into the file unit.
	TargetUnitMM float64
	// FileUnit is the length unit an EXPORT writes (and declares) — "mm", "cm",
	// "m", "in" or "ft". Empty ⇒ millimetres. It is ignored on import, where the
	// unit is read from the file.
	FileUnit string
	// ChordTolerance bounds curve/surface faceting used by validation/mass props.
	ChordTolerance float64
}

// DBUnitMM is the database-unit size in millimetres the model works in: the
// kernel's geometry is centimetres, so one database unit is 10 mm. Translators
// receive this via [TranslationOptions.TargetUnitMM]; it is the single place the
// model↔file unit scale is anchored (Oblikovati/Oblikovati#146).
const DBUnitMM = 10.0

// mmPerFileUnit is the millimetre size of a named export unit. It is the codec's
// own small unit vocabulary so the kernel exchange need not depend on the model's
// parameter engine. Unknown/empty names fall back to millimetres.
var mmPerFileUnit = map[string]float64{
	"mm": 1, "cm": 10, "m": 1000, "in": 25.4, "ft": 304.8,
}

// FileUnitMM returns the millimetre size of opts.FileUnit (millimetres when unset
// or unknown), plus whether the name was recognised.
func (o TranslationOptions) FileUnitMM() (float64, bool) {
	if o.FileUnit == "" {
		return 1, true
	}
	mm, ok := mmPerFileUnit[o.FileUnit]
	if !ok {
		return 1, false
	}
	return mm, true
}

// dbUnitMM returns the effective database-unit size in millimetres, defaulting a
// zero TargetUnitMM to 1 (the historical millimetre kernel) so existing direct
// callers are unaffected.
func (o TranslationOptions) dbUnitMM() float64 {
	if o.TargetUnitMM == 0 {
		return 1
	}
	return o.TargetUnitMM
}

// ImportScale is the file→database length factor: a length in the file's unit
// times this yields database units. The reader resolves the file unit (mm per
// file unit) and divides by the database unit size.
func (o TranslationOptions) ImportScale(mmPerFileUnit float64) float64 {
	return mmPerFileUnit / o.dbUnitMM()
}

// ExportScale is the database→file length factor: a database-unit length times
// this yields a value in opts.FileUnit.
func (o TranslationOptions) ExportScale() float64 {
	mm, _ := o.FileUnitMM()
	return o.dbUnitMM() / mm
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
