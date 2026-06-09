// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

func TestNopStandoffCSG(t *testing.T) {
	post := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 48, 0), 0, 1.2, "standoff-post")
	capsule, err := ops.ConvexHull(standoffSphereSamples(0.18, -0.06, 1.56), "standoff-capsule")
	if err != nil {
		t.Fatalf("ConvexHull(standoff capsule): %v", err)
	}
	body := joinOrFatal(t, post, capsule, "standoff capsule")

	requireValidNopSolid(t, "standoff", body)
	if got := vol(body); got <= vol(post) || got >= vol(post)+vol(capsule) {
		t.Errorf("standoff volume = %.6f, want between post %.6f and sum %.6f", got, vol(post), vol(post)+vol(capsule))
	}
}
