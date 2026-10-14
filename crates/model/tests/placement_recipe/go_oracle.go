// SPDX-License-Identifier: GPL-2.0-only
// Capture the pinned Go aa8297d1 constrained slot, rectangle and circle recipes and rank independently.
package main

import (
	"encoding/json"
	"oblikovati.org/api/types"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type record struct {
	Kind      string
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
	Radius            float64
	Kind              string
	Points            []int
	CCW, Construction bool
}
type relation struct {
	Points   []int
	Kind     string
	Entities []int
}
type field struct {
	Label       string
	Value       float64
	Witness     [2][2]float64
	Binding     string
	Points      [2]int
	Entity      int
	Orientation string
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
	for _, kind := range []string{"rectangle", "center", "threePoint"} {
		for _, sign := range [][2]float64{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			out = append(out, capture(record{Kind: kind, Input: [3][2]float64{{3, 4}, {3 + 6*sign[0], 4 + 8*sign[1]}, {-5, 10}}}))
		}
	}
	for _, center := range [][2]float64{{0, 0}, {3, -4}, {-20, 30}} {
		for _, radius := range []float64{0.125, 5, 100} {
			out = append(out, capture(record{Kind: "circle", Input: [3][2]float64{center}, Width: radius}))
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func xy(p m.Point2) [2]float64 { return [2]float64{p.X, p.Y} }
func capture(out record) record {
	out.Relations = []relation{}
	c, s, e := m.P2(out.Input[0][0], out.Input[0][1]), m.P2(out.Input[1][0], out.Input[1][1]), m.P2(out.Input[2][0], out.Input[2][1])
	r := sketch.StraightSlotRecipe(c, s, out.Width)
	if out.Curved {
		r = sketch.ArcSlotRecipe(c, s, e, out.Width, out.CCW)
	}
	switch out.Kind {
	case "circle":
		r = sketch.CircleRecipe(c, out.Width)
	case "rectangle":
		r = sketch.RectangleRecipe(c, s)
	case "center":
		r = sketch.CenterRectangleRecipe(c, s)
	case "threePoint":
		r = sketch.ThreePointRectangleRecipe(c, s, e)
	}
	for _, p := range r.Points {
		out.Points = append(out.Points, xy(p))
	}
	for _, v := range r.Entities {
		kind := "line"
		if v.Kind == sketch.RecipeArc {
			kind = "arc"
		}
		if v.Kind == sketch.RecipeCircle {
			kind = "circle"
		}
		out.Entities = append(out.Entities, entity{float64(v.Radius), kind, v.Points, v.CounterClockwise, v.Construction})
	}
	for _, v := range r.Constraints {
		out.Relations = append(out.Relations, relation{v.Points, string(v.Kind), v.Entities})
	}
	for _, v := range r.Fields {
		binding := "steering"
		if v.Dim.Kind == sketch.DistanceDim {
			binding = "distance"
		}
		if v.Dim.Kind == sketch.DiameterDim {
			binding = "diameter"
		}
		out.Fields = append(out.Fields, field{v.Label, v.Value, [2][2]float64{xy(v.Witness[0]), xy(v.Witness[1])}, binding, v.Dim.Points, v.Dim.Entity, sketch.DistanceOrientationName(v.Dim.Orientation)})
	}
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	if _, _, err := owner.Apply(r, types.OverConstrainedApplyDriven); err != nil {
		panic(err)
	}
	out.DOF = owner.DegreesOfFreedom()
	return out
}
