// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"
	"path/filepath"

	"oblikovati.org/yamlcodec"
)

// openpbrAppearanceLibraryFile is the on-disk shape of the project's OpenPBR appearance
// library — a sibling of [appearanceLibraryFile], persisted as its own readable file
// (openpbr-appearances.yaml) so the metallic-roughness and OpenPBR catalogs never
// interleave on disk.
type openpbrAppearanceLibraryFile struct {
	Appearances []OpenPBRAppearanceRecipe `yaml:"appearances"`
}

func (s *Store) openpbrAppearancePath() string {
	return filepath.Join(s.dir, "openpbr-appearances.yaml")
}

func (s *Store) loadOpenPBRAppearances(lib *Library) error {
	data, ok := s.read(s.openpbrAppearancePath())
	if !ok {
		return nil
	}
	var f openpbrAppearanceLibraryFile
	if err := yamlcodec.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("material: parse %q: %w", s.openpbrAppearancePath(), err)
	}
	for _, r := range f.Appearances {
		lib.AddOpenPBRAppearance(recipeToOpenPBRAppearance(r, SourceProject))
	}
	return nil
}

func (s *Store) saveOpenPBRAppearances(lib *Library) error {
	var f openpbrAppearanceLibraryFile
	for _, a := range sortOpenPBRAppearances(projectOpenPBRAppearances(lib)) {
		f.Appearances = append(f.Appearances, openPBRAppearanceToRecipe(a))
	}
	data, err := yamlcodec.Marshal(f)
	if err != nil {
		return fmt.Errorf("material: marshal openpbr appearances: %w", err)
	}
	return s.fs.WriteFile(s.openpbrAppearancePath(), data)
}

// projectOpenPBRAppearances filters a library to its project-source OpenPBR assets.
func projectOpenPBRAppearances(lib *Library) []*OpenPBRAppearance {
	var out []*OpenPBRAppearance
	for _, a := range lib.OpenPBRAppearances() {
		if a.Source() == SourceProject {
			out = append(out, a)
		}
	}
	return out
}
