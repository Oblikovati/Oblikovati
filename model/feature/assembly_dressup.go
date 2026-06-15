// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/param"
)

// Assembly edge dress-up features (M11-F08, #735): chamfer and fillet picked component
// edges on every participating placement. The edges are stored as component-local lineage
// SUFFIXES; at recompute each participant's placed body (transformed under a per-occurrence
// lineage prefix) is matched by suffix to recover that placement's full edge keys, which the
// existing exact-match part dress-up ops then operate on. A participant whose component does
// not carry a picked edge passes through unchanged.

// AssemblyChamferFeature flat-chamfers the picked component edges by a distance on every
// participant.
type AssemblyChamferFeature struct {
	edgeSuffixes [][]byte
	distance     func() float64
	flatCorners  bool
}

// NewAssemblyChamferFeature returns a chamfer over the component edge suffixes, applying
// distance (a closure, so an edit reflows it).
func NewAssemblyChamferFeature(edgeSuffixes [][]byte, distance func() float64) *AssemblyChamferFeature {
	return &AssemblyChamferFeature{edgeSuffixes: edgeSuffixes, distance: distance}
}

// Kind implements [Feature].
func (f *AssemblyChamferFeature) Kind() string { return kindAssemblyChamfer }

// Recompute chamfers the matched edges of every participant body.
func (f *AssemblyChamferFeature) Recompute(in Input) (Output, error) {
	dist := f.distance()
	bodies, err := dressParticipants(in.Bodies, edgeSuffixKeys(f.edgeSuffixes), func(body *topo.Body, keys [][]byte) (*topo.Body, error) {
		out, err := chamferEdges(Input{Bodies: []*topo.Body{body}}, keys, dist, dist, "asmChamfer", f.flatCorners)
		if err != nil {
			return nil, err
		}
		return soleBody(out.Bodies, kindAssemblyChamfer)
	})
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// EditableParams exposes the chamfer's setback distance (#752-style edit).
func (f *AssemblyChamferFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Distance", param.Length, &f.distance)}
}

// AssemblyFilletFeature rounds the picked component edges to a constant radius on every
// participant.
type AssemblyFilletFeature struct {
	edgeSuffixes [][]byte
	radius       func() float64
}

// NewAssemblyFilletFeature returns a fillet over the component edge suffixes, applying
// radius (a closure, so an edit reflows it).
func NewAssemblyFilletFeature(edgeSuffixes [][]byte, radius func() float64) *AssemblyFilletFeature {
	return &AssemblyFilletFeature{edgeSuffixes: edgeSuffixes, radius: radius}
}

// Kind implements [Feature].
func (f *AssemblyFilletFeature) Kind() string { return kindAssemblyFillet }

// Recompute rounds the matched edges of every participant body.
func (f *AssemblyFilletFeature) Recompute(in Input) (Output, error) {
	r := f.radius()
	bodies, err := dressParticipants(in.Bodies, edgeSuffixKeys(f.edgeSuffixes), func(body *topo.Body, keys [][]byte) (*topo.Body, error) {
		return ops.FilletEdges(body, keys, r)
	})
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// EditableParams exposes the fillet radius.
func (f *AssemblyFilletFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Radius", param.Length, &f.radius)}
}

// dressParticipants applies a face-edit/dress-up to every participant body, resolving the
// component reference suffixes to that body's full keys via keysFor first; a body with no
// matching entity passes through unchanged. Shared by the assembly chamfer, fillet, and
// move-face (#735).
func dressParticipants(bodies []*topo.Body, keysFor func(*topo.Body) [][]byte, dress func(body *topo.Body, keys [][]byte) (*topo.Body, error)) ([]*topo.Body, error) {
	out := make([]*topo.Body, 0, len(bodies))
	for _, body := range bodies {
		keys := keysFor(body)
		if len(keys) == 0 {
			out = append(out, body)
			continue
		}
		dressed, err := dress(body, keys)
		if err != nil {
			return nil, err
		}
		out = append(out, dressed)
	}
	return out, nil
}

// edgeSuffixKeys / faceSuffixKeys return a resolver that maps each component suffix to a
// participant body's full edge / face reference keys (the occurrence-relative resolver
// from #735).
func edgeSuffixKeys(suffixes [][]byte) func(*topo.Body) [][]byte {
	return func(body *topo.Body) [][]byte {
		var keys [][]byte
		for _, s := range suffixes {
			keys = append(keys, body.EdgeReferenceKeysWithLineageSuffix(s)...)
		}
		return keys
	}
}

func faceSuffixKeys(suffixes [][]byte) func(*topo.Body) [][]byte {
	return func(body *topo.Body) [][]byte {
		var keys [][]byte
		for _, s := range suffixes {
			keys = append(keys, body.FaceReferenceKeysWithLineageSuffix(s)...)
		}
		return keys
	}
}

// soleBody returns the single body of a dress-up result, erroring if the op did not yield
// exactly one (a participant body chamfers to one body).
func soleBody(bodies []*topo.Body, feat string) (*topo.Body, error) {
	if len(bodies) != 1 {
		return nil, fmt.Errorf("%s: dress-up produced %d bodies, want 1", feat, len(bodies))
	}
	return bodies[0], nil
}
