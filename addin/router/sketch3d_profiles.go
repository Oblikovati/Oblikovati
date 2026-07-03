// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/sketch"
)

// sketch3DProfiles enumerates the closed planar loops of a 3D sketch.
func sketch3DProfiles(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.ListProfiles3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListProfiles3DResult{}, err
	}
	profiles := sk.Profiles3D()
	out := make([]wire.Profile3DInfo, len(profiles))
	for i, p := range profiles {
		out[i] = profile3DInfo(i, p)
	}
	return wire.ListProfiles3DResult{Profiles: out}, nil
}

// sketch3DPaths enumerates the connected line/arc chains of a 3D sketch.
func sketch3DPaths(_ *app.Session, part *compdef.PartComponentDefinition, in wire.Sketch3DArgs) (wire.ListPaths3DResult, error) {
	sk, err := sketch3DAtIndex(part, in.SketchIndex)
	if err != nil {
		return wire.ListPaths3DResult{}, err
	}
	paths := sk.Paths3D()
	out := make([]wire.Path3DInfo, len(paths))
	for i, p := range paths {
		out[i] = wire.Path3DInfo{Index: i, Closed: p.IsClosed(), Points: p.Count()}
	}
	return wire.ListPaths3DResult{Paths: out}, nil
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
