// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"fmt"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step/geommap"
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/exchange/step/schema"
	"oblikovati.org/kernel/exchange/step/topomap"
	"oblikovati.org/kernel/topo"
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

// ExportSolids writes the bodies as one AP203 file in opts.FileUnit (millimetres
// when unset), scaling the kernel's centimetre geometry into that unit and
// declaring it so the lengths are unambiguous on re-import. A body with a surface
// type that cannot be exported yields an error (tessellated fallback is PBI-E);
// the geometry that does export is byte-stable for a given body and unit.
func (Writer) ExportSolids(bodies []*topo.Body, opts exchange.TranslationOptions) ([]byte, []string, error) {
	if _, ok := opts.FileUnitMM(); !ok {
		return nil, nil, fmt.Errorf("step export: unknown file unit %q", opts.FileUnit)
	}
	w := part21.NewWriter()
	emit := geommap.NewEmitter(w, opts.ExportScale()) // database-unit (cm) → file unit
	emitUnitContext(w, opts.FileUnit)
	for _, body := range bodies {
		if _, err := topomap.BodyToStep(emit, body); err != nil {
			return nil, nil, err
		}
	}
	return w.Emit(ap203Header()), nil, nil
}

// emitUnitContext emits the file's SI (mm/cm/m) or conversion-based (in/ft) length
// unit plus a unit-assigned context, so the file's lengths are unambiguous on
// re-import. An empty unit defaults to millimetres. The reader locates the unit by
// scanning for GLOBAL_UNIT_ASSIGNED_CONTEXT (units.go), so the context id needs no
// further reference here.
func emitUnitContext(w *part21.Writer, fileUnit string) {
	w.Add("GLOBAL_UNIT_ASSIGNED_CONTEXT", part21.FormatList(part21.Ref(lengthUnitRef(w, fileUnit))))
}

// lengthUnitRef emits the LENGTH_UNIT entity for fileUnit and returns its id.
func lengthUnitRef(w *part21.Writer, fileUnit string) int {
	switch fileUnit {
	case "cm":
		return w.AddRaw("(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.CENTI.,.METRE.))")
	case "m":
		return w.AddRaw("(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT($,.METRE.))")
	case "in":
		return conversionLengthUnit(w, "INCH", 0.0254)
	case "ft":
		return conversionLengthUnit(w, "FOOT", 0.3048)
	default: // "" or "mm"
		return w.AddRaw("(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT(.MILLI.,.METRE.))")
	}
}

// conversionLengthUnit emits a CONVERSION_BASED_UNIT of metresPerUnit metres (an
// imperial length), returning its id. The reader's units_conversion.go parses it.
func conversionLengthUnit(w *part21.Writer, name string, metresPerUnit float64) int {
	dim := w.AddRaw("DIMENSIONAL_EXPONENTS(1.,0.,0.,0.,0.,0.,0.)")
	metre := w.AddRaw("(LENGTH_UNIT()NAMED_UNIT(*)SI_UNIT($,.METRE.))")
	measure := w.AddRaw("LENGTH_MEASURE_WITH_UNIT(LENGTH_MEASURE(" +
		part21.FormatReal(metresPerUnit) + ")," + part21.Ref(metre) + ")")
	return w.AddRaw("(CONVERSION_BASED_UNIT(" + part21.QuoteString(name) + "," +
		part21.Ref(measure) + ")LENGTH_UNIT()NAMED_UNIT(" + part21.Ref(dim) + "))")
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
