// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// boxComponent is a named fake occurrence definition with a real bounding box, so its
// occurrences have a non-empty RangeBox for the collision-stop test.
type boxComponent struct{ box math.Box }

func (c boxComponent) RangeBox() math.Box { return c.box }

// placeBox adds an occurrence backed by a box-bearing definition.
func placeBox(occs *occurrence.Occurrences, name string, m math.Matrix4, box math.Box) *occurrence.Occurrence {
	return occs.AddByComponentDefinition(name, boxComponent{box}, m)
}

// rollAboutZ is the moving frame's roll about +Z: the angle of its mapped local X axis in the
// XY plane. The drive advances this as it pins the rotational variable.
func rollAboutZ(m math.Matrix4) float64 {
	x := m.TransformVector(math.V3(1, 0, 0))
	return stdmath.Atan2(x.Y, x.X)
}

// TestDriveRotationalJointAnimates is the F03 acceptance: driving a rotational joint advances
// the free component's rotation through the swept range, one step at a time, and restores the
// assembly afterwards.
func TestDriveRotationalJointAnimates(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 6)))
	before := moving.Transform()

	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	axis := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
	}
	joint := js.AddRotational(axis(base), axis(moving))

	span := stdmath.Pi / 2
	settings := NewDriveSettings(types.DriveAngular, 0, span, span/4, 1, false, false)
	res, err := DriveJoint(occs, cs, js, joint.ID(), settings)
	if err != nil {
		t.Fatalf("DriveJoint: %v", err)
	}
	if len(res.Frames) != 5 {
		t.Fatalf("frames = %d, want 5 (0, π/8, π/4, 3π/8, π/2)", len(res.Frames))
	}

	first := rollAboutZ(res.Frames[0].Placements[0].Transform)
	last := rollAboutZ(res.Frames[len(res.Frames)-1].Placements[0].Transform)
	if delta := stdmath.Abs(last - first); stdmath.Abs(delta-span) > 1e-3 {
		t.Errorf("rotation swept = %.4f rad, want ≈ %.4f", delta, span)
	}
	if got := moving.Transform(); !got.IsEqualTo(before, 1e-9) {
		t.Errorf("assembly not restored after drive: %v, want %v", got, before)
	}
}

// TestDriveStopsOnCollision drives an arm (its body offset from the joint axis) toward a fixed
// wall; collision detection must halt the sweep at the frame the arm's box first overlaps the
// wall.
func TestDriveStopsOnCollision(t *testing.T) {
	occs := occurrence.NewOccurrences()
	wall := placeBox(occs, "wall:1", math.Identity4(), math.NewBox(math.P3(-5, -1, -2), math.P3(-1, 1, 8)))
	wall.SetGrounded(true)
	arm := placeBox(occs, "arm:1", math.Identity4(), math.NewBox(math.P3(1, -0.5, -0.5), math.P3(3, 0.5, 0.5)))

	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	axis := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))}
	}
	joint := js.AddRotational(axis(wall), axis(arm))

	// Sweep the arm from +X (clear) toward -X (into the wall) with collision detection on.
	settings := NewDriveSettings(types.DriveAngular, 0, stdmath.Pi, stdmath.Pi/8, 1, false, true)
	res, err := DriveJoint(occs, cs, js, joint.ID(), settings)
	if err != nil {
		t.Fatalf("DriveJoint: %v", err)
	}
	if !res.StoppedByCollision {
		t.Fatalf("drive did not stop on collision: %+v", res)
	}
	if res.StoppedAtStep == 0 || res.StoppedAtStep >= 8 {
		t.Errorf("stopped at step %d, want a mid-sweep frame (arm clear at 0, in the wall near π)", res.StoppedAtStep)
	}
	if last := res.Frames[len(res.Frames)-1]; !last.Collided {
		t.Errorf("final frame not marked collided: %+v", last)
	}
}

// TestDriveValuesSequence checks the value expansion: a ramp, repetitions, and a ping-pong
// that plays alternate passes in reverse.
func TestDriveValuesSequence(t *testing.T) {
	ramp := driveValues(NewDriveSettings(types.DriveLinear, 0, 1, 0.5, 1, false, false))
	if want := []float64{0, 0.5, 1}; !floatsEqual(ramp, want) {
		t.Errorf("ramp = %v, want %v", ramp, want)
	}
	pong := driveValues(NewDriveSettings(types.DriveLinear, 0, 1, 0.5, 2, true, false))
	if want := []float64{0, 0.5, 1, 1, 0.5, 0}; !floatsEqual(pong, want) {
		t.Errorf("ping-pong = %v, want %v", pong, want)
	}
	twice := driveValues(NewDriveSettings(types.DriveLinear, 0, 1, 0.5, 2, false, false))
	if want := []float64{0, 0.5, 1, 0, 0.5, 1}; !floatsEqual(twice, want) {
		t.Errorf("repeat = %v, want %v", twice, want)
	}
}

// TestDrivableVariable checks which joint-kind/variable pairings are drivable and how natural
// resolves.
func TestDrivableVariable(t *testing.T) {
	cases := []struct {
		kind      types.AssemblyJointType
		requested types.DriveVariable
		want      types.DriveVariable
		ok        bool
	}{
		{types.JointRotational, types.DriveNatural, types.DriveAngular, true},
		{types.JointRotational, types.DriveLinear, types.DriveNatural, false},
		{types.JointSlider, types.DriveNatural, types.DriveLinear, true},
		{types.JointCylindrical, types.DriveNatural, types.DriveAngular, true},
		{types.JointCylindrical, types.DriveLinear, types.DriveLinear, true},
		{types.JointRigid, types.DriveNatural, types.DriveNatural, false},
		{types.JointPlanar, types.DriveNatural, types.DriveNatural, false},
		{types.JointBall, types.DriveAngular, types.DriveNatural, false},
	}
	for _, c := range cases {
		got, ok := drivableVariable(c.kind, c.requested)
		if got != c.want || ok != c.ok {
			t.Errorf("drivableVariable(%s, %s) = (%s, %v), want (%s, %v)", c.kind, c.requested, got, ok, c.want, c.ok)
		}
	}
}

// TestDriveJointErrors checks the engine rejects an unknown joint and an undrivable kind.
func TestDriveJointErrors(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Identity4())
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	frame := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
	}
	rigid := js.AddRigid(frame(base), frame(moving))

	if _, err := DriveJoint(occs, cs, js, 999, NewDriveSettings(types.DriveAngular, 0, 1, 0.5, 1, false, false)); err == nil {
		t.Error("DriveJoint(unknown id) = nil error, want failure")
	}
	if _, err := DriveJoint(occs, cs, js, rigid.ID(), NewDriveSettings(types.DriveNatural, 0, 1, 0.5, 1, false, false)); err == nil {
		t.Error("DriveJoint(rigid) = nil error, want not-drivable failure")
	}
}

// floatsEqual compares two float slices within a tolerance.
func floatsEqual(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if stdmath.Abs(a[i]-b[i]) > 1e-9 {
			return false
		}
	}
	return true
}
