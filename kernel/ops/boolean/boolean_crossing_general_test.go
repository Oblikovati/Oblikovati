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

// PARTIAL penetration: a stub whose cap ends inside the other cylinder, on its very axis. The crossing
// wraps the stub's azimuth and not the fat one's, and the stub's cap sections the fat wall in two
// straight RULINGS three units clear of the cap's own rim — a section with no contact in it, which the
// pairing used to refuse for not being a conic (ADR-0061 stage 4).
func TestGeneralPipelineTakesAPartialPenetration(t *testing.T) {
	t.Parallel()
	const fatR, fatH, rodR, rodL = 3.0, 12.0, 1.5, 6.0
	plug := partialPlugVolume(t)
	fat := stdmath.Pi * fatR * fatR * fatH
	rod := stdmath.Pi * rodR * rodR * rodL
	for _, row := range []struct {
		name string
		op   brep.Op
		want float64
	}{
		{"the plug", brep.Intersection, plug},
		{"the blind hole", brep.Difference, fat - plug},
		{"the stub joined on", brep.Union, fat + rod - plug},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			res := partiallyPenetrated(t, row.op)
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

// partialPlugVolume measures the shared plug on the intersection this pipeline builds; the other two
// rows are pinned to it by inclusion-exclusion, so the three are one statement.
func partialPlugVolume(t *testing.T) float64 {
	t.Helper()
	return query.BodyGeometryProperties(partiallyPenetrated(t, brep.Intersection), ops.DefaultQuality()).Volume
}

// partiallyPenetrated runs one boolean of a Ø6 cylinder with a Ø3 stub that stops on its axis.
func partiallyPenetrated(t *testing.T, op brep.Op) *topo.Body {
	t.Helper()
	fat, err := brep.SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	stub, err := brep.SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 6)
	if err != nil {
		t.Fatalf("stub: %v", err)
	}
	res, err := brep.Boolean(op, fat, stub)
	if err != nil {
		t.Fatalf("boolean %v: %v", op, err)
	}
	return res
}
