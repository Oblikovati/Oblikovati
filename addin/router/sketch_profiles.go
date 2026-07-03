// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// sketchProfiles enumerates the closed regions (profiles) a sketch yields, with their
// area and hole count — the seam between the sketcher and the features that consume them.
func sketchProfiles(_ *app.Session, part *compdef.PartComponentDefinition, in wire.SketchArgs) (wire.ListProfilesResult, error) {
	sk, err := sketchAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListProfilesResult{}, err
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
	return wire.ListProfilesResult{Profiles: out}, nil
}
