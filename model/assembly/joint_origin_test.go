// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// TestOriginOffsetApply: an offset origin shifts the primitive point by dx/dy in the frame's plane —
// the Z (normal) coordinate is untouched and the in-plane displacement is √(dx²+dy²).
func TestOriginOffsetApply(t *testing.T) {
	prim := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	od := originDef{mode: types.JointOriginOffset, dx: 2, dy: 3}
	p := od.apply(prim).point
	if stdmath.Abs(float64(p.Z)) > 1e-9 {
		t.Errorf("offset moved the point off its plane: Z = %v, want 0", p.Z)
	}
	if d := stdmath.Hypot(float64(p.X), float64(p.Y)); stdmath.Abs(d-stdmath.Sqrt(13)) > 1e-9 {
		t.Errorf("in-plane offset = %v, want √13", d)
	}
}

// TestOriginBetweenFacesApply: a between-two-faces origin sits at the midpoint of the two faces.
func TestOriginBetweenFacesApply(t *testing.T) {
	od := originDef{
		mode:     types.JointOriginBetweenTwoFaces,
		faceA:    PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)),
		faceB:    PlanePrimitive(math.P3(0, 0, 4), unit(t, 0, 0, 1)),
		hasFaces: true,
	}
	p := od.apply(PlanePrimitive(math.P3(9, 9, 9), unit(t, 0, 0, 1))).point
	if p.X != 0 || p.Y != 0 || p.Z != 2 {
		t.Errorf("midplane origin = %v, want (0,0,2)", p)
	}
}

// TestRigidJointHonoursOriginOffset drives a rigid joint whose grounded origin is offset by (2,3):
// the free component coincides with the offset frame, so it lands √13 from the base in-plane (#1973).
func TestRigidJointHonoursOriginOffset(t *testing.T) {
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Translation4(math.V3(0, 0, 6)))
	frame := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
	}
	cs := NewConstraintSet(occs, nil)
	js := NewJointSet(occs, nil)
	j := js.AddRigid(frame(base), frame(moving))
	j.SetOriginOneAsOffset(2, 3)

	if rep := SolveAssembly(cs, js); !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}
	tr := moving.Transform().Translation()
	if stdmath.Abs(float64(tr.Z)) > 1e-6 {
		t.Errorf("moving Z = %v, want 0 (offset is in-plane)", tr.Z)
	}
	if d := stdmath.Hypot(float64(tr.X), float64(tr.Y)); stdmath.Abs(d-stdmath.Sqrt(13)) > 1e-6 {
		t.Errorf("moving in-plane distance = %v, want √13 (the origin offset)", d)
	}
}

// TestJointOriginReadBack: the origin mode and offsets read back as set (#1973).
func TestJointOriginReadBack(t *testing.T) {
	occs := occurrence.NewOccurrences()
	a := place(occs, "a:1", math.Identity4())
	b := place(occs, "b:1", math.Identity4())
	frame := func(o *occurrence.Occurrence) Ref {
		return Ref{Occurrence: o, Primitive: FramePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), unit(t, 1, 0, 0))}
	}
	js := NewJointSet(occs, nil)
	j := js.AddRigid(frame(a), frame(b))
	j.SetOriginOneAsOffset(5, 7)
	j.SetOriginTwoAsInfer()

	am, bm := j.OriginModes()
	if am != types.JointOriginOffset || bm != types.JointOriginInfer {
		t.Errorf("origin modes = (%v, %v), want (offset, infer)", am, bm)
	}
	if ax, ay, _, _ := j.OriginOffsets(); ax != 5 || ay != 7 {
		t.Errorf("origin one offset = (%v, %v), want (5, 7)", ax, ay)
	}
}
