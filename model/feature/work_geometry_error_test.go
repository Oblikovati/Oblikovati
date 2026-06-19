// SPDX-License-Identifier: GPL-2.0-only

package feature

import "testing"

// TestWorkGeometryResolveMissingPlane covers the "no work plane" branches: resolving a user
// plane ref that names no existing plane (index 99 in an empty work-geometry) is rejected by
// both the plane and the typed work-plane resolvers.
func TestWorkGeometryResolveMissingPlane(t *testing.T) {
	g := NewWorkGeometry()
	if _, err := g.ResolvePlaneRef(WorkRef("plane/99")); err == nil {
		t.Error("ResolvePlaneRef of a missing user plane should error")
	}
	if _, err := g.WorkPlaneByRef(WorkRef("plane/99")); err == nil {
		t.Error("WorkPlaneByRef of a missing user plane should error")
	}
}
