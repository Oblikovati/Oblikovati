// SPDX-License-Identifier: GPL-2.0-only
package ops
import (
	"testing"
	"oblikovati.org/kernel/brep"
	m "oblikovati.org/math"
)
// TestShellIsVoidInBodyCurvedCavity: a solid block with a fully-enclosed SPHERICAL cavity — the void
// shell is a curved (sphere) surface, the case a per-face orientation cannot sign. The orientation-free
// ray-parity classification must still mark exactly the sphere shell as the void.
func TestShellIsVoidInBodyCurvedCavity(t *testing.T) {
	block, err := brep.SolidBlock(m.P3(0,0,0), m.P3(10,10,10), "block")
	if err != nil { t.Fatal(err) }
	ball, err := brep.SolidSphere(m.P3(5,5,5), 2, "ball")
	if err != nil { t.Fatal(err) }
	body, err := Boolean(Cut, block, ball)
	if err != nil { t.Fatalf("cut cavity: %v", err) }
	if r := Validate(body); !r.Valid { t.Fatalf("not valid: %+v", r.Issues) }
	voids := 0
	for _, sh := range body.Shells() {
		if ShellIsVoidInBody(body, sh) { voids++ }
	}
	if voids != 1 { t.Errorf("curved-cavity body has %d void shells, want 1", voids) }
}
