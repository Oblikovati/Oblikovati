// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/sketch"
)

// partWithSketches returns a session whose active document is an empty part ready to host profiles.
func partWithSketches(t *testing.T) (*Session, *compdef.PartComponentDefinition) {
	t.Helper()
	s := NewSession()
	def := compdef.NewPartComponentDefinition()
	pd, err := s.Workspace().Add(doc.Part, "network.obk", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	pd.SetContent(def)
	return s, def
}

// addSquareProfile adds a closed square profile (offset in x by ox) to the part and returns its handle.
func addSquareProfile(def interface {
	Sketches() *sketch.Sketches
}, side, ox float64) ProfileHandle {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0 := sk.Points().Add(math.P2(ox, 0))
	c1 := sk.Points().Add(math.P2(ox+side, 0))
	c2 := sk.Points().Add(math.P2(ox+side, side))
	c3 := sk.Points().Add(math.P2(ox, side))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	return ProfileHandle{Sketch: sk, ProfileIndex: 0}
}

func TestNetworkToolPickAssignsDirections(t *testing.T) {
	s, def := partWithSketches(t)
	u1 := addSquareProfile(def, 1, 0)
	u2 := addSquareProfile(def, 1, 2)
	v1 := addSquareProfile(def, 1, 4)
	tool := NewNetworkTool()
	tool.Pick(s, u1)
	tool.Pick(s, u1) // duplicate ignored
	tool.Pick(s, u2)
	if len(tool.uProfiles) != 2 {
		t.Fatalf("uProfiles = %d, want 2 (dedup)", len(tool.uProfiles))
	}
	if tool.CanCommit() {
		t.Error("two U and zero V curves should not be committable")
	}
	tool.Params().Bools[0].Set(true)
	if !tool.pickingV {
		t.Fatal("toggling Pick V curves should set pickingV")
	}
	tool.Pick(s, v1)
	if len(tool.vProfiles) != 1 {
		t.Fatalf("vProfiles = %d, want 1", len(tool.vProfiles))
	}
	if len(tool.Picks()) != 3 {
		t.Errorf("Picks() = %d, want 3", len(tool.Picks()))
	}
}

func TestNetworkToolPromptStages(t *testing.T) {
	tool := NewNetworkTool()
	if got := tool.Prompt(nil); got == "" {
		t.Error("U-stage prompt empty")
	}
	tool.pickingV = true
	if tool.Prompt(nil) == "" {
		t.Error("V-stage prompt empty")
	}
}

func TestNetworkToolBakesProfilesToModel(t *testing.T) {
	_, def := partWithSketches(t)
	h := addSquareProfile(def, 2, 0)
	lines := bakeProfiles([]ProfileHandle{h})
	if len(lines) != 1 || len(lines[0]) < 4 {
		t.Fatalf("bakeProfiles = %d polylines, first has %d points", len(lines), len(lines[0]))
	}
}

func TestNetworkToolCommitReportsBadGrid(t *testing.T) {
	s, def := partWithSketches(t)
	tool := NewNetworkTool()
	tool.uProfiles = []ProfileHandle{addSquareProfile(def, 1, 0), addSquareProfile(def, 1, 4)}
	tool.vProfiles = []ProfileHandle{addSquareProfile(def, 1, 8), addSquareProfile(def, 1, 12)}
	if !tool.CanCommit() {
		t.Fatal("two each way should be committable")
	}
	s.StartFeatureTool(tool)
	// The disjoint squares do not form a grid, so the draft previews sick and the commit gate
	// refuses OK — the unhealthy feature never enters the design (#1594, #1626).
	if err := s.OK(); err == nil {
		t.Error("committing non-intersecting profiles should report an error")
	}
	if tool.AddedFeature() != nil {
		t.Error("the sick network must be blocked by the commit gate, not committed")
	}
	if def.Features().Count() != 0 {
		t.Errorf("features count = %d, want 0 — a sick node must never persist in the tree", def.Features().Count())
	}
}

func TestNetworkToolAcceptsProfiles(t *testing.T) {
	if k := NewNetworkTool().AcceptedKinds(); len(k) != 1 || k[0] != SelectProfile {
		t.Errorf("AcceptedKinds = %v, want [SelectProfile]", k)
	}
}

// TestNetworkToolDraftFeature pins the #1626 commit-gate seam: no draft below two curves each
// way, a non-nil draft once the grid picks are complete.
func TestNetworkToolDraftFeature(t *testing.T) {
	_, def := partWithSketches(t)
	tool := NewNetworkTool()
	if _, ok := tool.DraftFeature(nil); ok {
		t.Error("DraftFeature must not build before two curves each way are picked")
	}
	tool.uProfiles = []ProfileHandle{addSquareProfile(def, 1, 0), addSquareProfile(def, 1, 4)}
	tool.vProfiles = []ProfileHandle{addSquareProfile(def, 1, 8), addSquareProfile(def, 1, 12)}
	if draft, ok := tool.DraftFeature(nil); !ok || draft == nil {
		t.Fatalf("DraftFeature = (%v, %v), want a non-nil draft once commit-ready", draft, ok)
	}
}
