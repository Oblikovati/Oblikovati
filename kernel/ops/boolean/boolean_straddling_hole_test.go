// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/internal/testcage"
	. "oblikovati.org/kernel/ops/boolean"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Regression for Oblikovati#2030 at the solid level: joining a block onto a plate whose HOLE
// straddles the block's footprint edge must leave a valid solid of the right genus.
//
// The plate's top plane then carries two openings that TOUCH — the block footprint and the hole —
// so they bound one connected region. Nesting them as two separate hole loops wrote their shared
// arcs twice, giving a face with edges whose two uses lay on that same face; χ = V−E+2F−L then
// drifted by one per doubled edge. With one hole that landed on an EVEN χ and slipped through
// [Validate]; with two it went odd and the solid was rejected outright (a Raspberry-Pi camera PCB
// whose two mounting holes straddled the flex-connector block: χ = −31 across 31 doubled edges).

// plateWithStraddledHole builds the configuration: a 2.5 × 2.4 × 0.1 plate whose top face is at
// z = 0, drilled by one Ø0.21 hole centred at (0.2, 0.96), FACETED (as an upstream join would
// leave it), then a block joined on top whose footprint edge at y = 1.05 crosses that hole — whose
// far side reaches y = 1.065, so a thin crescent stays outside the block.
func plateWithStraddledHole(t *testing.T, holeCenters ...math.Point2) *topo.Body {
	t.Helper()
	body, err := brep.SolidBlock(math.P3(-1.25, -1.2, -0.1), math.P3(1.25, 1.2, 0), "plate")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	for _, c := range holeCenters {
		drill, err := brep.SolidCylinder(math.P3(c.X, c.Y, -0.2), math.V3(0, 0, 1), 0.105, 0.4)
		if err != nil {
			t.Fatalf("SolidCylinder: %v", err)
		}
		if body, err = Boolean(Cut, body, drill); err != nil {
			t.Fatalf("drill at %v: %v", c, err)
		}
	}
	// A plate with ANALYTIC bores meets the block in a flush contact the mixed boolean does not model
	// yet (a coplanar face carrying conic loops), so the join would fall to triangle CSG; facet the
	// plate explicitly so this test pins the planar path — the #2030 hole nesting — and nothing else.
	body = testcage.Body(body, "facet")

	block, err := brep.SolidBlock(math.P3(-0.4, 0.55, 0), math.P3(0.4, 1.05, 0.1), "block")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	joined, err := Boolean(Join, body, block)
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	return joined
}

// sameFaceSeamEdges counts edges whose TWO uses lie on one face. A cylinder's seam is a legitimate
// instance (the face is still a disk once cut along it); one on a PLANAR face is the doubled
// boundary this regression guards against.
func sameFaceSeamEdges(b *topo.Body) int {
	n := 0
	for _, e := range b.Edges() {
		if us := e.Uses(); len(us) == 2 && us[0].Loop().Face() == us[1].Loop().Face() {
			n++
		}
	}
	return n
}

func TestJoinOverAStraddlingHoleKeepsOneOpening(t *testing.T) {
	t.Parallel()
	joined := plateWithStraddledHole(t, math.P2(0.2, 0.96))
	if got := sameFaceSeamEdges(joined); got != 0 {
		t.Errorf("%d edge(s) have both uses on one face — the shared boundary was written twice", got)
	}
	// One through-hole survives (the crescent keeps it open), so the solid is genus 1: χ = 2−2g = 0.
	if got := joined.EulerCharacteristic(); got != 0 {
		t.Errorf("χ = %d, want 0 (one through-hole ⇒ genus 1)", got)
	}
	if r := Validate(joined); !r.Valid {
		t.Errorf("joined solid invalid: %v", r.Issues)
	}
}

// TestTwoStraddlingHolesStayOnTheExactPath is the camera's own shape — TWO straddled holes — and
// pins the COST of the defect rather than only its symptom. With both holes the doubled edges made
// the planar result invalid, so [booleanGeneral]'s guard threw it away and adopted the triangle-soup
// CSG fallback instead: a valid solid of the right genus, but 5726 faces where the exact planar
// path gives 77. That silent 74× demotion is what shattered the camera into facets, and asserting
// a low face count is what makes this a real guard — the χ and validity checks alone pass either
// way, because the fallback rescues them.
func TestTwoStraddlingHolesStayOnTheExactPath(t *testing.T) {
	t.Parallel()
	joined := plateWithStraddledHole(t, math.P2(0.2, 0.96), math.P2(-0.2, 0.96))
	if got := len(joined.Faces()); got > 200 {
		t.Errorf("join produced %d faces — the exact planar path was abandoned for triangle CSG", got)
	}
	if got := sameFaceSeamEdges(joined); got != 0 {
		t.Errorf("%d edge(s) have both uses on one face", got)
	}
	// Two through-holes survive (each keeps a crescent open), so the solid is genus 2: χ = 2−2g.
	if got := joined.EulerCharacteristic(); got != -2 {
		t.Errorf("χ = %d, want -2 (two through-holes ⇒ genus 2)", got)
	}
	if r := Validate(joined); !r.Valid {
		t.Errorf("joined solid invalid: %v", r.Issues)
	}
}
