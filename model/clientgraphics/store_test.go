// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"github.com/Oblikovati/api/types"
	"github.com/Oblikovati/api/wire"
)

// meshArgs is a minimal one-triangle persistent group submitted under clientID.
func meshArgs(clientID string, lane types.GraphicsLane) wire.SetClientGraphicsArgs {
	return wire.SetClientGraphicsArgs{
		ClientId: clientID, Lane: string(lane),
		Nodes: []wire.GraphicsNode{{Primitives: []wire.GraphicsPrimitive{{
			Kind:        string(types.GraphicsTriangles),
			Coordinates: []float64{0, 0, 0, 1, 0, 0, 0, 1, 0},
			Indices:     []int{0, 1, 2},
		}}}},
	}
}

func mustDecode(t *testing.T, args wire.SetClientGraphicsArgs) Group {
	t.Helper()
	g, err := DecodeGroup(args)
	if err != nil {
		t.Fatalf("DecodeGroup: %v", err)
	}
	return g
}

func TestStoreSetReplacesByClientId(t *testing.T) {
	s := NewStore()
	s.Set(mustDecode(t, meshArgs("a", LanePersistent)))
	s.Set(mustDecode(t, meshArgs("a", LanePersistent))) // same id replaces, not appends
	if got := len(s.Groups()); got != 1 {
		t.Errorf("group count = %d, want 1 after replace", got)
	}
}

func TestStoreDeleteRemovesGroup(t *testing.T) {
	s := NewStore()
	s.Set(mustDecode(t, meshArgs("a", LanePersistent)))
	s.Delete("a")
	if got := len(s.Groups()); got != 0 {
		t.Errorf("group count = %d, want 0 after delete", got)
	}
}

func TestStoreSetVisibleUnknownErrors(t *testing.T) {
	s := NewStore()
	if err := s.SetVisible("missing", false); err == nil {
		t.Error("SetVisible on unknown id should error")
	}
}

func TestStoreClearInteractionKeepsPersistent(t *testing.T) {
	s := NewStore()
	s.Set(mustDecode(t, meshArgs("keep", LanePersistent)))
	s.Set(mustDecode(t, meshArgs("over", LaneOverlay)))
	s.Set(mustDecode(t, meshArgs("prev", LanePreview)))
	s.ClearInteraction()
	groups := s.Groups()
	if len(groups) != 1 || groups[0].Name() != "keep" {
		t.Errorf("after ClearInteraction groups = %v, want only persistent 'keep'", groups)
	}
}

func TestStoreReplaceLaneForcesLane(t *testing.T) {
	s := NewStore()
	g := mustDecode(t, meshArgs("p", LanePersistent)) // submitted persistent...
	s.ReplaceLane(LanePreview, []Group{g})            // ...but forced into preview
	groups := s.Groups()
	if len(groups) != 1 || groups[0].Lane() != string(LanePreview) {
		t.Errorf("ReplaceLane: group lane = %v, want preview", groups)
	}
}

func TestDecodeRejectsBadCoordinateArity(t *testing.T) {
	args := meshArgs("a", LanePersistent)
	args.Nodes[0].Primitives[0].Coordinates = []float64{0, 0} // not a multiple of 3
	if _, err := DecodeGroup(args); err == nil {
		t.Error("DecodeGroup should reject non-triple coordinates")
	}
}

func TestDecodeRejectsBadTransform(t *testing.T) {
	args := meshArgs("a", LanePersistent)
	args.Nodes[0].Transform = []float64{1, 2, 3} // not 16 cells
	if _, err := DecodeGroup(args); err == nil {
		t.Error("DecodeGroup should reject a non-16-cell transform")
	}
}

func TestGroupSatisfiesScalarContract(t *testing.T) {
	g := mustDecode(t, meshArgs("a", LanePersistent))
	if g.Name() != "a" || g.Lane() != string(LanePersistent) || !g.Visible() || g.NodeCount() != 1 || g.PrimitiveCount() != 1 {
		t.Errorf("scalar view wrong: name=%q lane=%q vis=%v nodes=%d prims=%d",
			g.Name(), g.Lane(), g.Visible(), g.NodeCount(), g.PrimitiveCount())
	}
}
