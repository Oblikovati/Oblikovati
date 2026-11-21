// SPDX-License-Identifier: GPL-2.0-only
// Run in the fixed Go checkout to export the complete eight-bit transfer domain and migration oracle.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"oblikovati.org/api/wire"
	"oblikovati.org/model/material"
	"oblikovati.org/yamlcodec"
	"os"
	"os/exec"
	"strings"
)

type migrationCase struct {
	Input    string              `json:"input"`
	Expected wire.AppearanceInfo `json:"expected"`
}
type receipt struct {
	License   string          `json:"license"`
	Baseline  string          `json:"baseline"`
	GammaBits []uint32        `json:"gammaBits"`
	Cases     []migrationCase `json:"cases"`
}

func main() {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	baseline := "aa8297d1b0ac33c024f19e54821536d76e8a447a"
	if err != nil || strings.TrimSpace(string(head)) != baseline {
		panic("expected fixed Go baseline")
	}
	out := receipt{License: "SPDX-License-Identifier: GPL-2.0-only", Baseline: baseline}
	for i := 0; i < 256; i++ {
		out.GammaBits = append(out.GammaBits, math.Float32bits(float32(math.Pow(float64(float32(i)/255), 2.2))))
		input := fmt.Sprintf("id: ramp-%d\nname: Ramp\nalbedo: '#%02x%02x%02xff'\nmetallic: 0.3\nroughness: 0.6\nemissive: '#%02x%02x%02x'\nopacity: 0.7\n", i, i, (i+83)%256, (i+173)%256, (i+29)%256, i, (i+113)%256)
		var recipe material.RecipeData
		if err := yamlcodec.Unmarshal([]byte("appearances:\n  - "+strings.ReplaceAll(strings.TrimSpace(input), "\n", "\n    ")), &recipe); err != nil {
			panic(err)
		}
		a := recipe.Appearances[0]
		expected := wire.AppearanceInfo{ID: a.ID, DisplayName: a.DisplayName, Source: "document", Base: a.Base, Specular: a.Specular, Transmission: a.Transmission, Subsurface: a.Subsurface, Coat: a.Coat, Fuzz: a.Fuzz, ThinFilm: a.ThinFilm, Emission: a.Emission, Geometry: a.Geometry}
		out.Cases = append(out.Cases, migrationCase{input, expected})
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
