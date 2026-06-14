// SPDX-License-Identifier: GPL-2.0-only

// Package options holds the typed per-user application options of M05-F11 (#618)
// and their YAML persistence — replacing ad-hoc preference keys with whole groups
// the Preferences window and the options.* wire methods both bind to. Only options
// with a live reader exist here (the issue's rule); the session applies them to its
// running state ([oblikovati.org/app] wires that).
package options

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati.org/api/types"
	"oblikovati.org/persistence/yamlcodec"
	"oblikovati.org/userconfig"
)

// General is application-level behavior: what opens at launch (read by the head's
// startup; honored on the NEXT start).
type General struct {
	StartupAction types.StartupActionType `yaml:"startupAction,omitempty"`
}

// Sketch is the grid and click snapping, applied live to the session's
// GridSettings. Spacing is in model/database units (cm), unit-independent.
type Sketch struct {
	GridSpacingCm  float64 `yaml:"gridSpacingCm"`
	GridVisible    bool    `yaml:"gridVisible"`
	GridMajorEvery int     `yaml:"gridMajorEvery"`
	SnapToPoints   bool    `yaml:"snapToPoints"`
	SnapToGrid     bool    `yaml:"snapToGrid"`
}

// Part is the part-modeling defaults, applied live to the session.
type Part struct {
	ChamferFlatCorners bool `yaml:"chamferFlatCorners"`
}

// Save is the save policy (M03-F09, #610): thumbnail capture on save (read by
// the head's save path), dependent saving (read by the session's save flow),
// and old-version retention (read by the package store).
type Save struct {
	Thumbnail         types.ThumbnailSaveOption `yaml:"thumbnail,omitempty"`
	SaveDependents    bool                      `yaml:"saveDependents,omitempty"`
	OldVersionsToKeep int                       `yaml:"oldVersionsToKeep,omitempty"`
}

// Updates is the software-update behavior (read by the head's startup auto-check and
// the CLI): whether to check GitHub for a newer release on launch. The user toggles it
// from the update notification or Help ▸ Check for Updates; a manual check ignores it.
type Updates struct {
	CheckOnStartup bool `yaml:"checkOnStartup"`
}

// All is every persisted option group. The ViewCube/display preferences live in
// their own store (persistence/userprefs) and the color scheme in the theme store —
// the options surface proxies those rather than duplicating their persistence.
type All struct {
	General General `yaml:"general"`
	Sketch  Sketch  `yaml:"sketch"`
	Part    Part    `yaml:"part"`
	Save    Save    `yaml:"save"`
	Updates Updates `yaml:"updates"`
}

// Defaults returns the out-of-the-box options, mirroring the session's historical
// defaults (10 mm visible grid with both snaps, flat chamfer corners, new part at
// launch) so a fresh install behaves exactly as before this file existed.
func Defaults() All {
	return All{
		General: General{StartupAction: types.StartupNewPart},
		Sketch:  Sketch{GridSpacingCm: 1, GridVisible: true, GridMajorEvery: 5, SnapToPoints: true, SnapToGrid: true},
		Part:    Part{ChamferFlatCorners: true},
		Save:    Save{Thumbnail: types.ThumbnailNone},
		Updates: Updates{CheckOnStartup: true},
	}
}

// Store persists the option groups across sessions.
type Store interface {
	Load() (All, error)
	Save(All) error
}

// FileStore persists the options to one YAML file in the user config directory.
type FileStore struct{ path string }

// DefaultPath is the per-user options file: ~/.oblikovati/options.yaml on
// Linux/macOS (the shared userconfig directory elsewhere).
func DefaultPath() (string, error) {
	return userconfig.File("options.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

// Load reads the stored options over the defaults: a missing file (fresh install)
// or an absent key keeps its default, so adding an option never breaks old files.
func (s *FileStore) Load() (All, error) {
	all := Defaults()
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return all, nil
	}
	if err != nil {
		return all, fmt.Errorf("options: read %q: %w", s.path, err)
	}
	if err := yamlcodec.Unmarshal(raw, &all); err != nil {
		return Defaults(), fmt.Errorf("options: parse %q: %w", s.path, err)
	}
	return all, nil
}

// Save writes the options, creating the config directory on first use.
func (s *FileStore) Save(all All) error {
	data, err := yamlcodec.Marshal(all)
	if err != nil {
		return fmt.Errorf("options: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("options: create config dir for %q: %w", s.path, err)
	}
	return os.WriteFile(s.path, data, 0o644)
}
