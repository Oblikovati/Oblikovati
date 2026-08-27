// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestReconstructionCutoverShadow is the ADR-0054 Layer-5 cutover gate: it shadow-validates the
// analytic reconstruction against a KNOWN analytic volume for a matrix of curved booleans that
// no analytic recognizer handles (so reconstructedCurvedBoolean is the path that fires). The
// invariant the cutover rests on is codified here — reconstruction is never ADOPTED WRONG:
//
//   - where it FIRES it is a valid closed manifold solid whose fine-tessellated volume matches
//     the analytic value within the facet budget, and keeps the analytic curved faces;
//   - where it CANNOT (a transversal crossing whose SSI edges do not yet weld), it DECLINES, so
//     the operation falls back to the guarded analytic/faceted path — never a wrong solid.
//
// The exact-volume checks are the shadow oracle the cutover's runtime guard (validity + the
// Requicha bracket) is trusted against.
func TestReconstructionCutoverShadow(t *testing.T) {
	const r = 3.0
	cases := []struct {
		name     string
		a, b     func(t *testing.T) *topo.Body
		op       meshbool.Op
		analytic float64 // exact volume when reconstruction fires; NaN when it must decline
		wantCyl  int     // analytic cylinder faces expected in the result (−1: don't assert)
	}{
		{
			"cocylindrical-D-join", // #2167: a cylinder with a stacked cocylindrical D-prism
			shCyl(math.P3(0, 0, 0), r, 6), shDPrism(r, 0.6, 6, 10),
			meshbool.Union, dJoinVolume(r, 0.6, 6, 4), 1,
		},
		{
			"coaxial-equal-cylinders", // a stepped/stacked shaft: two coaxial equal-radius cylinders
			shCyl(math.P3(0, 0, 0), r, 6), shCyl(math.P3(0, 0, 6), r, 4),
			meshbool.Union, stdmath.Pi * r * r * 10, 1,
		},
		{
			"box-union", // a purely planar union still reconstructs exactly (no curved face)
			shBox(0, 0, 0, 2, 2, 2), shBox(1, 1, 1, 3, 3, 3),
			meshbool.Union, 8 + 8 - 1, 0,
		},
		{
			"crossing-cylinders", // transversal SSI: reconstruction cannot weld the ellipse edges yet → must DECLINE
			shCyl(math.P3(0, 0, -6), r, 12), shCylAxis(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12),
			meshbool.Union, stdmath.NaN(), -1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body, ok := reconstructBoolean(c.a(t), c.b(t), c.op, DefaultQuality())
			if stdmath.IsNaN(c.analytic) {
				if ok && validBooleanSolid(body) {
					t.Fatalf("%s: reconstruction produced a solid but is expected to DECLINE (SSI edges do not weld yet)", c.name)
				}
				return
			}
			if !ok || !validBooleanSolid(body) {
				t.Fatalf("%s: reconstruction declined or produced an invalid solid (ok=%v)", c.name, ok)
			}
			if c.wantCyl >= 0 {
				if got := cylinderFaceCount(body); got != c.wantCyl {
					t.Errorf("%s: %d analytic cylinder walls, want %d", c.name, got, c.wantCyl)
				}
			}
			if v := soupVolume(bodyToSoup(body, PropertyQuality())); stdmath.Abs(v-c.analytic) > 5e-3*c.analytic {
				t.Errorf("%s: volume %.4f vs analytic %.4f (rel %.4f > 0.5%%)", c.name, v, c.analytic, stdmath.Abs(v-c.analytic)/c.analytic)
			}
		})
	}
}

// dJoinVolume is the exact volume of a radius-r cylinder of height h1 joined with a stacked
// D-segment prism (disc minus the minor segment the chord at half-angle theta cuts) of height h2.
func dJoinVolume(r, theta, h1, h2 float64) float64 {
	minor := 0.5 * r * r * (2*theta - stdmath.Sin(2*theta))
	return stdmath.Pi*r*r*h1 + (stdmath.Pi*r*r-minor)*h2
}

func shCyl(base math.Point3, r, h float64) func(*testing.T) *topo.Body {
	return shCylAxis(base, math.V3(0, 0, 1), r, h)
}

func shCylAxis(base math.Point3, axis math.Vector3, r, h float64) func(*testing.T) *topo.Body {
	return func(t *testing.T) *topo.Body {
		b, err := brep.SolidCylinder(base, axis, r, h)
		if err != nil {
			t.Fatalf("cylinder: %v", err)
		}
		return b
	}
}

func shDPrism(r, theta, z0, z1 float64) func(*testing.T) *topo.Body {
	return func(*testing.T) *topo.Body { return dPrismBody(r, theta, z0, z1, "d") }
}

func shBox(x0, y0, z0, x1, y1, z1 float64) func(*testing.T) *topo.Body {
	return func(t *testing.T) *topo.Body {
		b, err := brep.SolidBlock(math.P3(x0, y0, z0), math.P3(x1, y1, z1), "block")
		if err != nil {
			t.Fatalf("block: %v", err)
		}
		return b
	}
}
