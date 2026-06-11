// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/renderer"
)

// Public lighting/shadow enums are defined once in the Apache-2.0 contract and aliased here so
// call sites read app.X (ADR-0018). They map onto the renderer's internal lighting types.
type (
	LightTypeEnum           = types.LightTypeEnum
	LightDefinitionTypeEnum = types.LightDefinitionTypeEnum
	LightingStyleTypeEnum   = types.LightingStyleTypeEnum
	ShadowDirectionEnum     = types.ShadowDirectionEnum
	GroundShadowEnum        = types.GroundShadowEnum
)

// SceneLighting returns the session's live lighting rig (lights + ambient/exposure +
// environment + shadows) — what the renderer hands the GPU each frame. It is never empty: the
// session starts on the Three Point studio rig ([renderer.LightingThreePoint]).
func (s *Session) SceneLighting() renderer.SceneLighting { return s.lighting }

// LightingStyleName returns the active lighting style's user-facing name.
func (s *Session) LightingStyleName() string { return s.lightingStyle.String() }

// SetLightingStyle activates the named lighting preset, resolving its rig into the live
// lighting state, and errors on an unknown name (so a bad add-in request is rejected, not
// silently ignored — mirroring SetDisplayMode).
func (s *Session) SetLightingStyle(name string) error {
	id, ok := lightingStyleByName(name)
	if !ok {
		return fmt.Errorf("app: unknown lighting style %q", name)
	}
	prevEnv := s.lighting.Environment
	s.lightingStyle = id
	s.lighting = renderer.SceneLightingFor(id)
	if !s.lighting.Environment.IsActive() {
		// A style without its own environment keeps the current one — switching the
		// lighting rig must not silently drop the chosen (or default Sky) skymap.
		s.lighting.Environment = prevEnv
	}
	return nil
}

// lightingStyleByName resolves a style name to its preset id via the renderer gallery (the
// single source of style names), case-sensitively.
func lightingStyleByName(name string) (renderer.LightingStyleID, bool) {
	for _, opt := range renderer.LightingStyleGallery() {
		if opt.Name == name {
			return opt.Style, true
		}
	}
	return renderer.LightingDefault, false
}

// Environment returns the active image-based-lighting environment.
func (s *Session) Environment() renderer.Environment { return s.lighting.Environment }

// SetEnvironment sets the active environment (preset or loaded file) on the live rig.
func (s *Session) SetEnvironment(e renderer.Environment) { s.lighting.Environment = e }

// ShadowSettings returns the active shadow settings.
func (s *Session) ShadowSettings() renderer.ShadowSettings { return s.lighting.Shadows }

// SetShadowSettings sets the active shadow settings on the live rig.
func (s *Session) SetShadowSettings(sh renderer.ShadowSettings) { s.lighting.Shadows = sh }

// Exposure/Brightness/Ambience read the active rig's global tone controls; the Set* forms edit
// them in place (the Lighting settings panel's sliders).
func (s *Session) Exposure() float32     { return s.lighting.Exposure }
func (s *Session) SetExposure(v float32) { s.lighting.Exposure = v }
func (s *Session) Brightness() float32   { return s.lighting.Brightness }
func (s *Session) SetBrightness(v float32) {
	s.lighting.Brightness = v
}
func (s *Session) Ambience() float32     { return s.lighting.Ambience }
func (s *Session) SetAmbience(v float32) { s.lighting.Ambience = v }

// OpenLightingPanel / CloseLightingPanel / LightingPanelOpen drive the Lighting settings panel.
func (s *Session) OpenLightingPanel()      { s.lightingPanelOpen = true }
func (s *Session) CloseLightingPanel()     { s.lightingPanelOpen = false }
func (s *Session) LightingPanelOpen() bool { return s.lightingPanelOpen }

// RequestLoadEnvironment flags that the user asked to load an HDR file; the head opens its file
// dialog and TakeLoadEnvironmentRequest consumes the flag (one-shot, so the dialog opens once).
func (s *Session) RequestLoadEnvironment() { s.loadEnvRequested = true }

