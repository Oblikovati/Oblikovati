// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"encoding/json"
	"maps"
	"math"
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/kernel/topo"
	obkmath "oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
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
	t.Parallel()
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{sideFaceKey(t, s)}, "angle": "3 deg", "neutralPlane": "origin/plane/xy",
	}); err != nil {
		t.Fatalf("neutral-plane draft: %v", err)
	}
}

// TestDraftNeutralPlaneBadRef: an unresolvable neutralPlane is a clean error.
func TestDraftNeutralPlaneBadRef(t *testing.T) {
	t.Parallel()
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{face}, "angle": "3 deg", "neutralPlane": "origin/plane/nope",
	}); err == nil {
		t.Error("bad neutralPlane ref should error")
	}
}

// TestDraftNeutralPlanePullDirection: a neutral-plane draft accepts an explicit pull direction
// (the override branch), overriding the plane-normal default. #1866.
func TestDraftNeutralPlanePullDirection(t *testing.T) {
	t.Parallel()
	s, _, _ := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{sideFaceKey(t, s)}, "angle": "3 deg",
		"neutralPlane": "origin/plane/xy", "pullDirection": []float64{0, 0, 1},
	}); err != nil {
		t.Fatalf("neutral-plane draft with pullDirection: %v", err)
	}
}

// TestDraftNeutralPlaneBadPullDirection: a malformed pull direction on a neutral-plane draft is a
// clean error (a 2-component vector cannot be a [dx,dy,dz]).
func TestDraftNeutralPlaneBadPullDirection(t *testing.T) {
	t.Parallel()
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "draft", map[string]any{
		"faceRefs": []string{face}, "angle": "3 deg",
		"neutralPlane": "origin/plane/xy", "pullDirection": []float64{0, 1},
	}); err == nil {
		t.Error("a 2-component pullDirection should error")
	}
}

// TestHoleDrillPointErrors: an unknown drillPoint value and an angled point with an unparseable
// tipAngle are both clean errors, not silent successes. #1863.
func TestHoleDrillPointErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		extra map[string]any
	}{
		{"unknown drillPoint", map[string]any{"drillPoint": "banana"}},
		{"angled bad tipAngle", map[string]any{"drillPoint": "angled", "tipAngle": "banana"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, face := extrudedSolid(t)
			args := map[string]any{"faceRef": face, "diameter": "4 mm", "depth": "5 mm"}
			maps.Copy(args, tc.extra)
			if _, err := applyMap(t, s, "hole", args); err == nil {
				t.Errorf("%s should error", tc.name)
			}
		})
	}
}

// TestCoilEndConditionErrors: an unparseable transition or flat angle on either spring end is a
// clean error. #1883.
func TestCoilEndConditionErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		extra map[string]any
	}{
		{"bad startTransitionAngle", map[string]any{"startTransitionAngle": "banana"}},
		{"bad startFlatAngle", map[string]any{"startFlatAngle": "banana"}},
		{"bad endTransitionAngle", map[string]any{"endTransitionAngle": "banana"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := profiledPart(t)
			args := map[string]any{"sketchIndex": 0, "pitch": "5 mm", "revolutions": "3"}
			maps.Copy(args, tc.extra)
			if _, err := applyMap(t, s, "coil", args); err == nil {
				t.Errorf("%s should error", tc.name)
			}
		})
	}
}

// springProfilePart is a part whose sketch is a proper COIL profile: a small section on a plane
// CONTAINING the coil axis and standing clear of it. profiledPart's sketch is on XY while the coil
// axis defaults to Z, so its profile plane contains the sweep direction — a zero-rise flat end then
// rotates that profile through its own plane and the body is degenerate by construction (#2080).
func springProfilePart(t *testing.T) *app.Session {
	t.Helper()
	s := app.NewSession()
	d, err := s.Workspace().Add(doc.Part, "spring.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	def := compdef.NewPartComponentDefinition()
	d.SetContent(def)
	sk := def.Sketches().Add(sketch.XZPlane()) // u = X (radial), v = Z (along the coil axis)
	at := func(x, z float64) *sketch.Point { return sk.Points().Add(obkmath.P2(x, z)) }
	corners := []*sketch.Point{at(2, 0), at(2.5, 0), at(2.5, 0.5), at(2, 0.5)} // 0.5 deep, radius 2
	for i, c := range corners {
		sk.Lines().Add(c, corners[(i+1)%len(corners)])
	}
	def.Recompute()
	return s
}

// springCoilVolume builds a coil on the spring profile and returns its volume.
func springCoilVolume(t *testing.T, extra map[string]any) float64 {
	t.Helper()
	s := springProfilePart(t)
	// Pitch 20 mm against a 0.5 cm deep section. A 180° flat plus a 90° transition costs the worst
	// one-turn window all but 3/8 of the pitch, so the rise there is 0.75 cm — still clear of the
	// section's own 0.5 cm depth, which is what keeps consecutive turns apart (#2080).
	args := map[string]any{"sketchIndex": 0, "pitch": "20 mm", "revolutions": "3"}
	maps.Copy(args, extra)
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

// TestCoilEndConditions: start/end transition + flat sweeps are accepted and build a healthy coil
// with more geometry (the flat/transition turns) than the plain helix (#1883).
func TestCoilEndConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~12s): `make test-corpus`")
	}
	t.Parallel()
	plain := springCoilVolume(t, nil)
	ground := springCoilVolume(t, map[string]any{
		"startTransitionAngle": "90 deg", "startFlatAngle": "180 deg",
		"endTransitionAngle": "90 deg", "endFlatAngle": "180 deg",
	})
	if ground <= plain {
		t.Errorf("ground-ended coil volume %g should exceed plain %g (added flat/transition turns)", ground, plain)
	}
}

// TestHoleAngledDrillPointChangesVolume: an angled drill point produces a different blind-hole
// volume than a flat bottom — proving drillPoint/tipAngle reach the model's PointAngle (#1863).
func TestHoleAngledDrillPointChangesVolume(t *testing.T) {
	t.Parallel()
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
	maps.Copy(args, extra)
	if _, err := applyMap(t, s, "hole", args); err != nil {
		t.Fatalf("drill hole %v: %v", extra, err)
	}
	return bodyVolume(t, s)
}
