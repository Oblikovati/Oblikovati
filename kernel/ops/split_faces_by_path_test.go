// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops/query"

	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// TestSplitFacesByPathScoresFaceWithoutRemovingMaterial is the #2068 core: imprinting a projected
// polyline onto a plate scores its faces (adding edges/faces) while leaving the volume untouched —
// the split-FACE contract, along a path rather than a straight plane section.
func TestSplitFacesByPathScoresFaceWithoutRemovingMaterial(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 2) // plate [0,4]×[0,4]×[0,2]
	before := query.BodyGeometryProperties(box, ops.DefaultQuality()).Volume
	beforeFaces := len(box.Faces())

	// A zig-zag from the x=0 edge to the x=4 edge across the top cap (z=2); its ends touch the
	// boundary so the imprint divides the face rather than leaving a dangling scratch.
	path := []math.Point3{math.P3(0, 1, 2), math.P3(2, 3, 2), math.P3(4, 1, 2)}
	got, err := ops.SplitFacesByPath(box, path, math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("SplitFacesByPath: %v", err)
	}
	if r := ops.Validate(got); !r.Valid || !got.IsSolid() {
		t.Fatalf("imprinted body is not a valid solid: %+v", r)
	}
	if after := query.BodyGeometryProperties(got, ops.DefaultQuality()).Volume; stdmath.Abs(after-before) > 1e-6*before {
		t.Errorf("volume %g → %g: a face split must remove no material (#2068)", before, after)
	}
	if after := len(got.Faces()); after <= beforeFaces {
		t.Errorf("faces %d → %d: the path did not score any face", beforeFaces, after)
	}
}

// TestSplitFacesByPathRejectsBadInput: fewer than two points, or a projection direction with no
// length, are refused rather than producing a degenerate tool.
func TestSplitFacesByPathRejectsBadInput(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	if _, err := ops.SplitFacesByPath(box, []math.Point3{math.P3(0, 0, 2)}, math.V3(0, 0, 1)); err == nil {
		t.Error("a one-point path should be refused")
	}
	path := []math.Point3{math.P3(0, 1, 2), math.P3(2, 1, 2)}
	if _, err := ops.SplitFacesByPath(box, path, math.V3(0, 0, 0)); err == nil {
		t.Error("a zero-length projection direction should be refused")
	}
}
