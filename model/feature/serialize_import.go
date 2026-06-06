// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"os"

	"oblikovati/api/types"
	"oblikovati/kernel/exchange"
	"oblikovati/kernel/exchange/meshio"
	"oblikovati/kernel/exchange/step"
	"oblikovati/kernel/topo"
)

// ImportBodies reads path and translates it into kernel bodies, routing by format: a mesh format
// (STL/OBJ/3MF) yields one welded body; STEP yields every B-rep solid in the file. It is the shared
// reader used by both an interactive import and a recipe re-import.
func ImportBodies(format types.ExchangeFormat, path string) ([]*topo.Body, []string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("import: read %q: %w", path, err)
	}
	switch {
	case format == types.FormatSTEP:
		return step.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	case format.IsMesh():
		body, warns, err := meshio.ImportBody(format, data, fmt.Sprintf("import:%s#0", format), 0)
		if err != nil {
			return nil, nil, err
		}
		return []*topo.Body{body}, warns, nil
	default:
		return nil, nil, fmt.Errorf("import: unsupported format %q (want stl|obj|3mf|step)", format)
	}
}

// ImportData is an imported-body feature's recipe: the SOURCE file path + format. The
// translated *topo.Body itself is not serialized (it cannot round-trip through YAML); on
// open the file is re-imported from these coordinates, so editing the source file and
// reopening pulls the new geometry — the associative-import story (M17-F04, plan §4).
type ImportData struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
	Index  int    `yaml:"index,omitempty"` // which body of a multi-body (STEP) file
}

// serializeImportedBody projects an imported-body feature to its source recipe.
func serializeImportedBody(f *ImportedBodyFeature) *ImportData {
	return &ImportData{Path: f.Path, Format: f.Format, Index: f.Index}
}

// restoreImportedBody re-imports the source file recorded in the recipe and wraps the recorded
// body (by index, for multi-body STEP files) as a feature. A missing/unreadable file, a bad
// format, or a now-missing body index errors (no silent body loss), matching the no-silent-drop
// policy of the recipe codec.
func restoreImportedBody(fs *PartFeatures, d *ImportData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("imported-body feature is missing its payload")
	}
	bodies, _, err := ImportBodies(types.ExchangeFormat(d.Format), d.Path)
	if err != nil {
		return nil, fmt.Errorf("imported-body re-import %q: %w", d.Path, err)
	}
	if d.Index < 0 || d.Index >= len(bodies) {
		return nil, fmt.Errorf("imported-body re-import %q: body %d no longer present (file has %d)", d.Path, d.Index, len(bodies))
	}
	return NewImportedBodies(fs).AddAt(bodies[d.Index], d.Path, d.Format, d.Index), nil
}
