// SPDX-License-Identifier: GPL-2.0-only

// Package options holds the typed per-user application options of M05-F11 (#618)
// and their YAML persistence — replacing ad-hoc preference keys with whole groups
// the Preferences window and the options.* wire methods both bind to. Only options
// with a live reader exist here (the issue's rule); the session applies them to its
// running state ([oblikovati.org/app] wires that).
package options

import (
	"oblikovati.org/api/types"
	"oblikovati.org/persistence/filestore"
	"oblikovati.org/userconfig"
)

// General is application-level behavior: what opens at launch (read by the head's
// startup; honored on the NEXT start).
type General struct {
	StartupAction types.StartupActionType `yaml:"startupAction,omitempty"`
}

// Sketch is the grid and click snapping, applied live to the session's
// GridSettings. Spacing is in model/database units (cm), unit-independent.
// HeadsUpDisplay and RelaxMode are the 2D-sketch UI-environment toggles (M21-F12,
// #790/#791): the dynamic-input HUD shown while drawing geometry, and Relax Mode
// (drag fully/over-constrained geometry while the solver relaxes the dimensions).
type Sketch struct {
	GridSpacingCm  float64 `yaml:"gridSpacingCm"`
	GridVisible    bool    `yaml:"gridVisible"`
	GridMajorEvery int     `yaml:"gridMajorEvery"`
	SnapToPoints   bool    `yaml:"snapToPoints"`
	SnapToGrid     bool    `yaml:"snapToGrid"`
	HeadsUpDisplay bool    `yaml:"headsUpDisplay"`
	RelaxMode      bool    `yaml:"relaxMode"`

	// The in-canvas input surfaces the heads-up display offers while geometry is placed
	// (#2014). PointerInput is the coordinate entry for a shape's first point; DimensionInput
	// is the boxes that size the shape being placed; the Cartesian flags pick X/Y and
	// width/height over length and angle. CreateDimensionsOnValueInput makes a typed value a
	// persistent driving dimension on commit. Absent keys keep their defaults because Load
	// starts from Defaults(), so an existing options file gains them switched on.
	PointerInput                 bool `yaml:"pointerInput"`
	PointerInputCartesian        bool `yaml:"pointerInputCartesian"`
	DimensionInput               bool `yaml:"dimensionInput"`
	DimensionInputCartesian      bool `yaml:"dimensionInputCartesian"`
	CreateDimensionsOnValueInput bool `yaml:"createDimensionsOnValueInput"`

	// AutoProjectOrigin projects the part origin's centre point into every new sketch, so the
	// sketch opens with an anchor to constrain against at (0,0) — Inventor's "Autoproject part
	// origin on sketch create", on by default. Load merges over Defaults(), so an options file
	// written before this key keeps the default rather than reading as off (#2016).
	AutoProjectOrigin bool `yaml:"autoProjectOrigin"`

	// SuppressFormatOverrides is the Format panel's Show Format toggle (#2015). On means the
	// sketch draws with DEFAULT attributes, hiding per-entity line type, colour and thickness —
	// the documented behaviour, which is the inverse of what the button's label suggests.
	SuppressFormatOverrides bool `yaml:"suppressFormatOverrides"`
}

// Part is the part-modeling defaults, applied live to the session.
type Part struct {
	ChamferFlatCorners bool `yaml:"chamferFlatCorners"`
	// TangentChainSelect makes a Fillet/Chamfer pick select the whole tangent chain through the
	// clicked edge (Inventor's tangent propagation). Defaults true; Load merges over Defaults so a
	// stored file missing the key keeps the default. See #1798 follow-up (#1947).
	TangentChainSelect bool `yaml:"tangentChainSelect"`
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

// Telemetry is the anonymous usage-statistics behavior (#1182). When ShareUsageStatistics is
// on, the head submits one anonymous installation snapshot (OS, hardware, version, installed
// add-ins) to stats.oblikovati.org during the startup update-check. It is opt-out: on by
// default, toggled from the same preferences surface as the update check.
type Telemetry struct {
	ShareUsageStatistics bool `yaml:"shareUsageStatistics"`
}

// UI is the user-facing scale of the interface, applied live by the head every frame
// (#1232 follow-up: icons/text too small on high-resolution monitors). 1.0 = 100%.
// FontScale drives ImGui's style.FontScaleMain (live, no atlas rebuild on ImGui 1.92);
// IconScale multiplies every ribbon/nav/steering/property icon's rasterization size.
type UI struct {
	FontScale float64 `yaml:"fontScale"`
	IconScale float64 `yaml:"iconScale"`
}

// All is every persisted option group. The ViewCube/display preferences live in
// their own store (persistence/userprefs) and the color scheme in the theme store —
// the options surface proxies those rather than duplicating their persistence.
type All struct {
	General   General   `yaml:"general"`
	Sketch    Sketch    `yaml:"sketch"`
	Part      Part      `yaml:"part"`
	Save      Save      `yaml:"save"`
	Updates   Updates   `yaml:"updates"`
	Telemetry Telemetry `yaml:"telemetry"`
	UI        UI        `yaml:"ui"`
}

// Defaults returns the out-of-the-box options, mirroring the session's historical
// defaults (10 mm visible grid with both snaps, flat chamfer corners, new part at
// launch, 100% UI scale) so a fresh install behaves exactly as before this file existed.
func Defaults() All {
	return All{
		General: General{StartupAction: types.StartupNewPart},
		Sketch: Sketch{
			GridSpacingCm: 1, GridVisible: true, GridMajorEvery: 5, SnapToPoints: true, SnapToGrid: true,
			HeadsUpDisplay: true, PointerInput: true, PointerInputCartesian: true,
			DimensionInput: true, CreateDimensionsOnValueInput: true, AutoProjectOrigin: true,
		},
		Part:      Part{ChamferFlatCorners: true, TangentChainSelect: true},
		Save:      Save{Thumbnail: types.ThumbnailNone},
		Updates:   Updates{CheckOnStartup: true},
		Telemetry: Telemetry{ShareUsageStatistics: true},
		UI:        UI{FontScale: 1, IconScale: 1},
	}
}

// Store persists the option groups across sessions.
type Store interface {
	Load() (All, error)
	Save(All) error
}

// FileStore persists the options to one YAML file in the user config directory
// (the shared filestore core, #1651).
type FileStore struct{ file *filestore.FileStore[All] }

// DefaultPath is the per-user options file: ~/.oblikovati/options.yaml on
// Linux/macOS (the shared userconfig directory elsewhere).
func DefaultPath() (string, error) {
	return userconfig.File("options.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{file: filestore.New[All](path)}
}

// Load reads the stored options over the defaults (filestore.LoadInto): a missing
// file (fresh install) or an absent key keeps its default, so adding an option
// never breaks old files.
func (s *FileStore) Load() (All, error) {
	all := Defaults()
	if _, err := s.file.LoadInto(&all); err != nil {
		return Defaults(), err
	}
	return all, nil
}

// Save writes the options, creating the config directory on first use.
func (s *FileStore) Save(all All) error { return s.file.Save(all) }
