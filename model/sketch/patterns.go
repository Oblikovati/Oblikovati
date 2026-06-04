// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
)

// RectangularPattern duplicates a selection on a grid: count1 columns stepped by step1
// and count2 rows stepped by step2 (each step a full direction×spacing vector). The seed
// cell (0,0) is the original, so the returned copies number count1·count2 − 1. It errors
// for non-positive counts.
func (s *Sketch) RectangularPattern(ents []Entity, step1 math.Vector2, count1 int, step2 math.Vector2, count2 int) ([]Entity, error) {
	if count1 < 1 || count2 < 1 {
		return nil, fmt.Errorf("rectangular pattern: counts must be ≥ 1, got %d×%d", count1, count2)
	}
	var copies []Entity
	for i := 0; i < count1; i++ {
		for j := 0; j < count2; j++ {
			if i == 0 && j == 0 {
				continue // the seed
			}
			offset := step1.Scale(float64(i)).Add(step2.Scale(float64(j)))
			copies = append(copies, s.cloneEntities(ents, translation(offset))...)
		}
	}
	return copies, nil
}

// CircularPattern duplicates a selection around center: count instances (including the
// seed) evenly spread over totalAngle (radians), so the angular step is totalAngle/count.
// The returned copies number count − 1. It errors for a count below 2.
func (s *Sketch) CircularPattern(ents []Entity, center math.Point2, count int, totalAngle float64) ([]Entity, error) {
	if count < 2 {
		return nil, fmt.Errorf("circular pattern: count must be ≥ 2, got %d", count)
	}
	step := totalAngle / float64(count)
	var copies []Entity
	for k := 1; k < count; k++ {
		copies = append(copies, s.cloneEntities(ents, rotation(center, step*float64(k)))...)
	}
	return copies, nil
}
