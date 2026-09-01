// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/api/wire/featureargs"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// draftPullDir builds a draft feature via buildDraft and returns its resolved pull direction.
func draftPullDir(t *testing.T, in featureargs.Draft) math.Vector3 {
	t.Helper()
	s, _, _ := extrudedSolid(t)
	part := s.ActiveDocument().Content().(*compdef.PartComponentDefinition)
	pf, err := buildDraft(part, in, func() float64 { return 0.05 })
	if err != nil {
		t.Fatalf("buildDraft(%+v): %v", in, err)
	}
	return pf.Definition().(*feature.FaceDraftFeature).Definition().PullDir
}

// TestDraftDefaultPullDirection: without pullDirection, the draft uses the host's default +Z pull.
func TestDraftDefaultPullDirection(t *testing.T) {
	t.Parallel()
	got := draftPullDir(t, featureargs.Draft{FaceRefs: []string{"f"}})
	if got != math.V3(0, 0, 1) {
		t.Errorf("default pull = %v, want +Z", got)
	}
}

// TestDraftExplicitPullDirection: an explicit pullDirection maps straight onto the model's
// AddDraftPull, so the feature's pull direction is exactly the vector supplied over the wire.
func TestDraftExplicitPullDirection(t *testing.T) {
	t.Parallel()
	got := draftPullDir(t, featureargs.Draft{FaceRefs: []string{"f"}, PullDirection: []float64{1, 0, 0}})
	if got != math.V3(1, 0, 0) {
		t.Errorf("explicit pull = %v, want +X", got)
	}
}

// TestDraftPullDirectionThroughRegistry: the full applyDraft path accepts a pullDirection on a
// real face and builds a feature (recomputeResult never errors for a well-formed request).
func TestDraftPullDirectionThroughRegistry(t *testing.T) {
	t.Parallel()
	s, _, face := extrudedSolid(t)
	args := map[string]any{"faceRefs": []string{face}, "angle": "3 deg", "pullDirection": []float64{0, 0, 1}}
	if _, err := applyMap(t, s, "draft", args); err != nil {
		t.Fatalf("draft with pullDirection: %v", err)
	}
}

// TestDraftRejectsBadPullDirection: a pullDirection that is not a 3-vector is a clean error.
func TestDraftRejectsBadPullDirection(t *testing.T) {
	t.Parallel()
	s, _, face := extrudedSolid(t)
	args := map[string]any{"faceRefs": []string{face}, "angle": "3 deg", "pullDirection": []float64{1, 0}}
	if _, err := applyMap(t, s, "draft", args); err == nil {
		t.Error("draft with a 2-component pullDirection should error")
	}
}
