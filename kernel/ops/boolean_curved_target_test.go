// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
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
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3.5, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	full := ops.BodyGeometryProperties(cyl, ops.DefaultQuality()).Volume
	tool := csgBox(math.P3(-1, -1, -1), 2, 2, 6) // 2x2 cross-section, pokes through both caps

	res, err := ops.Boolean(ops.Cut, cyl, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut): %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("cut result not a valid solid: %+v", r)
	}
	got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if want := full - 16; stdmath.Abs(got-want) > 1e-3 {
		t.Errorf("curved-target cut volume = %.5f, want %.5f (full %.5f − 16)", got, want, full)
	}
}

// TestBooleanNearTangentCylindersStayManifold extends the curved-boolean suite per #1598
// (audit A2): perpendicular analytic cylinders with nearly equal radii approach the pinched
// Steinmetz configuration — the near-tangent booleans where the SSI marcher used to
// branch-jump or falsely close and the trim then followed the wrong locus. At |Δr|/r = 5e-4
// the general rod-band path must produce an Euler–Poincaré-valid solid within the analytic
// volume bracket. At |Δr|/r = 5e-5 (inside the near-pinch band) the general imprint DECLINES
// (input-sensitive saddle topology) and the boolean takes the recorded faceted fallback: the
// volume stays inside the bracket and the degradation is visible on the recorder — never a
// silently wrong analytic assembly (#1403 tracks unifying the band onto the exact path).
func TestBooleanNearTangentCylindersStayManifold(t *testing.T) {
	const r = 2.0
	vCyl := stdmath.Pi * r * r * 6
	steinmetz := 16.0 * r * r * r / 3
	want := 2*vCyl - steinmetz // union volume; the intersection is a hair under Steinmetz

	t.Run("general path at dr=1e-3", func(t *testing.T) {
		union, rec := nearTangentUnion(t, r, 1e-3)
		if res := ops.Validate(union); !res.Valid || !union.IsSolid() {
			t.Fatalf("near-tangent union not a valid solid: %+v", res)
		}
		if rec.Has(brep.CodeImprintNearPinchDeclined) {
			t.Fatal("dr=1e-3 must stay on the general path, not decline")
		}
		got := ops.BodyGeometryProperties(union, ops.DefaultQuality()).Volume
		if stdmath.Abs(got-want) > 0.02*want {
			t.Errorf("union volume = %.4f, want ≈ %.4f (2·cyl − Steinmetz)", got, want)
		}
	})

	t.Run("recorded fallback inside the near-pinch band", func(t *testing.T) {
		union, rec := nearTangentUnion(t, r, 1e-4)
		if !rec.Has(brep.CodeImprintNearPinchDeclined) {
			t.Fatal("near-pinch decline must be RECORDED, not silent (#1598)")
		}
		got := ops.BodyGeometryProperties(union, ops.DefaultQuality()).Volume
		// The faceted fallback inscribes the cylinders, so it systematically under-measures
		// by the facet sagitta — allow 3% where the general path gets 2%.
		if stdmath.Abs(got-want) > 0.03*want {
			t.Errorf("fallback union volume = %.4f, want ≈ %.4f — the faceted route must not lose material", got, want)
		}
	})
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
