// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TransformBody historically rejected multi-shell bodies and silently dropped
// each face's material sense; both broke transformed cavity bodies (#629/#628
// Copy rides on this). The cavity body is the regression vehicle: 4³ − 2³ = 56
// of material in two shells with reversed cavity walls.

func TestTransformBodyCavityPreservesShellsAndVolume(t *testing.T) {
	t.Parallel()
	body := cavityBody(t)
	moved, err := TransformBody(body, math.Translation4(math.V3(10, 0, 0)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if got := len(moved.Shells()); got != 2 {
		t.Errorf("moved cavity body has %d shells, want 2", got)
	}
	if v := BodyGeometryProperties(moved, DefaultQuality()).Volume; stdmath.Abs(v-56) > 0.1 {
		t.Errorf("moved cavity volume = %g, want 56 (face sense must survive)", v)
	}
	if r := Validate(moved); !r.Valid {
		t.Errorf("moved cavity invalid: %v", r.Issues)
	}
}

// TestTransformBodyCarriesWires: a wire-only body transforms with its wire.
func TestTransformBodyCarriesWires(t *testing.T) {
	t.Parallel()
	_, w := squareWireBody(1)
	src := w.Body()
	moved, err := TransformBody(src, math.Translation4(math.V3(0, 0, 5)), func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		t.Fatalf("TransformBody: %v", err)
	}
	if len(moved.Wires()) != 1 {
		t.Fatalf("moved body has %d wires, want 1", len(moved.Wires()))
	}
	p := moved.Wires()[0].Edges()[0].StartVertex().Point()
	if stdmath.Abs(float64(p.Z)-5) > 1e-12 {
		t.Errorf("moved wire start Z = %v, want 5", p.Z)
	}
}
