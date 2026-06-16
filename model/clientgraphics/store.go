// SPDX-License-Identifier: GPL-2.0-only

package clientgraphics

import (
	"fmt"
	"sort"
)

// Store holds the live graphics groups keyed by client id, across all lanes. It is the
// single owner of add-in-submitted graphics for a session; Build reads it each frame.
// Not safe for concurrent use — it is mutated and read on the session goroutine.
type Store struct {
	groups  map[string]*Group
	mappers map[string]*ColorMapper // named, reusable color mappers (M16-F05 #641)
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{groups: map[string]*Group{}, mappers: map[string]*ColorMapper{}}
}

// Set inserts or replaces a group (idempotent by client id) — the submit path.
func (s *Store) Set(g Group) {
	cp := g
	s.groups[g.clientID] = &cp
}

// Delete removes a group; it is a no-op if the id is unknown.
func (s *Store) Delete(clientID string) { delete(s.groups, clientID) }

// SetVisible toggles a group's visibility without resubmitting geometry; it errors if the
// id is unknown so a mis-keyed add-in call is diagnosable.
func (s *Store) SetVisible(clientID string, visible bool) error {
	g, ok := s.groups[clientID]
	if !ok {
		return fmt.Errorf("clientGraphics: no group %q to set visibility", clientID)
	}
	g.visible = visible
	return nil
}

// ReplaceLane drops every group in a lane and inserts the given ones — the
// InteractionGraphics update path, which replaces a transient lane wholesale. The
// supplied groups are forced into the lane.
func (s *Store) ReplaceLane(lane Lane, groups []Group) {
	s.clearLane(lane)
	for i := range groups {
		groups[i].lane = lane
		s.Set(groups[i])
	}
}

// ClearInteraction drops the transient overlay and preview lanes — called when a command
// or tool deactivates so previews vanish on commit/cancel.
func (s *Store) ClearInteraction() {
	s.clearLane(LaneOverlay)
	s.clearLane(LanePreview)
}

// clearLane removes every group in one lane.
func (s *Store) clearLane(lane Lane) {
	for id, g := range s.groups {
		if g.lane == lane {
			delete(s.groups, id)
		}
	}
}

// Groups returns every group sorted by client id, for deterministic enumeration and
// frame building (the renderer may reorder for batching, but the source order is stable).
func (s *Store) Groups() []*Group {
	out := make([]*Group, 0, len(s.groups))
	for _, g := range s.groups {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].clientID < out[j].clientID })
	return out
}
