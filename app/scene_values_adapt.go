// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// This file is the ONE renderer/scene wall crossing for the scene value types (audit B10,
// #1621): every translation between the app-owned [Light]/[ShadowRig]/[Environment]/
// [CameraFrame] and the renderer's internal GPU/scene representations lives here. Session
// accessors and the head go through these so no other consumer — the router above all — needs
// to import renderer or scene.

// lightValue converts a renderer scene light into the app value type.
func lightValue(l renderer.SceneLight) Light {
	return Light{
		Definition:  DefinitionForLightKind(l.Kind),
		On:          l.On,
		Color:       l.Color,
		Intensity:   l.Intensity,
		Direction:   l.Direction,
		Position:    l.Position,
		SpotInner:   l.SpotInner,
		SpotOuter:   l.SpotOuter,
		Attenuation: l.Attenuation,
	}
}

// renderLight converts an app light value into the renderer's internal light.
func renderLight(l Light) renderer.SceneLight {
	return renderer.SceneLight{
		Kind:        LightKindForDefinition(l.Definition),
		On:          l.On,
		Color:       l.Color,
		Intensity:   l.Intensity,
		Direction:   l.Direction,
		Position:    l.Position,
		SpotInner:   l.SpotInner,
		SpotOuter:   l.SpotOuter,
		Attenuation: l.Attenuation,
	}
}

// shadowRigValue converts renderer shadow settings into the app value type (field-for-field).
func shadowRigValue(s renderer.ShadowSettings) ShadowRig {
	return ShadowRig{
		GroundShadows: s.GroundShadows, GroundXRay: s.GroundXRay,
		ObjectShadows: s.ObjectShadows, AmbientShadows: s.AmbientShadows,
		Density: s.Density, Softness: s.Softness,
	}
}

// renderShadowRig converts an app shadow rig into the renderer's internal settings.
func renderShadowRig(s ShadowRig) renderer.ShadowSettings {
	return renderer.ShadowSettings{
		GroundShadows: s.GroundShadows, GroundXRay: s.GroundXRay,
		ObjectShadows: s.ObjectShadows, AmbientShadows: s.AmbientShadows,
		Density: s.Density, Softness: s.Softness,
	}
}

// environmentValue converts a renderer environment into the app value type, naming the preset.
func environmentValue(e renderer.Environment) EnvironmentState {
	return EnvironmentState{
		Preset: e.Preset.String(), FilePath: e.FilePath,
		Rotation: e.Rotation, Intensity: e.Intensity, ShowImage: e.ShowImage,
	}
}

// renderEnvironment converts an app environment into the renderer's internal environment,
// resolving the preset name back to its enum (unknown ⇒ EnvNone).
func renderEnvironment(e EnvironmentState) renderer.Environment {
	preset, _ := EnvironmentPresetByName(e.Preset)
	return renderer.Environment{
		Preset: preset, FilePath: e.FilePath,
		Rotation: e.Rotation, Intensity: e.Intensity, ShowImage: e.ShowImage,
	}
}

// RenderEnvironment exposes the app→renderer environment translation for the head's render loop
// (the composition root legitimately lives across the wall). Application-internal code and the
// router never need it.
func RenderEnvironment(e EnvironmentState) renderer.Environment { return renderEnvironment(e) }

// cameraFrameValue converts a renderer scene camera into the app value type.
func cameraFrameValue(c scene.Camera) CameraFrame {
	return CameraFrame{
		Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV,
		Width: c.Width, Height: c.Height, Orthographic: c.Orthographic,
	}
}

// CameraFrame returns the live viewport camera as an app value (audit B10, #1621): the
// router-facing accessor, so a caller need not name scene.Camera. The head keeps the
// scene-typed [Session.Camera] for the render loop.
func (s *Session) CameraFrame() CameraFrame { return cameraFrameValue(s.camera) }

// renderCamera converts an app camera frame into the renderer's scene camera.
func renderCamera(c CameraFrame) scene.Camera {
	return scene.Camera{
		Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV,
		Width: c.Width, Height: c.Height, Orthographic: c.Orthographic,
	}
}

// LightingStyleGallery returns the lighting styles in gallery order as app options (name only) —
// the application-typed view of the renderer gallery, for the router and pickers.
func LightingStyleGallery() []LightingStyleOption {
	g := renderer.LightingStyleGallery()
	out := make([]LightingStyleOption, len(g))
	for i, o := range g {
		out[i] = LightingStyleOption{Name: o.Name}
	}
	return out
}

// EnvironmentGallery returns the built-in environment presets in gallery order as app options.
func EnvironmentGallery() []EnvironmentOption {
	g := renderer.EnvironmentGallery()
	out := make([]EnvironmentOption, len(g))
	for i, o := range g {
		out[i] = EnvironmentOption{Name: o.Name}
	}
	return out
}
