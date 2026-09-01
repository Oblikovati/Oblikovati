// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// AssemblyMoveFaceFeature translates picked component faces by a vector (in assembly space)
// on every participating placement — the assembly-context Move Face (M11-F08, #735). The
// faces are stored as component-local lineage suffixes and resolved per participant through
// the occurrence-relative resolver, then moved by the existing [transform.MoveFaces]. A participant
// whose component does not carry a picked face passes through unchanged.
type AssemblyMoveFaceFeature struct {
	faceSuffixes [][]byte
	translation  math.Vector3
}

// NewAssemblyMoveFaceFeature returns a move-face over the component face suffixes,
// translating them by delta.
func NewAssemblyMoveFaceFeature(faceSuffixes [][]byte, delta math.Vector3) *AssemblyMoveFaceFeature {
	return &AssemblyMoveFaceFeature{faceSuffixes: faceSuffixes, translation: delta}
}

// Kind implements [Feature].
func (f *AssemblyMoveFaceFeature) Kind() string { return kindAssemblyMoveFace }

// Recompute moves the matched faces of every participant body by the translation.
func (f *AssemblyMoveFaceFeature) Recompute(in Input) (Output, error) {
	bodies, err := dressParticipants(in.Bodies, faceSuffixKeys(f.faceSuffixes), func(body *topo.Body, keys [][]byte) (*topo.Body, error) {
		return transform.MoveFaces(body, keys, f.translation)
	})
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// EditableParams exposes the move's displacement magnitude (the part move-face's edit
// scalar), so assemblyFeatures.edit can re-dimension it.
func (f *AssemblyMoveFaceFeature) EditableParams() []EditableParam {
	return []EditableParam{spacingParam("Distance", &f.translation, math.V3(0, 0, 1))}
}
