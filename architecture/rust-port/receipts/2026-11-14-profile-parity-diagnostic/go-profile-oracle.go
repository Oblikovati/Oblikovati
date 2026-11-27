// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	m "oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

func main() {
	cases := []struct {
		name       string
		x, y, side float64
	}{{"cube", 0, 0, 1}, {"tiny", 0, 0, 0.00002}, {"small", 0, 0, 0.0001}, {"offset", 1e8, 1e8, 1}, {"offset_small", 1e8, 1e8, 0.01}}
	for _, c := range cases {
		s := sketch.NewSketches().Add(sketch.XYPlane())
		p := []*sketch.Point{s.Points().Add(m.P2(c.x, c.y)), s.Points().Add(m.P2(c.x+c.side, c.y)), s.Points().Add(m.P2(c.x+c.side, c.y+c.side)), s.Points().Add(m.P2(c.x, c.y+c.side))}
		for i := range p {
			s.Lines().Add(p[i], p[(i+1)%4])
		}
		ps := s.Profiles()
		fmt.Printf("%s count=%d", c.name, ps.Count())
		for _, p := range ps.All() {
			fmt.Printf(" closed=%t", p.IsClosed())
		}
		fmt.Println()
	}
}
