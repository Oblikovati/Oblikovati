// SPDX-License-Identifier: GPL-2.0-only
package main

import (
	"fmt"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

func main() {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(8, 4))
	s.GeometricConstraints().AddCoincident(a, b)
	s.Solve()
	fmt.Printf("first=%v second=%v\n", a.Position(), b.Position())
}
