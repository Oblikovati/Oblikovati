// SPDX-License-Identifier: GPL-2.0-only

package blend_test

import (
	"testing"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// countTorus counts a body's toroidal faces.
func countTorus(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
}

// TestFilletTangentStripeTopPerimeter is the #1797 acceptance: filleting the whole top perimeter of a
// box whose vertical edges are already rounded — a closed tangent chain of 4 straight (plane∩plane)
// and 4 arc (plane∩cylinder) edges — builds ONE continuous blend stripe (4 cylinder + 4 torus faces
// meeting at shared section circles), a valid closed manifold solid. Before P4b this failed in the
// miter corner solver ("outer face must be planar"); the tangent junctions are G1, not miters.
func TestFilletTangentStripeTopPerimeter(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4)
	var verts [][]byte
	for _, e := range box.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			verts = append(verts, e.ReferenceKey())
		}
	}
	filleted, err := blend.FilletEdges(box, verts, 0.5)
	if err != nil {
		t.Fatalf("vertical fillet setup: %v", err)
	}
	before := query.BodyGeometryProperties(filleted, ops.DefaultQuality()).Volume

	top := topPerimeterKeys(t, filleted)
	if len(top) != 8 {
		t.Fatalf("expected 8 top-perimeter edges, got %d", len(top))
	}

	res, err := blend.FilletEdges(filleted, top, 0.25)
	if err != nil {
		t.Fatalf("tangent-stripe top-perimeter fillet failed: %v", err)
	}
	rep := ops.Validate(res)
	if !rep.Valid || !res.IsSolid() || !rep.Manifold || !rep.Closed || !rep.OrientationOK {
		t.Fatalf("stripe result invalid: valid=%v solid=%v manifold=%v closed=%v orient=%v issues=%v",
			rep.Valid, res.IsSolid(), rep.Manifold, rep.Closed, rep.OrientationOK, rep.Issues)
	}
	if rep.EulerCharacteristic != 2 {
		t.Errorf("Euler characteristic = %d, want 2 (genus-0 solid)", rep.EulerCharacteristic)
	}
	if tori := countTorus(res); tori != 4 {
		t.Errorf("torus faces = %d, want 4 (one per arc segment)", tori)
	}
	if cyls := brepfixture.CountCylinderFaces(res); cyls != 8 {
		t.Errorf("cylinder faces = %d, want 8 (4 vertical + 4 straight blends)", cyls)
	}
	after := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
	if removed := before - after; removed < 0.18 || removed > 0.22 {
		// OCCT BRepFilletAPI_MakeFillet oracle removes 0.198382 for this exact fixture; we read 0.2027
		// — a ~2% tessellation overshoot on the curved blend faces at DefaultQuality (verified against a
		// locally-built OCCT TKFillet). The tolerance brackets that agreement.
		t.Errorf("removed volume = %g, want ≈0.198 (OCCT oracle for the top-rim stripe)", removed)
	}
}

// topPerimeterKeys returns the reference keys of every edge lying entirely on the body's top plane.
func topPerimeterKeys(t *testing.T, b *topo.Body) [][]byte {
	t.Helper()
	maxZ := 0.0
	for _, v := range b.Vertices() {
		if v.Point().Z > maxZ {
			maxZ = v.Point().Z
		}
	}
	var keys [][]byte
	for _, e := range b.Edges() {
		if e.StartVertex().Point().Z >= maxZ-1e-9 && e.EndVertex().Point().Z >= maxZ-1e-9 {
			keys = append(keys, e.ReferenceKey())
		}
	}
	return keys
}
