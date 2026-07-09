// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
)

// Unreachable-API quick-wins from the M33 audit: draft neutral plane (#1866), coil end conditions
// (#1883) and hole angled drill point (#1863) — wiring model capability that had no authoring
// surface.

// sideFaceKey returns the reference key of a vertical side face (normal ⟂ Z) of the active body —
// one that actually meets the XY neutral plane in a line (a face parallel to it would not draft).
func sideFaceKey(t *testing.T, s *app.Session) string {
	t.Helper()
	for _, f := range activePartBody(t, s).Faces() {
		if n := topo.DescribeFace(f).Normal; math.Abs(n.Z) < 0.5 {
			return string(f.ReferenceKey())
		}
	}
	t.Fatal("no vertical side face found")
	return ""
}

// TestDraftNeutralPlane: a fixed-plane draft about origin/plane/xy is accepted and healthy — the
// faces pivot on their intersection with the neutral plane (#1866).
func TestDraftNeutralPlane(t *testing.T) {
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{sideFaceKey(t, s)}, "angle": "3 deg", "neutralPlane": "origin/plane/xy",
	}); err != nil {
		t.Fatalf("neutral-plane draft: %v", err)
	}
}

// TestDraftNeutralPlaneBadRef: an unresolvable neutralPlane is a clean error.
func TestDraftNeutralPlaneBadRef(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{face}, "angle": "3 deg", "neutralPlane": "origin/plane/nope",
	}); err == nil {
		t.Error("bad neutralPlane ref should error")
	}
}

// TestCoilEndConditions: start/end transition + flat sweeps are accepted and build a healthy coil
// with more geometry (the flat/transition turns) than the plain helix (#1883).
func TestCoilEndConditions(t *testing.T) {
	plain := coilVolume(t, nil)
	ground := coilVolume(t, map[string]any{
		"startTransitionAngle": "90 deg", "startFlatAngle": "180 deg",
		"endTransitionAngle": "90 deg", "endFlatAngle": "180 deg",
	})
	if ground <= plain {
		t.Errorf("ground-ended coil volume %g should exceed plain %g (added flat/transition turns)", ground, plain)
	}
}

func coilVolume(t *testing.T, extra map[string]any) float64 {
	t.Helper()
	s := profiledPart(t)
	args := map[string]any{"sketchIndex": 0, "pitch": "5 mm", "revolutions": "3"}
	for k, v := range extra {
		args[k] = v
	}
	raw, err := applyMap(t, s, "coil", args)
	if err != nil {
		t.Fatalf("coil %v: %v", extra, err)
	}
	var res struct {
		Healthy bool `json:"healthy"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || !res.Healthy {
		t.Fatalf("coil %v not healthy: %s", extra, raw)
	}
	return bodyVolume(t, s)
}

// TestHoleAngledDrillPointChangesVolume: an angled drill point produces a different blind-hole
// volume than a flat bottom — proving drillPoint/tipAngle reach the model's PointAngle (#1863).
func TestHoleAngledDrillPointChangesVolume(t *testing.T) {
	vFlat := blindHoleVolume(t, nil)
	vAngled := blindHoleVolume(t, map[string]any{"drillPoint": "angled"})
	if math.Abs(vAngled-vFlat) < 1e-6*vFlat {
		t.Errorf("angled drill-point volume %g == flat %g; drillPoint did not take effect", vAngled, vFlat)
	}
}

func blindHoleVolume(t *testing.T, extra map[string]any) float64 {
	t.Helper()
	s, _, face := extrudedSolid(t)
	args := map[string]any{"faceRef": face, "diameter": "4 mm", "depth": "5 mm"}
	for k, v := range extra {
		args[k] = v
	}
	if _, err := applyMap(t, s, "hole", args); err != nil {
		t.Fatalf("drill hole %v: %v", extra, err)
	}
	return bodyVolume(t, s)
}
