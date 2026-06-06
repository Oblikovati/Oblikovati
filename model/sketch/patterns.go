// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati/math"
)

// RectangularPattern duplicates a selection on a grid: count1 columns stepped by step1
// and count2 rows stepped by step2 (each step a full direction×spacing vector). The seed
// cell (0,0) is the original, so the returned copies number count1·count2 − 1. It errors
// for non-positive counts.
func (s *Sketch) RectangularPattern(ents []Entity, step1 math.Vector2, count1 int, step2 math.Vector2, count2 int) ([]Entity, error) {
	return s.RectangularPatternLive(ents, func() math.Vector2 { return step1 }, count1, func() math.Vector2 { return step2 }, count2)
}

// RectangularPatternLive is RectangularPattern with live step providers: each clone is
// linked to its seed by a parameter-driven offset (step()×index), so editing the
// spacing repositions the copies on the next solve. A clone's other dimensions (e.g. a
// hole's radius) are copied at clone time and do not track the seed — only the grid
// spacing is parametric.
func (s *Sketch) RectangularPatternLive(ents []Entity, step1 func() math.Vector2, count1 int, step2 func() math.Vector2, count2 int) ([]Entity, error) {
	if count1 < 1 || count2 < 1 {
		return nil, fmt.Errorf("rectangular pattern: counts must be ≥ 1, got %d×%d", count1, count2)
	}
	g := s.GeometricConstraints()
	var copies []Entity
	for i := 0; i < count1; i++ {
		for j := 0; j < count2; j++ {
			if i == 0 && j == 0 {
				continue // the seed
			}
			ii, jj := i, j
			offset := step1().Scale(float64(ii)).Add(step2().Scale(float64(jj)))
			clones, pmap := s.cloneEntitiesMapped(ents, translation(offset))
			off := func() (float64, float64) {
				o := step1().Scale(float64(ii)).Add(step2().Scale(float64(jj)))
				return float64(o.X), float64(o.Y)
			}
			for seed, clone := range pmap {
				g.AddPatternLinkLive(seed, clone, off)
			}
			// A clone's non-point DOFs (a circle/arc radius) are not pinned by the
			// point links, so tie each clone's radius to its seed — this both removes
			// the free DOF and makes the clone track the seed's (parametric) radius.
			for k, clone := range clones {
				if sc, ok := ents[k].(CircularCurve); ok {
					if cc, ok := clone.(CircularCurve); ok {
						g.AddEqualRadius(sc, cc)
					}
				}
			}
			copies = append(copies, clones...)
		}
	}
	return copies, nil
}

// CircularPattern duplicates a selection around center: count instances (including the
// seed) evenly spread over totalAngle (radians), so the angular step is totalAngle/count.
// The returned copies number count − 1. It errors for a count below 2.
//
// Like [RectangularPatternLive], each clone is constrained back to its seed so the array is
// rigid (0 DOF when the seed is): every clone point is tied to the seed point by a LIVE
// rotational offset (clone = seed rotated by the member's angle about center, recomputed each
// solve so moving the seed — e.g. a parametric bolt-circle radius — carries the whole array),
// and a circular clone's radius is tied equal to its seed's. Without these links the clones'
// coordinates and radii float free (the bug that left a 3-hole bolt circle 6 DOF short).
func (s *Sketch) CircularPattern(ents []Entity, center math.Point2, count int, totalAngle float64) ([]Entity, error) {
	if count < 2 {
		return nil, fmt.Errorf("circular pattern: count must be ≥ 2, got %d", count)
	}
	g := s.GeometricConstraints()
	step := totalAngle / float64(count)
	var copies []Entity
	for k := 1; k < count; k++ {
		rot := rotation(center, step*float64(k))
		clones, pmap := s.cloneEntitiesMapped(ents, rot)
		for seed, clone := range pmap {
			g.AddPatternLinkLive(seed, clone, rotationOffset(seed, rot))
		}
		for i, clone := range clones {
			if sc, ok := ents[i].(CircularCurve); ok {
				if cc, ok := clone.(CircularCurve); ok {
					g.AddEqualRadius(sc, cc)
				}
			}
		}
		copies = append(copies, clones...)
	}
	return copies, nil
}

// rotationOffset returns the live seed→member offset for a circular-pattern member: the
// vector from the seed's current position to that position rotated by rot. Read each solve so
// the member tracks the (possibly parametric) seed about the pattern centre.
func rotationOffset(seed *Point, rot affine2) func() (float64, float64) {
	return func() (float64, float64) {
		p := seed.Position()
		r := rot.point(p)
		return float64(r.X - p.X), float64(r.Y - p.Y)
	}
}
