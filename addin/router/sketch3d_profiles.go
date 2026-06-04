// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/model/sketch"
)

// sketch3DProfiles enumerates the closed planar loops of a 3D sketch.
func sketch3DProfiles(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	profiles := sk.Profiles3D()
	out := make([]wire.Profile3DInfo, len(profiles))
	for i, p := range profiles {
		out[i] = profile3DInfo(i, p)
	}
	return json.Marshal(wire.ListProfiles3DResult{Profiles: out})
}

// sketch3DPaths enumerates the connected line/arc chains of a 3D sketch.
func sketch3DPaths(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	sk, _, err := resolveSketch3D(s, raw)
	if err != nil {
		return nil, err
	}
	paths := sk.Paths3D()
	out := make([]wire.Path3DInfo, len(paths))
	for i, p := range paths {
		out[i] = wire.Path3DInfo{Index: i, Closed: p.IsClosed(), Points: p.Count()}
	}
	return json.Marshal(wire.ListPaths3DResult{Paths: out})
}

// profile3DInfo renders a 3D profile as its wire summary.
func profile3DInfo(index int, p *sketch.Profile3D) wire.Profile3DInfo {
	n := p.Normal()
	return wire.Profile3DInfo{
		Index: index, Area: p.Area(),
		Normal:   []float64{float64(n.X), float64(n.Y), float64(n.Z)},
		Vertices: len(p.Points()),
	}
}
