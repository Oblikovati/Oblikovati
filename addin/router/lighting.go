// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// getLightingStyle returns the active lighting style's global controls
// (wire.MethodLightingGetStyle). No args and no active-model context, so it stays a raw handler.
func getLightingStyle(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(lightingStyleView(s))
}

// setLightingStyle activates a lighting style by name and echoes it, erroring on an unknown
// name (wire.MethodLightingSetStyle).
func setLightingStyle(s *app.Session, in wire.SetLightingStyleArgs) (wire.LightingStyleView, error) {
	if err := s.SetLightingStyle(in.Name); err != nil {
		return wire.LightingStyleView{}, err
	}
	return lightingStyleView(s), nil
}

// listLightingStyles enumerates every lighting style in gallery order, flagging the active one
// (wire.MethodLightingListStyles).
func listLightingStyles(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.LightingStyleName()
	gallery := app.LightingStyleGallery()
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
func addLight(s *app.Session, in wire.AddLightArgs) (wire.LightInfo, error) {
	l, err := s.AddLight(in.DefinitionType)
	if err != nil {
		return wire.LightInfo{}, err
	}
	return lightInfo(len(s.Lights())-1, l), nil
}

// setLight replaces the light at the given index and returns it (wire.MethodLightingSetLight).
func setLight(s *app.Session, in wire.SetLightArgs) (wire.LightInfo, error) {
	if err := s.SetLight(in.Index, appLight(in.Light)); err != nil {
		return wire.LightInfo{}, err
	}
	return lightInfo(in.Index, s.Lights()[in.Index]), nil
}

// getShadows returns the viewport's shadow settings (wire.MethodViewGetShadows).
func getShadows(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(shadowSettings(s.ShadowSettings()))
}

// setShadows applies the viewport's shadow settings and echoes them (wire.MethodViewSetShadows).
func setShadows(s *app.Session, in wire.ShadowSettings) (wire.ShadowSettings, error) {
	s.SetShadowSettings(shadowRigFromWire(in))
	return shadowSettings(s.ShadowSettings()), nil
}

// getEnvironment returns the active environment (wire.MethodEnvironmentGet).
func getEnvironment(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(environmentView(s.Environment()))
}

// setEnvironment activates a built-in preset with display parameters, erroring on an unknown
// preset name (wire.MethodEnvironmentSet).
func setEnvironment(s *app.Session, in wire.SetEnvironmentArgs) (wire.EnvironmentView, error) {
	if _, ok := app.EnvironmentPresetByName(in.Preset); !ok {
		return wire.EnvironmentView{}, fmt.Errorf("router: unknown environment preset %q", in.Preset)
	}
	s.SetEnvironment(app.EnvironmentState{
		Preset:    in.Preset,
		Rotation:  float32(in.Rotation),
		Intensity: float32(in.Intensity),
		ShowImage: in.ShowImage,
	})
	return environmentView(s.Environment()), nil
}

// listEnvironmentPresets enumerates the built-in environments, flagging the active one
// (wire.MethodEnvironmentListPresets).
func listEnvironmentPresets(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	active := s.Environment()
	gallery := app.EnvironmentGallery()
	out := make([]wire.EnvironmentPresetInfo, len(gallery))
	for i, opt := range gallery {
		isActive := active.FilePath == "" && active.Preset == opt.Name
		out[i] = wire.EnvironmentPresetInfo{Name: opt.Name, Active: isActive}
	}
	return json.Marshal(wire.EnvironmentPresetListResult{Presets: out})
}

// loadEnvironmentImage sets a user HDR file as the environment (wire.MethodEnvironmentLoadImage).
func loadEnvironmentImage(s *app.Session, in wire.LoadEnvironmentImageArgs) (wire.EnvironmentView, error) {
	if in.FilePath == "" {
		return wire.EnvironmentView{}, fmt.Errorf("router: environment.loadImage requires a non-empty filePath")
	}
	env := s.Environment()
	env.FilePath = in.FilePath
	env.ShowImage = true
	if env.Intensity == 0 {
		env.Intensity = 1
	}
	s.SetEnvironment(env)
	return environmentView(s.Environment()), nil
}

// lightingStyleView builds the active style's controls as the shared LightingStyleView.
func lightingStyleView(s *app.Session) wire.LightingStyleView {
	style := app.LightingStyleOf(s.LightingStyleName(), s.SceneLighting())
	return wire.LightingStyleView{
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
	}
}

// lightInfo converts an app light value at index into the wire DTO.
func lightInfo(index int, l app.Light) wire.LightInfo {
	return wire.LightInfo{
		Index:               index,
		LightType:           types.ModelSpaceLight,
		LightDefinitionType: l.Definition,
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

// appLight converts a wire light DTO into the app light value.
func appLight(in wire.LightInfo) app.Light {
	return app.Light{
		Definition:  in.LightDefinitionType,
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

// shadowSettings converts an app shadow rig into the wire DTO (folding the ground flags into
// the public GroundShadowEnum).
func shadowSettings(sh app.ShadowRig) wire.ShadowSettings {
	return wire.ShadowSettings{
		GroundShadow:   app.GroundShadowForSettings(sh),
		ObjectShadows:  sh.ObjectShadows,
		AmbientShadows: sh.AmbientShadows,
		Density:        float64(sh.Density),
		Softness:       float64(sh.Softness),
	}
}

// shadowRigFromWire converts a wire shadow DTO into the app shadow rig.
func shadowRigFromWire(in wire.ShadowSettings) app.ShadowRig {
	sh := app.ShadowRig{
		ObjectShadows:  in.ObjectShadows,
		AmbientShadows: in.AmbientShadows,
		Density:        float32(in.Density),
		Softness:       float32(in.Softness),
	}
	app.ApplyGroundShadow(&sh, in.GroundShadow)
	return sh
}

// environmentView converts an app environment into the wire DTO.
func environmentView(e app.EnvironmentState) wire.EnvironmentView {
	preset := ""
	if e.FilePath == "" {
		preset = e.Preset
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
