// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// #2038: a drilled wall's seam sits wherever its builder's angle-0 happened to point, so a lens hole
// can straddle it. holesIntoBranch then fits the lens into no branch, holedConicWallMesh declined, and
// the face fell through to the flat best-fit-plane CDT — which covers only half a full wrap. These
// tests pin the re-cut (wall_seam_recut.go) that removes the dependence on where the seam landed.

// bridgedWallLoop builds the seam-bridged outer loop of a full-wrap wall between v=vBot and v=vTop:
// the seam's bottom vertex, the top rim all the way round, then the bottom rim back — the shape the
// curved boolean hands the tessellator for a drilled cylinder wall.
func bridgedWallLoop(s geom.Surface, vBot, vTop float64, n int) []math.Point3 {
	step := 2 * stdmath.Pi / float64(n)
	out := []math.Point3{s.PointAt(0, vBot)}
	for k := 0; k <= n; k++ { // top rim, θ: 0 → −2π (closing on its own start)
		out = append(out, s.PointAt(-step*float64(k), vTop))
	}
	for k := n; k > 0; k-- { // bottom rim back, θ: −2π → 0 (exclusive: index 0 already holds it)
		out = append(out, s.PointAt(-step*float64(k), vBot))
	}
	return out
}

// TestHoledWallMeshesLensStraddlingTheSeam is the #2038 regression: the wall must mesh to its full
// curved area whether the lens sits ON the seam or clear of it. Before the re-cut the straddling case
// declined here and the caller's flat-CDT fallback covered ~half the wrap.
func TestHoledWallMeshesLensStraddlingTheSeam(t *testing.T) {
	s := bandCylinder(10)
	q := Quality{ChordTolerance: 0.005, AngleTolerance: 2 * stdmath.Pi / 180}
	full := 2 * stdmath.Pi * 3 * 10 // 2πRh = 188.50, minus one small lens
	for _, tc := range []struct {
		name  string
		theta float64
	}{
		{"lens clear of the seam", stdmath.Pi},
		{"lens straddling the seam", 0},
		{"lens just inside the seam", 0.05},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outer := bridgedWallLoop(s, 0, 10, 96)
			lens := lensLoop(s, tc.theta, 5, 0.12, 0.35, 40)
			m, ok := holedConicWallMesh(s, outer, [][]math.Point3{lens}, q)
			if !ok {
				t.Fatal("holedConicWallMesh declined a drilled wall; the flat CDT fallback covers half the wrap (#2038)")
			}
			if area := meshArea(m); area < full-3 || area > full+0.5 {
				t.Errorf("wall area %.3f, want ≈%.3f (full wrap minus a small lens) — half-covered wrap?", area, full)
			}
		})
	}
}

// TestSplitSeamBridgedRimsRecoversBothRims: the split must hand back the two rims intact — every
// sampled vertex, each exactly once (the seam vertex is walked twice and must not come back doubled),
// and separated by their v level.
func TestSplitSeamBridgedRimsRecoversBothRims(t *testing.T) {
	s := bandCylinder(10)
	top, bot, ok := splitSeamBridgedRims(s, bridgedWallLoop(s, 0, 10, 96))
	if !ok {
		t.Fatal("splitSeamBridgedRims declined a seam-bridged wall loop")
	}
	if len(top) != 96 || len(bot) != 96 {
		t.Errorf("rims came back with %d and %d points, want 96 each (a doubled seam vertex?)", len(top), len(bot))
	}
	if v := meanVParam(s, top); stdmath.Abs(v-10) > 1e-9 {
		t.Errorf("top rim mean v = %v, want 10 — the rims are swapped or mixed", v)
	}
	if v := meanVParam(s, bot); stdmath.Abs(v) > 1e-9 {
		t.Errorf("bottom rim mean v = %v, want 0 — the rims are swapped or mixed", v)
	}
}

// TestSplitSeamBridgedRimsDeclinesNonWrappingLoop: a loop that does not wrap the period is not a
// bridged wall, so the split must defer rather than invent rims.
func TestSplitSeamBridgedRimsDeclinesNonWrappingLoop(t *testing.T) {
	s := bandCylinder(10)
	quarter := []math.Point3{s.PointAt(0, 0), s.PointAt(1, 0), s.PointAt(1, 10), s.PointAt(0, 10)}
	if _, _, ok := splitSeamBridgedRims(s, quarter); ok {
		t.Error("splitSeamBridgedRims accepted a quarter-wall loop")
	}
}

