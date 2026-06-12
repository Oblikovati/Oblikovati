// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/renderer"
)

// getLightingStyle returns the active lighting style's global controls
// (wire.MethodLightingGetStyle).
func getLightingStyle(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return marshalLightingStyle(s)
}

// setLightingStyle activates a lighting style by name and echoes it, erroring on an unknown
// name (wire.MethodLightingSetStyle).
func setLightingStyle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetLightingStyleArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.SetLightingStyle(a.Name); err != nil {
		return nil, err
	}
	return marshalLightingStyle(s)
}

// listLightingStyles enumerates every lighting style in gallery order, flagging the active one
// (wire.MethodLightingListStyles).
func listLightingStyles(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.LightingStyleName()
	gallery := renderer.LightingStyleGallery()
	out := make([]wire.LightingStyleInfo, len(gallery))
	for i, opt := range gallery {
		out[i] = wire.LightingStyleInfo{Name: opt.Name, Active: opt.Name == active}
	}
	return json.Marshal(wire.LightingStyleListResult{Styles: out})
}

// listLights returns the active rig's lights (wire.MethodLightingListLights).
func listLights(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	lights := s.Lights()
	out := make([]wire.LightInfo, len(lights))
	for i, l := range lights {
		out[i] = lightInfo(i, l)
	}
	return json.Marshal(wire.LightListResult{Lights: out})
}

// addLight adds a light of the requested emission shape and returns it
// (wire.MethodLightingAddLight).
func addLight(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.AddLightArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	l, err := s.AddLight(app.LightKindForDefinition(a.DefinitionType))
	if err != nil {
		return nil, err
	}
	return json.Marshal(lightInfo(len(s.Lights())-1, l))
}

// setLight replaces the light at the given index and returns it (wire.MethodLightingSetLight).
func setLight(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetLightArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if err := s.SetLight(a.Index, sceneLight(a.Light)); err != nil {
		return nil, err
	}
	return json.Marshal(lightInfo(a.Index, s.Lights()[a.Index]))
}

// getShadows returns the viewport's shadow settings (wire.MethodViewGetShadows).
func getShadows(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(shadowSettings(s.ShadowSettings()))
}

// setShadows applies the viewport's shadow settings and echoes them (wire.MethodViewSetShadows).
func setShadows(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.ShadowSettings
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	s.SetShadowSettings(rendererShadows(a))
	return json.Marshal(shadowSettings(s.ShadowSettings()))
}

// getEnvironment returns the active environment (wire.MethodEnvironmentGet).
func getEnvironment(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(environmentView(s.Environment()))
}

// setEnvironment activates a built-in preset with display parameters, erroring on an unknown
// preset name (wire.MethodEnvironmentSet).
func setEnvironment(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.SetEnvironmentArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	preset, ok := app.EnvironmentPresetByName(a.Preset)
	if !ok {
		return nil, fmt.Errorf("router: unknown environment preset %q", a.Preset)
	}
	s.SetEnvironment(renderer.Environment{
		Preset:    preset,
		Rotation:  float32(a.Rotation),
		Intensity: float32(a.Intensity),
		ShowImage: a.ShowImage,
	})
	return json.Marshal(environmentView(s.Environment()))
}

// listEnvironmentPresets enumerates the built-in environments, flagging the active one
// (wire.MethodEnvironmentListPresets).
func listEnvironmentPresets(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.Environment()
	gallery := renderer.EnvironmentGallery()
	out := make([]wire.EnvironmentPresetInfo, len(gallery))
	for i, opt := range gallery {
		isActive := active.FilePath == "" && active.Preset == opt.Preset
		out[i] = wire.EnvironmentPresetInfo{Name: opt.Name, Active: isActive}
	}
	return json.Marshal(wire.EnvironmentPresetListResult{Presets: out})
}

// loadEnvironmentImage sets a user HDR file as the environment (wire.MethodEnvironmentLoadImage).
func loadEnvironmentImage(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var a wire.LoadEnvironmentImageArgs
	if err := decode(args, &a); err != nil {
		return nil, err
	}
	if a.FilePath == "" {
		return nil, fmt.Errorf("router: environment.loadImage requires a non-empty filePath")
	}
	env := s.Environment()
	env.FilePath = a.FilePath
	env.ShowImage = true
	if env.Intensity == 0 {
		env.Intensity = 1
	}
	s.SetEnvironment(env)
	return json.Marshal(environmentView(s.Environment()))
}

