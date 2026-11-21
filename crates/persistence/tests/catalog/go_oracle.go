// SPDX-License-Identifier: GPL-2.0-only
// Disposable exporter: run from the fixed Go checkout; it does not modify the baseline.
package main

import (
	"encoding/json"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/material"
	"os"
	"os/exec"
	"strings"
)

type oracleReceipt struct {
	License     string                `json:"license"`
	Baseline    string                `json:"baseline"`
	Materials   []wire.MaterialInfo   `json:"materials"`
	Appearances []wire.AppearanceInfo `json:"appearances"`
}

func main() {
	verifyBaseline()
	library := material.NewLibrary()
	receipt := oracleReceipt{License: "SPDX-License-Identifier: GPL-2.0-only", Baseline: "aa8297d1b0ac33c024f19e54821536d76e8a447a"}
	for _, asset := range library.Materials() {
		receipt.Materials = append(receipt.Materials, materialInfo(asset))
	}
	for _, asset := range library.Appearances() {
		receipt.Appearances = append(receipt.Appearances, appearanceInfo(asset))
	}
	if err := json.NewEncoder(os.Stdout).Encode(receipt); err != nil {
		panic(err)
	}
}

func materialInfo(asset *material.Material) wire.MaterialInfo {
	return wire.MaterialInfo{
		ID: asset.ID(), DisplayName: asset.DisplayName(), Source: string(asset.Source()),
		Density: asset.Density(), Mechanical: asset.Mechanical(), Thermal: asset.Thermal(),
		Electrical: asset.Electrical(), Magnetic: asset.Magnetic(), IsotropyClass: string(asset.IsotropyClass()),
		Anisotropic: asset.Anisotropic(), AppearanceID: asset.AppearanceID(),
	}
}

func appearanceInfo(asset *material.Appearance) wire.AppearanceInfo {
	return wire.AppearanceInfo{
		ID: asset.ID(), DisplayName: asset.DisplayName(), Source: string(asset.Source()),
		Base: asset.Base(), Specular: asset.Specular(), Transmission: asset.Transmission(),
		Subsurface: asset.Subsurface(), Coat: asset.Coat(), Fuzz: asset.Fuzz(),
		ThinFilm: asset.ThinFilm(), Emission: asset.Emission(), Geometry: asset.Geometry(),
	}
}

func verifyBaseline() {
	output, err := exec.Command("git", "rev-parse", "HEAD").Output()
	expected := "aa8297d1b0ac33c024f19e54821536d76e8a447a"
	actual := strings.TrimSpace(string(output))
	if err != nil || actual != expected {
		panic("catalog baseline " + actual + ": expected " + expected)
	}
}
