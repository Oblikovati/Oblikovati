// SPDX-License-Identifier: GPL-2.0-only

package renderer

// LightingStyleID selects a built-in lighting rig — the renderer-internal mirror of an
// Inventor LightingStyle preset. Like [VisualStyle], every value resolves through a single
// table ([SceneLightingFor]); the app maps the public lighting-style enum onto it
// (ADR-0026 §1,§2).
type LightingStyleID uint8

const (
	// LightingDefault reproduces the pre-IBL hardcoded headlight + analytic ambient so an
	// un-configured Realistic frame is unchanged (ADR-0026 §7). It is the zero value.
	LightingDefault LightingStyleID = iota
	// LightingCool is a balanced key/fill rig with a cool (bluish) cast.
	LightingCool
	// LightingWarm is the same rig with a warm (amber) cast.
	LightingWarm
	// LightingThreePoint is a studio key/fill/back rig for even product lighting.
	LightingThreePoint
	// LightingSun is a single strong directional sun with low ambient (hard shadows).
	LightingSun
	// LightingOutdoors pairs a sun with sky fill and the Outdoors environment for IBL.
	LightingOutdoors
)

// directional builds an On directional light from a (not-necessarily-unit) direction, a
// linear RGB color, and an intensity — the common case the preset rigs are made of.
func directional(dir, color [3]float32, intensity float32) SceneLight {
	return SceneLight{
		Kind:      DirectionalLight,
		Direction: dir,
		Color:     color,
		Intensity: intensity,
		On:        true,
	}
}

// defaultShadows is the shadow rig shared by the soft studio styles: object + ambient
// shadows at moderate density/softness, no hard ground shadow.
var defaultShadows = ShadowSettings{
	ObjectShadows:  true,
	AmbientShadows: true,
	Density:        0.5,
	Softness:       0.5,
}

// lightingStyleSpec is one row of the resolver table: a style id, its label, and a builder
// that returns a fresh [SceneLighting] (a function so each call yields an independent slice
// the caller may mutate). Builders stay short by composing [directional].
type lightingStyleSpec struct {
	id    LightingStyleID
	name  string
	build func() SceneLighting
}

// white is the neutral light color the casted rigs tint away from.
var white = [3]float32{1, 1, 1}

// lightingStyleTable defines every lighting style exactly once, in gallery order. Adding a
// style is one row here plus an enum value above — there is no default fall-through for a
// known id (see TestSceneLightingForIsTotal), mirroring the display-mode resolver.
var lightingStyleTable = []lightingStyleSpec{
	{LightingDefault, "Default", func() SceneLighting {
		// The pre-IBL constants from mesh.frag: one headlight at (0.4,0.6,0.8), sun ≈ 3,
		// ambient 0.18, no environment or shadows (ADR-0026 §7).
		return SceneLighting{
			Lights:     []SceneLight{directional([3]float32{0.4, 0.6, 0.8}, white, 3)},
			Ambience:   0.18,
			Brightness: 1,
			Exposure:   1,
		}
	}},
	{LightingCool, "Cool Light", func() SceneLighting {
		return SceneLighting{
			Lights: []SceneLight{
				directional([3]float32{0.5, 0.7, 0.8}, [3]float32{0.85, 0.92, 1.0}, 2.6),
				directional([3]float32{-0.6, 0.3, 0.5}, [3]float32{0.7, 0.78, 0.9}, 1.0),
			},
			Ambience: 0.22, Brightness: 1, Exposure: 1, Shadows: defaultShadows,
		}
	}},
	{LightingWarm, "Warm Light", func() SceneLighting {
		return SceneLighting{
			Lights: []SceneLight{
				directional([3]float32{0.5, 0.7, 0.8}, [3]float32{1.0, 0.95, 0.85}, 2.6),
				directional([3]float32{-0.6, 0.3, 0.5}, [3]float32{0.95, 0.85, 0.7}, 1.0),
			},
			Ambience: 0.22, Brightness: 1, Exposure: 1, Shadows: defaultShadows,
		}
	}},
	{LightingThreePoint, "Three Point", func() SceneLighting {
		return SceneLighting{
			Lights: []SceneLight{
				directional([3]float32{0.5, 0.6, 0.9}, white, 2.8),   // key
				directional([3]float32{-0.7, 0.2, 0.6}, white, 1.2),  // fill
				directional([3]float32{0.1, -0.8, -0.6}, white, 1.5), // back/rim
			},
			Ambience: 0.16, Brightness: 1, Exposure: 1, Shadows: defaultShadows,
		}
	}},
	{LightingSun, "Sun", func() SceneLighting {
		return SceneLighting{
			Lights:   []SceneLight{directional([3]float32{0.6, 0.8, 0.5}, [3]float32{1.0, 0.97, 0.9}, 4.0)},
			Ambience: 0.08, Brightness: 1, Exposure: 1,
			Shadows: ShadowSettings{GroundShadows: true, ObjectShadows: true, Density: 0.7, Softness: 0.2},
		}
	}},
	{LightingOutdoors, "Outdoors", func() SceneLighting {
		return SceneLighting{
			Lights: []SceneLight{
				directional([3]float32{0.6, 0.8, 0.5}, [3]float32{1.0, 0.97, 0.9}, 3.2),
				directional([3]float32{-0.3, 0.2, 0.9}, [3]float32{0.7, 0.8, 1.0}, 0.8),
			},
			Ambience: 0.2, Brightness: 1, Exposure: 1,
			Environment: Environment{Preset: EnvOutdoors, Intensity: 1, ShowImage: true},
			Shadows:     ShadowSettings{GroundShadows: true, ObjectShadows: true, AmbientShadows: true, Density: 0.6, Softness: 0.35},
		}
	}},
}

// SceneLightingFor resolves a lighting style to its rig. It is total over every defined
// [LightingStyleID]; an unknown value falls back to [LightingDefault] so a corrupt id never
// yields a black, unlit scene (mirroring PassSetFor, ADR-0026 §1).
func SceneLightingFor(id LightingStyleID) SceneLighting {
	for _, sp := range lightingStyleTable {
		if sp.id == id {
			return sp.build()
		}
	}
	return lightingStyleTable[0].build()
}

// DefaultSceneLighting is the rig the renderer uses when the app supplies none — the
// [LightingDefault] headlight that reproduces the pre-IBL look (ADR-0023 §6, ADR-0026 §7).
func DefaultSceneLighting() SceneLighting { return SceneLightingFor(LightingDefault) }

// String returns the style's stable, user-facing name (the Lighting Style gallery label).
func (id LightingStyleID) String() string {
	for _, sp := range lightingStyleTable {
		if sp.id == id {
			return sp.name
		}
	}
	return "Default"
}

// LightingStyleOption pairs a style with its gallery label, for building the View-tab
// Lighting Style selection box.
type LightingStyleOption struct {
	Style LightingStyleID
	Name  string
}

// LightingStyleGallery returns every lighting style with its label, in gallery order — the
// source list for the lighting-style picker (the analogue of [VisualStyleGallery]).
func LightingStyleGallery() []LightingStyleOption {
	opts := make([]LightingStyleOption, len(lightingStyleTable))
	for i, sp := range lightingStyleTable {
		opts[i] = LightingStyleOption{Style: sp.id, Name: sp.name}
	}
	return opts
}
