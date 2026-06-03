// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/kernel/topo"
	"github.com/Oblikovati/oblikovati/math"
)

// Modify / direct-edit features operate on whole bodies or picked faces. Combine is
// real in phase A (it is exactly a boolean over two bodies, which the kernel
// already does for the non-overlapping cases); the face-level direct edits
// (move/offset/delete/replace/thicken) and split need the general boolean / tolerant
// machinery (phase C), so they resolve their inputs then defer.

// CombineDefinition combines two running bodies (by index) under a boolean op.
type CombineDefinition struct {
	TargetIndex int
	ToolIndex   int
	Operation   ops.PartFeatureOperation
}

// CombineFeature booleans two bodies in the running state into one result.
type CombineFeature struct{ def *CombineDefinition }

// Definition returns the combine recipe.
func (c *CombineFeature) Definition() *CombineDefinition { return c.def }

// Kind implements [Feature].
func (c *CombineFeature) Kind() string { return "combine" }

// Recompute booleans the target and tool bodies, replacing them with the result.
func (c *CombineFeature) Recompute(in Input) (Output, error) {
	ti, oi := c.def.TargetIndex, c.def.ToolIndex
	if !validIndex(ti, in.Bodies) || !validIndex(oi, in.Bodies) || ti == oi {
		return Output{}, fmt.Errorf("combine: invalid body indices %d,%d (have %d)", ti, oi, len(in.Bodies))
	}
	res, err := ops.Boolean(c.def.Operation, in.Bodies[ti], in.Bodies[oi])
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceTwo(in.Bodies, ti, oi, res)}, nil
}

func validIndex(i int, bodies []*topo.Body) bool { return i >= 0 && i < len(bodies) }

// replaceTwo returns the bodies with the target/tool removed and the result (if
// non-empty) appended.
func replaceTwo(bodies []*topo.Body, ti, oi int, res *topo.Body) []*topo.Body {
	var out []*topo.Body
	for i, b := range bodies {
		if i != ti && i != oi {
			out = append(out, b)
		}
	}
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out
}

// faceEditFeature is the shared shape of the deferred direct-edit features: they
// resolve picked face keys against the running body, then defer the geometry.
type faceEditFeature struct {
	kind     string
	faceKeys [][]byte
}

func (f *faceEditFeature) Kind() string { return f.kind }
func (f *faceEditFeature) Recompute(in Input) (Output, error) {
	return resolveFacesThenDefer(in, f.faceKeys, f.kind)
}

// FaceKeys returns the reference keys of the faces this direct edit acts on. It lets
// the recipe serialize every face-edit feature uniformly (they share this shape).
func (f *faceEditFeature) FaceKeys() [][]byte { return f.faceKeys }

// SplitFeature, ReplaceFaceFeature and ThickenFeature are the direct edits whose geometry
// still defers (phase C).
type (
	SplitFeature       struct{ faceEditFeature }
	ReplaceFaceFeature struct{ faceEditFeature }
	ThickenFeature     struct{ faceEditFeature }
)

// DeleteFaceFeature removes the picked faces and heals the openings (extends neighbours).
type DeleteFaceFeature struct{ faceEditFeature }

// Recompute deletes the picked faces from the running body and heals it (see
// kernel/ops/delete_face.go); a non-healable selection makes the feature go Sick.
func (f *DeleteFaceFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, ops.DeleteFaces)
}

// MoveFaceFeature translates the picked faces by a vector, retrimming the neighbours.
type MoveFaceFeature struct {
	faceEditFeature
	translation math.Vector3
}

// Translation returns the move-face displacement (for the UI / serialization).
func (f *MoveFaceFeature) Translation() math.Vector3 { return f.translation }

// Recompute moves the picked faces on the running body (see kernel/ops/move_face.go).
func (f *MoveFaceFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		return ops.MoveFaces(b, keys, f.translation)
	})
}

// FaceOffsetFeature moves the picked faces along their own normals by a distance.
type FaceOffsetFeature struct {
	faceEditFeature
	distance float64
}

// Distance returns the face-offset distance (for the UI / serialization).
func (f *FaceOffsetFeature) Distance() float64 { return f.distance }

// Recompute offsets the picked faces on the running body (see kernel/ops/move_face.go).
func (f *FaceOffsetFeature) Recompute(in Input) (Output, error) {
	return retopoFacesBody(in, f.faceKeys, f.kind, func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		return ops.OffsetFaces(b, keys, f.distance)
	})
}

// retopoFacesBody applies a face retopology op to the running body and replaces it; a lost
// key (surfaced by the op) makes the feature go Sick.
func retopoFacesBody(in Input, keys [][]byte, feat string, op func(*topo.Body, [][]byte) (*topo.Body, error)) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	result, err := op(body, keys)
	if err != nil {
		return Output{}, fmt.Errorf("%s: %w", feat, err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// ModifyFeatures adds modify/direct-edit features into the engine.
type ModifyFeatures struct{ engine *PartFeatures }

// NewModifyFeatures binds the collection to an engine.
func NewModifyFeatures(engine *PartFeatures) *ModifyFeatures { return &ModifyFeatures{engine} }

// AddCombine booleans two running bodies (by index) under op.
func (c *ModifyFeatures) AddCombine(targetIndex, toolIndex int, op ops.PartFeatureOperation) *PartFeature {
	return c.engine.Add(&CombineFeature{def: &CombineDefinition{TargetIndex: targetIndex, ToolIndex: toolIndex, Operation: op}})
}

// AddSplit/AddMoveFace/AddFaceOffset/AddDeleteFace/AddReplaceFace/AddThicken add the
// deferred direct edits over the given face keys.
func (c *ModifyFeatures) AddSplit(faceKeys [][]byte) *PartFeature {
	return c.engine.Add(&SplitFeature{faceEditFeature{kind: "split", faceKeys: faceKeys}})
}

func (c *ModifyFeatures) AddMoveFace(faceKeys [][]byte, translation math.Vector3) *PartFeature {
	return c.engine.Add(&MoveFaceFeature{faceEditFeature: faceEditFeature{kind: "move-face", faceKeys: faceKeys}, translation: translation})
}

func (c *ModifyFeatures) AddFaceOffset(faceKeys [][]byte, distance float64) *PartFeature {
	return c.engine.Add(&FaceOffsetFeature{faceEditFeature: faceEditFeature{kind: "face-offset", faceKeys: faceKeys}, distance: distance})
}

func (c *ModifyFeatures) AddDeleteFace(faceKeys [][]byte) *PartFeature {
	return c.engine.Add(&DeleteFaceFeature{faceEditFeature{kind: "delete-face", faceKeys: faceKeys}})
}

func (c *ModifyFeatures) AddReplaceFace(faceKeys [][]byte) *PartFeature {
	return c.engine.Add(&ReplaceFaceFeature{faceEditFeature{kind: "replace-face", faceKeys: faceKeys}})
}

func (c *ModifyFeatures) AddThicken(faceKeys [][]byte) *PartFeature {
	return c.engine.Add(&ThickenFeature{faceEditFeature{kind: "thicken", faceKeys: faceKeys}})
}
