// SPDX-License-Identifier: GPL-2.0-only

package topo

import "testing"

// TestFacePickTessRoundTrip fences the opaque pick-tessellation memo on a Face: it starts nil, stores
// whatever ops hands it, and reads it back unchanged. The ops package holds the payload type and the
// hover-pick reuse logic (kernel/ops/pick.go); topo only owns the field's lifetime, so this test lives
// here to credit the accessors' coverage in package topo (a cross-package ops_test does not). The memo
// exists so orbiting a curved model does not re-tessellate every face every frame — see PickTess.
func TestFacePickTessRoundTrip(t *testing.T) {
	face := buildTetra().Faces()[0]
	if got := face.PickTess(); got != nil {
		t.Fatalf("a fresh face's PickTess = %v, want nil (memo empty until the first hover-pick)", got)
	}
	type stubMesh struct{ triangles int }
	memo := stubMesh{triangles: 42}
	face.SetPickTess(memo)
	got, ok := face.PickTess().(stubMesh)
	if !ok || got != memo {
		t.Fatalf("PickTess after SetPickTess = %v (ok=%v), want %v", got, ok, memo)
	}
}
