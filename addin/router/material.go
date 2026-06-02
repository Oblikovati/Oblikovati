// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/material"
)

// appearanceInfo marshals a model appearance into its wire DTO.
func appearanceInfo(a *material.Appearance) wire.AppearanceInfo {
	return wire.AppearanceInfo{
		ID: a.ID(), DisplayName: a.DisplayName(), Source: string(a.Source()),
		Albedo: a.Albedo().Hex(), Metallic: a.Metallic(), Roughness: a.Roughness(),
		Emissive: a.Emissive().Hex(), Opacity: a.Opacity(),
	}
}

// materialInfo marshals a model material into its wire DTO.
func materialInfo(m *material.Material) wire.MaterialInfo {
	return wire.MaterialInfo{
		ID: m.ID(), DisplayName: m.DisplayName(), Source: string(m.Source()),
		Density: m.Density(), Mechanical: m.Mechanical(), Thermal: m.Thermal(),
		Electrical: m.Electrical(), AppearanceID: m.AppearanceID(),
	}
}

func listAppearances(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	apprs := s.Materials().Appearances()
	out := make([]wire.AppearanceInfo, len(apprs))
	for i, a := range apprs {
		out[i] = appearanceInfo(a)
	}
	return json.Marshal(wire.ListAppearancesResult{Appearances: out})
}

func listMaterials(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	mats := s.Materials().Materials()
	out := make([]wire.MaterialInfo, len(mats))
	for i, m := range mats {
		out[i] = materialInfo(m)
	}
	return json.Marshal(wire.ListMaterialsResult{Materials: out})
}

func getAppearance(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AssetRefArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	a, ok := s.Materials().Appearance(in.ID)
	if !ok {
		return nil, fmt.Errorf("appearances.get: unknown appearance %q", in.ID)
	}
	return json.Marshal(appearanceInfo(a))
}

func getMaterial(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AssetRefArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	m, ok := s.Materials().Material(in.ID)
	if !ok {
		return nil, fmt.Errorf("materials.get: unknown material %q", in.ID)
	}
	return json.Marshal(materialInfo(m))
}

func createAppearance(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.DuplicateAssetArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	a, err := s.DuplicateAppearance(in.BaseID, in.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(appearanceInfo(a))
}

func createMaterial(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.DuplicateAssetArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	m, err := s.DuplicateMaterial(in.BaseID, in.Name)
	if err != nil {
		return nil, err
	}
	return json.Marshal(materialInfo(m))
}

func updateAppearance(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AppearanceInfo
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	albedo, err := material.ParseColor(in.Albedo)
	if err != nil {
		return nil, fmt.Errorf("appearances.update: albedo: %w", err)
	}
	emissive, err := material.ParseColor(in.Emissive)
	if err != nil {
		return nil, fmt.Errorf("appearances.update: emissive: %w", err)
	}
	s.UpdateAppearance(in.ID, material.AppearanceSpec{
		DisplayName: in.DisplayName, Albedo: albedo, Metallic: in.Metallic,
		Roughness: in.Roughness, Emissive: emissive, Opacity: in.Opacity,
	})
	a, ok := s.Materials().Appearance(in.ID)
	if !ok {
		return nil, fmt.Errorf("appearances.update: unknown appearance %q", in.ID)
	}
	return json.Marshal(appearanceInfo(a))
}

func updateMaterial(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.MaterialInfo
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	s.UpdateMaterial(in.ID, material.MaterialSpec{
		DisplayName: in.DisplayName, Density: in.Density, Mechanical: in.Mechanical,
		Thermal: in.Thermal, Electrical: in.Electrical, AppearanceID: in.AppearanceID,
	})
	m, ok := s.Materials().Material(in.ID)
	if !ok {
		return nil, fmt.Errorf("materials.update: unknown material %q", in.ID)
	}
	return json.Marshal(materialInfo(m))
}

func assignMaterial(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AssignMaterialArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := s.AssignMaterial(in.BodyKey, in.MaterialID); err != nil {
		return nil, err
	}
	return ok()
}

func assignAppearance(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.AssignAppearanceArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := s.AssignAppearance(in.Scope, in.Key, in.AppearanceID); err != nil {
		return nil, err
	}
	return ok()
}

func physicalProperties(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	props, ok := s.PhysicalProperties()
	if !ok {
		return nil, fmt.Errorf("model.physicalProperties: no active part")
	}
	return json.Marshal(props)
}
