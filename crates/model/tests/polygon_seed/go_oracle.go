// SPDX-License-Identifier: GPL-2.0-only
// Capture Go aa8297d1 polygon seeds independently of the Rust implementation.
package main

import (
	"encoding/json"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type polygonRecord struct {
	Center    [2]float64
	Through   [2]float64
	Sides     int
	Inscribed bool
	Vertices  [][2]float64
}

func main() {
	records := []polygonRecord{}
	for _, pair := range [][2][2]float64{{{0, 0}, {3, 0}}, {{1, -2}, {-4, 7}}} {
		for _, sides := range []int{3, 4, 5, 8, 17} {
			for _, inscribed := range []bool{true, false} {
				records = append(records, capturePolygon(pair, sides, inscribed))
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(records); err != nil {
		panic(err)
	}
}
func capturePolygon(pair [2][2]float64, sides int, inscribed bool) polygonRecord {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	center := gmath.P2(gmath.Scalar(pair[0][0]), gmath.Scalar(pair[0][1]))
	through := gmath.P2(gmath.Scalar(pair[1][0]), gmath.Scalar(pair[1][1]))
	lines, _, err := s.AddPolygon(center, through, sides, inscribed)
	if err != nil {
		panic(err)
	}
	record := polygonRecord{Center: pair[0], Through: pair[1], Sides: sides, Inscribed: inscribed}
	for _, line := range lines {
		point := line.(*sketch.Line).StartPoint().Position()
		record.Vertices = append(record.Vertices, [2]float64{float64(point.X), float64(point.Y)})
	}
	return record
}
