// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// The iso-rectilinear trim's acceptance. The gate is deliberately narrow — the shape only appears when
// an obstacle imprint notches a developable band — so these pin BOTH sides of it: the shapes it must
// take, and the shapes it must leave to the paths that already own them.

// rectilinearUnitLoop is the [0,4]×[0,10] rectangle with the [3,4]×[8,10] corner removed, sampled the
// way ToUVLoops delivers a trim (many collinear samples along each run, as an arc discretization does).
func rectilinearUnitLoop() []math.Point2 {
	corners := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 8), math.P2(3, 8), math.P2(3, 10), math.P2(0, 10),
	}
	var loop []math.Point2
	for i := range corners {
		a, b := corners[i], corners[(i+1)%len(corners)]
		for k := range 7 {
			loop = append(loop, a.Lerp(b, math.Scalar(float64(k)/7)))
		}
	}
	return loop
}

// TestIsoRectilinearGridDecomposesANotchedRectangle is the shape the imprint produces: the grid lines
// are the loop's own coordinates, and the cells it keeps reproduce the loop's area exactly.
func TestIsoRectilinearGridDecomposesANotchedRectangle(t *testing.T) {
	t.Parallel()
	loop := rectilinearUnitLoop()
	us, vs, skip, ok := tessellate.IsoRectilinearGrid(loop)
	if !ok {
		t.Fatal("tessellate.IsoRectilinearGrid declined a notched rectangle")
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
	t.Parallel()
	var loop []math.Point2
	corners := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 10), math.P2(0, 10)}
	for i := range corners {
		a, b := corners[i], corners[(i+1)%len(corners)]
		for k := range 5 {
			loop = append(loop, a.Lerp(b, math.Scalar(float64(k)/5)))
		}
	}
	if _, _, _, ok := tessellate.IsoRectilinearGrid(loop); ok {
		t.Error("tessellate.IsoRectilinearGrid claimed a plain rectangle, which belongs to isoRectangleGrid")
	}
}

// TestIsoRectilinearGridDeclinesAnObliqueRun is the guard on the decomposition's own premise: one run
// that is neither an iso-u nor an iso-v line means the trim is NOT a union of grid cells, and filling
// cells would cover area the trim does not have.
func TestIsoRectilinearGridDeclinesAnObliqueRun(t *testing.T) {
	t.Parallel()
	loop := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 8), math.P2(3, 10), math.P2(0, 10)}
	if _, _, _, ok := tessellate.IsoRectilinearGrid(loop); ok {
		t.Error("tessellate.IsoRectilinearGrid accepted a trim with an oblique run")
	}
}

// TestIsoRectilinearGridDeclinesASelfTouchingTrim exercises the AREA falsification directly: a loop
// whose runs are all iso-lines but which pinches shut encloses less than its cells would fill, so the
// cover check must reject it rather than mesh the pinched-off lobe.
func TestIsoRectilinearGridDeclinesASelfTouchingTrim(t *testing.T) {
	t.Parallel()
	loop := []math.Point2{
		math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4),
		math.P2(0, 8), math.P2(4, 8), math.P2(4, 4), math.P2(0, 4),
	}
	if _, _, _, ok := tessellate.IsoRectilinearGrid(loop); ok {
		t.Error("tessellate.IsoRectilinearGrid accepted a self-touching trim")
	}
}

// TestIsoRectilinearGridMeshesTheImprintedBandExactly is the end-to-end reason the path exists: the
// simple/Y4 band's notched trim, meshed as iso quads, reproduces its closed-form developed area
// (π/2·25·100 − 25·(asin .6 − asin .2)·10 = 3816.455) to the boundary's own chord budget. Through the
// generic CDT the same face costs ~40 s and lands on the boundary-only ear-clip, 0.59 % OVER.
func TestIsoRectilinearGridMeshesTheImprintedBandExactly(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		bandCase
		bite float64
	}{
		{bandCase{"Y2", 15}, stdmath.Asin(1.0 / 3)},
		{bandCase{"Y3", 20}, stdmath.Asin(0.5)},
		{bandCase{"Y4", 25}, stdmath.Asin(0.6) - stdmath.Asin(0.2)},
	} {
		body, ef, maps, caps := bandSlotFillet(t, c.bandCase)
		set, ok := bandImprintFacesFor(body, ef, maps, caps)
		if !ok {
			t.Fatalf("%s: the imprint declined", c.name)
		}
		bandBody := assembleBody([]filletFace{set.band})
		mesh := tessellate.TessellateFaceSurface(bandBody.Faces()[0], PropertyQuality())
		want := c.r*stdmath.Pi/2*100 - c.r*c.bite*10
		got := meshTriangleArea(mesh)
		// The mesh chords the band's own boundary discretization, so it under-measures by that
		// sampling's sagitta — bounded here at 1e-4 relative, two decades under the corpus gate.
		if rel := (want - got) / want; rel < 0 || rel > 1e-4 {
			t.Errorf("%s band mesh area %.6f, closed form %.6f (rel %.3g, want a small UNDER-measure)", c.name, got, want, rel)
		}
	}
}

// meshTriangleArea is the summed triangle area of a mesh.
func meshTriangleArea(m *tessellate.Mesh) float64 {
	a := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		p0, p1, p2 := m.Positions[m.Indices[i]], m.Positions[m.Indices[i+1]], m.Positions[m.Indices[i+2]]
		a += float64(p0.VectorTo(p1).Cross(p0.VectorTo(p2)).Length()) / 2
	}
	return a
}
