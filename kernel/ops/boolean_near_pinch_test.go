// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearPinchIntersectVolume is the exact volume of {y²+z² ≤ ra²} ∩ {x²+y² ≤ rb²} — a rod of radius ra
// (axis x) crossing a cylinder of radius rb (axis z) through the centre. Used as the analytic oracle for
// the near-pinch band (#1781): the crossing intersection must match it within the DefaultQuality budget.
func nearPinchIntersectVolume(ra, rb float64) float64 {
	const n = 20000
	sum := 0.0
	for i := 0; i < n; i++ {
		phi := 2 * stdmath.Pi * (float64(i) + 0.5) / n
		c := stdmath.Cos(phi) * stdmath.Cos(phi)
		if c < 1e-9 {
			sum += rb * ra * ra
			continue
		}
		sum += (2.0 / (3 * c)) * (rb*rb*rb - stdmath.Pow(rb*rb-ra*ra*c, 1.5))
	}
	return sum * 2 * stdmath.Pi / n
}

// crossingCyls builds two perpendicular solid cylinders (axis x radius R, axis z radius R+dr), sized so the
// bicylinder lies well between the caps.
func crossingCyls(t *testing.T, r, dr float64) (cx, cz *topo.Body) {
	t.Helper()
	h := 4 * r
	var err error
	if cx, err = brep.SolidCylinder(math.P3(-2*r, 0, 0), math.V3(1, 0, 0), r, h); err != nil {
		t.Fatalf("SolidCylinder x: %v", err)
	}
	if cz, err = brep.SolidCylinder(math.P3(0, 0, -2*r), math.V3(0, 0, 1), r+dr, h); err != nil {
		t.Fatalf("SolidCylinder z: %v", err)
	}
	return cx, cz
}

// TestNearPinchRecoveredBandWatertight is the #1781 acceptance gate: crossing cylinders whose imprint neck
// is well-separated (the band recovered above the near-pinch gate) must Intersect to the EXACT three-face
// analytic solid — watertight (0 free edges), a valid closed manifold, volume within the DefaultQuality
// budget of the analytic oracle — and must NOT decline to the faceted fallback. Swept at two model scales
// (R=3 and R=30) with |Δr| scaled by R, so the dimensionless gate is exercised identically at both — the
// scale-invariance the near-pinch ratio gate promises.
func TestNearPinchRecoveredBandWatertight(t *testing.T) {
	for _, r := range []float64{3.0, 30.0} {
		for _, k := range []float64{1.2e-4, 1.6e-4, 2.4e-4, 4.0e-4, 6.0e-4} {
			dr := k * (r / 3.0) // scale |Δr| with R so |Δr|/R (hence the loop gap/chord ratio) matches
			t.Run(fmt.Sprintf("recovered/R=%g/dr=%g", r, dr), func(t *testing.T) {
				cx, cz := crossingCyls(t, r, dr)
				rec := &diag.Recorder{}
				got, err := BooleanWithDiagnostics(Intersect, cx, cz, rec)
				if err != nil {
					t.Fatalf("Boolean(Intersect): %v", err)
				}
				if rec.Has(brep.CodeImprintNearPinchDeclined) {
					t.Fatalf("R=%g dr=%g must ship the analytic path, not decline (#1781)", r, dr)
				}
				assertWatertightCrossing(t, got, r, dr)
			})
		}
	}
}

// assertWatertightCrossing pins the analytic three-face crossing result: valid closed manifold solid, mesh
// with zero free edges (the #1781 watertightness win), and volume within 4% of the analytic oracle.
func assertWatertightCrossing(t *testing.T, got *topo.Body, r, dr float64) {
	t.Helper()
	if v := Validate(got); !v.Valid || !v.Closed || !v.Manifold || !got.IsSolid() {
		t.Fatalf("crossing not a valid closed manifold solid: %+v", v)
	}
	if n := len(got.Faces()); n != 3 {
		t.Errorf("result has %d faces, want 3 (rod band + two lens caps)", n)
	}
	m, _ := TessellateBody(got, DefaultQuality())
	if free := freeEdgeCount(m); free != 0 {
		t.Errorf("R=%g dr=%g meshed with %d free edges; want 0 (watertight)", r, dr, free)
	}
	vol := BodyGeometryProperties(got, DefaultQuality()).Volume
	want := nearPinchIntersectVolume(r, r+dr)
	if rel := stdmath.Abs(vol-want) / want; rel > 0.04 {
		t.Errorf("R=%g dr=%g volume %.4f, want %.4f (analytic) — rel %.3f > 4%%", r, dr, vol, want, rel)
	}
}

// TestNearPinchResidualBandDeclinesCleanly pins the residual lower band (#1818): where the neck is below the
// gate the boolean must DECLINE the analytic path (recorded, never silent) and take the faceted fallback,
// whose VOLUME must not collapse (a user gets degraded-but-correct bulk, not a wrong tiny lump — the ~98%
// collapse the analytic arrangement would produce here). The fallback is knowingly NON-watertight in this
// band (the ~6300-face CSG has sliver edges, #1780) — making it watertight analytically is #1818; this test
// pins only that the decline is recorded and the volume is preserved, so #1818's arrival is a clean upgrade.
func TestNearPinchResidualBandDeclinesCleanly(t *testing.T) {
	const r = 3.0
	for _, dr := range []float64{4e-5, 6e-5, 8e-5} { // |Δr|/r ≈ 1.3e-5 .. 2.7e-5, above the snap ceiling
		t.Run(fmt.Sprintf("residual/dr=%g", dr), func(t *testing.T) {
			cx, cz := crossingCyls(t, r, dr)
			rec := &diag.Recorder{}
			got, err := BooleanWithDiagnostics(Intersect, cx, cz, rec)
			if err != nil {
				t.Fatalf("Boolean(Intersect): %v", err)
			}
			if !rec.Has(brep.CodeImprintNearPinchDeclined) {
				t.Fatalf("dr=%g (below the near-pinch gate) must record the decline, not ship a silent analytic result", dr)
			}
			vol := BodyGeometryProperties(got, DefaultQuality()).Volume
			want := nearPinchIntersectVolume(r, r+dr)
			if rel := stdmath.Abs(vol-want) / want; rel > 0.05 {
				t.Errorf("fallback volume %.4f collapsed vs analytic %.4f — rel %.3f > 5%% (the analytic arrangement would collapse it ~98%% here)", vol, want, rel)
			}
		})
	}
}
