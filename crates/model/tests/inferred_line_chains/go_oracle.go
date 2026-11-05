// SPDX-License-Identifier: GPL-2.0-only
// Run from Go aa8297d1's module; capture its LineTool.Commit ordering and ApplyLineInference.
package main

import (
	"encoding/json"
	"oblikovati.org/api/types"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type segment struct {
	TargetPositions [2][2]float64
	Endpoints       [2]int
	Positions       [2][2]float64
	Snaps           [][2]int
	Kind            string
	Target          int
	Constraints     int
}
type fixture struct {
	Input    [][2]float64
	Closed   bool
	Infer    bool
	Apply    bool
	Priority int
	Points   [][2]float64
	Members  []int
	Segments []segment
}

func main() {
	var out []fixture
	for _, input := range [][][2]float64{
		{{0, 0}, {4, 0.01}}, {{0.0005, 0.0005}, {3.0005, 2.0005}, {7, 2.0005}},
		{{10, 10}, {13, 12}, {11, 15}}, {{0, 0}, {4, 0}, {4, 3}, {0, 3}},
		{{10, 10}, {10.0001, 10.0001}, {10, 10}}, {{3, 2}, {6, 4}, {9, 6}},
		{{3.0005, 2.0005}, {6, 4}, {3, 2}}, {{0, 0}, {0, 0}, {1, 1}},
	} {
		for _, closed := range []bool{false, true} {
			for _, mode := range [][3]int{{1, 1, 0}, {1, 1, 1}, {1, 1, 2}, {1, 0, 0}, {0, 1, 0}} {
				out = append(out, capture(fixture{Input: input, Closed: closed, Infer: mode[0] == 1, Apply: mode[1] == 1, Priority: mode[2]}))
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func capture(out fixture) fixture {
	owner := sketch.NewSketches().Add(sketch.XYPlane())
	a := owner.Points().Add(m.P2(0, 0))
	b := owner.Points().Add(m.P2(3, 2))
	owner.Lines().Add(a, b)
	owner.Points().Add(m.P2(0, 0)) // Equal-distance targets retain last-authored precedence.
	points := []*sketch.Point{owner.Points().Add(m.P2(out.Input[0][0], out.Input[0][1]))}
	opts := sketch.InferenceOptions{InferEnabled: out.Infer, ConstrainEnabled: out.Apply, Priority: types.ConstraintInferencePriority(out.Priority)}
	for _, xy := range out.Input[1:] {
		next := owner.Points().Add(m.P2(xy[0], xy[1]))
		out.Segments = append(out.Segments, appendSegment(owner, points[len(points)-1], next, opts))
		points = append(points, next)
	}
	if out.Closed && len(points) >= 3 {
		out.Segments = append(out.Segments, appendSegment(owner, points[len(points)-1], points[0], opts))
	}
	for _, p := range owner.AllPoints() {
		out.Points = append(out.Points, [2]float64{p.X, p.Y})
	}
	for _, entity := range owner.Entities() {
		out.Members = append(out.Members, entityIndex(owner, entity))
	}
	return out
}
func appendSegment(owner *sketch.Sketch, a, b *sketch.Point, opts sketch.InferenceOptions) segment {
	line := owner.Lines().Add(a, b)
	constraints, snaps := owner.ApplyLineInference(line, opts)
	out := segment{Endpoints: [2]int{pointIndex(owner, a), pointIndex(owner, b)}, Positions: [2][2]float64{{a.X, a.Y}, {b.X, b.Y}}, Snaps: [][2]int{}, Target: -1, Constraints: owner.GeometricConstraints().Count()}
	for _, snap := range snaps {
		out.Snaps = append(out.Snaps, [2]int{pointIndex(owner, snap.Point), pointIndex(owner, snap.Entities[0].(*sketch.Point))})
	}
	if len(constraints) > 0 {
		kinds := map[types.ConstraintInferenceKind]string{types.InferHorizontal: "horizontal", types.InferVertical: "vertical", types.InferParallel: "parallel", types.InferPerpendicular: "perpendicular"}
		out.Kind = kinds[constraints[0].Kind]
		if len(constraints[0].Entities) == 2 {
			out.Target = entityIndex(owner, constraints[0].Entities[1])
			target := constraints[0].Entities[1].(*sketch.Line)
			out.TargetPositions = [2][2]float64{{target.A.X, target.A.Y}, {target.B.X, target.B.Y}}
		}
	}
	return out
}
func pointIndex(owner *sketch.Sketch, point *sketch.Point) int {
	for i, p := range owner.AllPoints() {
		if p == point {
			return i
		}
	}
	panic("point missing")
}
func entityIndex(owner *sketch.Sketch, entity sketch.Entity) int {
	for i, e := range owner.Entities() {
		if e == entity {
			return i
		}
	}
	panic("entity missing")
}
