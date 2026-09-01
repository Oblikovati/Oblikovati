// SPDX-License-Identifier: GPL-2.0-only

package transform_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// identityLineage keeps lineage unchanged under a deform (the test bodies carry none anyway).
func identityLineage(l topo.Lineage) topo.Lineage { return l }

// TestDeformBodyIdentity an identity map reproduces the body exactly — same volume, still a
// valid closed solid.
func TestDeformBodyIdentity(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 3, 4) // volume 24
	out, err := transform.DeformBody(box, func(p math.Point3) math.Point3 { return p }, identityLineage)
	if err != nil {
		t.Fatalf("DeformBody: %v", err)
	}
	if r := ops.Validate(out); !r.Valid {
		t.Fatalf("identity deform invalid: %v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(out, ops.DefaultQuality()).Volume; stdmath.Abs(v-24) > 1e-6 {
		t.Errorf("identity deform volume = %v, want 24", v)
	}
}

// TestDeformBodyShearPreservesVolume a shear (x += 0.5·z) keeps a box a valid solid of the same
// volume — the non-affine path the affine TransformBody cannot take but DeformBody must.
func TestDeformBodyShearPreservesVolume(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2) // volume 8
	shear := func(p math.Point3) math.Point3 { return math.P3(p.X+0.5*p.Z, p.Y, p.Z) }
	out, err := transform.DeformBody(box, shear, identityLineage)
	if err != nil {
		t.Fatalf("DeformBody: %v", err)
	}
	if r := ops.Validate(out); !r.Valid {
		t.Fatalf("sheared box invalid: %v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(out, ops.DefaultQuality()).Volume; stdmath.Abs(v-8) > 1e-6 {
		t.Errorf("sheared box volume = %v, want 8 (shear preserves volume)", v)
	}
}

// TestDeformBodyRigidMatchesMove a rigid map (translate) leaves the volume unchanged and the
// body valid — the deform must agree with a Move on the affine subset.
func TestDeformBodyRigidMatchesMove(t *testing.T) {
	t.Parallel()
	box := csgBox(math.P3(0, 0, 0), 1, 1, 1)
	out, err := transform.DeformBody(box, func(p math.Point3) math.Point3 {
		return math.P3(p.X+5, p.Y-2, p.Z+1)
	}, identityLineage)
	if err != nil {
		t.Fatalf("DeformBody: %v", err)
	}
	if r := ops.Validate(out); !r.Valid {
		t.Fatalf("translated box invalid: %v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(out, ops.DefaultQuality()).Volume; stdmath.Abs(v-1) > 1e-6 {
		t.Errorf("translated box volume = %v, want 1", v)
	}
	if c := out.RangeBox().Center(); stdmath.Abs(c.X-5.5) > 1e-6 || stdmath.Abs(c.Y-(-1.5)) > 1e-6 {
		t.Errorf("translated box centre = %v, want ~(5.5,-1.5,1.5)", c)
	}
}
