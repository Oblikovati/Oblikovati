// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Region-correctness assertions for half-space cuts. Watertightness and analytic face counts are
// necessary-but-not-sufficient: a cut that keeps the wrong half or clips to the wrong depth stays
// watertight with the same face count. These helpers make "which region survived" a first-class
// assertion — volume against an analytic/oracle value and centroid on the kept side of the plane
// (Oblikovati/Oblikovati#1497, guarding the known "OUTSIDE-keep wrong region" failure mode).

// assertKeptVolume fails unless the body's tessellated volume is within relTol of want. Tessellation
// inscribes the true surface, so the measured volume runs slightly UNDER analytic; relTol absorbs that.
func assertKeptVolume(t *testing.T, body *topo.Body, want, relTol float64, label string) {
	t.Helper()
	got := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume
	if rel := stdmath.Abs(got-want) / want; rel > relTol {
		t.Errorf("%s: kept volume %.4f, want %.4f — rel %.4f > %.2f%% (wrong region kept or wrong depth)",
			label, got, want, rel, relTol*100)
	}
}

// assertCentroidKeptSide fails unless the body's centroid lies on the kept side of the cutting plane.
// HalfSpaceCut keeps { p : normal·(p−origin) ≤ 0 } (the side the normal points away from), so the
// centroid must satisfy the same inequality with a margin. This catches a cut that ignores the plane
// normal even when the complementary regions happen to have equal volume (e.g. a symmetric mid-plane).
func assertCentroidKeptSide(t *testing.T, body *topo.Body, origin math.Point3, normal math.Vector3, label string) {
	t.Helper()
	c := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Centroid
	signed := origin.VectorTo(c).Dot(normal)
	if signed > -1e-6 {
		t.Errorf("%s: centroid signed distance %.4f from plane is not on the kept side (want < 0; wrong half kept)",
			label, signed)
	}
}
