// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"oblikovati/kernel/exchange"
	"oblikovati/kernel/exchange/step/geommap"
	"oblikovati/kernel/exchange/step/part21"
	"oblikovati/kernel/exchange/step/schema"
	"oblikovati/kernel/exchange/step/topomap"
	"oblikovati/kernel/topo"
)

// Writer exports kernel bodies as a minimal valid AP203 (CONFIG_CONTROL_DESIGN)
// STEP file. It satisfies exchange.BodyExporter. The emitted file re-imports
// without degradation (the PBI-C round-trip invariant). The zero value is usable.
//
// Example:
//
//	data, warns, err := step.Writer{}.ExportSolids([]*topo.Body{b}, exchange.TranslationOptions{})
type Writer struct{}

// compile-time assertion that Writer is a BodyExporter.
var _ exchange.BodyExporter = Writer{}

// ExportSolids writes the bodies as one AP203 file. Lengths are emitted in mm (the
// kernel's database unit) so the file declares a millimeter unit. A body with a
// surface type that cannot be exported yields an error (tessellated fallback is
// PBI-E); the geometry that does export is byte-stable for a given body.
func (Writer) ExportSolids(bodies []*topo.Body, opts exchange.TranslationOptions) ([]byte, []string, error) {
	w := part21.NewWriter()
	emit := geommap.NewEmitter(w, 1.0) // 1 file unit = 1 mm
	emitUnitContext(w)
	for _, body := range bodies {
		if _, err := topomap.BodyToStep(emit, body); err != nil {
			return nil, nil, err
		}
	}
	return w.Emit(ap203Header()), nil, nil
}

// emitUnitContext emits the millimeter SI length unit + a unit-assigned context, so
// the file's lengths are unambiguous on re-import.
func emitUnitContext(w *part21.Writer) int {
	mm := w.AddRaw("(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.))")
	return w.Add("GLOBAL_UNIT_ASSIGNED_CONTEXT", part21.FormatList(part21.Ref(mm)))
}

// ap203Header builds the HEADER for an exported AP203 file.
func ap203Header() part21.Header {
	id, _ := schema.SchemaIdentifier(schema.AP203)
	return part21.Header{
		Description:         []string{"Oblikovati STEP export"},
		ImplementationLevel: "2;1",
		Name:                "oblikovati.step",
		PreprocessorVersion: "Oblikovati",
		OriginatingSystem:   "Oblikovati",
		SchemaIdentifiers:   []string{id},
	}
}
