// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestPartAppearanceResolvesThroughLiveAssignPath drives the exact path the UI/MCP uses —
// AssignAppearance(ScopePart) — then asks SurfaceLookup for the body's surface, asserting it
// carries the assigned appearance's albedo rather than the neutral default. This isolates
// model-side resolution from the head's render/cache for the part-document grey-appearance bug.
func TestPartAppearanceResolvesThroughLiveAssignPath(t *testing.T) {
	t.Parallel()
	s, _ := extrudedBoxPart(t)
	want, ok := s.Materials().Appearance("steel")
	if !ok {
		t.Fatal("steel appearance missing from the seeded library")
	}
	if err := s.AssignAppearance(ScopePart, "", "steel"); err != nil {
		t.Fatalf("AssignAppearance: %v", err)
	}

	look := s.SurfaceLookup()
	if look == nil {
		t.Fatal("part SurfaceLookup is nil")
	}
	body := s.VisibleBodies()[0]
	got := look(body)
	wantAlbedo := appearanceSurface(want).Albedo
	if got.Albedo != wantAlbedo {
		t.Errorf("body albedo = %v, want steel %v — part appearance not resolved (renders grey)",
			got.Albedo, wantAlbedo)
	}
}
