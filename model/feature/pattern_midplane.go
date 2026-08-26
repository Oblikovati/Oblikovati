// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/math"

// Spreading a pattern to BOTH sides of its seed (Oblikovati#1889) — Inventor's
// XDirectionMidPlanePattern / YDirectionMidPlanePattern / MidPlanePattern.
//
// A plain pattern runs one way from the seed: the seed, then count−1 more along the step. A
// mid-plane pattern puts occurrences on both sides of it instead. The seed itself does NOT move —
// Inventor is explicit that "the transform returned for the first element in a pattern (the parent
// feature(s)) will be an identity matrix", mid-plane or not, and it must be, because the seed is the
// source features' own material, already applied to the running body before the pattern ran. There
// is no pre-source body to re-place it from.
//
// So a mid-plane pattern is not a moved grid; it is the same grid re-indexed about the seed. Where a
// plain pattern occupies steps 0…n−1, a mid-plane one occupies −s…n−1−s for a shift s, and s is
// chosen to split the OTHER n−1 occurrences as evenly as the count allows.
//
// With an odd count the split is exact. With an even count one side must take the extra, and
// Inventor gives it to the natural direction. Here the step vector IS the direction the author gave,
// so the extra goes on the +step side; reversing the step moves it to the other, which is the same
// control by a different name.

// patternIndexShift is how many occurrences a mid-plane pattern places on the step's negative side.
// Zero — the plain one-way pattern — when midPlane is false.
func patternIndexShift(count int, midPlane bool) int {
	if !midPlane || count < 2 {
		return 0
	}
	return (count - 1) / 2 // the +step side keeps the extra when count is even
}

// seedFirst reorders a list of per-occurrence offsets so the seed's own cell leads it, which is
// what element 0 means everywhere else in the pattern code (and what replicateTools relies on when
// it starts at k=1). The remaining occurrences keep their relative order, so element indices stay
// stable and a suppressed occurrence stays the same occurrence.
func seedFirst[T any](cells []T, seed int) []T {
	if seed <= 0 || seed >= len(cells) {
		return cells
	}
	out := make([]T, 0, len(cells))
	out = append(out, cells[seed])
	out = append(out, cells[:seed]...)
	return append(out, cells[seed+1:]...)
}

// rectTransforms returns the grid of occurrence transforms; element 0 is the identity (the seed).
// shiftX/shiftY move the grid's origin cell to the seed so a mid-plane direction straddles it.
func rectTransforms(nx, ny int, stepX, stepY math.Vector3, shiftX, shiftY int) []math.Matrix4 {
	out := make([]math.Matrix4, 0, nx*ny)
	for iy := range ny {
		for ix := range nx {
			offset := stepX.Scale(float64(ix - shiftX)).Add(stepY.Scale(float64(iy - shiftY)))
			out = append(out, math.Translation4(offset))
		}
	}
	return seedFirst(out, shiftX+shiftY*nx)
}

// circTransforms returns count occurrence rotations about the axis stepping by inc radians per
// occurrence; element 0 is the identity. shift rotates the run back so a mid-plane pattern spans
// the seed. The increment is chosen by the spacing type ([PatternOptions.circIncrement]).
func circTransforms(count int, inc float64, axisPoint math.Point3, axisDir math.Vector3,
	shift int) ([]math.Matrix4, error) {
	dir, err := math.UnitVector3FromVector(axisDir)
	if err != nil {
		return nil, err
	}
	out := make([]math.Matrix4, count)
	for k := range count {
		out[k] = math.Rotation4(inc*float64(k-shift), dir, axisPoint)
	}
	return seedFirst(out, shift), nil
}
