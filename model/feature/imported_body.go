// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati/kernel/topo"

// ImportedBodyFeature wraps a body translated from a foreign mesh file (STL/OBJ/3MF,
// M17-F04) as a feature-tree participant, so downstream parametric features (fillet,
// hole, boolean cut, …) can operate on it. It is the mesh-exchange counterpart of
// NonParametricBaseFeature in derived.go, adding the SOURCE (file path + format) so the
// recipe re-imports the body on reopen — a *topo.Body cannot itself round-trip through
// the YAML recipe, so re-import from the recorded source is the persistence path.
//
// The wrapped body is frozen (non-parametric): editing the file and re-importing is a
// new import; this feature just injects the body it was built with.
type ImportedBodyFeature struct {
	body   *topo.Body
	Path   string // source file path (recorded in the recipe; re-imported on open)
	Format string // source format ("stl"|"obj"|"3mf"), an api/types.ExchangeFormat string
}

// Body returns the imported body (the geometry injected into recompute).
func (f *ImportedBodyFeature) Body() *topo.Body { return f.body }

// Kind implements [Feature].
func (f *ImportedBodyFeature) Kind() string { return "importedBody" }

// Recompute appends the imported body to the running state, leaving earlier bodies in
// place so a part can hold both modeled and imported geometry.
func (f *ImportedBodyFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), f.body)}, nil
}

// ImportedBodies adds imported-body features into the engine — the model-layer seam the
// mesh-exchange importer calls after translating a file to a body.
type ImportedBodies struct{ engine *PartFeatures }

// NewImportedBodies binds the collection to a feature engine.
func NewImportedBodies(engine *PartFeatures) *ImportedBodies { return &ImportedBodies{engine} }

// Add wraps a translated body with its source coordinates and records it as a feature.
func (c *ImportedBodies) Add(body *topo.Body, path, format string) *PartFeature {
	return c.engine.Add(&ImportedBodyFeature{body: body, Path: path, Format: format})
}
