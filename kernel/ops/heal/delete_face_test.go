// SPDX-License-Identifier: GPL-2.0-only

package heal_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops/heal"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/feature"
)

// chamferedBox bevels one vertical edge of a 2×2×2 box (setback 0.5) via the chamfer
// feature, returning the resulting solid (vol 7.75) — a body with a chamfer face to delete.
func chamferedBox(t *testing.T) *topo.Body {
	t.Helper()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	var edge []byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			edge = e.ReferenceKey()
			break
		}
	}
	fs := feature.NewPartFeatures(nil)
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
	t.Parallel()
	chamfered := chamferedBox(t)
	healed, err := heal.DeleteFaces(chamfered, [][]byte{chamferFaceKey(t, chamfered)})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(healed); !r.Valid || !healed.IsSolid() {
		t.Fatalf("healed body not a valid solid: %+v", r)
	}
	if got := query.BodyGeometryProperties(healed, ops.DefaultQuality()).Volume; stdmath.Abs(got-8) > 1e-6 {
		t.Errorf("healed volume = %g, want 8 (sharp box restored)", got)
	}
}

// TestDeleteFaceKeepsSurvivingIdentity is ADR-0043 P3: healing a deleted face rebuilds the body,
// but the faces and edges it carries through unchanged must keep their ORIGINAL identity rather
// than be renumbered to the rebuild's delface:* ordinals — so a selection on an untouched face or
// edge survives the operation. Before P3 every result key was a fresh delface ordinal.
func TestDeleteFaceKeepsSurvivingIdentity(t *testing.T) {
	t.Parallel()
	chamfered := chamferedBox(t)
	inFaces, inEdges := map[string]bool{}, map[string]bool{}
	for _, f := range chamfered.Faces() {
		inFaces[string(f.ReferenceKey())] = true
	}
	for _, e := range chamfered.Edges() {
		inEdges[string(e.ReferenceKey())] = true
	}

	healed, err := heal.DeleteFaces(chamfered, [][]byte{chamferFaceKey(t, chamfered)})
	if err != nil {
		t.Fatal(err)
	}
	faceKept, edgeKept := 0, 0
	for _, f := range healed.Faces() {
		if inFaces[string(f.ReferenceKey())] {
			faceKept++
		}
	}
	for _, e := range healed.Edges() {
		if inEdges[string(e.ReferenceKey())] {
			edgeKept++
		}
	}
	if faceKept == 0 {
		t.Error("no surviving face kept its identity — all renamed to delface:f#N")
	}
	if edgeKept == 0 {
		t.Error("no surviving edge kept its identity — all renamed to delface:e#N")
	}
}

// TestDeleteFaceLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestDeleteFaceLostKeyErrors(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	if _, err := heal.DeleteFaces(box, [][]byte{[]byte("ghost")}); err == nil {
		t.Error("delete-face with a lost key should error")
	}
}
