// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// guardBlock builds an axis-aligned block body with an exact, easily reasoned volume.
func guardBlock(t *testing.T, min, max math.Point3, name string) *topo.Body {
	t.Helper()
	b, err := brep.SolidBlock(min, max, name)
	if err != nil {
		t.Fatalf("SolidBlock(%v,%v): %v", min, max, err)
	}
	return b
}

// TestBooleanVolumeGuardTwoSided pins the Requicha brackets on the boolean volume guard (#1601):
// V(A∖B) ∈ [V(A)−V(B), V(A)] and V(A∪B) ∈ [max(V(A),V(B)), V(A)+V(B)]. The guard used to be
// one-sided — a cut that removed too much material, or a join that fabricated material, passed
// silently and shipped a materially wrong body.
func TestBooleanVolumeGuardTwoSided(t *testing.T) {
	target := guardBlock(t, math.P3(0, 0, 0), math.P3(2, 1, 1), "target") // V(A) = 2
	tool := guardBlock(t, math.P3(1.5, 0, 0), math.P3(2.5, 1, 1), "tool") // V(B) = 1, overlap 0.5

	cases := []struct {
		name    string
		op      PartFeatureOperation
		result  *topo.Body
		invalid bool
	}{
		{"cut within bracket", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "ok"), false},   // 1.5 ∈ [1, 2]
		{"cut removed too much", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(0.5, 1, 1), "low"), true}, // 0.5 < V(A)−V(B) = 1
		{"cut removed nothing extra", Cut, guardBlock(t, math.P3(0, 0, 0), math.P3(2.5, 1, 1), "high"), true},
		{"join within bracket", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(2.5, 1, 1), "ok"), false},         // 2.5 ∈ [2, 3]
		{"join fabricated material", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(4, 1, 1), "high"), true},     // 4 > V(A)+V(B) = 3
		{"join lost material", Join, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "low"), true},          // 1.5 < max = 2
		{"intersect within bound", Intersect, guardBlock(t, math.P3(1.5, 0, 0), math.P3(2, 1, 1), "ok"), false}, // 0.5 ≤ min = 1
		{"intersect too big", Intersect, guardBlock(t, math.P3(0, 0, 0), math.P3(1.5, 1, 1), "high"), true},     // 1.5 > min = 1
	}
	for _, c := range cases {
		if got := invalidBooleanVolume(c.op, target, tool, c.result); got != c.invalid {
			t.Errorf("%s: invalidBooleanVolume = %v, want %v", c.name, got, c.invalid)
		}
	}
}
