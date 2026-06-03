// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/model/feature"
)

// chamferedBox bevels one vertical edge of a 2×2×2 box (setback 0.5) via the chamfer
// feature, returning the resulting solid (vol 7.75) — a body with a chamfer face to delete.
func chamferedBox(t *testing.T) *topo.Body {
	t.Helper()
	box := shellBox(2, 2, 2)
	var edge []byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			edge = e.ReferenceKey()
			break
		}
	}
	fs := feature.NewPartFeatures(nil, nil)
	feature.NewBaseFeatures(fs).AddBase(box)
	feature.NewDressUpFeatures(fs).AddChamfer([][]byte{edge}, func() float64 { return 0.5 })
	fs.Recompute()
	return fs.Result()[0]
}

// chamferFaceKey returns the bevel face — its normal is diagonal in XY (neither axis).
func chamferFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if n := f.Geometry().NormalAt(0, 0); stdmath.Abs(n.X) > 0.1 && stdmath.Abs(n.Y) > 0.1 {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no chamfer face")
	return nil
}

// TestDeleteFaceHealsChamfer deletes the chamfer face of a beveled box: the two side faces
// extend to meet at the original sharp edge, healing the box back to vol 8 — a valid solid.
// The headline delete-face-then-heal acceptance.
func TestDeleteFaceHealsChamfer(t *testing.T) {
	chamfered := chamferedBox(t)
	healed, err := ops.DeleteFaces(chamfered, [][]byte{chamferFaceKey(t, chamfered)})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(healed); !r.Valid || !healed.IsSolid() {
		t.Fatalf("healed body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(healed, ops.DefaultQuality()).Volume; stdmath.Abs(got-8) > 1e-6 {
		t.Errorf("healed volume = %g, want 8 (sharp box restored)", got)
	}
}

// TestDeleteFaceLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestDeleteFaceLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.DeleteFaces(box, [][]byte{[]byte("ghost")}); err == nil {
		t.Error("delete-face with a lost key should error")
	}
}
