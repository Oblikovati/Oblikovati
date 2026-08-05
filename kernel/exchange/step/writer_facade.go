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
	// The product structure comes first so the file reads top-down, and because the geometric
	// representation context it emits carries the unit declaration the geometry is expressed in.
	ps := emitProductStructure(w, opts.Name, opts.FileUnit)
	shapes := make([]int, 0, len(bodies))
	solids := true
	for _, body := range bodies {
		id, err := topomap.BodyToStep(emit, body)
		if err != nil {
			return nil, nil, err
		}
		shapes = append(shapes, id)
		solids = solids && body.IsSolid()
	}
	// Without this the DATA section is unreachable geometry: a conformant reader enters at
	// SHAPE_DEFINITION_REPRESENTATION and would find nothing to show (#2055).
	emitShapeRepresentation(w, ps, shapes, solids)
	return w.Emit(ap203Header()), mixedBodyWarnings(bodies), nil
}

// mixedBodyWarnings reports a body set that mixes solids and open shells. One shape
// representation cannot hold both — ADVANCED_BREP_SHAPE_REPRESENTATION takes solid b-reps and
// MANIFOLD_SURFACE_SHAPE_REPRESENTATION takes surface models — so the file declares the surface
// form, which readers accept for both, and the user is told the distinction was flattened.
func mixedBodyWarnings(bodies []*topo.Body) []string {
	solids, shells := 0, 0
	for _, b := range bodies {
		if b.IsSolid() {
			solids++
			continue
		}
		shells++
	}
	if solids > 0 && shells > 0 {
		return []string{fmt.Sprintf(
			"step export: %d solid and %d surface bodies share one shape representation; "+
				"the file declares the surface form, which readers accept for both", solids, shells)}
	}
	return nil
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
