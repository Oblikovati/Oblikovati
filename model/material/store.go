// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/yamlcodec"
)

// FileSystem is the small filesystem seam the project [Store] depends on, so library IO is
// testable with an in-memory fake. ReadFile returns an error for a missing file (first
// run); the caller treats that as "no library yet".
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error // create/overwrite, making parent dirs
}

// Store loads and saves the project's shared asset library — the catalog every document of
// a project picks from (ADR-0022). It lives in the project's design-data directory as two
// readable YAML files, appearances.yaml and materials.yaml.
type Store struct {
	dir string
	fs  FileSystem
}

// NewStore builds a store over a project design-data directory.
func NewStore(designDataDir string, fs FileSystem) *Store {
	return &Store{dir: designDataDir, fs: fs}
}

type appearanceLibraryFile struct {
	Appearances []AppearanceRecipe `yaml:"appearances"`
}

type materialLibraryFile struct {
	Materials []MaterialRecipe `yaml:"materials"`
}

// Load folds the project library's appearances, OpenPBR appearances, and materials into
// lib as project-source assets. A missing library (first run) is not an error. A
// malformed color aborts the load (a corrupt library should fail loudly, not render
// black).
func (s *Store) Load(lib *Library) error {
	if err := s.loadAppearances(lib); err != nil {
		return err
	}
	if err := s.loadOpenPBRAppearances(lib); err != nil {
		return err
	}
	return s.loadMaterials(lib)
}

func (s *Store) loadAppearances(lib *Library) error {
	data, ok := s.read(s.appearancePath())
	if !ok {
		return nil
	}
	var f appearanceLibraryFile
	if err := yamlcodec.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("material: parse %q: %w", s.appearancePath(), err)
	}
	for _, r := range f.Appearances {
		a, err := recipeToAppearance(r, SourceProject)
		if err != nil {
			return fmt.Errorf("material: %q: %w", s.appearancePath(), err)
		}
		lib.AddAppearance(a)
	}
	return nil
}

func (s *Store) loadMaterials(lib *Library) error {
	data, ok := s.read(s.materialPath())
	if !ok {
		return nil
	}
	var f materialLibraryFile
	if err := yamlcodec.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("material: parse %q: %w", s.materialPath(), err)
	}
	for _, r := range f.Materials {
		lib.AddMaterial(recipeToMaterial(r, SourceProject))
	}
	return nil
}

// Save writes the library's project-source assets back to the design-data directory. Only
// project assets are persisted — built-ins are reproducible and document assets belong to
// their .obk.
func (s *Store) Save(lib *Library) error {
	if err := s.saveAppearances(lib); err != nil {
		return err
	}
	if err := s.saveOpenPBRAppearances(lib); err != nil {
		return err
	}
	return s.saveMaterials(lib)
}

func (s *Store) saveAppearances(lib *Library) error {
	var af appearanceLibraryFile
	for _, a := range sortAppearances(projectAppearances(lib)) {
		af.Appearances = append(af.Appearances, appearanceToRecipe(a))
	}
	appr, err := yamlcodec.Marshal(af)
	if err != nil {
		return fmt.Errorf("material: marshal appearances: %w", err)
	}
	return s.fs.WriteFile(s.appearancePath(), appr)
}

func (s *Store) saveMaterials(lib *Library) error {
	var mf materialLibraryFile
	for _, m := range sortMaterials(projectMaterials(lib)) {
		mf.Materials = append(mf.Materials, materialToRecipe(m))
	}
	mats, err := yamlcodec.Marshal(mf)
	if err != nil {
		return fmt.Errorf("material: marshal materials: %w", err)
	}
	return s.fs.WriteFile(s.materialPath(), mats)
}

// read returns a file's bytes, or (nil,false) when it is absent/unreadable.
func (s *Store) read(path string) ([]byte, bool) {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

func (s *Store) appearancePath() string { return filepath.Join(s.dir, "appearances.yaml") }
func (s *Store) materialPath() string   { return filepath.Join(s.dir, "materials.yaml") }

// projectAppearances / projectMaterials filter a library to its project-source assets.
func projectAppearances(lib *Library) []*Appearance {
	var out []*Appearance
	for _, a := range lib.Appearances() {
		if a.Source() == SourceProject {
			out = append(out, a)
		}
	}
	return out
}

func projectMaterials(lib *Library) []*Material {
	var out []*Material
	for _, m := range lib.Materials() {
		if m.Source() == SourceProject {
			out = append(out, m)
		}
	}
	return out
}

// OSFileSystem is the production [FileSystem]: real files under the project design-data
// directory.
type OSFileSystem struct{}

// ReadFile reads a file's contents.
func (OSFileSystem) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

// WriteFile writes data, creating parent directories as needed.
func (OSFileSystem) WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