// TakeLoadEnvironmentRequest returns and clears the pending load-HDR request.
func (s *Session) TakeLoadEnvironmentRequest() bool {
	req := s.loadEnvRequested
	s.loadEnvRequested = false
	return req
}

// LoadEnvironmentFile sets a user HDR file as the active environment (shown as the background).
// It is what the head calls when the load-HDR file dialog is confirmed.
func (s *Session) LoadEnvironmentFile(path string) {
	s.SetEnvironment(renderer.Environment{FilePath: path, Intensity: 1, ShowImage: true})
}

// Lights returns the live rig's lights.
func (s *Session) Lights() []renderer.SceneLight { return s.lighting.Lights }

// AddLight appends a new light of the given kind with neutral defaults (white, intensity 1,
// on), returning it; it is a no-op past [renderer.MaxSceneLights] (returning the last light).
func (s *Session) AddLight(kind renderer.LightKind) (renderer.SceneLight, error) {
	if len(s.lighting.Lights) >= renderer.MaxSceneLights {
		return renderer.SceneLight{}, fmt.Errorf("app: cannot add light, at the %d-light maximum",
			renderer.MaxSceneLights)
	}
	l := renderer.SceneLight{
		Kind:      kind,
		Direction: [3]float32{0, 0, 1},
		Color:     [3]float32{1, 1, 1},
		Intensity: 1,
		On:        true,
	}
	s.lighting.Lights = append(s.lighting.Lights, l)
	return l, nil
}

// SetLight replaces the light at index, erroring on an out-of-range index (so a bad request is
// rejected rather than panicking).
func (s *Session) SetLight(index int, l renderer.SceneLight) error {
	if index < 0 || index >= len(s.lighting.Lights) {
		return fmt.Errorf("app: light index %d out of range [0,%d)", index, len(s.lighting.Lights))
	}
	s.lighting.Lights[index] = l
	return nil
}

// LightKindForDefinition maps the public LightDefinitionTypeEnum onto the renderer's LightKind.
func LightKindForDefinition(d LightDefinitionTypeEnum) renderer.LightKind {
	switch d {
	case types.PointLight:
		return renderer.PointLight
	case types.SpotLight:
		return renderer.SpotLight
	default:
		return renderer.DirectionalLight
	}
}

// DefinitionForLightKind maps the renderer's LightKind back onto the public enum.
func DefinitionForLightKind(k renderer.LightKind) LightDefinitionTypeEnum {
	switch k {
	case renderer.PointLight:
		return types.PointLight
	case renderer.SpotLight:
		return types.SpotLight
	default:
		return types.DirectionalLight
	}
}

// EnvironmentPresetByName resolves a built-in environment name to its preset via the renderer
// gallery (the single source of preset names), case-sensitively.
func EnvironmentPresetByName(name string) (renderer.EnvironmentPreset, bool) {
	for _, opt := range renderer.EnvironmentGallery() {
		if opt.Name == name {
			return opt.Preset, true
		}
	}
	return renderer.EnvNone, false
}

// GroundShadowForSettings maps the renderer's ground-shadow flags back onto the public
// GroundShadowEnum (None / Ground / X-Ray).
func GroundShadowForSettings(sh renderer.ShadowSettings) GroundShadowEnum {
	switch {
	case !sh.GroundShadows:
		return types.NoGroundShadow
	case sh.GroundXRay:
		return types.XRayGroundShadow
	default:
		return types.GroundShadow
	}
}

// ApplyGroundShadow sets the renderer's ground-shadow flags from the public GroundShadowEnum.
func ApplyGroundShadow(sh *renderer.ShadowSettings, g GroundShadowEnum) {
	sh.GroundShadows = g != types.NoGroundShadow
	sh.GroundXRay = g == types.XRayGroundShadow
}
