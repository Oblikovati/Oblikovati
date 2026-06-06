// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati/api/contract"
	"oblikovati/api/types"
	"oblikovati/renderer"
)

// lightView adapts a renderer.SceneLight to the contract.Light interface so the in-process Go
// API exposes scene lights without leaking the renderer type (ADR-0018). Lights are
// model-space in our rig; the public color is the renderer's linear RGB as an opaque Rgba.
type lightView struct{ light renderer.SceneLight }

var _ contract.Light = lightView{}

// LightOf wraps a renderer light as a contract.Light.
func LightOf(l renderer.SceneLight) contract.Light { return lightView{l} }

func (v lightView) LightType() types.LightTypeEnum { return types.ModelSpaceLight }

func (v lightView) LightDefinitionType() types.LightDefinitionTypeEnum {
	return DefinitionForLightKind(v.light.Kind)
}

func (v lightView) On() bool { return v.light.On }

func (v lightView) Color() types.Rgba {
	c := v.light.Color
	return types.Rgba{R: c[0], G: c[1], B: c[2], A: 1}
}

func (v lightView) Intensity() float64 { return float64(v.light.Intensity) }

func (v lightView) Direction() [3]float64 { return vec3to64(v.light.Direction) }

func (v lightView) Position() [3]float64 { return vec3to64(v.light.Position) }

func (v lightView) SpotInnerAngle() float64 { return float64(v.light.SpotInner) }

func (v lightView) SpotOuterAngle() float64 { return float64(v.light.SpotOuter) }

func (v lightView) Attenuation() [3]float64 { return vec3to64(v.light.Attenuation) }

// lightingStyleView adapts the session's live rig + active style name to contract.LightingStyle.
type lightingStyleView struct {
	name string
	rig  renderer.SceneLighting
}

var _ contract.LightingStyle = lightingStyleView{}

// LightingStyleOf wraps the session's lighting state under the given style name as a
// contract.LightingStyle.
func LightingStyleOf(name string, rig renderer.SceneLighting) contract.LightingStyle {
	return lightingStyleView{name: name, rig: rig}
}

func (v lightingStyleView) Name() string { return v.name }

func (v lightingStyleView) StyleType() types.LightingStyleTypeEnum {
	if v.rig.Environment.IsActive() {
		return types.ImageBasedLightingStyle
	}
	return types.StandardLightingStyle
}

func (v lightingStyleView) Ambience() float64   { return float64(v.rig.Ambience) }
func (v lightingStyleView) Brightness() float64 { return float64(v.rig.Brightness) }
func (v lightingStyleView) Exposure() float64   { return float64(v.rig.Exposure) }

func (v lightingStyleView) Lights() []contract.Light {
	out := make([]contract.Light, len(v.rig.Lights))
	for i, l := range v.rig.Lights {
		out[i] = LightOf(l)
	}
	return out
}

func (v lightingStyleView) ImageBasedLightingBrightness() float64 {
	return float64(v.rig.Environment.Intensity)
}

func (v lightingStyleView) ImageBasedLightingRotation() float64 {
	return float64(v.rig.Environment.Rotation)
}

func (v lightingStyleView) ShadowDensity() float64  { return float64(v.rig.Shadows.Density) }
func (v lightingStyleView) ShadowSoftness() float64 { return float64(v.rig.Shadows.Softness) }

// ShadowDirection has no renderer-side source yet (Phase 5); report the environment-following
// default so the contract is total.
func (v lightingStyleView) ShadowDirection() types.ShadowDirectionEnum {
	return types.EnvironmentShadow
}

// vec3to64 widens a renderer [3]float32 to the contract's [3]float64.
func vec3to64(v [3]float32) [3]float64 {
	return [3]float64{float64(v[0]), float64(v[1]), float64(v[2])}
}
