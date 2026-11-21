// SPDX-License-Identifier: GPL-2.0-only
// Capture authored seeds from Go aa8297d1; no Rust implementation is consulted.
package main

import (
	"encoding/json"
	"math"
	"os"

	gmath "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

type rectangleRecord struct {
	Variant string
	Inputs  [][2]float64
	Bits    [4][2]uint64
}

func main() {
	records := []rectangleRecord{}
	for _, points := range [][3][2]float64{
		{{0, 0}, {3, 4}, {7, 2}}, {{2, -3}, {-1, -7}, {8, 9}},
		{{1, 2}, {5, 2}, {9, -3}}, {{1, 2}, {1, 6}, {8, 9}},
		{{0, 0}, {3, 4}, {6, 8}}, {{1e40, -1e40}, {2e40, 3e40}, {-4e40, 5e40}},
		{{1e-40, -1e-40}, {2e-40, 3e-40}, {-4e-40, 5e-40}},
	} {
		for _, variant := range []string{"corners", "center", "threePoint"} {
			records = append(records, captureRectangle(variant, points))
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(records); err != nil {
		panic(err)
	}
}
func captureRectangle(variant string, points [3][2]float64) rectangleRecord {
	lines, count := rectangleLines(variant, points)
	record := rectangleRecord{Variant: variant, Inputs: points[:count]}
	for index, line := range lines {
		point := line.(*sketch.Line).StartPoint().Position()
		record.Bits[index] = [2]uint64{math.Float64bits(float64(point.X)), math.Float64bits(float64(point.Y))}
	}
	return record
}
func rectangleLines(variant string, points [3][2]float64) ([]sketch.Entity, int) {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a, b, c := oraclePoint(points[0]), oraclePoint(points[1]), oraclePoint(points[2])
	switch variant {
	case "corners":
		return s.AddRectangleByCorners(a, b), 2
	case "center":
		return s.AddRectangleByCenter(a, b), 2
	case "threePoint":
		lines, err := s.AddRectangleByThreePoints(a, b, c)
		if err != nil {
			panic(err)
		}
		return lines, 3
	default:
		panic("unknown variant " + variant + "; expected corners, center, or threePoint")
	}
}
func oraclePoint(point [2]float64) gmath.Point2 {
	return gmath.P2(gmath.Scalar(point[0]), gmath.Scalar(point[1]))
}
