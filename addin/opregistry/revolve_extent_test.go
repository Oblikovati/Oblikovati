// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// The revolve's geometric extents over the JSON API (#1860). The model tests
// (model/feature/revolve_extent_test.go) pin the angles these resolve to; these pin the wiring —
// that the extent discriminator reaches the definition and that the terminators resolve from the
// same reference vocabulary extrude's extents use.
//
// Every case revolves the radius-2..4, height-3 rectangle about the origin Y axis, so a sweep of θ
// holds θ/2π of the 36π washer. The origin YZ plane contains that axis and stands at 90° from the
// profile, which makes it the natural terminator.

// revolveWasher is the volume the fixture profile sweeps in a complete turn: π(4²−2²)·3.
const revolveWasher = math.Pi * (4*4 - 2*2) * 3

// TestRevolveToFaceStopsOnARadialPlane: "to-face" sweeps until the profile reaches the named face,
// so the 90° origin YZ plane leaves a quarter of the washer — the angle comes from the model, not
// from a number the caller had to work out.
func TestRevolveToFaceStopsOnARadialPlane(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "axisRef": "origin/axis/y",
		"extent": "to-face", "toFace": "origin/plane/yz", "operation": "new",
	})
	if _, err := apply(t, s, "revolve", string(args)); err != nil {
		t.Fatalf("to-face revolve: %v", err)
	}
	got, want := bodyVolume(t, s), revolveWasher/4
	if math.Abs(got-want) > 0.01*want {
		t.Errorf("to-face volume = %g, want ≈%g (a 90° stop on the YZ plane)", got, want)
	}
}

// TestRevolveFromToBoundsBothWays: "from-to" runs backwards to fromFace and forwards to toFace. The
// YZ plane stands 90° from the profile on BOTH sides, so naming it at each end gives a half washer
// straddling the profile — proof that the "from" terminator is honoured, not ignored.
func TestRevolveFromToBoundsBothWays(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "axisRef": "origin/axis/y", "extent": "from-to",
		"fromFace": "origin/plane/yz", "toFace": "origin/plane/yz", "operation": "new",
	})
	if _, err := apply(t, s, "revolve", string(args)); err != nil {
		t.Fatalf("from-to revolve: %v", err)
	}
	got, want := bodyVolume(t, s), revolveWasher/2
	if math.Abs(got-want) > 0.01*want {
		t.Errorf("from-to volume = %g, want ≈%g (90° back plus 90° forward)", got, want)
	}
}

// TestRevolveAngleExtentStillNeedsAnAngle: relaxing "angle" out of the schema's required set (so a
// geometric extent need not carry a meaningless number) must not let the ANGLE extent through
// without one — that would silently build a full revolution.
func TestRevolveAngleExtentStillNeedsAnAngle(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{"sketchIndex": 0, "axisRef": "origin/axis/y"})
	_, err := apply(t, s, "revolve", string(args))
	if err == nil || !strings.Contains(err.Error(), "angle") {
		t.Errorf("angle-extent revolve with no angle gave %v; want an error naming \"angle\"", err)
	}
}

// TestRevolveGeometricExtentRejectsAngle2: angle2 is the asymmetric TWO-ANGLE mode, which names
// both sides itself. Combined with an extent that measures its own sweep, one of the two inputs
// would have to be dropped — so the op refuses instead of picking a winner silently.
func TestRevolveGeometricExtentRejectsAngle2(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "axisRef": "origin/axis/y", "extent": "to-face",
		"toFace": "origin/plane/yz", "angle2": "30 deg",
	})
	if _, err := apply(t, s, "revolve", string(args)); err == nil {
		t.Error("to-face revolve accepted angle2; the two name conflicting sweeps and must not combine")
	}
}

// TestRevolveToFaceNeedsATarget: an extent that terminates on a face and is given none is a caller
// error naming the missing field, not a silent fallback to some angle.
func TestRevolveToFaceNeedsATarget(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{"sketchIndex": 0, "axisRef": "origin/axis/y", "extent": "to-face"})
	_, err := apply(t, s, "revolve", string(args))
	if err == nil || !strings.Contains(err.Error(), "toFace") {
		t.Errorf("to-face revolve with no target gave %v; want an error naming \"toFace\"", err)
	}
}

// TestRevolveUnknownExtentIsRejected keeps a typo from degrading to the angle extent.
func TestRevolveUnknownExtentIsRejected(t *testing.T) {
	t.Parallel()
	s := revolveReadyPart(t)
	args, _ := json.Marshal(map[string]any{
		"sketchIndex": 0, "axisRef": "origin/axis/y", "angle": "90 deg", "extent": "through-all",
	})
	if _, err := apply(t, s, "revolve", string(args)); err == nil {
		t.Error("revolve accepted extent \"through-all\"; a revolve has no such termination")
	}
}
