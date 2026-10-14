// SPDX-License-Identifier: GPL-2.0-only
// Capture the pinned Go aa8297d1 constrained slot, rectangle and circle, arc, ellipse, point and line-chain recipes and rank independently.
package main

import (
	"encoding/json"
	"fmt"
	"oblikovati.org/api/types"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type record struct {
	Chain        [][2]float64 `json:",omitempty"`
	Closed       bool         `json:",omitempty"`
	Sides        int
	Inscribed    bool
	Locked       []lockedArc
	SolvedPoints [][2]float64
	Kind         string
	Curved       bool
	Input        [3][2]float64
	Width        float64
	CCW          bool
	Points       [][2]float64
	Entities     []entity
	Relations    []relation
	Fields       []field
	DOF          int
}
type lockedArc struct {
	Radius float64
	Points [][2]float64
	DOF    int
}
type entity struct {
	MajorAxis         [2]float64
	MinorRadius       float64
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
	for _, input := range [][3][2]float64{
		{{0, 0}, {5, 0}, {0, 5}}, {{0, 0}, {5, 0}, {0, 2}},
		{{3, -4}, {8, -4}, {3, -9}}, {{0, 0}, {5, 0}, {-5, 0}}, {{3, 4}, {-2, 4}, {3, 9}},
	} {
		for _, ccw := range []bool{false, true} {
			out = append(out, capture(record{Kind: "arc", Input: input, CCW: ccw}))
		}
	}
	for _, axis := range [][2]float64{{1, 0}, {0, 1}, {0.6, 0.8}, {-0.6, -0.8}} {
		for _, radii := range [][2]float64{{5, 2}, {5, 5}, {100, 0.125}} {
			out = append(out, capture(record{Kind: "ellipse", Input: [3][2]float64{{3, -4}, axis, radii}}))
		}
	}
	for _, sides := range []int{3, 4, 5, 8, 17} {
		for _, inscribed := range []bool{true, false} {
			for _, through := range [][2]float64{{8, -4}, {6, 0}} {
				out = append(out, capture(record{Kind: "polygon", Sides: sides, Inscribed: inscribed, Input: [3][2]float64{{3, -4}, through}}))
			}
		}
	}
	for _, point := range [][2]float64{{0, 0}, {3, -4}, {-20, 30}, {0.125, 100}} {
		out = append(out, capture(record{Kind: "point", Input: [3][2]float64{point}}))
	}
	for _, sign := range [][2]float64{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		chain := [][2]float64{{3, -4}, {9, -4}, {9, 4}, {-2, 4}, {-2, -1}}
		for i, p := range chain {
			chain[i] = [2]float64{p[0] * sign[0], p[1] * sign[1]}
		}
		for _, count := range []int{2, 3, 5} {
			out = append(out, capture(record{Kind: "chain", Chain: chain[:count]}))
			if count >= 3 {
				out = append(out, capture(record{Kind: "chain", Chain: chain[:count], Closed: true}))
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func xy(p m.Point2) [2]float64 { return [2]float64{p.X, p.Y} }
func capture(out record) record {
	out.Relations = []relation{}
	out.Fields = []field{}
	c, s, e := m.P2(out.Input[0][0], out.Input[0][1]), m.P2(out.Input[1][0], out.Input[1][1]), m.P2(out.Input[2][0], out.Input[2][1])
	r := sketch.StraightSlotRecipe(c, s, out.Width)
	if out.Curved {
		r = sketch.ArcSlotRecipe(c, s, e, out.Width, out.CCW)
	}
	switch out.Kind {
	case "chain":
		r = chainRecipe(out)
	case "point":
		r = sketch.PointRecipe(c)
	case "polygon":
		r = sketch.PolygonRecipe(c, s, out.Sides, out.Inscribed)
	case "ellipse":
		r = sketch.EllipseRecipe(c, m.V2(s.X, s.Y), e.X, e.Y)
	case "arc":
		r = sketch.ArcRecipe(c, s, e, out.CCW)
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
		if v.Kind == sketch.RecipePoint {
			kind = "point"
		}
		if v.Kind == sketch.RecipeArc {
			kind = "arc"
		}
		if v.Kind == sketch.RecipeCircle {
			kind = "circle"
		}
		if v.Kind == sketch.RecipeEllipse {
			kind = "ellipse"
		}
		out.Entities = append(out.Entities, entity{[2]float64{v.MajorAxis.X, v.MajorAxis.Y}, float64(v.MinorRadius), float64(v.Radius), kind, v.Points, v.CounterClockwise, v.Construction})
	}
	for _, v := range r.Constraints {
		out.Relations = append(out.Relations, relation{v.Points, string(v.Kind), v.Entities})
	}
	for _, v := range r.Fields {
		binding := "steering"
		if v.Dim.Kind == sketch.DistanceDim {
			binding = "distance"
		}
		if v.Dim.Kind == sketch.RadiusDim {
			binding = "radius"
		}
		if v.Dim.Kind == sketch.DiameterDim {
			binding = "diameter"
		}
		if v.Dim.Kind == sketch.EllipseRadiusDim {
			binding = "ellipseRadius"
			if v.Dim.Entity == sketch.NoRecipeEntity {
				binding = "steeringLength"
			}
		}
		out.Fields = append(out.Fields, field{v.Label, v.Value, [2][2]float64{xy(v.Witness[0]), xy(v.Witness[1])}, binding, v.Dim.Points, v.Dim.Entity, sketch.DistanceOrientationName(v.Dim.Orientation)})
	}
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	_, pts, err := owner.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		panic(err)
	}
	out.DOF = owner.DegreesOfFreedom()
	if out.Kind == "arc" {
		if !owner.Solve().Converged {
			panic("arc solve failed")
		}
		for _, p := range pts {
			out.SolvedPoints = append(out.SolvedPoints, xy(p.Position()))
		}
		if out.Input[2][0] == out.Input[0][0] {
			out.Locked = captureLockedArc(r)
		}
	}
	return out
}
func captureLockedArc(r sketch.Recipe) []lockedArc {
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	ents, pts, err := owner.Apply(r, types.OverConstrainedApplyDriven)
	if err != nil {
		panic(err)
	}
	owner.GeometricConstraints().AddFix(pts[0])
	owner.GeometricConstraints().AddHorizontal(pts[0], pts[1])
	owner.GeometricConstraints().AddVertical(pts[0], pts[2])
	dimension, err := owner.DimensionConstraints().AddRadius(ents[0].(sketch.CircularCurve), "5 cm")
	if err != nil {
		panic(err)
	}
	out := []lockedArc{}
	for _, radius := range []float64{5, 7, 1.5} {
		if err := dimension.Parameter().SetExpression(fmt.Sprintf("%g cm", radius)); err != nil {
			panic(err)
		}
		if !owner.Solve().Converged {
			panic("locked arc solve failed")
		}
		sample := lockedArc{Radius: radius, DOF: owner.DegreesOfFreedom()}
		for _, point := range pts {
			sample.Points = append(sample.Points, xy(point.Position()))
		}
		out = append(out, sample)
	}
	return out
}

// Closure is read from Go AddPolyline's actual shared point identities, not rebuilt from indices.
func chainRecipe(out record) sketch.Recipe {
	points := make([]m.Point2, len(out.Chain))
	for i, p := range out.Chain {
		points[i] = m.P2(p[0], p[1])
	}
	if len(points) == 2 {
		return sketch.LineRecipe(points[0], points[1])
	}
	if !out.Closed {
		return sketch.LineChainRecipe(points)
	}
	return closedChainRecipe(points)
}
func closedChainRecipe(points []m.Point2) sketch.Recipe {
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	members, err := owner.AddPolyline(points, true)
	if err != nil {
		panic(err)
	}
	recipe := sketch.Recipe{}
	ids := map[*sketch.Point]int{}
	for _, member := range members {
		line := member.(*sketch.Line)
		pair := []int{captureChainPoint(&recipe, ids, line.A), captureChainPoint(&recipe, ids, line.B)}
		recipe.Entities = append(recipe.Entities, sketch.RecipeEntity{Kind: sketch.RecipeLine, Points: pair})
	}
	last := members[len(members)-1].(*sketch.Line)
	recipe.Fields = sketch.LineRecipe(last.A.Position(), last.B.Position()).Fields
	ends := recipe.Entities[len(recipe.Entities)-1].Points
	recipe.Fields[0].Dim.Points = [2]int{ends[0], ends[1]}
	if len(recipe.Points) != len(points) || owner.DegreesOfFreedom() != 2*len(points) {
		panic("closed chain ownership or rank changed")
	}
	return recipe
}
func captureChainPoint(recipe *sketch.Recipe, ids map[*sketch.Point]int, point *sketch.Point) int {
	if id, found := ids[point]; found {
		return id
	}
	id := len(recipe.Points)
	ids[point] = id
	recipe.Points = append(recipe.Points, point.Position())
	return id
}
