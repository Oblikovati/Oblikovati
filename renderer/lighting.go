// SPDX-License-Identifier: GPL-2.0-only

package renderer

// LightKind is the emission shape of a scene light — the renderer-internal mirror of
// Inventor's LightDefinitionTypeEnum (Directional/Point/Spot). The renderer owns its own
// enum; the app maps the public type onto it (ADR-0022 §6, ADR-0026 §2).
type LightKind uint8

const (
	// DirectionalLight emits parallel rays (a sun); only Direction matters.
	DirectionalLight LightKind = iota
	// PointLight emits from Position in all directions with distance attenuation.
	PointLight
	// SpotLight emits from Position within a cone about Direction (inner/outer angles).
	SpotLight
)

// MaxSceneLights is the fixed upper bound on lights drawn in one frame — the size of the GPU
// scene-UBO light array (ADR-0026 §1). CAD scenes rarely need more; the app clamps to it.
const MaxSceneLights = 8

// SceneLight is one light feeding the Realistic shader. Direction is the unit vector from a
// lit surface toward the light (normalized by the shader). Color is linear RGB scaled by
// Intensity. Position and Attenuation (constant, linear, quadratic) apply to Point/Spot
// kinds; SpotInner/SpotOuter are the cone half-angles in radians for SpotLight.
type SceneLight struct {
	Kind        LightKind
	Direction   [3]float32
	Position    [3]float32
	Color       [3]float32
	Intensity   float32
	On          bool
	SpotInner   float32
	SpotOuter   float32
	Attenuation [3]float32
}

// ShadowSettings controls how the scene casts and receives shadows — the renderer-internal
// mirror of Inventor's View shadow toggles plus the LightingStyle shadow properties
// (ADR-0026 §6). Density is the shadow darkness [0,1]; Softness is the edge blur [0,1].
type ShadowSettings struct {
	GroundShadows  bool
	GroundXRay     bool // ground shadow is the see-through (X-ray) style (Inventor kXRayGroundShadow)
	ObjectShadows  bool
	AmbientShadows bool
	Density        float32
	Softness       float32
}

// SceneLighting is the complete per-frame lighting environment the renderer hands the native
// layer: the active lights, the global ambient/brightness/exposure controls, the IBL
// [Environment], and the [ShadowSettings]. It is pure data (ADR-0014); the app resolves it
// from the active lighting style and the GPU consumes it (scene UBO + cubemaps + shadow map).
type SceneLighting struct {
	Lights      []SceneLight
	Ambience    float32
	Brightness  float32
	Exposure    float32
	Environment Environment
	Shadows     ShadowSettings
}

// ActiveLights returns the lights that are On, clamped to [MaxSceneLights]. The native layer
// uploads exactly these into the scene UBO, so a style with more lights than the array can
// hold is truncated deterministically rather than overflowing the GPU buffer.
func (s SceneLighting) ActiveLights() []SceneLight {
	on := make([]SceneLight, 0, len(s.Lights))
	for _, l := range s.Lights {
		if l.On {
			on = append(on, l)
		}
		if len(on) == MaxSceneLights {
			break
		}
	}
	return on
}
