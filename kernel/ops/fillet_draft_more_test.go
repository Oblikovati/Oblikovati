// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// horizontalEdgeKey returns a top horizontal edge (start/end share Z = top) of a box.
func horizontalEdgeKey(t *testing.T, b *topo.Body, topZ float64) []byte {
	t.Helper()
	for _, e := range b.Edges() {
		s, en := e.StartVertex().Point(), e.EndVertex().Point()
		if stdmath.Abs(float64(s.Z)-topZ) < 1e-9 && stdmath.Abs(float64(en.Z)-topZ) < 1e-9 {
			return e.ReferenceKey()
		}
	}
	t.Fatalf("no top horizontal edge found")
	return nil
}

// TestFilletHorizontalEdge fillets a top edge (between the +Z face and a side), exercising
// the fillet's corner/end-face handling for a non-vertical edge.
func TestFilletHorizontalEdge(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	res, err := ops.FilletEdges(box, [][]byte{horizontalEdgeKey(t, box, 2)}, 0.4)
	if err != nil {
		t.Fatalf("fillet top edge: %v", err)
	}
	if r := ops.Validate(res); !r.Valid {
		t.Fatalf("filleted body invalid: %+v", r)
	}
	if hasCylinderFaces(res) == 0 {
		t.Error("a fillet should introduce a cylindrical face")
	}
}

// TestDraftPositiveAngle drafts a side face outward (positive angle), the opposite sign of
// the existing inward-draft test.
func TestDraftPositiveAngle(t *testing.T) {
	box := csgBox(math.P3(0, 0, 0), 2, 2, 2)
	var side []byte
	for _, f := range box.Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(float64(n.X)-1) < 1e-9 {
			side = f.ReferenceKey()
		}
	}
	res, err := ops.DraftFaces(box, [][]byte{side}, math.V3(0, 0, 1), stdmath.Atan(0.2))
	if err != nil {
		t.Fatalf("positive draft: %v", err)
	}
	if r := ops.Validate(res); !r.Valid {
		t.Fatalf("drafted body invalid: %+v", r)
	}
}
