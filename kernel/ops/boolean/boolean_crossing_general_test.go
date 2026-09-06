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

// Two crossing cylinders, all three ways round, through the GENERAL pipeline (ADR-0061 stage 4). The
// mixed boolean declined every overlapping wall pair on box overlap alone until pairWallWallImprints;
// the union then still came back open, over two zero-length edges that no chart's wrapping emission
// dropped. Through brep.Boolean, BELOW the recognizer list, so it measures the general pipeline.
func TestGeneralPipelineCrossesTwoCylinders(t *testing.T) {
	t.Parallel()
	const bigR, bigH, rodR, rodL = 3.0, 12.0, 1.5, 12.0
	// Both run right through the other, so each meets it in two closed crossings.
	inter := crossedCylinderVolume(t)
	big := stdmath.Pi * bigR * bigR * bigH
	rod := stdmath.Pi * rodR * rodR * rodL
	for _, row := range []struct {
		name string
		op   brep.Op
		want float64
	}{
		{"the lens they share", brep.Intersection, inter},
		{"the drilled cylinder", brep.Difference, big - inter},
		{"the cross", brep.Union, big + rod - inter},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			res := crossedCylinders(t, row.op)
			if v := ops.Validate(res); !v.Valid || !v.Closed || !res.IsSolid() {
				t.Fatalf("not a valid closed solid: %+v", v.Issues)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-row.want) / row.want; rel > 0.001 {
				t.Errorf("volume %.6f, want %.6f — rel %.5f", got, row.want, rel)
			}
		})
	}
}

// crossedCylinderVolume is the shared lens, measured on the intersection the same pipeline builds. The
// union and the difference are then pinned to it by inclusion-exclusion, which is what makes the three
// rows one consistent statement rather than three independent numbers.
func crossedCylinderVolume(t *testing.T) float64 {
	t.Helper()
	return query.BodyGeometryProperties(crossedCylinders(t, brep.Intersection), ops.DefaultQuality()).Volume
}

// crossedCylinders runs one boolean of a Ø6 cylinder up the z axis with a Ø3 rod along x through it.
func crossedCylinders(t *testing.T, op brep.Op) *topo.Body {
	t.Helper()
	a, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	b, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	if err != nil {
		t.Fatalf("rod: %v", err)
	}
	res, err := brep.Boolean(op, a, b)
	if err != nil {
		t.Fatalf("boolean %v: %v", op, err)
	}
	return res
}

// The Steinmetz degeneracy — two cylinders of EQUAL radius — through the general pipeline. Its section
// is two planar ellipses crossing at the folds, which the ruled∩quadric form declines; the closed form
// inside the intersector supplies them, and the bicylinder's volume is the classical 16r³/3 exactly
// (ADR-0061 stage 4).
func TestGeneralPipelineBuildsTheSteinmetzBicylinder(t *testing.T) {
	t.Parallel()
	const r, h = 3.0, 12.0
	bicyl := 16 * r * r * r / 3
	one := stdmath.Pi * r * r * h
	for _, row := range []struct {
		name string
		op   brep.Op
		want float64
	}{
		{"the bicylinder", brep.Intersection, bicyl},
		{"one cylinder less the other", brep.Difference, one - bicyl},
		{"the cross", brep.Union, 2*one - bicyl},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			cx, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), r, h)
			if err != nil {
				t.Fatalf("cylinder x: %v", err)
			}
			cz, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), r, h)
			if err != nil {
				t.Fatalf("cylinder z: %v", err)
			}
			res, err := brep.Boolean(row.op, cx, cz)
			if err != nil {
				t.Fatalf("boolean: %v", err)
			}
			if v := ops.Validate(res); !v.Valid || !v.Closed || !res.IsSolid() {
				t.Fatalf("not a valid closed solid: %+v", v.Issues)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			if rel := stdmath.Abs(got-row.want) / row.want; rel > 0.001 {
				t.Errorf("volume %.6f, want %.6f — rel %.5f", got, row.want, rel)
			}
		})
	}
}
