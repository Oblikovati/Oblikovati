// SPDX-License-Identifier: GPL-2.0-only

package renderer

// EnvironmentPreset identifies a built-in HDR sky generated procedurally by the head (no
// file I/O), so the default image-based-lighting path is dependency-free and headless
// (ADR-0026 §4). A user file overrides the preset (see [Environment.FilePath]).
type EnvironmentPreset uint8

const (
	// EnvNone draws no sky and contributes no IBL — the viewport keeps its flat themed
	// clear color and the Realistic shader falls back to the analytic ambient term. It is
	// the zero value so an un-configured scene is unchanged (ADR-0026 §7).
	EnvNone EnvironmentPreset = iota
	// EnvStudio is a soft neutral three-tone studio surround (even, low-contrast).
	EnvStudio
	// EnvOutdoors is a sunny sky-to-ground gradient (bright blue zenith, warm horizon).
	EnvOutdoors
	// EnvOvercast is a dim, near-uniform grey sky (flat, shadow-light).
	EnvOvercast
	// EnvSunset is a warm low-sun gradient (orange horizon, deep blue zenith).
	EnvSunset
)

// environmentNames is the user-facing label for each preset (the Environment gallery).
var environmentNames = map[EnvironmentPreset]string{
	EnvNone:     "None",
	EnvStudio:   "Studio",
	EnvOutdoors: "Outdoors",
	EnvOvercast: "Overcast",
	EnvSunset:   "Sunset",
}

// Environment is the image-based-lighting source the scene reflects — either a built-in
// [EnvironmentPreset] or, when FilePath is non-empty, a user equirectangular HDR file that
// overrides it. It is pure data (ADR-0014/§1): the native layer resolves it into the cubemaps
// + BRDF LUT the Realistic shader samples and into the optional skybox background.
//
// Rotation spins the sky about the vertical (+Z) axis in radians; Intensity scales the IBL
// contribution (1 = as authored); ShowImage draws the environment as the viewport background
// instead of the themed clear color (ADR-0026 §5).
type Environment struct {
	Preset    EnvironmentPreset
	FilePath  string
	Rotation  float32
	Intensity float32
	ShowImage bool
}

// String returns the environment's user-facing label: the file's marker when a file is set,
// otherwise the preset name.
func (e Environment) String() string {
	if e.FilePath != "" {
		return "File"
	}
	return e.Preset.String()
}

// IsActive reports whether the environment contributes IBL/skybox — a file is set or the
// preset is not [EnvNone]. An inactive environment leaves the pre-IBL look unchanged.
func (e Environment) IsActive() bool {
	return e.FilePath != "" || e.Preset != EnvNone
}

// String returns the preset's user-facing label (the Environment gallery entry).
func (p EnvironmentPreset) String() string {
	if name, ok := environmentNames[p]; ok {
		return name
	}
	return "environment(?)"
}

// IsValid reports whether p is a defined preset.
func (p EnvironmentPreset) IsValid() bool {
	_, ok := environmentNames[p]
	return ok
}

// EnvironmentOption pairs a preset with its label, for building the Environment picker.
type EnvironmentOption struct {
	Preset EnvironmentPreset
	Name   string
}

// EnvironmentGallery returns every built-in preset with its label, in picker order — the
// source list for the View-tab Environment selection box.
func EnvironmentGallery() []EnvironmentOption {
	order := []EnvironmentPreset{EnvNone, EnvStudio, EnvOutdoors, EnvOvercast, EnvSunset}
	opts := make([]EnvironmentOption, len(order))
	for i, p := range order {
		opts[i] = EnvironmentOption{Preset: p, Name: p.String()}
	}
	return opts
}
