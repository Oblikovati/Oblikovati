// SPDX-License-Identifier: GPL-2.0-only
// Capture Go aa8297d1 slot ownership and seeds without calling Rust.
package main

import (
	"encoding/json"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"os"
)

type record struct {
	Variant      string
	Input        [][2]float64
	Width, Sweep float64
	CCW          bool
	Points       [][2]float64
	Members      []member
}
type member struct {
	Kind   string
	Center int
	Ends   [2]int
	CCW    bool
}

func main() {
	out := []record{}
	for _, width := range []float64{0.5, 2} {
		for _, sweep := range []float64{-1.1, 1.1} {
			for _, variant := range []string{"centers", "overall", "slotCenter", "arc", "arcCenterPoint"} {
				r := record{Variant: variant, Input: [][2]float64{{1, -2}, {6, -2}, {1, 3}}, Width: width, Sweep: sweep, CCW: sweep > 0}
				out = append(out, capture(r))
			}
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		panic(err)
	}
}
func point(p [2]float64) m.Point2 { return m.P2(p[0], p[1]) }
func capture(r record) record {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a, b, c := point(r.Input[0]), point(r.Input[1]), point(r.Input[2])
	var entities []sketch.Entity
	var err error
	switch r.Variant {
	case "centers":
		entities, err = s.AddStraightSlot(a, b, r.Width)
	case "overall":
		entities, err = s.AddStraightSlotByOverall(a, b, r.Width)
	case "slotCenter":
		entities, err = s.AddStraightSlotBySlotCenter(a, b, r.Width)
	case "arc":
		entities, err = s.AddArcSlot(a, b, c, r.Width, r.CCW)
	case "arcCenterPoint":
		entities, err = s.AddArcSlotByCenterPoint(a, b, r.Sweep, r.Width)
	}
	if err != nil {
		panic(err)
	}
	indices := map[*sketch.Point]int{}
	for i, p := range s.AllPoints() {
		indices[p] = i
		v := p.Position()
		r.Points = append(r.Points, [2]float64{v.X, v.Y})
	}
	for _, e := range entities {
		switch v := e.(type) {
		case *sketch.Line:
			r.Members = append(r.Members, member{Kind: "line", Ends: [2]int{indices[v.StartPoint()], indices[v.EndPoint()]}})
		case *sketch.Arc:
			r.Members = append(r.Members, member{Kind: "arc", Center: indices[v.Center], Ends: [2]int{indices[v.Start], indices[v.End]}, CCW: v.CounterClockwise})
		}
	}
	return r
}
