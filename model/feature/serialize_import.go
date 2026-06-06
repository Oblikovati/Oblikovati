// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	"os"

	"oblikovati/api/types"
	"oblikovati/kernel/exchange/meshio"
)

// ImportData is an imported-body feature's recipe: the SOURCE file path + format. The
// translated *topo.Body itself is not serialized (it cannot round-trip through YAML); on
// open the file is re-imported from these coordinates, so editing the source file and
// reopening pulls the new geometry — the associative-import story (M17-F04, plan §4).
type ImportData struct {
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
}

// serializeImportedBody projects an imported-body feature to its source recipe.
func serializeImportedBody(f *ImportedBodyFeature) *ImportData {
	return &ImportData{Path: f.Path, Format: f.Format}
}

// restoreImportedBody re-imports the source file recorded in the recipe and wraps the
// resulting body as a feature. A missing/unreadable file or a bad format errors (no
// silent body loss), matching the no-silent-drop policy of the recipe codec.
func restoreImportedBody(fs *PartFeatures, d *ImportData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("imported-body feature is missing its payload")
	}
	data, err := os.ReadFile(d.Path)
	if err != nil {
		return nil, fmt.Errorf("imported-body re-import %q: %w", d.Path, err)
	}
	feat := fmt.Sprintf("import:%s#0", d.Format)
	body, _, err := meshio.ImportBody(types.ExchangeFormat(d.Format), data, feat, 0)
	if err != nil {
		return nil, fmt.Errorf("imported-body re-import %q: %w", d.Path, err)
	}
	return NewImportedBodies(fs).Add(body, d.Path, d.Format), nil
}
