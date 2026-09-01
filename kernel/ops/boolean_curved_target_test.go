// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestBooleanCutCurvedTargetRemovesTunnel is a regression for the boolean that
// did nothing when the *target* (minuend) was a curved body. classify() bounded
// the cylinder by its seam vertices only, judged it disjoint from the tool, and
// returned the target uncut. With curved-edge-aware RangeBox the cut runs: a box
// bored through a cylinder removes its volume.
//
// cyl(r=3.5, h=4) volume = pi*r^2*h = 153.94 (faceted slightly less); a 2x2 box
// tunnel through it removes 2*2*4 = 16.
func TestBooleanCutCurvedTargetRemovesTunnel(t *testing.T) {
	t.Parallel()
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	full := query.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	tool := csgBox(math.P3(-1, -1, -1), 2, 2, 6) // 2x2 cross-section, pokes through both caps

	res, err := ops.Boolean(ops.Cut, cyl, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("cut result not a valid solid: %+v", r)
	}
	got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if want := full - 16; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("curved-target cut volume = %.5f, want %.5f (full %.5f − 16)", got, want, full)
	}
}

// TestBooleanNearTangentCylindersStayManifold extends the curved-boolean suite per #1598
// (audit A2): perpendicular analytic cylinders with nearly equal radii approach the pinched
// Steinmetz configuration — the near-tangent booleans where the SSI marcher used to
// branch-jump or falsely close and the trim then followed the wrong locus. Two regimes, split
// by the near-pinch gate (#1781): where the two imprint loops' neck is well-separated relative
// to the imprint chord the general rod-band path builds an Euler–Poincaré-valid analytic solid
// (|Δr|/r = 5e-4 AND the recovered |Δr|/r = 5e-5 upper-band case); only where the neck is too
// narrow for the (u,v) arrangement (|Δr|/r = 2.5e-5, the residual lower band) does the imprint
// DECLINE to the recorded faceted fallback — never a silently wrong analytic assembly. Folding
// that residual band onto the analytic path is #1817.
func TestBooleanNearTangentCylindersStayManifold(t *testing.T) {
	t.Parallel()
	const r = 2.0
	vCyl := stdmath.Pi * r * r * 6
	steinmetz := 16.0 * r * r * r / 3
	want := 2*vCyl - steinmetz // union volume; the intersection is a hair under Steinmetz

	// The general analytic path holds across the clean crossing AND the recovered upper near-pinch
	// band (dr=5e-5 → |Δr|/r=2.5e-5 was declined before #1781, now ships watertight).
	for _, dr := range []float64{1e-3, 1e-4} {
		t.Run("general path at dr="+fmtDr(dr), func(t *testing.T) {
			union, rec := nearTangentUnion(t, r, dr)
			if res := ops.Validate(union); !res.Valid || !union.IsSolid() {
				t.Fatalf("near-tangent union not a valid solid: %+v", res)
			}
			if rec.Has(brep.CodeImprintNearPinchDeclined) {
				t.Fatalf("dr=%g must stay on the general analytic path, not decline (#1781)", dr)
			}
			got := query.BodyGeometryProperties(union, ops.DefaultQuality()).Volume
			if stdmath.Abs(got-want) > 0.02*want {
				t.Errorf("union volume = %.4f, want ≈ %.4f (2·cyl − Steinmetz)", got, want)
			}
		})
	}

	t.Run("analytic path in the residual lower band", func(t *testing.T) {
		// dr=5e-5 (|Δr|/r = 2.5e-5) is deep in the near-pinch band that #1781 declined; #1818 now ships it
		// analytically (nearPinchCrossingJoin: raw whole-loop stubs + corridor-seeded keyhole wall).
		union, rec := nearTangentUnion(t, r, 5e-5)
		if res := ops.Validate(union); !res.Valid || !union.IsSolid() {
			t.Fatalf("residual near-pinch union not a valid solid: %+v", res)
		}
		if rec.Has(brep.CodeImprintNearPinchDeclined) {
			t.Fatalf("residual near-pinch union must ship the analytic path, not decline (#1818)")
		}
		if got := query.BodyGeometryProperties(union, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 0.02*want {
			t.Errorf("union volume = %.4f, want ≈ %.4f (2·cyl − Steinmetz)", got, want)
		}
	})
}

// fmtDr formats a radius gap for a subtest name without pulling in a format verb per call site.
func fmtDr(dr float64) string {
	switch dr {
	case 1e-3:
		return "1e-3"
	case 1e-4:
		return "1e-4"
	default:
		return "dr"
	}
}

// nearTangentUnion joins two perpendicular solid cylinders of radius r and r−dr with a recorder.
func nearTangentUnion(t *testing.T, r, dr float64) (*topo.Body, *diag.Recorder) {
	t.Helper()
	a, err := brep.SolidCylinder(math.P3(0, 0, -3), math.V3(0, 0, 1), r, 6)
	if err != nil {
		t.Fatalf("SolidCylinder a: %v", err)
	}
	b, err := brep.SolidCylinder(math.P3(-3, 0, 0), math.V3(1, 0, 0), r-dr, 6)
	if err != nil {
		t.Fatalf("SolidCylinder b: %v", err)
	}
	rec := &diag.Recorder{}
	union, err := ops.BooleanWithDiagnostics(ops.Join, a, b, rec)
	if err != nil {
		t.Fatalf("Boolean(Join): %v", err)
	}
	return union, rec
}