// TestCrossingCylinderCutIsSeamIndependent is the invariant the defect broke: the volume of a bored
// disk must not depend on the ANGLE the bore happens to make with the disk's angle-0 seam. The disk's
// seam is where axisFrame put it (+Y for a +Z axis), so the φ=90°/270° rods drill straight through it.
// Before the re-cut those two read ~10.07 cm³ against an analytic 30.71 — a −67% error on a valid,
// Validate-clean solid with no diagnostics.
func TestCrossingCylinderCutIsSeamIndependent(t *testing.T) {
	want := stdmath.Pi*25*0.4 - stdmath.Pi*0.15*0.15*10 // disk − tunnel = 30.709068
	for _, deg := range []float64{0, 45, 90, 135, 180, 270} {
		body := boredDiskAtAzimuth(t, deg)
		if r := Validate(body); !r.Valid || !body.IsSolid() {
			t.Fatalf("φ=%g°: not a valid solid: %+v", deg, r.Issues)
		}
		got := BodyGeometryProperties(body, PropertyQuality()).Volume
		if rel := stdmath.Abs(got-want) / want; rel > 1e-3 {
			t.Errorf("φ=%g°: volume %.6f, want %.6f (rel %+.4f) — the bore straddles the disk's seam (#2038)",
				deg, got, want, (got-want)/want)
		}
	}
}

// boredDiskAtAzimuth cuts a Ø0.3 rod clean through a R=5 × 0.4 disk along the azimuth deg (degrees
// from the +X axis, in the disk's mid-plane), through the exact analytic crossing-cylinder path.
func boredDiskAtAzimuth(t *testing.T, deg float64) *topo.Body {
	t.Helper()
	disk, err := brep.SolidCylinder(math.P3(0, 0, -0.2), math.V3(0, 0, 1), 5, 0.4)
	if err != nil {
		t.Fatalf("disk: %v", err)
	}
	rad := deg * stdmath.Pi / 180
	axis := math.V3(math.Scalar(stdmath.Cos(rad)), math.Scalar(stdmath.Sin(rad)), 0)
	rod, err := brep.SolidCylinder(math.P3(0, 0, 0).TranslateBy(axis.Scale(-6)), axis, 0.15, 12)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	body, ok := CurvedBoolean(Cut, disk, rod)
	if !ok {
		t.Fatalf("φ=%g°: the exact crossing-cylinder path declined", deg)
	}
	return body
}

// TestCrossingCylinderBooleanStaysExactAtScale covers #2038's "adjacent finding at scale". Going
// through the full Boolean entry, the seam-straddling wall's bad mesh made the Requicha volume guard
// (curvedExactGuarded) REJECT a perfectly good analytic result, so the cut fell back to triangle-soup
// CSG: 408 all-planar faces at k=1 and a NON-MANIFOLD body from k=5 up. With the wall meshed correctly
// the analytic result is accepted, so the cut stays exact — and valid — at every scale.
func TestCrossingCylinderBooleanStaysExactAtScale(t *testing.T) {
	for _, k := range []float64{1, 5, 20} {
		disk, err := brep.SolidCylinder(math.P3(0, 0, math.Scalar(-0.2*k)), math.V3(0, 0, 1), 5*k, 0.4*k)
		if err != nil {
			t.Fatalf("k=%v disk: %v", k, err)
		}
		rod, err := brep.SolidCylinder(math.P3(0, math.Scalar(-6*k), 0), math.V3(0, 1, 0), 0.15*k, 12*k)
		if err != nil {
			t.Fatalf("k=%v rod: %v", k, err)
		}
		body, err := Boolean(Cut, disk, rod)
		if err != nil {
			t.Fatalf("k=%v: %v", k, err)
		}
		if r := Validate(body); !r.Valid || !body.IsSolid() {
			t.Fatalf("k=%v: not a valid solid (%d issues, first %v) — CSG fallback?", k, len(r.Issues), r.Issues[0])
		}
		if n := analyticCylinderFaceCount(body); n != 2 {
			t.Errorf("k=%v: %d cylindrical faces on a %d-face body, want 2 (rim + bore) — the cut was faceted",
				k, n, len(body.Faces()))
		}
		want := stdmath.Pi*25*k*k*0.4*k - stdmath.Pi*0.15*0.15*k*k*10*k
		got := BodyGeometryProperties(body, PropertyQuality()).Volume
		if rel := stdmath.Abs(got-want) / want; rel > 1e-3 {
			t.Errorf("k=%v: volume %.6f, want %.6f (rel %+.4f)", k, got, want, (got-want)/want)
		}
	}
}

// analyticCylinderFaceCount tallies the body's true cylindrical faces — zero means the boolean
// abandoned the analytic surfaces for facets.
func analyticCylinderFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			n++
		}
	}
	return n
}
