// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"os"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
)

// ImportBodies reads path and translates it via [ImportBodiesFromData]. The interactive
// importer uses it to read a file once before embedding its bytes as a document resource.
func ImportBodies(format types.ExchangeFormat, path string) ([]*topo.Body, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("import: read %q: %w", path, err)
	}
	return ImportBodiesFromData(format, data)
}

// ImportBodiesFromData translates raw file bytes into kernel bodies, routing by format: a mesh
// format (STL/OBJ/3MF) yields one welded body; STEP yields every B-rep solid in the file. It is
// the shared reader for both an interactive import and a recipe re-import-from-resource — the
// bytes come from the file the first time and from the embedded document resource on reopen.
func ImportBodiesFromData(format types.ExchangeFormat, data []byte) ([]*topo.Body, []string, error) {
	switch {
	case format == types.FormatSTEP:
		// The kernel works in centimetres; the reader scales the file's declared unit into it.
		return step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
	case format.IsMesh():
		body, warns, err := meshio.ImportBody(format, data, fmt.Sprintf("import:%s#0", format), 0,
			exchange.TranslationOptions{TargetUnitMM: exchange.DBUnitMM})
		if err != nil {
			return nil, nil, err
		}
		return []*topo.Body{body}, warns, nil
	default:
		return nil, nil, fmt.Errorf("import: unsupported format %q (want stl|obj|3mf|step)", format)
	}
}

// ImportData is an imported-body feature's recipe: the document resource UUID holding the source
// bytes + the format + which body (multi-body STEP). The translated *topo.Body itself is not
// serialized (it cannot round-trip through YAML); on open the body is re-derived from the
// embedded resource (ADR-0031), so the document is self-contained — no external file path.
type ImportData struct {
	Resource string `yaml:"resource"`
	Format   string `yaml:"format"`
	Index    int    `yaml:"index,omitempty"` // which body of a multi-body (STEP) file
}

// serializeImportedBody projects an imported-body feature to its source recipe.
func serializeImportedBody(f *ImportedBodyFeature) *ImportData {
	return &ImportData{Resource: f.Resource, Format: f.Format, Index: f.Index}
}

// restoreImportedBody re-derives the body from the document resource the recipe cites and wraps
// it (by index, for multi-body STEP files) as a feature. A missing resource, a bad format, or a
// now-missing body index errors (no silent body loss), matching the recipe codec's no-silent-drop
// policy. The bytes come from the document itself, so reopening needs no external source file.
func restoreImportedBody(fs *PartFeatures, d *ImportData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("imported-body feature is missing its payload")
	}
	if fs.resources == nil {
		return nil, fmt.Errorf("imported-body re-import: no resource store wired into the engine")
	}
	data, ok := fs.resources.ResourceBytes(d.Resource)
	if !ok {
		return nil, fmt.Errorf("imported-body re-import: resource %q not present in the document", d.Resource)
	}
	bodies, _, err := ImportBodiesFromData(types.ExchangeFormat(d.Format), data)
	if err != nil {
		return nil, fmt.Errorf("imported-body re-import (resource %q): %w", d.Resource, err)
	}
	if d.Index < 0 || d.Index >= len(bodies) {
		return nil, fmt.Errorf("imported-body re-import (resource %q): body %d no longer present (file has %d)", d.Resource, d.Index, len(bodies))
	}
	return NewImportedBodies(fs).AddAt(bodies[d.Index], d.Resource, d.Format, d.Index), nil
}
