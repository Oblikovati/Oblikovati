// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/hlr"
	gmath "oblikovati.org/math"
)

// Retrieve model dimensions (#1991): pull the referenced part's existing parametric (sketch)
// dimensions onto a drawing view, instead of re-picking every dimension by hand. The host resolves
// the part's dimensions to world geometry (a name, its current value, and the two 3D endpoints it
// spans); this package projects them onto a view and materialises a drawing dimension flagged
// Retrieved, which re-measures with the model on recompute (associative to the parameter).

// ModelDimension is one of the referenced model's parametric dimensions resolved to world geometry:
// its parameter name, current value (model units, cm), and the two 3D endpoints it spans.
type ModelDimension struct {
	Name  string
	Value float64
	A, B  gmath.Point3
}

// modelDimLookup resolves the referenced model's retrievable dimensions; nil ⇒ none.
type modelDimLookup func() ([]ModelDimension, bool)

// ModelDimensionResolver is the host seam resolving a referenced model document to its retrievable
// parametric dimensions (#1991) — the model↔drawing coupling the retrieve workflow needs, mirroring
// BodyResolver.
type ModelDimensionResolver interface {
	// ModelDimensions returns the named model document's retrievable dimensions, and whether it
	// resolved.
	ModelDimensions(fullDocumentName string) ([]ModelDimension, bool)
}

// RetrievableDimension is one candidate the model offers a view: its name, current value (mm) and the
// sheet position of its midpoint (so a UI can list/preview it before retrieving).
type RetrievableDimension struct {
	Name    string
	ValueMM float64
	SheetX  float64
	SheetY  float64
}

// ListRetrievable returns the referenced model's parametric dimensions projected onto the named base
// view — the candidates a retrieve can materialise (#1991).
func (ds *DrawingDimensions) ListRetrievable(viewName string) ([]RetrievableDimension, error) {
	view, _, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	dims, ok := ds.modelDimensions()
	if !ok {
		return nil, fmt.Errorf("drawing: the referenced model has no retrievable dimensions")
	}
	out := make([]RetrievableDimension, 0, len(dims))
	for _, md := range dims {
		mid := view.place(hlr.ProjectPoint(basis, md.A.Midpoint(md.B)))
		out = append(out, RetrievableDimension{
			Name: md.Name, ValueMM: md.Value * cmToMM, SheetX: float64(mid.X), SheetY: float64(mid.Y),
		})
	}
	return out, nil
}

// Retrieve materialises the named model dimensions as drawing dimensions on the view, each flagged
// Retrieved with a back-reference to its parameter; offset stands the dimension lines off (#1991). An
// unknown name is a clean error. Passing no names retrieves every model dimension.
func (ds *DrawingDimensions) Retrieve(viewName string, names []string, offset float64) ([]*DrawingDimension, error) {
	view, _, basis, err := ds.dimensionBasis(viewName)
	if err != nil {
		return nil, err
	}
	dims, ok := ds.modelDimensions()
	if !ok {
		return nil, fmt.Errorf("drawing: the referenced model has no retrievable dimensions")
	}
	byName := indexModelDimensions(dims)
	if len(names) == 0 {
		names = modelDimensionNames(dims)
	}
	var out []*DrawingDimension
	for _, n := range names {
		md, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("drawing: no retrievable model dimension named %q", n)
		}
		out = append(out, ds.addRetrieved(view, basis, md, offset))
	}
	return out, nil
}

// addRetrieved builds and appends a retrieved drawing dimension from a model dimension.
func (ds *DrawingDimensions) addRetrieved(view *DrawingView, basis hlr.View, md ModelDimension, offset float64) *DrawingDimension {
	d := &DrawingDimension{
		name: ds.uniqueName(""), dimType: types.AlignedDimension, viewName: view.name,
		retrievedFrom: md.Name, worldA: md.A, worldB: md.B, offset: offset,
	}
	ds.recomputeRetrievedFrom(d, view, basis, md)
	ds.items = append(ds.items, d)
	return d
}

// recomputeRetrieved re-fetches the retrieved dimension's source model dimension by name (so the
// endpoints and value track the parameter) and rebuilds its glyph; with the source gone it keeps the
// last stored endpoints (#1991).
func (ds *DrawingDimensions) recomputeRetrieved(d *DrawingDimension, view *DrawingView, basis hlr.View) {
	md := ModelDimension{Name: d.retrievedFrom, A: d.worldA, B: d.worldB, Value: d.valueMM / cmToMM}
	if dims, ok := ds.modelDimensions(); ok {
		if live, found := indexModelDimensions(dims)[d.retrievedFrom]; found {
			md = live
			d.worldA, d.worldB = live.A, live.B
		}
	}
	ds.recomputeRetrievedFrom(d, view, basis, md)
}

// recomputeRetrievedFrom projects a model dimension's endpoints onto the view and builds the glyph,
// using the model dimension's parametric value (not the projected distance) as the measured value.
func (ds *DrawingDimensions) recomputeRetrievedFrom(d *DrawingDimension, view *DrawingView, basis hlr.View, md ModelDimension) {
	p1 := hlr.ProjectPoint(basis, md.A)
	p2 := hlr.ProjectPoint(basis, md.B)
	ds.buildLinearGlyph(d, view, p1, p2, md.Value*cmToMM)
}

// SetRetrievedValue rejects driving the model from a retrieved dimension this increment, with a clear
// error naming the parameter — the acceptance path that defers model editing (#1991).
func (ds *DrawingDimensions) SetRetrievedValue(name string, _ float64) error {
	d, ok := ds.ByName(name)
	if !ok {
		return fmt.Errorf("drawing: no dimension named %q", name)
	}
	if d.retrievedFrom == "" {
		return fmt.Errorf("drawing: %q is not a retrieved dimension; its value is measured from the model", name)
	}
	return fmt.Errorf("drawing: editing retrieved dimension %q would drive model parameter %q — not supported this increment", name, d.retrievedFrom)
}

// modelDimensions resolves the referenced model's retrievable dimensions through the injected hook.
func (ds *DrawingDimensions) modelDimensions() ([]ModelDimension, bool) {
	if ds.modelDims == nil {
		return nil, false
	}
	return ds.modelDims()
}

// indexModelDimensions maps model dimensions by name (last wins on a duplicate name).
func indexModelDimensions(dims []ModelDimension) map[string]ModelDimension {
	out := make(map[string]ModelDimension, len(dims))
	for _, md := range dims {
		out[md.Name] = md
	}
	return out
}

// modelDimensionNames lists the dimension names in order.
func modelDimensionNames(dims []ModelDimension) []string {
	out := make([]string, len(dims))
	for i, md := range dims {
		out[i] = md.Name
	}
	return out
}

// Retrieved reports whether the dimension was retrieved from a model dimension (#1991).
func (d *DrawingDimension) Retrieved() bool { return d.retrievedFrom != "" }

// RetrievedFrom returns the source model-dimension (parameter) name, or "" for a picked dimension.
func (d *DrawingDimension) RetrievedFrom() string { return d.retrievedFrom }
