// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// The iso-rectilinear trim's acceptance. The gate is deliberately narrow — the shape only appears when
// an obstacle imprint notches a developable band — so these pin BOTH sides of it: the shapes it must
// take, and the shapes it must leave to the paths that already own them.

// rectilinearUnitLoop is the [0,4]×[0,10] rectangle with the [3,4]×[8,10] corner removed, sampled the
// way toUVLoops delivers a trim (many collinear samples along each run, as an arc discretization does).
func rectilinearUnitLoop() []math.Point2 {
	corners := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 8), math.P2(3, 8), math.P2(3, 10), math.P2(0, 10),
	}
	var loop []math.Point2
	for i := range corners {
		a, b := corners[i], corners[(i+1)%len(corners)]
		for k := 0; k < 7; k++ {
			loop = append(loop, a.Lerp(b, math.Scalar(float64(k)/7)))
		}
	}
	return loop
}

// TestIsoRectilinearGridDecomposesANotchedRectangle is the shape the imprint produces: the grid lines
// are the loop's own coordinates, and the cells it keeps reproduce the loop's area exactly.
func TestIsoRectilinearGridDecomposesANotchedRectangle(t *testing.T) {
	loop := rectilinearUnitLoop()
	us, vs, skip, ok := isoRectilinearGrid(loop)
	if !ok {
		t.Fatal("isoRectilinearGrid declined a notched rectangle")
	}
	kept := 0.0
	for i := 0; i+1 < len(us); i++ {
		for j := 0; j+1 < len(vs); j++ {
			if !skip(i, j) {
				kept += (us[i+1] - us[i]) * (vs[j+1] - vs[j])
			}
		}
	}
	if want := 4.0*10 - 1*2; stdmath.Abs(kept-want) > 1e-12 {
		t.Errorf("kept cell area %.12f, closed form %.12f", kept, want)
	}
}

// TestIsoRectilinearGridDeclinesAPlainRectangle keeps the new path off the faces isoRectangleGrid
// already meshes: with no cell removed there is nothing this decomposition adds, and answering here
// would divert an existing green.
func TestIsoRectilinearGridDeclinesAPlainRectangle(t *testing.T) {
	var loop []math.Point2
	corners := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 10), math.P2(0, 10)}
	for i := range corners {
		a, b := corners[i], corners[(i+1)%len(corners)]
		for k := 0; k < 5; k++ {
			loop = append(loop, a.Lerp(b, math.Scalar(float64(k)/5)))
		}
	}
	if _, _, _, ok := isoRectilinearGrid(loop); ok {
		t.Error("isoRectilinearGrid claimed a plain rectangle, which belongs to isoRectangleGrid")
	}
}

// TestIsoRectilinearGridDeclinesAnObliqueRun is the guard on the decomposition's own premise: one run
// that is neither an iso-u nor an iso-v line means the trim is NOT a union of grid cells, and filling
// cells would cover area the trim does not have.
func TestIsoRectilinearGridDeclinesAnObliqueRun(t *testing.T) {
	loop := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 8), math.P2(3, 10), math.P2(0, 10)}
	if _, _, _, ok := isoRectilinearGrid(loop); ok {
		t.Error("isoRectilinearGrid accepted a trim with an oblique run")
	}
}

// TestIsoRectilinearGridDeclinesASelfTouchingTrim exercises the AREA falsification directly: a loop
// whose runs are all iso-lines but which pinches shut encloses less than its cells would fill, so the
// cover check must reject it rather than mesh the pinched-off lobe.
func TestIsoRectilinearGridDeclinesASelfTouchingTrim(t *testing.T) {
	loop := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4),
		math.P2(0, 8), math.P2(4, 8), math.P2(4, 4), math.P2(0, 4),
	}
	if _, _, _, ok := isoRectilinearGrid(loop); ok {
		t.Error("isoRectilinearGrid accepted a self-touching trim")
	}
}
