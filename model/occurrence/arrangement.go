// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// Arrangement computes the per-element placements of an occurrence pattern: one
// world-space offset transform per position, in order, applied to the seed's base
// placement. Position 0 is always the identity — the seed itself. Circular,
// rectangular, and feature-based arrangements implement it (M11-F04, PBI-121).
type Arrangement interface {
	// Placements returns one offset transform per pattern position, in order.
	Placements() []math.Matrix4
}

// CircularArrangement places Count copies around an axis line (Origin, Axis) with
// Step radians between adjacent elements (position i sits at i·Step). A full ring of
// N uses Step = 2π/N; an arc uses a smaller span. Editing Count adds or drops trailing
// positions at the same spacing.
type CircularArrangement struct {
	Origin math.Point3
	Axis   math.UnitVector3
	Step   math.Scalar
	Count  int
}

// Placements returns Count rotations about the axis, the first being the identity.
func (c CircularArrangement) Placements() []math.Matrix4 {
	out := make([]math.Matrix4, max(c.Count, 1))
	for i := range out {
		out[i] = math.Rotation4(math.Scalar(i)*c.Step, c.Axis, c.Origin)
	}
	return out
}

// RectangularArrangement places a Count1×Count2 grid: Count1 columns spaced Spacing1
// along Dir1 and Count2 rows spaced Spacing2 along Dir2. Positions run column-fastest,
// so position 0 (column 0, row 0) is the identity.
type RectangularArrangement struct {
	Dir1     math.UnitVector3
	Spacing1 math.Scalar
	Count1   int
	Dir2     math.UnitVector3
	Spacing2 math.Scalar
	Count2   int
}

// Placements returns Count1·Count2 translations on the grid, the first the identity.
func (r RectangularArrangement) Placements() []math.Matrix4 {
	c1, c2 := max(r.Count1, 1), max(r.Count2, 1)
	out := make([]math.Matrix4, 0, c1*c2)
	for row := 0; row < c2; row++ {
		for col := 0; col < c1; col++ {
			offset := r.Dir1.AsVector().Scale(math.Scalar(col) * r.Spacing1).
				Add(r.Dir2.AsVector().Scale(math.Scalar(row) * r.Spacing2))
			out = append(out, math.Translation4(offset))
		}
	}
	return out
}

// FeatureArrangement follows a part feature pattern: its element placements are
// supplied directly (the offsets of the driving feature pattern's elements, extracted
// by the caller), so an assembly pattern tracks a part pattern without this package
// depending on the feature engine. Offsets[0] should be the identity (the seed).
type FeatureArrangement struct {
	Offsets []math.Matrix4
}

// Placements returns the supplied feature-driven offsets.
func (f FeatureArrangement) Placements() []math.Matrix4 {
	out := make([]math.Matrix4, len(f.Offsets))
	copy(out, f.Offsets)
	return out
}
