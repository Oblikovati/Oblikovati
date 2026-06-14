// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"testing"

	"oblikovati.org/api/contract"
	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// recordingJointListener is a named fake capturing joint notifications.
type recordingJointListener struct{ added, deleted int }

func (r *recordingJointListener) JointAdded(contract.AssemblyJoint)   { r.added++ }
func (r *recordingJointListener) JointDeleted(contract.AssemblyJoint) { r.deleted++ }

// jointDOF builds a grounded base + free moving box, adds the joint via add, solves the
// assembly (constraints + joints together), and returns the free box's reported DOF.
func jointDOF(t *testing.T, add func(js *JointSet, base, moving *occurrence.Occurrence)) (SolveReport, uint64) {
	t.Helper()
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 6)))
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	add(js, base, moving)
	return SolveAssembly(cs, js), moving.ID()
}

// TestJointDegreesOfFreedom checks each joint type leaves the free component exactly its
// nominal degrees of freedom — the F02 acceptance generalized to the whole family.
func TestJointDegreesOfFreedom(t *testing.T) {
	axis := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
	}
	frame := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
	}
	plane := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
	}
	point := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: PointPrimitive(math.P3(0, 0, 0))}
	}
	cases := []struct {
		name string
		add  func(js *JointSet, a, b *occurrence.Occurrence)
		want int
	}{
		{"rigid", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddRigid(frame(a), frame(b)) }, 0},
		{"rotational", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddRotational(axis(a), axis(b)) }, 1},
		{"slider", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddSlider(frame(a), frame(b)) }, 1},
		{"cylindrical", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddCylindrical(axis(a), axis(b)) }, 2},
		{"planar", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddPlanar(plane(a), plane(b)) }, 3},
		{"ball", func(js *JointSet, a, b *occurrence.Occurrence) { js.AddBall(point(a), point(b)) }, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rep, movingID := jointDOF(t, c.add)
			if got := dofOf(rep, movingID); got != c.want {
				t.Errorf("%s joint free DOF = %d, want %d (report %+v)", c.name, got, c.want, rep)
			}
		})
	}
}

// TestRotationalJointPositionsAndLimits checks a rotational joint positions the free
// component on the axis (leaving rotation free) and round-trips its angular limits — the
// PBI-126 acceptance.
func TestRotationalJointPositionsAndLimits(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(3, 4, 6)))
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	axis := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
	}
	j := js.AddRotational(axis(base), axis(moving))
	rep := SolveAssembly(cs, js)

	if !rep.Converged {
		t.Fatalf("rotational joint solve did not converge: %+v", rep)
	}
	// The axes are collinear and seated: the moving origin sits on the grounded axis (x,y,z→0).
	if p := moving.Transform().Translation(); !math.V3(p.X, p.Y, p.Z).IsEqualTo(math.V3(0, 0, 0), 1e-6) {
		t.Errorf("moving origin = %+v, want on the axis at the origin", p)
	}
	lim := NewJointLimits(nil, &limits{min: -1, max: 1, hasMin: true, hasMax: true})
	if err := js.SetLimits(j.ID(), lim); err != nil {
		t.Fatalf("SetLimits: %v", err)
	}
	if j.Limits() == nil {
		t.Fatal("Limits() = nil after SetLimits")
	}
	if v, ok := j.Limits().AngularMaximum(); !ok || v != 1 {
		t.Errorf("angular max = %v (ok=%v), want 1", v, ok)
	}
	if _, ok := j.Limits().LinearMinimum(); ok {
		t.Error("linear min should be unset on a rotational joint")
	}
}

// TestJointFlipAndListener checks flip toggles and the set notifies its listener.
func TestJointFlipAndListener(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	rec := &recordingJointListener{}
	js := NewJointSet(occs, rec)
	j := js.AddRotational(axisFrameOn(t, a), axisFrameOn(t, b))
	if j.Flip() {
		t.Error("new joint should not be flipped")
	}
	if err := js.SetFlip(j.ID(), true); err != nil {
		t.Fatalf("SetFlip: %v", err)
	}
	if !j.Flip() {
		t.Error("SetFlip(true) did not flip the joint")
	}
	if !js.Delete(j.ID()) {
		t.Fatal("Delete returned false for a known joint")
	}
	if rec.added != 1 || rec.deleted != 1 {
		t.Errorf("notifications = added %d deleted %d, want 1/1", rec.added, rec.deleted)
	}
}

// axisFrameOn is a +Z frame joint origin on occurrence o.
func axisFrameOn(t *testing.T, o *occurrence.Occurrence) Ref {
	t.Helper()
	return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
}

// TestCombinedConstraintAndJointSolve checks constraints and joints solve together over one
// system: a mate + a rotational joint on the same free component converge.
func TestCombinedConstraintAndJointSolve(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 8)))
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	// A mate stacking the moving box's bottom face onto the base's top face.
	cs.AddMate(
		Ref{Occurrence: base, Primitive: PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))},
		Ref{Occurrence: moving, Primitive: PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, -1))},
		0, types.MateSolutionOpposed)
	// A cylindrical joint along Z (axes collinear) further constrains it.
	js.AddCylindrical(
		Ref{Occurrence: base, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))},
		Ref{Occurrence: moving, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))})

	rep := SolveAssembly(cs, js)
	if !rep.Converged {
		t.Fatalf("combined solve did not converge: %+v", rep)
	}
	if rep.Constraints != 2 {
		t.Errorf("active relationships = %d, want 2 (mate + joint)", rep.Constraints)
	}
	// Mate (plane, removes 3) + cylindrical joint axis-collinear: leaves a spin about Z (1 DOF).
	if got := dofOf(rep, moving.ID()); got != 1 {
		t.Errorf("free DOF under mate + cylindrical joint = %d, want 1", got)
	}
}

// TestDSJointImposedMotion checks a DS joint exposes its DOF and that locking one reduces the
// free count.
func TestDSJointImposedMotion(t *testing.T) {
	ds := NewDSJointSet()
	a := Ref{Occurrence: nil, Primitive: PointPrimitive(math.P3(0, 0, 0))}
	j := ds.Add(types.DSJointCylindrical, a, a)
	if j.DOFCount() != 2 {
		t.Fatalf("cylindrical DS joint DOF count = %d, want 2", j.DOFCount())
	}
	if j.FreeDegreesOfFreedom() != 2 {
		t.Errorf("free DOF = %d, want 2 (all free)", j.FreeDegreesOfFreedom())
	}
	if err := ds.SetImposedMotion(j.ID(), 0, types.DSDOFLocked, 0); err != nil {
		t.Fatalf("SetImposedMotion: %v", err)
	}
	if j.FreeDegreesOfFreedom() != 1 {
		t.Errorf("free DOF after locking one = %d, want 1", j.FreeDegreesOfFreedom())
	}
	if j.DOF(0).ImposedMotion() != types.DSDOFLocked {
		t.Errorf("DOF 0 imposed motion = %v, want locked", j.DOF(0).ImposedMotion())
	}
	if err := ds.SetImposedMotion(j.ID(), 9, types.DSDOFFree, 0); err == nil {
		t.Error("SetImposedMotion on an out-of-range DOF should error")
	}
}
