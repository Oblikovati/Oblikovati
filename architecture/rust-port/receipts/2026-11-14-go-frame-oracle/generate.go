// SPDX-License-Identifier: GPL-2.0-only
// Print Go's analytic plane frames for independent Rust compatibility tests.
package main

import (
	"encoding/json"
	"os"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

type sample struct {
	Input [3]float64 `json:"input"`
	U     [3]float64 `json:"u"`
	V     [3]float64 `json:"v"`
}

func main() {
	normals := [][3]float64{
		{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		{0.899999999, 0.435889896, 0}, {0.900000001, 0.435889894, 0},
		{-0.899999999, 0.435889896, 0}, {-0.900000001, 0.435889894, 0},
		{1, 2, 3}, {-2, -3, 1}, {0.01, -0.8, 0.6},
	}
	rows := make([]sample, 0, len(normals))
	for _, n := range normals {
		p, err := geom.NewPlane(m.P3(0, 0, 0), m.V3(n[0], n[1], n[2]))
		if err != nil {
			panic(err)
		}
		rows = append(rows, sample{n,
			[3]float64{p.UAxis.X(), p.UAxis.Y(), p.UAxis.Z()},
			[3]float64{p.VAxis.X(), p.VAxis.Y(), p.VAxis.Z()},
		})
	}
	if err := json.NewEncoder(os.Stdout).Encode(rows); err != nil {
		panic(err)
	}
}
