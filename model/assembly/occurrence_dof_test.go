// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// splitOf returns the DOF split reported for occurrence id, failing when it is absent.
func splitOf(t *testing.T, rep SolveReport, id uint64) DOFSplit {
	t.Helper()
	for _, o := range rep.Occurrences {
		if o.Occurrence == id {
			if o.Split.TranslationCount+o.Split.RotationCount != o.DegreesOfFreedom {
				t.Fatalf("split %d+%d ≠ scalar DOF %d (occ %d)", o.Split.TranslationCount, o.Split.RotationCount, o.DegreesOfFreedom, id)
			}
			return o.Split
		}
	}
	t.Fatalf("no DOF report for occurrence %d", id)
	return DOFSplit{}
}

// axisAligned reports whether v is (anti)parallel to the unit axis (ax, ay, az).
func axisAligned(v math.Vector3, ax, ay, az float64) bool {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		return false
	}
	w := u.AsVector()
	dot := float64(w.X)*ax + float64(w.Y)*ay + float64(w.Z)*az
	return stdmath.Abs(dot) > 0.999
}

// TestSliderDOFSplit: a slider joint leaves translation 1 / rotation 0 with the slide axis (#1980).
func TestSliderDOFSplit(t *testing.T) {
	rep, movingID := jointDOF(t, func(js *JointSet, a, b *occurrence.Occurrence) {
		js.AddSlider(dofFrame(t, a), dofFrame(t, b))
	})
	s := splitOf(t, rep, movingID)
	if s.TranslationCount != 1 || s.RotationCount != 0 {
		t.Fatalf("slider split = %d/%d, want 1/0", s.TranslationCount, s.RotationCount)
	}
	if len(s.TranslationAxes) != 1 || !axisAligned(s.TranslationAxes[0], 0, 0, 1) {
		t.Errorf("slider translation axis = %v, want ±Z", s.TranslationAxes)
	}
}

// TestRotationalDOFSplit: a rotational joint leaves rotation 1 / translation 0 with the axis and a
// centre on that axis (#1980).
func TestRotationalDOFSplit(t *testing.T) {
	rep, movingID := jointDOF(t, func(js *JointSet, a, b *occurrence.Occurrence) {
		js.AddRotational(dofAxis(t, a), dofAxis(t, b))
	})
	s := splitOf(t, rep, movingID)
	if s.RotationCount != 1 || s.TranslationCount != 0 {
		t.Fatalf("rotational split = %d/%d, want 0/1", s.TranslationCount, s.RotationCount)
	}
	if len(s.RotationAxes) != 1 || !axisAligned(s.RotationAxes[0], 0, 0, 1) {
		t.Errorf("rotational axis = %v, want ±Z", s.RotationAxes)
	}
	// The DOF centre sits on the Z axis (x = y = 0 for an axis through the origin).
	if stdmath.Abs(float64(s.Center.X)) > 1e-6 || stdmath.Abs(float64(s.Center.Y)) > 1e-6 {
		t.Errorf("rotational DOF centre = %v, want on the Z axis (x=y=0)", s.Center)
	}
}

// TestGroundedAndFreeDOFSplit: a grounded occurrence is 0/0; an unconstrained one is 3/3 (#1980).
func TestGroundedAndFreeDOFSplit(t *testing.T) {
	occs := occurrence.NewOccurrences()
	grounded := place(occs, "grounded:1", math.Identity4())
	grounded.SetGrounded(true)
	free := place(occs, "free:1", math.Translation4(math.V3(1, 2, 3)))
	cs := NewConstraintSet(occs, nil)
	rep := SolveAssembly(cs, NewJointSet(occs, nil))

	if g := splitOf(t, rep, grounded.ID()); g.TranslationCount != 0 || g.RotationCount != 0 {
		t.Errorf("grounded split = %d/%d, want 0/0", g.TranslationCount, g.RotationCount)
	}
	if f := splitOf(t, rep, free.ID()); f.TranslationCount != 3 || f.RotationCount != 3 {
		t.Errorf("free split = %d/%d, want 3/3", f.TranslationCount, f.RotationCount)
	}
}

// dofFrame / dofAxis build joint-origin refs for the DOF tests.
func dofFrame(t *testing.T, o *occurrence.Occurrence) Ref {
	return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
}

func dofAxis(t *testing.T, o *occurrence.Occurrence) Ref {
	return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
}
