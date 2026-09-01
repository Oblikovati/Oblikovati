// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The squat-rim bore (Oblikovati/Oblikovati#2038): a Ø0.3 rod bored clean through a R=5 × 0.4 disk.
// An extruded circle's analytic cylinder pins angle-0 to its sketch +X (extrude_analytic.go), so an
// XY-sketched disk carries its seam at +X — and a bore sketched on YZ runs straight through it. The
// wall's lens holes then straddled the seam, the tessellator fell back to a flat CDT that covers half
// the wrap, and the part reported 7.1 cm³ against an analytic 30.7 with a valid solid and no
// diagnostic. See kernel/ops/wall_seam_recut.go.

// boredDiskVolume is the analytic volume of the disk minus the full tunnel.
func boredDiskVolume() float64 { return stdmath.Pi*25*0.4 - stdmath.Pi*0.15*0.15*10 }

// boredDisk builds the disk on XY and cuts the bore from borePlane under the given extent direction.
func boredDisk(t *testing.T, borePlane sketch.Plane, dir ExtentDirection) *PartFeatures {
	t.Helper()
	fs := NewPartFeatures(nil)
	disk := sketch.NewSketches().Add(sketch.XYPlane())
	disk.Circles().AddByCenterRadius(math.P2(0, 0), 5)
	NewExtrudeFeatures(fs).AddExtrude(disk, []int{0}, ops.NewBody,
		Extent{Type: DistanceExtent, Direction: SymmetricDir, Distance: func() float64 { return 0.4 }}, 0)
	bore := sketch.NewSketches().Add(borePlane)
	bore.Circles().AddByCenterRadius(math.P2(0, 0), 0.15)
	NewExtrudeFeatures(fs).AddExtrude(bore, []int{0}, ops.Cut, Extent{Type: ThroughAllExtent, Direction: dir}, 0)
	fs.Recompute()
	if n := len(fs.Result()); n != 1 {
		t.Fatalf("recompute gave %d bodies, want 1", n)
	}
	return fs
}

// TestBoredDiskVolumeIsIndependentOfTheBoreAxis: the same part must measure the same whether the bore
// runs along Y (clear of the disk's +X seam) or along X (straight through it). #2038 read the second
// 77% low.
func TestBoredDiskVolumeIsIndependentOfTheBoreAxis(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		plane sketch.Plane
	}{
		{"bore along Y, clear of the seam", sketch.XZPlane()},
		{"bore along X, through the seam", sketch.YZPlane()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := boredDisk(t, tc.plane, SymmetricDir).Result()[0]
			if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
				t.Fatalf("not a valid solid: %+v", r.Issues)
			}
			want := boredDiskVolume()
			got := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume
			if rel := stdmath.Abs(got-want) / want; rel > 1e-3 {
				t.Errorf("volume %.6f, want %.6f (rel %+.4f) — half-covered wall mesh? (#2038)",
					got, want, (got-want)/want)
			}
		})
	}
}

// TestSymmetricThroughAllBoreIsOpenAlongItsWholeLength probes the bore axis at several stations rather
// than trusting a volume band: a volume check alone passes a solid that is missing an end of the
// tunnel by less than its tolerance (#2038's acceptance).
func TestSymmetricThroughAllBoreIsOpenAlongItsWholeLength(t *testing.T) {
	t.Parallel()
	body := boredDisk(t, sketch.XZPlane(), SymmetricDir).Result()[0]
	for _, y := range []float64{-4.5, -3, -1.5, -0.5, 0.5, 1.5, 3, 4.5} {
		p := math.P3(0, math.Scalar(y), 0)
		if c := ops.BodyContainment(body, p, ops.PropertyQuality(), 1e-7); c != ops.ContainOutside {
			t.Errorf("y=%+.1f inside the bore reads %v, want outside — the tunnel is filled there", y, c)
		}
	}
}

// TestPositiveThroughAllBoreCutsOneSideOnly locks the Inventor semantics #2038 misread as a bug: a
// through-all extent takes a DIRECTION (ExtrudeDefinition.SetThroughAllExtent), and
// kPositiveExtentDirection cuts only the side of the sketch plane the normal points to. The XZ plane's
// normal is −Y, so the material at +Y must survive and the −Y half must go.
func TestPositiveThroughAllBoreCutsOneSideOnly(t *testing.T) {
	t.Parallel()
	body := boredDisk(t, sketch.XZPlane(), PositiveDir).Result()[0]
	for _, tc := range []struct {
		y    float64
		want ops.PointContainment
	}{{-4, ops.ContainOutside}, {-2, ops.ContainOutside}, {2, ops.ContainInside}, {4, ops.ContainInside}} {
		p := math.P3(0, math.Scalar(tc.y), 0)
		if c := ops.BodyContainment(body, p, ops.PropertyQuality(), 1e-7); c != tc.want {
			t.Errorf("y=%+.1f reads %v, want %v — a one-direction through-all must not cut both ways",
				tc.y, c, tc.want)
		}
	}
	// Exactly half the tunnel, so a caller measuring against the full tunnel is measuring the wrong thing.
	want := stdmath.Pi*25*0.4 - stdmath.Pi*0.15*0.15*5
	got := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > 2e-3 {
		t.Errorf("volume %.6f, want %.6f (disk minus HALF the tunnel)", got, want)
	}
}
