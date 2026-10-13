// SPDX-License-Identifier: GPL-2.0-only
// Capture the pinned Go aa8297d1 constrained-slot recipes and rank independently.
package main

import (
	"encoding/json"
	"oblikovati.org/api/types"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type record struct {
	Curved    bool
	Input     [3][2]float64
	Width     float64
	CCW       bool
	Points    [][2]float64
	Entities  []entity
	Relations []relation
	Fields    []field
	DOF       int
}
type entity struct {
	Kind              string
	Points            []int
	CCW, Construction bool
}
type relation struct {
	Kind     string
	Entities []int
}
type field struct {
	Label   string
	Value   float64
	Witness [2][2]float64
	Binding string
	Points  [2]int
	Entity  int
}

func main() {
	out := []record{}
	for _, input := range [][3][2]float64{{{0, 0}, {10, 0}, {0, 10}}, {{3, 4}, {9, 12}, {-5, 10}}} {
		for _, width := range []float64{1, 2} {
			for _, ccw := range []bool{false, true} {
				for _, curved := range []bool{false, true} {
					out = append(out, capture(record{Curved: curved, Input: input, Width: width, CCW: ccw}))
				}
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func xy(p m.Point2) [2]float64 { return [2]float64{p.X, p.Y} }
func capture(out record) record {
	c, s, e := m.P2(out.Input[0][0], out.Input[0][1]), m.P2(out.Input[1][0], out.Input[1][1]), m.P2(out.Input[2][0], out.Input[2][1])
	r := sketch.StraightSlotRecipe(c, s, out.Width)
	if out.Curved {
		r = sketch.ArcSlotRecipe(c, s, e, out.Width, out.CCW)
	}
	for _, p := range r.Points {
		out.Points = append(out.Points, xy(p))
	}
	for _, v := range r.Entities {
		kind := "line"
		if v.Kind == sketch.RecipeArc {
			kind = "arc"
		}
		out.Entities = append(out.Entities, entity{kind, v.Points, v.CounterClockwise, v.Construction})
	}
	for _, v := range r.Constraints {
		out.Relations = append(out.Relations, relation{string(v.Kind), v.Entities})
	}
	for _, v := range r.Fields {
		binding := "steering"
		if v.Dim.Kind == sketch.DistanceDim {
			binding = "distance"
		}
		if v.Dim.Kind == sketch.DiameterDim {
			binding = "diameter"
		}
		out.Fields = append(out.Fields, field{v.Label, v.Value, [2][2]float64{xy(v.Witness[0]), xy(v.Witness[1])}, binding, v.Dim.Points, v.Dim.Entity})
	}
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	if _, _, err := owner.Apply(r, types.OverConstrainedApplyDriven); err != nil {
		panic(err)
	}
	out.DOF = owner.DegreesOfFreedom()
	return out
}
