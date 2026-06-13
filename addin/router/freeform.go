// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// The freeform cage-editing surface (M10-F03 PBI-114, #699): edit a placed sub-D
// feature's control cage by its stable feature id, recompute, and return the refreshed
// detail — the wire face of FreeformBody.SetLevel/MoveVertices/CreaseEdges.

// cageEditContext is the resolved target of one cage edit: the part, the placed
// feature (with its history index, for the detail reply) and its editable cage.
type cageEditContext struct {
	part  *compdef.PartComponentDefinition
	pf    *feature.PartFeature
	index int
	body  *feature.FreeformBody
}

// resolveFreeform resolves an id to the freeform feature, rejecting other kinds with
// the offending kind name.
func resolveFreeform(s *app.Session, id uint64, method string) (*cageEditContext, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	f, idx, err := partFeatureByID(part, id, method)
	if err != nil {
		return nil, err
	}
	ff, ok := f.Definition().(*feature.FreeformFeature)
	if !ok {
		return nil, fmt.Errorf("%s: feature %d is a %s, not a freeform body", method, id, f.Kind())
	}
	return &cageEditContext{part: part, pf: f, index: idx, body: ff.FreeformBody()}, nil
}

// commitCageEdit recomputes after a cage mutation and replies with the refreshed detail.
func (c *cageEditContext) commitCageEdit(s *app.Session) (json.RawMessage, error) {
	if err := s.CommitFeatureEdit(c.pf); err != nil {
		return nil, err
	}
	return featureDetailReply(c.part, c.pf, c.index)
}

// freeformSetLevel changes the subdivision level the cage is evaluated at.
func freeformSetLevel(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetFreeformLevelArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	c, err := resolveFreeform(s, in.ID, wire.MethodFreeformSetLevel)
	if err != nil {
		return nil, err
	}
	c.body.SetLevel(in.Level)
	return c.commitCageEdit(s)
}

// freeformMoveVertices translates the selected cage vertices by the given vector.
func freeformMoveVertices(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.MoveFreeformVerticesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	c, err := resolveFreeform(s, in.ID, wire.MethodFreeformMoveVertices)
	if err != nil {
		return nil, err
	}
	if err := checkCageIndices(c.body, in.Vertices, wire.MethodFreeformMoveVertices); err != nil {
		return nil, err
	}
	t := math.V3(in.Translation[0], in.Translation[1], in.Translation[2])
	c.body.MoveVertices(in.Vertices, t)
	return c.commitCageEdit(s)
}

// freeformCreaseEdges sets the crease sharpness on the selected cage edges.
func freeformCreaseEdges(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.CreaseFreeformEdgesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	c, err := resolveFreeform(s, in.ID, wire.MethodFreeformCreaseEdges)
	if err != nil {
		return nil, err
	}
	if in.Sharpness < 0 || in.Sharpness > 1 {
		return nil, fmt.Errorf("%s: sharpness %g out of range (want 0..1)", wire.MethodFreeformCreaseEdges, in.Sharpness)
	}
	for _, e := range in.Edges {
		if err := checkCageIndices(c.body, e[:], wire.MethodFreeformCreaseEdges); err != nil {
			return nil, err
		}
	}
	c.body.CreaseEdges(in.Edges, in.Sharpness)
	return c.commitCageEdit(s)
}

// checkCageIndices rejects a vertex index outside the cage, naming the offender and
// the cage size (the model layer applies edits unchecked).
func checkCageIndices(b *feature.FreeformBody, indices []int, method string) error {
	n := b.Vertices().Count()
	for _, i := range indices {
		if i < 0 || i >= n {
			return fmt.Errorf("%s: cage vertex %d out of range (cage has %d vertices)", method, i, n)
		}
	}
	if len(indices) == 0 {
		return fmt.Errorf("%s: empty selection; expected at least one cage vertex index", method)
	}
	return nil
}
