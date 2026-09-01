// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The ISO-RECTILINEAR curved trim — a rectangle with rectangular bites taken out of it.
//
// isoRectangleGrid + structuredGridMesh already mesh the plain iso-rectangle (a cylinder/cone wall, an
// uninterrupted fillet band) as an exact structured grid. The band∩obstacle imprint
// (fillet_band_imprint.go) produces the next shape up: the same rectangle with the obstacle's own
// footprint removed, still bounded entirely by iso-lines. That trim has to leave the structured path,
// and the generic fallback for it (metricPatchMesh's metric-scaled CDT with curvature-adaptive interior
// nodes) is the wrong tool twice over on simple/Y4's band: it lays 11539 interior nodes, spends ~40 s in
// constrained-edge recovery, returns a 242-triangle mesh with a fold, and is then discarded for the
// boundary-only ear-clip — whose best-fit-plane triangulation of a 90°-wrapping band over-measures the
// face by 0.59 %. The trim is a union of grid cells, so it does not need a triangulator at all.
//
// The grid lines are the loop's OWN sample coordinates, exactly as isoRectangleGrid takes them, so every
// boundary vertex of the trim is reproduced by the grid and the cells inside it are filled with the same
// iso quads the rectangle path emits. Cells outside the trim are skipped.
//
// FALSIFIABLE: the decomposition is accepted only when the included cells' area equals the loop's own
// shoelace area (isoRectilinearAreaTol). A loop that is not really a union of these cells — a stray
// oblique run inside the sampling tolerance, a self-touching trim — fails that check and falls through
// to the existing path untouched.

// isoRectilinearAreaTol is how exactly the included cells must reproduce the loop's own (u,v) area,
// relative to that area. The loop is rectilinear by construction when the test passes at all, so the
// two agree to rounding; 1e-9 leaves room only for float64 summation noise.
const isoRectilinearAreaTol = 1e-9

// IsoRectilinearGrid reports whether a curved face's (u,v) trim is an axis-aligned RECTILINEAR region
// with at least one cell removed — the notched band the obstacle imprint leaves — and returns the grid
// lines plus the per-cell skip predicate structuredGridMeshSkip takes.
//
// It deliberately declines a plain rectangle (no cell removed): that is isoRectangleGrid's case, and
// answering it here would divert faces the existing path already meshes.
//
// Example: the simple/Y4 fillet band, whose 100×(π/2) trim loses the slot's [90,100]×[asin .2, asin .6]
// footprint, meshes as 3 strips of iso quads instead of an 11539-node CDT.
func IsoRectilinearGrid(loop []math.Point2) (us, vs []float64, skip func(i, j int) bool, ok bool) {
	uMin, uMax, vMin, vMax := bounds2D(loop)
	tolU, tolV := trimGridRelTol*float64(uMax-uMin), trimGridRelTol*float64(vMax-vMin)
	if tolU <= 0 || tolV <= 0 || !loopIsRectilinear(loop, tolU, tolV) {
		return nil, nil, nil, false
	}
	us, vs = isoRectilinearLines(loop, tolU, tolV)
	if len(us) < 2 || len(vs) < 2 {
		return nil, nil, nil, false
	}
	in := isoRectilinearCells(loop, us, vs)
	if !isoRectilinearCellsCover(loop, us, vs, in) {
		return nil, nil, nil, false
	}
	return us, vs, func(i, j int) bool { return !in[i][j] }, true
}

// loopIsRectilinear reports whether every step of the loop is axis-aligned in (u,v).
func loopIsRectilinear(loop []math.Point2, tolU, tolV float64) bool {
	for i := range loop {
		a, b := loop[i], loop[(i+1)%len(loop)]
		du, dv := stdmath.Abs(float64(a.X-b.X)), stdmath.Abs(float64(a.Y-b.Y))
		if du > tolU && dv > tolV {
			return false // an oblique run: the trim is not a union of grid cells
		}
	}
	return true
}

// isoRectilinearLines is the grid the loop's own samples define — every distinct u and every distinct
// v the boundary visits, so the grid reproduces each boundary vertex exactly.
func isoRectilinearLines(loop []math.Point2, tolU, tolV float64) ([]float64, []float64) {
	su, sv := make([]float64, 0, len(loop)), make([]float64, 0, len(loop))
	for _, p := range loop {
		su, sv = append(su, float64(p.X)), append(sv, float64(p.Y))
	}
	return sortUnique(su, tolU), sortUnique(sv, tolV)
}

// isoRectilinearCells classifies each grid cell by whether its centre lies inside the trim loop.
func isoRectilinearCells(loop []math.Point2, us, vs []float64) [][]bool {
	in := make([][]bool, len(us)-1)
	for i := range in {
		in[i] = make([]bool, len(vs)-1)
		for j := range in[i] {
			c := math.P2((us[i]+us[i+1])/2, (vs[j]+vs[j+1])/2)
			in[i][j] = pointInsideUVLoop(loop, c)
		}
	}
	return in
}

// pointInsideUVLoop is the even-odd crossing test for a point against a closed (u,v) loop.
func pointInsideUVLoop(loop []math.Point2, p math.Point2) bool {
	inside := false
	for i := range loop {
		a, b := loop[i], loop[(i+1)%len(loop)]
		if (a.Y > p.Y) == (b.Y > p.Y) {
			continue
		}
		if float64(p.X) < float64(a.X)+float64(b.X-a.X)*float64(p.Y-a.Y)/float64(b.Y-a.Y) {
			inside = !inside
		}
	}
	return inside
}

// isoRectilinearCellsCover is the falsification: the included cells must reproduce the loop's own
// shoelace area, and at least one cell must be excluded (a full rectangle belongs to isoRectangleGrid).
func isoRectilinearCellsCover(loop []math.Point2, us, vs []float64, in [][]bool) bool {
	cells, skipped := 0.0, false
	for i := range in {
		for j := range in[i] {
			if !in[i][j] {
				skipped = true
				continue
			}
			cells += (us[i+1] - us[i]) * (vs[j+1] - vs[j])
		}
	}
	want := shoelaceArea2D(loop)
	return skipped && want > 0 && stdmath.Abs(cells-want) <= isoRectilinearAreaTol*want
}

// shoelaceArea2D is the unsigned enclosed area of a closed (u,v) loop.
func shoelaceArea2D(loop []math.Point2) float64 {
	sum := 0.0
	for i := range loop {
		a, b := loop[i], loop[(i+1)%len(loop)]
		sum += float64(a.X*b.Y - b.X*a.Y)
	}
	return stdmath.Abs(sum) / 2
}
