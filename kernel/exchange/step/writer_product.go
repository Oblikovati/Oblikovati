// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"oblikovati.org/kernel/exchange/step/part21"
)

// The AP203 product structure and shape representation — the entry point every conformant STEP
// reader walks to find geometry (#2055).
//
// The exported file used to be a DATA section of raw geometry ending at MANIFOLD_SOLID_BREP,
// with no APPLICATION_CONTEXT, no PRODUCT and no shape representation. Our own reader scans for
// the b-rep entity directly, so the round-trip test passed — but a reader that enters the file
// the standard way, from SHAPE_DEFINITION_REPRESENTATION down, found nothing to display and
// opened an empty model. That is the shape of "no geometry in the STEP file" (#2055).
//
// The chain a reader follows, and therefore the chain emitted here:
//
//	SHAPE_DEFINITION_REPRESENTATION
//	  → PRODUCT_DEFINITION_SHAPE → PRODUCT_DEFINITION → … → PRODUCT
//	  → ADVANCED_BREP_SHAPE_REPRESENTATION (in a GEOMETRIC_REPRESENTATION_CONTEXT)
//	      → MANIFOLD_SOLID_BREP / SHELL_BASED_SURFACE_MODEL

// productStructure holds the ids the shape representation needs to reference.
type productStructure struct {
	definitionShape int // PRODUCT_DEFINITION_SHAPE — what the representation is OF
	context         int // the geometric representation context the shapes live in
}

// emitProductStructure writes the AP203 product/application context chain and the geometric
// representation context, returning the ids the shape representation binds to. name labels the
// product; an empty name becomes "Oblikovati part" so a reader's tree shows something.
func emitProductStructure(w *part21.Writer, name, fileUnit string) productStructure {
	if name == "" {
		name = defaultProductName
	}
	appCtx := w.Add("APPLICATION_CONTEXT", part21.QuoteString(ap203ContextName))
	w.Add("APPLICATION_PROTOCOL_DEFINITION", part21.QuoteString("international standard"),
		part21.QuoteString("config_control_design"), "1994", part21.Ref(appCtx))

	prodCtx := w.Add("PRODUCT_CONTEXT", part21.QuoteString(""), part21.Ref(appCtx),
		part21.QuoteString("mechanical"))
	product := w.Add("PRODUCT", part21.QuoteString(name), part21.QuoteString(name),
		part21.QuoteString(""), part21.FormatList(part21.Ref(prodCtx)))
	formation := w.Add("PRODUCT_DEFINITION_FORMATION_WITH_SPECIFIED_SOURCE",
		part21.QuoteString(""), part21.QuoteString(""), part21.Ref(product), ".NOT_KNOWN.")
	defCtx := w.Add("PRODUCT_DEFINITION_CONTEXT", part21.QuoteString("part definition"),
		part21.Ref(appCtx), part21.QuoteString("design"))
	definition := w.Add("PRODUCT_DEFINITION", part21.QuoteString("design"),
		part21.QuoteString(""), part21.Ref(formation), part21.Ref(defCtx))
	defShape := w.Add("PRODUCT_DEFINITION_SHAPE", part21.QuoteString(""),
		part21.QuoteString(""), part21.Ref(definition))

	return productStructure{definitionShape: defShape, context: emitGeometricContext(w, fileUnit)}
}

// emitShapeRepresentation binds the emitted b-rep entities to the product, so a reader that
// starts at SHAPE_DEFINITION_REPRESENTATION reaches the geometry.
//
// solids selects ADVANCED_BREP_SHAPE_REPRESENTATION over MANIFOLD_SURFACE_SHAPE_REPRESENTATION:
// the first is for MANIFOLD_SOLID_BREPs, the second for SHELL_BASED_SURFACE_MODELs, and a reader
// rejects a representation whose items are of the wrong kind.
func emitShapeRepresentation(w *part21.Writer, ps productStructure, shapeIDs []int, solids bool) int {
	items := make([]string, 0, len(shapeIDs)+1)
	items = append(items, part21.Ref(originPlacement(w))) // the part's own coordinate frame
	for _, id := range shapeIDs {
		items = append(items, part21.Ref(id))
	}
	keyword := "MANIFOLD_SURFACE_SHAPE_REPRESENTATION"
	if solids {
		keyword = "ADVANCED_BREP_SHAPE_REPRESENTATION"
	}
	rep := w.Add(keyword, part21.QuoteString(""), part21.FormatList(items...), part21.Ref(ps.context))
	return w.Add("SHAPE_DEFINITION_REPRESENTATION", part21.Ref(ps.definitionShape), part21.Ref(rep))
}

// originPlacement emits the identity axis placement every shape representation carries as its
// first item — the frame the geometry is expressed in.
func originPlacement(w *part21.Writer) int {
	origin := w.AddShared("CARTESIAN_POINT", part21.QuoteString(""), "(0.,0.,0.)")
	z := w.AddShared("DIRECTION", part21.QuoteString(""), "(0.,0.,1.)")
	x := w.AddShared("DIRECTION", part21.QuoteString(""), "(1.,0.,0.)")
	return w.Add("AXIS2_PLACEMENT_3D", part21.QuoteString(""),
		part21.Ref(origin), part21.Ref(z), part21.Ref(x))
}

// emitGeometricContext emits the combined representation context the shape representation lives
// in: three dimensions, the length/angle/solid-angle units, and the modelling tolerance.
//
// It replaces the bare GLOBAL_UNIT_ASSIGNED_CONTEXT the writer used to emit. A representation
// must sit in a GEOMETRIC_REPRESENTATION_CONTEXT that also assigns units and an uncertainty; a
// standalone unit context satisfies neither the schema nor a reader. Our own reader already
// handles both forms — it scans the complex instance's GLOBAL_UNIT_ASSIGNED_CONTEXT component
// and picks the length-bearing unit out of the list (units.go) — so the round trip is unaffected.
func emitGeometricContext(w *part21.Writer, fileUnit string) int {
	length := lengthUnitRef(w, fileUnit)
	angle := w.AddRaw("(NAMED_UNIT(*)PLANE_ANGLE_UNIT()SI_UNIT($,.RADIAN.))")
	solid := w.AddRaw("(NAMED_UNIT(*)SI_UNIT($,.STERADIAN.)SOLID_ANGLE_UNIT())")
	tol := w.AddRaw("UNCERTAINTY_MEASURE_WITH_UNIT(LENGTH_MEASURE(" +
		part21.FormatReal(modelTolerance) + ")," + part21.Ref(length) +
		",'distance_accuracy_value','confusion accuracy')")
	return w.AddRaw("(GEOMETRIC_REPRESENTATION_CONTEXT(3)" +
		"GLOBAL_UNCERTAINTY_ASSIGNED_CONTEXT((" + part21.Ref(tol) + "))" +
		"GLOBAL_UNIT_ASSIGNED_CONTEXT((" + part21.Ref(length) + "," + part21.Ref(angle) +
		"," + part21.Ref(solid) + "))REPRESENTATION_CONTEXT('',''))")
}

const (
	// ap203ContextName is the APPLICATION_CONTEXT description AP203 prescribes.
	ap203ContextName = "configuration controlled 3d designs of mechanical parts and assemblies"
	// defaultProductName labels an exported part that carries no name of its own.
	defaultProductName = "Oblikovati part"
	// modelTolerance is the distance accuracy declared for the geometry, in FILE units. It is
	// the conventional 1e-6 relative value readers expect rather than a kernel tolerance.
	modelTolerance = 1e-6
)
