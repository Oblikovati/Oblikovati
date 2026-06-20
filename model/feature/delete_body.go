// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bytes"
	"fmt"

	"oblikovati.org/kernel/topo"
)

// DeleteBodyDefinition removes one running body — identified by its persistent reference key — from
// the part (#1078). Keying by reference key (not index) anchors the deletion to the same body
// across a recompute, even when earlier edits reorder the body list; it is the body-set counterpart
// of the index-based Combine/Split features.
type DeleteBodyDefinition struct {
	BodyKey []byte
}

// DeleteBodyFeature drops the referenced body from the running state.
type DeleteBodyFeature struct {
	def *DeleteBodyDefinition
}

// Definition returns the delete-body recipe.
func (f *DeleteBodyFeature) Definition() *DeleteBodyDefinition { return f.def }

// Kind implements [Feature].
func (f *DeleteBodyFeature) Kind() string { return "delete-body" }

// Recompute returns the running bodies minus the one whose reference key matches; a missing target
// (its body was already consumed, or never existed) makes the feature go Sick rather than silently
// deleting nothing.
func (f *DeleteBodyFeature) Recompute(in Input) (Output, error) {
	kept := make([]*topo.Body, 0, len(in.Bodies))
	found := false
	for _, b := range in.Bodies {
		if bytes.Equal(b.ReferenceKey(), f.def.BodyKey) {
			found = true
			continue
		}
		kept = append(kept, b)
	}
	if !found {
		return Output{}, fmt.Errorf("delete-body: the body to delete is no longer in the part (key %x)", f.def.BodyKey)
	}
	return Output{Bodies: kept}, nil
}

// AddDeleteBody appends a delete-body feature removing the body with the given reference key.
func (fs *PartFeatures) AddDeleteBody(bodyKey []byte) *PartFeature {
	return fs.Add(&DeleteBodyFeature{def: &DeleteBodyDefinition{BodyKey: append([]byte(nil), bodyKey...)}})
}
