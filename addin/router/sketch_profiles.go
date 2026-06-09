// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// sketchProfiles enumerates the closed regions (profiles) a sketch yields, with their
// area and hole count — the seam between the sketcher and the features that consume them.
func sketchProfiles(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch(s, raw)
	if err != nil {
		return nil, err
	}
	profiles := sk.Profiles()
	out := make([]wire.ProfileInfo, profiles.Count())
	for i := 0; i < profiles.Count(); i++ {
		p := profiles.Item(i)
		out[i] = wire.ProfileInfo{
			Index:  i,
			Area:   p.Area(),
			Closed: p.IsClosed(),
			Holes:  len(p.InnerLoops()),
		}
	}
	return json.Marshal(wire.ListProfilesResult{Profiles: out})
}
