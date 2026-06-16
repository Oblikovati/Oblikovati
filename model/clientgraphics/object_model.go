// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"fmt"
	"sort"

	"oblikovati.org/math"
)

// SetNodeTransform replaces one node's transform without resubmitting its geometry — the
// retained-mode move path (M16-F05 #641). An identity matrix clears the transform. It errors
// when the group or node id is unknown so a mis-keyed add-in call is diagnosable.
func (s *Store) SetNodeTransform(clientID, nodeID string, transform math.Matrix4, hasTransform bool) error {
	n, err := s.node(clientID, nodeID)
	if err != nil {
		return err
	}
	n.Transform, n.HasTransform = transform, hasTransform
	return nil
}

// SetNodeVisible toggles one node's visibility within a group without resubmitting geometry.
func (s *Store) SetNodeVisible(clientID, nodeID string, visible bool) error {
	n, err := s.node(clientID, nodeID)
	if err != nil {
		return err
	}
	v := visible
	n.Visible = &v
	return nil
}

// SetNodeSelectable toggles whether one node's primitives participate in picking.
func (s *Store) SetNodeSelectable(clientID, nodeID string, selectable bool) error {
	n, err := s.node(clientID, nodeID)
	if err != nil {
		return err
	}
	n.Selectable = selectable
	return nil
}

// node finds a node by group client id and node id, erroring when either is unknown.
func (s *Store) node(clientID, nodeID string) (*Node, error) {
	g, ok := s.groups[clientID]
	if !ok {
		return nil, fmt.Errorf("clientGraphics: no group %q", clientID)
	}
	for i := range g.nodes {
		if g.nodes[i].Id == nodeID {
			return &g.nodes[i], nil
		}
	}
	return nil, fmt.Errorf("clientGraphics: group %q has no node %q", clientID, nodeID)
}

// RegisterMapper stores a named, reusable color mapper that heatmap primitives reference by
// name instead of carrying an inline legend (M16-F05 #641). A blank name is rejected.
func (s *Store) RegisterMapper(name string, m *ColorMapper) error {
	if name == "" {
		return fmt.Errorf("clientGraphics: a color mapper must have a non-empty name")
	}
	s.mappers[name] = m
	return nil
}

// Mapper returns the registered mapper by name (nil when absent) — the Build path resolves a
// primitive's MapperName through this.
func (s *Store) Mapper(name string) *ColorMapper { return s.mappers[name] }

// MapperInfo is one entry of [Store.Mappers]: a registered mapper's name and stop count.
type MapperInfo struct {
	Name      string
	StopCount int
}

// Mappers returns the registered color mappers, sorted by name for a stable enumeration.
func (s *Store) Mappers() []MapperInfo {
	out := make([]MapperInfo, 0, len(s.mappers))
	for name, m := range s.mappers {
		out = append(out, MapperInfo{Name: name, StopCount: len(m.Values)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
