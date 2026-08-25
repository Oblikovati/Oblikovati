// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/material"
)

// appearanceInfo marshals a model appearance into its wire DTO.
func appearanceInfo(a *material.Appearance) wire.AppearanceInfo {
	return wire.AppearanceInfo{
		ID: a.ID(), DisplayName: a.DisplayName(), Source: string(a.Source()),
		Base: a.Base(), Specular: a.Specular(), Transmission: a.Transmission(),
		Subsurface: a.Subsurface(), Coat: a.Coat(), Fuzz: a.Fuzz(), ThinFilm: a.ThinFilm(),
		Emission: a.Emission(), Geometry: a.Geometry(),
	}
}

// materialInfo marshals a model material into its wire DTO.
func materialInfo(m *material.Material) wire.MaterialInfo {
	return wire.MaterialInfo{
		ID: m.ID(), DisplayName: m.DisplayName(), Source: string(m.Source()),
		Density: m.Density(), Mechanical: m.Mechanical(), Thermal: m.Thermal(),
		Electrical: m.Electrical(), Magnetic: m.Magnetic(), IsotropyClass: string(m.IsotropyClass()),
		Anisotropic: m.Anisotropic(), AppearanceID: m.AppearanceID(),
	}
}

// listAppearances reads no args and no active-model context (the appearance library is
// session-scoped), so it stays a raw handler.
func listAppearances(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	apprs := s.Materials().Appearances()
	out := make([]wire.AppearanceInfo, len(apprs))
	for i, a := range apprs {
		out[i] = appearanceInfo(a)
	}
	return json.Marshal(wire.ListAppearancesResult{Appearances: out})
}

// listMaterials reads no args and no active-model context, so it stays a raw handler.
func listMaterials(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	mats := s.Materials().Materials()
	out := make([]wire.MaterialInfo, len(mats))
	for i, m := range mats {
		out[i] = materialInfo(m)
	}
	return json.Marshal(wire.ListMaterialsResult{Materials: out})
}

func getAppearance(s *app.Session, in wire.AssetRefArgs) (wire.AppearanceInfo, error) {
	a, ok := s.Materials().Appearance(in.ID)
	if !ok {
		return wire.AppearanceInfo{}, fmt.Errorf("appearances.get: unknown appearance %q", in.ID)
	}
	return appearanceInfo(a), nil
}

func getMaterial(s *app.Session, in wire.AssetRefArgs) (wire.MaterialInfo, error) {
	m, ok := s.Materials().Material(in.ID)
	if !ok {
		return wire.MaterialInfo{}, fmt.Errorf("materials.get: unknown material %q", in.ID)
	}
	return materialInfo(m), nil
}

func createAppearance(s *app.Session, in wire.CreateAppearanceArgs) (wire.AppearanceInfo, error) {
	a, err := s.DuplicateAppearance(in.BaseID, in.Name)
	if err != nil {
		return wire.AppearanceInfo{}, err
	}
	return appearanceInfo(a), nil
}

func createMaterial(s *app.Session, in wire.DuplicateAssetArgs) (wire.MaterialInfo, error) {
	m, err := s.DuplicateMaterial(in.BaseID, in.Name)
	if err != nil {
		return wire.MaterialInfo{}, err
	}
	return materialInfo(m), nil
}

func updateAppearance(s *app.Session, in wire.UpdateAppearanceArgs) (wire.AppearanceInfo, error) {
	s.UpdateAppearance(in.ID, material.AppearanceSpec{
		DisplayName: in.DisplayName,
		Base:        in.Base, Specular: in.Specular, Transmission: in.Transmission,
		Subsurface: in.Subsurface, Coat: in.Coat, Fuzz: in.Fuzz, ThinFilm: in.ThinFilm,
		Emission: in.Emission, Geometry: in.Geometry,
	})
	a, ok := s.Materials().Appearance(in.ID)
	if !ok {
		return wire.AppearanceInfo{}, fmt.Errorf("appearances.update: unknown appearance %q", in.ID)
	}
	return appearanceInfo(a), nil
}

func updateMaterial(s *app.Session, in wire.MaterialInfo) (wire.MaterialInfo, error) {
	s.UpdateMaterial(in.ID, material.MaterialSpec{
		DisplayName: in.DisplayName, Density: in.Density, Mechanical: in.Mechanical,
		Thermal: in.Thermal, Electrical: in.Electrical, Magnetic: in.Magnetic,
		IsotropyClass: material.IsotropyClass(in.IsotropyClass), Anisotropic: in.Anisotropic,
		AppearanceID: in.AppearanceID,
	})
	m, ok := s.Materials().Material(in.ID)
	if !ok {
		return wire.MaterialInfo{}, fmt.Errorf("materials.update: unknown material %q", in.ID)
	}
	return materialInfo(m), nil
}

func assignMaterial(s *app.Session, in wire.AssignMaterialArgs) (wire.OKResult, error) {
	if err := s.AssignMaterial(in.BodyKey, in.MaterialID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

func assignAppearance(s *app.Session, in wire.AssignAppearanceArgs) (wire.OKResult, error) {
	if err := s.AssignAppearance(in.Scope, in.Key, in.AppearanceID); err != nil {
		return wire.OKResult{}, err
	}
	return wire.OKResult{OK: true}, nil
}

// physicalProperties reads no args; it resolves the active part's aggregate physical properties or
// reports there is none, so it stays a raw handler.
func physicalProperties(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	props, ok := s.PhysicalProperties()
	if !ok {
		return nil, fmt.Errorf("model.physicalProperties: no active part")
	}
	return json.Marshal(props)
}
