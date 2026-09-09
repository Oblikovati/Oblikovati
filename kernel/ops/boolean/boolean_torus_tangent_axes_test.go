// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A plane tangent to a torus's inner equator cuts a figure-eight, and the two lobes TOUCH. The island
// gate that refuses two islands "not nested and not apart" answered that touch with an even-odd parity
// test, which has no answer ON a boundary — so the same geometry was refused for a Y-axis and an X-axis
// torus and admitted for a Z-axis one. The corpus takes every axis, because the defect was a coin toss
// (ADR-0061).
func TestTorusTangentCutIsExactAboutEveryAxis(t *testing.T) {
	t.Parallel()
	// R = 5, r = 2: the tangency is at |offset| = R − r = 3, on the axis the cut normal names.
	for _, row := range []struct {
		name   string
		axis   math.Vector3
		bmin   math.Point3
		expect float64 // the kept volume of the INTERSECT, from OCC
	}{
		{"torus about z, tangent on +y", math.V3(0, 0, 1), math.P3(-20, 3, -20), 114.886326},
		{"torus about z, tangent on +x", math.V3(0, 0, 1), math.P3(3, -20, -20), 114.886326},
		{"torus about y, tangent on +x", math.V3(0, 1, 0), math.P3(3, -20, -20), 114.886326},
		{"torus about y, tangent on +z", math.V3(0, 1, 0), math.P3(-20, -20, 3), 114.886326},
		{"torus about x, tangent on +y", math.V3(1, 0, 0), math.P3(-20, 3, -20), 114.886326},
		{"torus about x, tangent on +z", math.V3(1, 0, 0), math.P3(-20, -20, 3), 114.886326},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			res := tangentCut(t, ops.Intersect, row.axis, row.bmin)
			if n := len(res.Faces()); n > 8 {
				t.Fatalf("%d faces — the tangent cut fell to faceted CSG, want the exact analytic path", n)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-row.expect) / row.expect; rel > 0.02 {
				t.Errorf("volume %.6f, want %.6f (OCC) — rel %.4f", got, row.expect, rel)
			}
		})
	}
}

// tangentCut intersects (or cuts) the R=5 r=2 torus about axis with the box [bmin, (20,20,20)].
func tangentCut(t *testing.T, op ops.PartFeatureOperation, axis math.Vector3, bmin math.Point3) *topo.Body {
	t.Helper()
	tor, err := brep.SolidTorus(math.P3(0, 0, 0), axis, 5, 2, "torus")
	if err != nil {
		t.Fatalf("torus: %v", err)
	}
	box, err := brep.SolidBlock(bmin, math.P3(20, 20, 20), "box")
	if err != nil {
		t.Fatalf("box: %v", err)
	}
	res, err := ops.Boolean(op, tor, box)
	if err != nil {
		t.Fatalf("boolean: %v", err)
	}
	return res
}
