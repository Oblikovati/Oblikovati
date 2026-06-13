// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// squareProfileSketch adds a closed rectangle [0,w]×[0,h] to sk (assembly coords).
func squareProfileSketch(sk *sketch.Sketch, w, h float64) {
	p0 := sk.Points().Add(math.P2(0, 0))
	p1 := sk.Points().Add(math.P2(w, 0))
	p2 := sk.Points().Add(math.P2(w, h))
	p3 := sk.Points().Add(math.P2(0, h))
	sk.Lines().Add(p0, p1)
	sk.Lines().Add(p1, p2)
	sk.Lines().Add(p2, p3)
	sk.Lines().Add(p3, p0)
}

// TestAssemblySketchHostsClosedProfile checks the assembly hosts a sketch on a work
// plane, shares its parameter DAG, and yields a closed profile after solving.
func TestAssemblySketchHostsClosedProfile(t *testing.T) {
	asm := NewAssemblyComponentDefinition()
	sk := asm.AddSketch(sketch.XYPlane(), nil)
	if asm.Sketches().Count() != 1 {
		t.Fatalf("assembly sketch count = %d, want 1", asm.Sketches().Count())
	}
	if sk.Parameters() != asm.Parameters() {
		t.Error("sketch does not share the assembly's parameter DAG")
	}
	squareProfileSketch(sk, 0.5, 1)
	sk.Solve()

	profiles := sk.Profiles()
	if profiles.Count() != 1 || !profiles.Item(0).IsClosed() {
		t.Fatalf("profiles = %d, want one closed region", profiles.Count())
	}
	if got := profiles.Item(0).Area(); stdmath.Abs(got-0.5) > 1e-9 {
		t.Errorf("profile area = %g, want 0.5", got)
	}
}

// TestAssemblySketchExtrudeCutsParticipant is the subsystem end-to-end: a profile
// sketched in assembly space is extruded into a prism and cut from a participant,
// gated against the analytic value (unit box minus a 0.5×1×0.6 pocket = 0.7).
func TestAssemblySketchExtrudeCutsParticipant(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm := NewAssemblyComponentDefinition()
	occ := asm.Place("box:1", part, math.Identity4())

	sk := asm.AddSketch(sketch.XYPlane(), nil)
	squareProfileSketch(sk, 0.5, 1)

	asm.AddFeature(feature.NewAssemblyExtrudeFeature(sk, 0, ops.Cut, func() float64 { return 0.6 }))
	asm.RecomputeFeatures()

	if got := resultVolume(asm.Features(), occ); stdmath.Abs(got-0.7) > 1e-6 {
		t.Errorf("extrude-cut volume = %g, want 0.7 (unit box minus a 0.5×1×0.6 pocket)", got)
	}
}

// TestAssemblyExtrudeJoinsBoss checks the join variant adds a profiled boss to each
// participant (unit box plus a 0.25×0.25×0.5 stud standing on its top face).
func TestAssemblyExtrudeJoinsBoss(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm := NewAssemblyComponentDefinition()
	occ := asm.Place("box:1", part, math.Identity4())

	// Sketch the stud footprint on the box's top face (z=1) and extrude up by 0.5.
	top, _ := sketch.NewPlane(math.P3(0, 0, 1), unitX(t), unitY(t))
	sk := asm.AddSketch(top, nil)
	squareProfileSketch(sk, 0.25, 0.25)

	asm.AddFeature(feature.NewAssemblyExtrudeFeature(sk, 0, ops.Join, func() float64 { return 0.5 }))
	asm.RecomputeFeatures()

	want := 1.0 + 0.25*0.25*0.5
	if got := resultVolume(asm.Features(), occ); stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("extrude-join volume = %g, want %g (box plus a 0.25×0.25×0.5 stud)", got, want)
	}
}

// TestAssemblyRevolveBoresParticipant is the revolve subsystem end-to-end: a profile
// sketched in assembly space (radius 0–0.25, height 0–1) is spun about its centerline into a
// drilled cylinder and cut from a participant, gated against the analytic value — a unit box
// minus a faceted r=0.25 through-bore ≈ 1 − π·0.25² (within facet tolerance).
func TestAssemblyRevolveBoresParticipant(t *testing.T) {
	part := partWithBlock(t, math.P3(0, 0, 0), math.P3(1, 1, 1))
	asm := NewAssemblyComponentDefinition()
	occ := asm.Place("box:1", part, math.Identity4())

	// Plane below the box whose local +x is world +x and local +y is world +z, so the profile's
	// local (x,y) maps to (radius from the vertical centreline, height up the box). The profile
	// runs z∈[-0.5,1.5] so the bore overshoots both box faces — no coplanar end caps, which would
	// be a coincident-face boolean hazard — leaving a clean through-hole over the box's z∈[0,1].
	bore, _ := sketch.NewPlane(math.P3(0.5, 0.5, -0.5), unitX(t), worldZ(t))
	sk := asm.AddSketch(bore, nil)
	squareProfileSketch(sk, 0.25, 2) // radius 0–0.25, height 0–2 (overshoots the box)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true) // the vertical bore axis

	asm.AddFeature(feature.NewAssemblyRevolveFeature(sk, 0, nil, ops.Cut, func() float64 { return 2 * stdmath.Pi }))
	asm.RecomputeFeatures()

	// This asserts the assembly plumbing — sketch hosting, centreline axis, AddFeature,
	// RecomputeFeatures, participant placement — drove a through-bore. The boolean re-facets
	// the revolved cylinder coarsely at this radius (~an 8-gon), so it removes a bit less than
	// the true circle; the tight analytic gate is the unit test TestAssemblyRevolveToolVolume.
	removed := 1.0 - resultVolume(asm.Features(), occ)
	trueBore := stdmath.Pi * 0.25 * 0.25 // π·r²·h, h=1
	if removed < 0.85*trueBore || removed > 1.02*trueBore {
		t.Errorf("revolve-bore removed = %g, want within [0.85,1.02]·%g of the analytic bore", removed, trueBore)
	}
}

// worldZ returns the world Z unit vector for a sketch plane frame.
func worldZ(t *testing.T) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatalf("worldZ: %v", err)
	}
	return u
}

// unitX / unitY return the world X / Y unit vectors for a sketch plane frame.
func unitX(t *testing.T) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(1, 0, 0)
	if err != nil {
		t.Fatalf("unitX: %v", err)
	}
	return u
}
func unitY(t *testing.T) math.UnitVector3 {
	t.Helper()
	u, err := math.NewUnitVector3(0, 1, 0)
	if err != nil {
		t.Fatalf("unitY: %v", err)
	}
	return u
}
