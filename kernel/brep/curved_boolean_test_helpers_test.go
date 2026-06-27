// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Shared curved-boolean test helpers. (Previously defined alongside the bespoke per-pair handlers, which the
// general pipeline replaced; kept here so the general-path and primitive tests still use them — #1403.)

// assertWatertight fails the test unless every edge of the body is used by exactly two faces (a closed
// orientable manifold's edge-use invariant).
func assertWatertight(t *testing.T, b *topo.Body) {
	t.Helper()
	if !b.IsSolid() {
		t.Fatalf("body is not a solid: %+v", b)
	}
	for _, e := range b.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
}

// faceTypeCounts tallies a body's faces by analytic surface type, failing the test on any non-analytic face.
func faceTypeCounts(t *testing.T, b *topo.Body) (cones, cyls, planes int) {
	t.Helper()
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	return
}
