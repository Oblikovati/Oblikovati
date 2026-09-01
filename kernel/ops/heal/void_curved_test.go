// SPDX-License-Identifier: GPL-2.0-only
package heal

import (
	"testing"

	"oblikovati.org/kernel/ops/boolean"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/brep"
	m "oblikovati.org/math"
)

// TestShellIsVoidInBodyCurvedCavity: a solid block with a fully-enclosed SPHERICAL cavity — the void
// shell is a curved (sphere) surface, the case a per-face orientation cannot sign. The orientation-free
// ray-parity classification must still mark exactly the sphere shell as the void.
func TestShellIsVoidInBodyCurvedCavity(t *testing.T) {
	t.Parallel()
	block, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 10), "block")
	if err != nil {
		t.Fatal(err)
	}
	ball, err := brep.SolidSphere(m.P3(5, 5, 5), 2, "ball")
	if err != nil {
		t.Fatal(err)
	}
	body, err := boolean.Boolean(boolean.Cut, block, ball)
	if err != nil {
		t.Fatalf("cut cavity: %v", err)
	}
	if r := Validate(body); !r.Valid {
		t.Fatalf("not valid: %+v", r.Issues)
	}
	voids := 0
	for _, sh := range body.Shells() {
		if ShellIsVoidInBody(body, sh) {
			voids++
		}
	}
	if voids != 1 {
		t.Errorf("curved-cavity body has %d void shells, want 1", voids)
	}
}

// TestShellIsVoidRejectsOpenShell: an open shell bounds no region, so it is never a void — the guard
// that keeps the ray-parity seed from being asked for an interior that does not exist.
func TestShellIsVoidRejectsOpenShell(t *testing.T) {
	t.Parallel()
	patch := brepfixture.QuadBody("patch", m.P3(0, 0, 0), m.P3(4, 0, 0), m.P3(4, 4, 0), m.P3(0, 4, 0))
	for _, sh := range patch.Shells() {
		if ShellIsVoidInBody(patch, sh) {
			t.Error("an open shell was classified as a void")
		}
	}
}

// TestShellInteriorPointRejectsNonPositiveEpsilon: the probe offset must exceed the classifier's
// on-surface band, so a non-positive offset is refused instead of seeding a point ON the surface.
func TestShellInteriorPointRejectsNonPositiveEpsilon(t *testing.T) {
	t.Parallel()
	ball, err := brep.SolidSphere(m.P3(0, 0, 0), 2, "ball")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := shellInteriorPoint(ball.Shells()[0], 0, 1e-6); ok {
		t.Error("shellInteriorPoint accepted a zero offset")
	}
}

// TestShellIsVoidSkipsFaceWithHole: on a face whose edge-midpoint average falls in a HOLE, the seed is
// rejected and another face is tried — the outer shell of a drilled plate is still not a void.
func TestShellIsVoidSkipsFaceWithHole(t *testing.T) {
	t.Parallel()
	plate, err := brep.SolidBlock(m.P3(-5, -5, 0), m.P3(5, 5, 2), "plate")
	if err != nil {
		t.Fatal(err)
	}
	drill, err := brep.SolidCylinder(m.P3(0, 0, -1), m.V3(0, 0, 1), 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	holed, err := boolean.Boolean(boolean.Cut, plate, drill)
	if err != nil {
		t.Fatalf("drill: %v", err)
	}
	for _, sh := range holed.Shells() {
		if ShellIsVoidInBody(holed, sh) {
			t.Error("the outer shell of a drilled plate was classified as a void")
		}
	}
}
