// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/material"
)

// openPBRAppearanceInfo marshals a model OpenPBR appearance into its wire DTO. Mirrors
// [appearanceInfo] for the full OpenPBR lobe set (M45, ADR-0053).
func openPBRAppearanceInfo(a *material.OpenPBRAppearance) wire.OpenPBRAppearanceInfo {
	return wire.OpenPBRAppearanceInfo{
		ID: a.ID(), DisplayName: a.DisplayName(), Source: string(a.Source()),
		Base: a.Base(), Specular: a.Specular(), Transmission: a.Transmission(),
		Subsurface: a.Subsurface(), Coat: a.Coat(), Fuzz: a.Fuzz(), ThinFilm: a.ThinFilm(),
		Emission: a.Emission(), Geometry: a.Geometry(),
	}
}

// listOpenPBRAppearances reads no args and no active-model context (the appearance
// library is session-scoped), so it stays a raw handler.
func listOpenPBRAppearances(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	apprs := s.Materials().OpenPBRAppearances()
	out := make([]wire.OpenPBRAppearanceInfo, len(apprs))
	for i, a := range apprs {
		out[i] = openPBRAppearanceInfo(a)
	}
	return json.Marshal(wire.ListOpenPBRAppearancesResult{Appearances: out})
}

func getOpenPBRAppearance(s *app.Session, in wire.AssetRefArgs) (wire.OpenPBRAppearanceInfo, error) {
	a, ok := s.Materials().OpenPBRAppearance(in.ID)
	if !ok {
		return wire.OpenPBRAppearanceInfo{}, fmt.Errorf("openpbrAppearances.get: unknown appearance %q", in.ID)
	}
	return openPBRAppearanceInfo(a), nil
}

func createOpenPBRAppearance(s *app.Session, in wire.CreateOpenPBRAppearanceArgs) (wire.OpenPBRAppearanceInfo, error) {
	a, err := s.DuplicateOpenPBRAppearance(in.BaseID, in.Name)
	if err != nil {
		return wire.OpenPBRAppearanceInfo{}, err
	}
	return openPBRAppearanceInfo(a), nil
}

func updateOpenPBRAppearance(s *app.Session, in wire.UpdateOpenPBRAppearanceArgs) (wire.OpenPBRAppearanceInfo, error) {
	s.UpdateOpenPBRAppearance(in.ID, material.OpenPBRAppearanceSpec{
		DisplayName: in.DisplayName,
		Base:        in.Base, Specular: in.Specular, Transmission: in.Transmission,
		Subsurface: in.Subsurface, Coat: in.Coat, Fuzz: in.Fuzz, ThinFilm: in.ThinFilm,
		Emission: in.Emission, Geometry: in.Geometry,
	})
	a, ok := s.Materials().OpenPBRAppearance(in.ID)
	if !ok {
		return wire.OpenPBRAppearanceInfo{}, fmt.Errorf("openpbrAppearances.update: unknown appearance %q", in.ID)
	}
	return openPBRAppearanceInfo(a), nil
}
