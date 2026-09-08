// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// windowedWallCover lays the rim-bounded windowed wall's chains and interior into a covering, returning
// it with the constraint loops — the state keptWithoutRimEars starts from.
func windowedWallCover(t *testing.T) (*chartCover, [][]int, []chartChain) {
	t.Helper()
	f := rimBoundedWindowedWall(t, 3, 4, 3, 6)
	s := f.Geometry()
	r, ok := newChartRegion(f, s)
	if !ok {
		t.Fatal("the windowed wall carries no chart region")
	}
	chains := chartBoundaryChains(f, s, r, DefaultQuality())
	b := newChartCover(s, r, DefaultQuality())
	loops := b.addChains(chains)
	b.addInterior(chains)
	return b, loops, chains
}

// windowRimIndices is the first three rim vertices laid inside the branch window that are three
// DISTINCT 3D points — the chains are laid at every period shift, so the lowest indices are a period
// away, and a shifted copy's closing sample welds onto the canonical copy's opening one.
func windowRimIndices(t *testing.T, b *chartCover) [3]int {
	t.Helper()
	grid := weldGrid([][]math.Point3{b.pos})
	var out [3]int
	seen := map[[3]int64]bool{}
	n := 0
	for i := 0; i < b.rim && n < 3; i++ {
		if k := quantizePoint(b.pos[i], grid); b.r.inWindow(b.uu[i], b.vv[i]) && !seen[k] {
			seen[k], out[n], n = true, i, n+1
		}
	}
	if n < 3 {
		t.Fatalf("only %d distinct rim vertices lie in the window; the test needs three", n)
	}
	return out
}

// weldedRimPair is a pair of rim vertices that are ONE 3D point — a chain's opening sample and its
// closing one, or the same sample a period along.
func weldedRimPair(t *testing.T, b *chartCover, grid float64) (int, int) {
	t.Helper()
	for i := 0; i < b.rim; i++ {
		for j := i + 1; j < b.rim; j++ {
			if quantizePoint(b.pos[i], grid) == quantizePoint(b.pos[j], grid) {
				return i, j
			}
		}
	}
	t.Fatal("no two rim vertices weld; the test needs a collapsing pair")
	return 0, 0
}

// TestIsRimEarReadsBoundaryIndicesThatSurviveTheWeld: three distinct rim vertices are an ear; a
// triangle with an interior vertex is not; a triangle two of whose rim vertices are one 3D point
// collapses at the weld and is not.
func TestIsRimEarReadsBoundaryIndicesThatSurviveTheWeld(t *testing.T) {
	t.Parallel()
	b, _, _ := windowedWallCover(t)
	grid := weldGrid([][]math.Point3{b.pos})
	if b.rim == 0 || b.rim == len(b.pos) {
		t.Fatalf("the covering holds %d rim vertices of %d; the test needs both kinds", b.rim, len(b.pos))
	}
	tri := windowRimIndices(t, b)
	if !b.isRimEar(tri, grid) {
		t.Error("three distinct rim vertices were not read as an ear")
	}
	if b.isRimEar([3]int{tri[0], tri[1], b.rim}, grid) {
		t.Error("a triangle with an interior vertex was read as an ear")
	}
	i, j := weldedRimPair(t, b, grid)
	if b.isRimEar([3]int{i, j, tri[2]}, grid) {
		t.Error("a triangle the weld collapses was read as an ear")
	}
}

// TestAddEarCentreAddsASurfacePointInsideTheWindow: the split point is evaluated ON the surface at the
// ear's own centroid — nothing is moved — and it lands inside the branch window, where addReplicas
// carries it.
func TestAddEarCentreAddsASurfacePointInsideTheWindow(t *testing.T) {
	t.Parallel()
	b, _, _ := windowedWallCover(t)
	before := len(b.pos)
	b.addEarCentre(windowRimIndices(t, b))
	if len(b.pos) <= before {
		t.Fatal("addEarCentre added no point")
	}
	p := b.pos[before]
	u, v := b.s.ParamAt(p)
	if d := float64(b.s.PointAt(u, v).DistanceTo(p)); d > geom.ResolutionForPoints(b.pos).Weld() {
		t.Errorf("the ear centre is %g off the surface; it must be evaluated on it", d)
	}
	if !b.r.inWindow(b.uu[before], b.vv[before]) {
		t.Errorf("the ear centre (%g, %g) is outside the branch window", b.uu[before], b.vv[before])
	}
}

// TestKeptWithoutRimEarsClearsEveryEar: what comes back holds no rim-only triangle, and on the windowed
// wall — whose window corners are exactly the kind of corner an ear closes — it does not decline.
func TestKeptWithoutRimEarsClearsEveryEar(t *testing.T) {
	t.Parallel()
	b, loops, _ := windowedWallCover(t)
	kept, ok := b.keptWithoutRimEars(loops)
	if !ok {
		t.Fatal("keptWithoutRimEars declined the windowed wall")
	}
	if ears := b.rimEars(kept); len(ears) != 0 {
		t.Errorf("%d rim-only ear(s) survive in the kept triangulation", len(ears))
	}
	if len(kept) == 0 {
		t.Error("the kept triangulation is empty")
	}
}
