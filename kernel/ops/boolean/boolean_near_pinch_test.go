// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/tessellate"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// nearPinchIntersectVolume is the exact volume of {y²+z² ≤ ra²} ∩ {x²+y² ≤ rb²} — a rod of radius ra
// (axis x) crossing a cylinder of radius rb (axis z) through the centre. Used as the analytic oracle for
// the near-pinch band (#1781): the crossing intersection must match it within the DefaultQuality budget.
func nearPinchIntersectVolume(ra, rb float64) float64 {
	const n = 20000
	sum := 0.0
	for i := range n {
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
	if testing.Short() {
		t.Skip("corpus tier (~8s): `make test-corpus`")
	}
	t.Parallel()
	// The whole band down to just above the snap ceiling (|Δr|≈2e-5 at R=3): #1781 recovered the upper part
	// (gap/chord ≥ the gate) and #1818 the residual near-pinch part (per-loop fat-wall trim). All must ship
	// the exact three-face watertight solid; none may decline.
	for _, r := range []float64{3.0, 30.0} {
		for _, k := range []float64{2e-5, 4e-5, 8e-5, 1.6e-4, 3.2e-4, 6.0e-4} {
			dr := k * (r / 3.0) // scale |Δr| with R so |Δr|/R (hence the loop gap/chord ratio) matches
			t.Run(fmt.Sprintf("R=%g/dr=%g", r, dr), func(t *testing.T) {
				cx, cz := crossingCyls(t, r, dr)
				rec := &diag.Recorder{}
				got, err := BooleanWithDiagnostics(Intersect, cx, cz, rec)
				if err != nil {
					t.Fatalf("Boolean(Intersect): %v", err)
				}
				if rec.Has(brep.CodeImprintNearPinchDeclined) {
					t.Fatalf("R=%g dr=%g must ship the analytic per-loop path, not decline (#1818)", r, dr)
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
	for _, gq := range gateQualities() {
		m, _ := tessellate.TessellateBody(got, gq.q)
		if free := freeEdgeCount(m); free != 0 {
			t.Errorf("R=%g dr=%g at %s quality meshed with %d free edges; want 0 (watertight)", r, dr, gq.name, free)
		}
	}
	vol := query.BodyGeometryProperties(got, DefaultQuality()).Volume
	want := nearPinchIntersectVolume(r, r+dr)
	if rel := stdmath.Abs(vol-want) / want; rel > 0.04 {
		t.Errorf("R=%g dr=%g volume %.4f, want %.4f (analytic) — rel %.3f > 4%%", r, dr, vol, want, rel)
	}
}

// TestNearPinchCutJoinWatertight is the #1818 cut/join acceptance gate: an unequal-radius near-pinch crossing
// (below the #1781 upper-band gate, down to just above the Steinmetz snap ceiling) must SUBTRACT and UNION to
// an EXACT watertight analytic solid — a valid closed manifold, 0 free mesh edges, volume within the
// DefaultQuality budget of the analytic oracle — and must NOT decline. Cut (cx−cz) severs the near-equal rod
// into two stubs (rod − intersection); Join fuses both cylinders (fat + thin − intersection). Every near-pinch
// face is built from the raw whole imprint loops so each shared loop welds as one topo edge (the drilled fat
// wall is the corridor-seeded keyhole; the thin walls are raw two-rim stubs), a global reorient fixing the
// winding. Swept at two model scales with |Δr| scaled by R, so the dimensionless near-pinch gate is exercised
// identically at both.
func TestNearPinchCutJoinWatertight(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~8s): `make test-corpus`")
	}
	t.Parallel()
	for _, r := range []float64{3.0, 30.0} {
		for _, op := range []PartFeatureOperation{Cut, Join} {
			for _, k := range []float64{4e-5, 6e-5, 1.6e-4, 3.2e-4} {
				dr := k * (r / 3.0)
				t.Run(fmt.Sprintf("%v/R=%g/dr=%g", op, r, dr), func(t *testing.T) {
					cx, cz := crossingCyls(t, r, dr)
					rec := &diag.Recorder{}
					got, err := BooleanWithDiagnostics(op, cx, cz, rec)
					if err != nil {
						t.Fatalf("Boolean(%v): %v", op, err)
					}
					if rec.Has(brep.CodeImprintNearPinchDeclined) {
						t.Fatalf("%v R=%g dr=%g must ship the analytic near-pinch path, not decline (#1818)", op, r, dr)
					}
					assertWatertightSolid(t, got)
					assertCutJoinVolume(t, op, got, r, dr)
				})
			}
		}
	}
}

// assertWatertightSolid pins a boolean result as a valid closed manifold whose mesh has zero free edges.
func assertWatertightSolid(t *testing.T, got *topo.Body) {
	t.Helper()
	if v := Validate(got); !v.Valid || !v.Closed || !v.Manifold || !got.IsSolid() {
		t.Fatalf("result not a valid closed manifold solid: %+v", v)
	}
	for _, gq := range gateQualities() {
		if m, _ := tessellate.TessellateBody(got, gq.q); freeEdgeCount(m) != 0 {
			t.Errorf("%s quality: meshed with %d free edges; want 0 (watertight)", gq.name, freeEdgeCount(m))
		}
	}
}

// assertCutJoinVolume checks the cut/join bulk against the analytic oracle: cx (rod, radius r) − cz (radius
// r+Δr) is the rod minus the crossing intersection; the union adds both cylinders less that intersection.
func assertCutJoinVolume(t *testing.T, op PartFeatureOperation, got *topo.Body, r, dr float64) {
	t.Helper()
	h := 4 * r
	rod, fat := stdmath.Pi*r*r*h, stdmath.Pi*(r+dr)*(r+dr)*h
	inter := nearPinchIntersectVolume(r, r+dr)
	want := rod - inter
	if op == Join {
		want = rod + fat - inter
	}
	vol := query.BodyGeometryProperties(got, DefaultQuality()).Volume
	if rel := stdmath.Abs(vol-want) / want; rel > 0.04 {
		t.Errorf("%v R=%g dr=%g volume %.4f, want %.4f (analytic) — rel %.3f > 4%%", op, r, dr, vol, want, rel)
	}
}
