// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
)

// partWithCylinder is the curved counterpart of partWithBlock: a solid cylinder on the Z axis,
// whose overlap volume has a closed form (πr²h) the analytic integral must reproduce exactly.
func partWithCylinder(t *testing.T, baseZ, radius, height float64) *PartComponentDefinition {
	t.Helper()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, math.Scalar(baseZ)), math.V3(0, 0, 1), math.Scalar(radius), math.Scalar(height))
	if err != nil {
		t.Fatalf("SolidCylinder(baseZ=%g, r=%g, h=%g): %v", baseZ, radius, height, err)
	}
	p := NewPartComponentDefinition()
	feature.NewBaseFeatures(p.Features()).AddBase(cyl)
	p.Recompute()
	return p
}

// TestAnalyzeInterferenceMeasuresACurvedOverlap is the M48/C3 regression
// (Oblikovati/Oblikovati#3451): two coaxial cylinders overlapping over one unit of height interfere
// by πr²h = π. The retired measurement summed the intersection's shells at the DISPLAY quality, and
// an inscribed N-gon under-reports a curved face by ~π²/(3N²): it read 3.121445 — 0.64% low, four
// times the tolerance asserted here — while the analytic-first path reads 3.141277.
//
// The residual ~1e-4 is not this site's: query.BodyGeometryProperties still declines a body whose
// cylindrical wall the boolean left as a seam-wrapping loop, and falls back to the mesh. Closing
// that coverage gap (#3453) makes this assertion exact with no change here.
func TestAnalyzeInterferenceMeasuresACurvedOverlap(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	lower := partWithCylinder(t, 0, 1, 2) // z ∈ [0, 2]
	upper := partWithCylinder(t, 0, 1, 2)
	asm := NewAssemblyComponentDefinition()
	asm.Place("lower:1", lower, math.Identity4())
	asm.Place("upper:1", upper, math.Translation4(math.V3(0, 0, 1))) // z ∈ [1, 3]

	res := asm.AnalyzeInterference(nil)
	if res.Count() != 1 {
		t.Fatalf("interference count = %d, want 1", res.Count())
	}
	if got := res.Results[0].Vol; stdmath.Abs(got-stdmath.Pi) > 5e-3 {
		t.Errorf("overlap volume = %.9f, want %.9f (πr²h, r=1 h=1)", got, stdmath.Pi)
	}
}

// TestAnalyzeInterferenceCenterIsOverlapCentroid pins the representative point to the overlap's
// own centroid: the cylinders above meet in z ∈ [1, 2], so the point sits at (0, 0, 1.5) — the
// bounding-box centre of the intersection lump the retired code reported only coincides here
// because the lump is symmetric, so the assertion is on the value, not on the method.
func TestAnalyzeInterferenceCenterIsOverlapCentroid(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4s): `make test-corpus`")
	}
	t.Parallel()
	asm := NewAssemblyComponentDefinition()
	asm.Place("lower:1", partWithCylinder(t, 0, 1, 2), math.Identity4())
	asm.Place("upper:1", partWithCylinder(t, 0, 1, 2), math.Translation4(math.V3(0, 0, 1)))

	res := asm.AnalyzeInterference(nil)
	if res.Count() != 1 {
		t.Fatalf("interference count = %d, want 1", res.Count())
	}
	if got, want := res.Results[0].Center, math.P3(0, 0, 1.5); !got.IsEqualTo(want, 1e-9) {
		t.Errorf("overlap centre = %v, want %v", got, want)
	}
}

// TestOverlapSumWeightsLumpsByVolume covers overlapSum directly: a negligible lump is dropped, and
// two real lumps average into a VOLUME-weighted centre, not an unweighted midpoint.
func TestOverlapSumWeightsLumpsByVolume(t *testing.T) {
	t.Parallel()
	var s overlapSum
	s.fold(ops.GeometryProperties{Volume: interferenceVolumeEps / 2, Centroid: math.P3(100, 0, 0)})
	s.fold(ops.GeometryProperties{Volume: 3, Centroid: math.P3(0, 0, 0)})
	s.fold(ops.GeometryProperties{Volume: 1, Centroid: math.P3(4, 0, 0)})

	if s.volume != 4 {
		t.Errorf("folded volume = %g, want 4 (the negligible lump must be dropped)", s.volume)
	}
	if got, want := s.centroid(), math.P3(1, 0, 0); !got.IsEqualTo(want, 1e-9) {
		t.Errorf("weighted centre = %v, want %v", got, want)
	}
}

// TestOverlapSumCentroidWithoutLumps covers the empty accumulator: no lump folded ⇒ the origin,
// never a division by zero.
func TestOverlapSumCentroidWithoutLumps(t *testing.T) {
	t.Parallel()
	var s overlapSum
	if got := s.centroid(); !got.IsEqualTo(math.P3(0, 0, 0), 0) {
		t.Errorf("empty overlap centre = %v, want the origin", got)
	}
}

// twoBoxAssembly places two unit-ish boxes; boxB is offset by dx along X so the caller controls
// whether they overlap. Returns the assembly and the two occurrence ids.
func twoBoxAssembly(t *testing.T, dx float64) (*AssemblyComponentDefinition, uint64, uint64) {
	t.Helper()
	a := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	b := partWithBlock(t, math.P3(0, 0, 0), math.P3(2, 2, 2))
	asm := NewAssemblyComponentDefinition()
	oa := asm.Place("a:1", a, math.Identity4())
	ob := asm.Place("b:1", b, math.Translation4(math.V3(dx, 0, 0)))
	return asm, oa.ID(), ob.ID()
}

// TestAnalyzeInterferenceDetectsOverlap covers AnalyzeInterference and its helpers: overlapping
// occurrences are reported with positive volume; separated ones are not.
func TestAnalyzeInterferenceDetectsOverlap(t *testing.T) {
	t.Parallel()
	overlap, _, _ := twoBoxAssembly(t, 1) // B spans x∈[1,3], overlaps A's x∈[0,2]
	res := overlap.AnalyzeInterference(nil)
	if res.Count() != 1 {
		t.Fatalf("interference count = %d, want 1", res.Count())
	}
	if res.Total <= 0 {
		t.Errorf("interference total volume = %g, want > 0", res.Total)
	}

	apart, _, _ := twoBoxAssembly(t, 10) // B spans x∈[10,12], no overlap
	if got := apart.AnalyzeInterference(nil).Count(); got != 0 {
		t.Errorf("separated parts report %d interferences, want 0", got)
	}
}

// TestAnalyzeInterferenceSubsetFilter covers pairInSubset/idInSet: an empty-but-non-nil subset
// (a pair not naming both ids) excludes the pair.
func TestAnalyzeInterferenceSubsetFilter(t *testing.T) {
	t.Parallel()
	asm, a, b := twoBoxAssembly(t, 1)
	if got := asm.AnalyzeInterference([]uint64{a, b}).Count(); got != 1 {
		t.Errorf("subset naming both ids: count = %d, want 1", got)
	}
	if got := asm.AnalyzeInterference([]uint64{a, a}).Count(); got != 0 {
		t.Errorf("subset not naming b: count = %d, want 0", got)
	}
}

// TestWouldContactBlockWithoutContactSet covers the early no-block path: an occurrence in no
// contact set never blocks a move.
func TestWouldContactBlockWithoutContactSet(t *testing.T) {
	t.Parallel()
	asm, a, _ := twoBoxAssembly(t, 1)
	if asm.WouldContactBlock(a, math.Translation4(math.V3(0.5, 0, 0))) {
		t.Error("WouldContactBlock should be false when the occurrence shares no contact set")
	}
}
