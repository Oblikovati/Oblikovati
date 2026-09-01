// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/topo"
	gmath "oblikovati.org/math"
	"oblikovati.org/model/param"
)

// holedVolume recomputes f against a unit box and returns the remaining volume.
func holedVolume(t *testing.T, f *AssemblyHoleFeature) float64 {
	t.Helper()
	out, err := f.Recompute(Input{Bodies: []*topo.Body{unitBlock(t)}})
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if len(out.Bodies) != 1 {
		t.Fatalf("result = %d bodies, want 1", len(out.Bodies))
	}
	return bodyVolume(out.Bodies[0])
}

// TestAssemblyHoleExposesEditableDiameterDepth checks the parametric hole advertises its
// Diameter and Depth as editable length scalars, and that widening the bore (the edit
// path) re-drills a larger hole that removes more material (#752).
func TestAssemblyHoleExposesEditableDiameterDepth(t *testing.T) {
	t.Parallel()
	axis, _ := gmath.NewUnitVector3(0, 0, 1)
	f, err := NewAssemblyHoleFeature(gmath.P3(0.5, 0.5, 0), axis, 0.3, 1.5)
	if err != nil {
		t.Fatalf("NewAssemblyHoleFeature: %v", err)
	}
	ps := f.EditableParams()
	if len(ps) != 2 || ps[0].Label != "Diameter" || ps[1].Label != "Depth" {
		t.Fatalf("EditableParams = %+v, want a Diameter and a Depth scalar", ps)
	}
	if ps[0].Unit != param.Length || ps[1].Unit != param.Length {
		t.Error("hole scalars should both be lengths")
	}

	narrow := holedVolume(t, f)
	ps[0].Set(0.6) // widen the bore via the editable param
	if wide := holedVolume(t, f); wide >= narrow {
		t.Errorf("widening the hole should remove more material: narrow=%g wide=%g", narrow, wide)
	}
}
