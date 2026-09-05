// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"oblikovati.org/math"
)

// A plane exactly tangent to a torus's tube makes the two boundary roots of |w(v)| = 1 a DOUBLE root,
// and arccos is infinitely steep there: a ratio short of 1 by half an ulp put the root's two halves
// 1.5e-08 apart in v, so each lobe of the section came back as an arc that misses closing on itself by
// 1.03e-07. Downstream that is not a hair — the stitch stores a near-closed edge as an OPEN one and
// recovers its direction by inverting the curve at two endpoints 1e-07 apart, which does not round-trip,
// and one lobe's loop came back wound against its own material (ADR-0061).
func TestTangentSectionArcsCloseOnThemselves(t *testing.T) {
	t.Parallel()
	// tol:calibrated — a few ulps of the tube radius: a tangency is an exact coincidence, not a near one.
	const ulps = 8 * 2.220446049250313e-16 * 2
	for _, row := range []struct {
		name   string
		axis   math.Vector3
		origin math.Point3
		normal math.Vector3
	}{
		{"axis-parallel, tangent to the inner equator", math.V3(0, 0, 1), math.P3(0, 3, 0), math.V3(0, 1, 0)},
		{"oblique, at the one/two-oval transition", math.V3(0, 0.6, 0.8), math.P3(0, 0, 1), math.V3(0, 0, 1)},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			arcs := tangentSectionArcs(t, row.axis, row.origin, row.normal)
			for i, c := range arcs {
				lo, hi := c.Domain()
				if gap := float64(c.PointAt(lo).DistanceTo(c.PointAt(hi))); gap > ulps {
					t.Errorf("lobe %d misses closing by %.3g, want ≤ %.3g", i, gap, ulps)
				}
			}
			if a, b := arcs[0].PointAt(0), arcs[1].PointAt(0); float64(a.DistanceTo(b)) > ulps {
				t.Errorf("the two lobes meet at %v and %v — one tangency, one point", a, b)
			}
		})
	}
}

// A section that is NOT tangent keeps its two open arcs: the snap must fire at the limit and nowhere
// else, or every ordinary oval would collapse.
func TestANonTangentSectionKeepsItsOpenArcs(t *testing.T) {
	t.Parallel()
	arcs := tangentSectionArcs(t, math.V3(0, 0, 1), math.P3(0, 3.01, 0), math.V3(0, 1, 0))
	for i, c := range arcs {
		lo, hi := c.Domain()
		if gap := float64(c.PointAt(lo).DistanceTo(c.PointAt(hi))); gap < 0.1 {
			t.Errorf("arc %d closed to %.3g on a plane 0.01 clear of the tangency", i, gap)
		}
	}
}

// tangentSectionArcs is the two-lobe spiric section of the R=5 r=2 torus cut by the given plane.
func tangentSectionArcs(t *testing.T, axis math.Vector3, origin math.Point3, normal math.Vector3) []Curve3 {
	t.Helper()
	tor, err := NewTorus(math.P3(0, 0, 0), axis, 5, 2)
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	pl, err := NewPlane(origin, normal)
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	arcs, ok := TorusPlaneSection(tor, pl)
	if !ok || len(arcs) != 2 {
		t.Fatalf("section: ok=%v, %d arcs, want 2 lobes", ok, len(arcs))
	}
	return arcs
}
