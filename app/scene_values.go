// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/math"

// App-owned scene value types (audit B10, #1621): the lighting / environment / camera settings
// the add-in router (and any other application consumer) speaks, so nothing outside the
// renderer wall (ADR-0014) has to name a renderer.* or scene.* type. The Session translates
// these to and from the renderer's internal GPU representations in scene_values_adapt.go — the
// single wall crossing — keeping the router an adapter that imports only app + api.

// Light is one scene light in application terms: its public definition kind plus emission
// parameters. It mirrors the renderer's per-light data without the renderer import; the wall
// maps Definition onto the renderer's LightKind.
type Light struct {
	Definition  LightDefinitionTypeEnum
	On          bool
	Color       [3]float32
	Intensity   float32
	Direction   [3]float32
	Position    [3]float32
	SpotInner   float32
	SpotOuter   float32
	Attenuation [3]float32
}

// ShadowRig is the application's shadow settings: the ground / object / ambient toggles and the
// density / softness controls. The GroundShadows+GroundXRay pair folds into the public
// GroundShadowEnum via [GroundShadowForSettings] / [ApplyGroundShadow].
type ShadowRig struct {
	GroundShadows  bool
	GroundXRay     bool
	ObjectShadows  bool
	AmbientShadows bool
	Density        float32
	Softness       float32
}

// EnvironmentState is the application's image-based-lighting environment: a built-in preset by
// NAME ("" or "None" when inactive), an optional user HDR file that overrides it, and the
// display controls. Naming the preset as a string keeps this type free of the renderer's preset
// enum. (Named EnvironmentState, not Environment, because app.Environment is already the ribbon
// context alias types.Environment.)
type EnvironmentState struct {
	Preset    string
	FilePath  string
	Rotation  float32
	Intensity float32
	ShowImage bool
}

// IsActive reports whether the environment contributes IBL/skybox — a file is set or a real
// preset (not the empty/"None" preset) is selected. Mirrors renderer.Environment.IsActive so
// call sites read the same across the wall.
func (e EnvironmentState) IsActive() bool {
	return e.FilePath != "" || (e.Preset != "" && e.Preset != "None")
}

// CameraFrame is the application's view camera: a look-at frame plus the transient viewport
// pixel size and projection mode. It mirrors the renderer's scene camera without the scene
// import (the router only edits the look-at frame; Width/Height/Orthographic pass through).
type CameraFrame struct {
	Eye          math.Point3
	Target       math.Point3
	Up           math.Vector3
	FOV          float64
	Width        int
	Height       int
	Orthographic bool
}

// LightingStyleOption is one lighting-style gallery entry — its user-facing name (the router
// flags the active one by comparing against [Session.LightingStyleName]).
type LightingStyleOption struct{ Name string }

// EnvironmentOption is one environment-preset gallery entry — its user-facing name.
type EnvironmentOption struct{ Name string }