// marshalLightingStyle encodes the active style's controls as the shared LightingStyleView.
func marshalLightingStyle(s *app.Session) (json.RawMessage, error) {
	style := app.LightingStyleOf(s.LightingStyleName(), s.SceneLighting())
	return json.Marshal(wire.LightingStyleView{
		Name:           style.Name(),
		StyleType:      style.StyleType(),
		Ambience:       style.Ambience(),
		Brightness:     style.Brightness(),
		Exposure:       style.Exposure(),
		IBLBrightness:  style.ImageBasedLightingBrightness(),
		IBLRotation:    style.ImageBasedLightingRotation(),
		ShadowDensity:  style.ShadowDensity(),
		ShadowSoftness: style.ShadowSoftness(),
		ShadowDir:      style.ShadowDirection(),
	})
}

// lightInfo converts a renderer light at index into the wire DTO.
func lightInfo(index int, l renderer.SceneLight) wire.LightInfo {
	return wire.LightInfo{
		Index:               index,
		LightType:           types.ModelSpaceLight,
		LightDefinitionType: app.DefinitionForLightKind(l.Kind),
		On:                  l.On,
		Color:               types.Rgba{R: l.Color[0], G: l.Color[1], B: l.Color[2], A: 1},
		Intensity:           float64(l.Intensity),
		Direction:           pointVec(l.Direction),
		Position:            pointPos(l.Position),
		SpotInnerAngle:      float64(l.SpotInner),
		SpotOuterAngle:      float64(l.SpotOuter),
		Attenuation:         vec64(l.Attenuation),
	}
}

// sceneLight converts a wire light DTO into a renderer light.
func sceneLight(in wire.LightInfo) renderer.SceneLight {
	return renderer.SceneLight{
		Kind:        app.LightKindForDefinition(in.LightDefinitionType),
		Direction:   [3]float32{float32(in.Direction.X), float32(in.Direction.Y), float32(in.Direction.Z)},
		Position:    [3]float32{float32(in.Position.X), float32(in.Position.Y), float32(in.Position.Z)},
		Color:       [3]float32{in.Color.R, in.Color.G, in.Color.B},
		Intensity:   float32(in.Intensity),
		On:          in.On,
		SpotInner:   float32(in.SpotInnerAngle),
		SpotOuter:   float32(in.SpotOuterAngle),
		Attenuation: vec32(in.Attenuation),
	}
}

// shadowSettings converts renderer shadow flags into the wire DTO (folding the ground flags
// into the public GroundShadowEnum).
func shadowSettings(sh renderer.ShadowSettings) wire.ShadowSettings {
	return wire.ShadowSettings{
		GroundShadow:   app.GroundShadowForSettings(sh),
		ObjectShadows:  sh.ObjectShadows,
		AmbientShadows: sh.AmbientShadows,
		Density:        float64(sh.Density),
		Softness:       float64(sh.Softness),
	}
}

// rendererShadows converts a wire shadow DTO into renderer flags.
func rendererShadows(in wire.ShadowSettings) renderer.ShadowSettings {
	sh := renderer.ShadowSettings{
		ObjectShadows:  in.ObjectShadows,
		AmbientShadows: in.AmbientShadows,
		Density:        float32(in.Density),
		Softness:       float32(in.Softness),
	}
	app.ApplyGroundShadow(&sh, in.GroundShadow)
	return sh
}

// environmentView converts a renderer environment into the wire DTO.
func environmentView(e renderer.Environment) wire.EnvironmentView {
	preset := ""
	if e.FilePath == "" {
		preset = e.Preset.String()
	}
	return wire.EnvironmentView{
		Preset:    preset,
		FilePath:  e.FilePath,
		Rotation:  float64(e.Rotation),
		Intensity: float64(e.Intensity),
		ShowImage: e.ShowImage,
	}
}

func vec64(v [3]float32) [3]float64 { return [3]float64{float64(v[0]), float64(v[1]), float64(v[2])} }

// pointVec / pointPos lift the renderer's float32 triples into the typed wire
// geometry (M01-F05).
func pointVec(v [3]float32) types.Vector {
	return types.Vector{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2])}
}

func pointPos(v [3]float32) types.Point {
	return types.Point{X: float64(v[0]), Y: float64(v[1]), Z: float64(v[2])}
}

func vec32(v [3]float64) [3]float32 { return [3]float32{float32(v[0]), float32(v[1]), float32(v[2])} }
