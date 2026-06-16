// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"testing"

	"oblikovati.org/math"
)

// groupWithNode seeds a store with a one-node group whose node carries id nodeID.
func groupWithNode(nodeID string) *Store {
	s := NewStore()
	s.Set(Group{clientID: "g", lane: LanePersistent, visible: true, nodes: []Node{{Id: nodeID}}})
	return s
}

// TestSetNodeTransformVisibleSelectable mutates a node by id and checks each setter.
func TestSetNodeTransformVisibleSelectable(t *testing.T) {
	s := groupWithNode("n1")

	if err := s.SetNodeTransform("g", "n1", math.Identity4(), true); err != nil {
		t.Fatalf("SetNodeTransform: %v", err)
	}
	if err := s.SetNodeVisible("g", "n1", false); err != nil {
		t.Fatalf("SetNodeVisible: %v", err)
	}
	if err := s.SetNodeSelectable("g", "n1", true); err != nil {
		t.Fatalf("SetNodeSelectable: %v", err)
	}
	n, err := s.node("g", "n1")
	if err != nil {
		t.Fatalf("node lookup: %v", err)
	}
	if !n.HasTransform || n.Visible == nil || *n.Visible || !n.Selectable {
		t.Errorf("node = %+v, want transform set, hidden, selectable", n)
	}
}

// TestNodeMutationErrors checks unknown group / node ids are diagnosable.
func TestNodeMutationErrors(t *testing.T) {
	s := groupWithNode("n1")
	if err := s.SetNodeVisible("missing", "n1", true); err == nil {
		t.Error("unknown group should error")
	}
	if err := s.SetNodeVisible("g", "missing", true); err == nil {
		t.Error("unknown node should error")
	}
}

// TestColorMapperRegistry registers named mappers and lists them sorted, rejecting a blank name.
func TestColorMapperRegistry(t *testing.T) {
	s := NewStore()
	m := &ColorMapper{Values: []float64{0, 1}, Colors: [][4]float32{{0, 0, 0, 1}, {1, 1, 1, 1}}}
	if err := s.RegisterMapper("heat", m); err != nil {
		t.Fatalf("RegisterMapper: %v", err)
	}
	if err := s.RegisterMapper("", m); err == nil {
		t.Error("a blank mapper name should error")
	}
	if got := s.Mapper("heat"); got == nil || len(got.Values) != 2 {
		t.Errorf("Mapper(heat) = %+v, want the registered 2-stop mapper", got)
	}
	list := s.Mappers()
	if len(list) != 1 || list[0].Name != "heat" || list[0].StopCount != 2 {
		t.Errorf("Mappers = %+v, want one heat/2-stop entry", list)
	}
}
