// SPDX-License-Identifier: GPL-2.0-only
// Run against the pinned Go module; see implementation-offset-splines.md.
package main

import (
	"encoding/json"
	"os"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

type splineOracle struct {
	Method   int
	Closed   bool
	Degree   int
	Poles    [][2]float64
	Knots    []float64
	Stations []float64
	Samples  [][5]float64
}

func main() {
	points := []math.Point2{{X: 0, Y: 0}, {X: 0.2, Y: 1}, {X: 2, Y: 1.5}, {X: 3, Y: -0.5}, {X: 1, Y: -2}}
	records := []splineOracle{}
	for _, closed := range []bool{false, true} {
		for _, method := range []geom.FitParameterization{geom.FitCentripetal, geom.FitChordLength, geom.FitUniform} {
			records = append(records, oracleRecord(points, method, closed))
		}
	}
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(records); err != nil {
		panic(err)
	}
}

func oracleRecord(points []math.Point2, method geom.FitParameterization, closed bool) splineOracle {
	curve, stations, err := geom.NewFittedBSplineCurve2dParam(points, method)
	if closed {
		curve, stations, err = geom.NewClosedFittedBSplineCurve2d(points, method)
	}
	if err != nil {
		panic(err)
	}
	record := splineOracle{Method: int(method), Closed: closed, Degree: curve.Degree, Knots: curve.Knots, Stations: stations}
	for _, pole := range curve.Ctrl {
		record.Poles = append(record.Poles, [2]float64{float64(pole.X), float64(pole.Y)})
	}
	record.Samples = oracleSamples(curve, stations)
	return record
}

func oracleSamples(curve geom.BSplineCurve2d, stations []float64) [][5]float64 {
	samples := [][5]float64{}
	for index := 0; index <= 16; index++ {
		samples = append(samples, oracleSample(curve, float64(index)/16))
	}
	for _, parameter := range stations {
		samples = append(samples, oracleSample(curve, parameter))
	}
	return samples
}

func oracleSample(curve geom.BSplineCurve2d, parameter float64) [5]float64 {
	position, tangent := curve.PointAt(parameter), curve.TangentAt(parameter)
	return [5]float64{parameter, float64(position.X), float64(position.Y), float64(tangent.X), float64(tangent.Y)}
}
