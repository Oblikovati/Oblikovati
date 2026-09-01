// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"strings"
	"testing"

	"oblikovati.org/math"
)

func TestConvexHullDegenerateDiagnostics(t *testing.T) {
	t.Parallel()
	if got := itoa(42); got != "42" {
		t.Fatalf("itoa(42) = %q", got)
	}
	_, err := ConvexHull([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}, "hull")
	if err == nil || !strings.Contains(err.Error(), "convex hull: need at least 4 distinct points") {
		t.Fatalf("ConvexHull too-few error = %v", err)
	}
	_, err = ConvexHull([]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0), math.P3(3, 0, 0)}, "hull")
	if err == nil || !strings.Contains(err.Error(), "collinear or coplanar") {
		t.Fatalf("ConvexHull collinear error = %v", err)
	}
}

func TestConvexHullSeedHelpersDegenerate(t *testing.T) {
	t.Parallel()
	pts := []math.Point3{math.P3(1, 1, 1), math.P3(1, 1, 1)}
	if got := boundsDiagonal(pts); got != 1 {
		t.Fatalf("boundsDiagonal coincident = %v, want 1", got)
	}
	if a, b := farthestPair(pts); a != b {
		t.Fatalf("farthestPair coincident = %d,%d, want same index", a, b)
	}
	if got := farthestFromLine(pts, 0, 1); got != -1 {
		t.Fatalf("farthestFromLine zero line = %d, want -1", got)
	}
	planar := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(1, 1, 0)}
	if got := farthestFromPlane(planar, 0, 1, 2); got != -1 {
		t.Fatalf("farthestFromPlane planar = %d, want -1", got)
	}
}
