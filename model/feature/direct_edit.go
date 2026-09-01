// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The consolidated direct-edit feature (M09-F04 PBI-108, #332): one feature whose
// operation — move / size / rotate / delete on picked faces, or a uniform scale of the
// running body — is discriminated by the frozen DirectEditOperationType. The face
// operations reuse the same plane-rebuild kernel ops as the standalone face edits; scale
// is a similarity TransformBody about a base point with identity lineage (an in-place
// edit, so downstream picks survive).

// DirectEditDefinition is the direct-edit recipe. Per operation:
// move (FaceKeys+Translation), size (FaceKeys+Direction+Distance — push/pull along the
// direction), rotate (FaceKeys+AxisPoint/AxisDir/Angle), delete (FaceKeys),
// scale (ScaleFactor about BasePoint, whole body).
type DirectEditDefinition struct {
	Operation   types.DirectEditOperationType
	FaceKeys    [][]byte
	Translation math.Vector3
	Direction   math.Vector3
	Distance    func() float64
	AxisPoint   math.Point3
	AxisDir     math.Vector3
	Angle       func() float64
	ScaleFactor func() float64
	BasePoint   math.Point3
}

// DirectEditFeature applies one direct-edit operation to the running body.
type DirectEditFeature struct{ def *DirectEditDefinition }

// Definition returns the direct-edit recipe.
func (f *DirectEditFeature) Definition() *DirectEditDefinition { return f.def }

// Kind implements [Feature].
func (f *DirectEditFeature) Kind() string { return "directEdit" }

// Recompute dispatches on the operation; an unknown operation is a precise error.
func (f *DirectEditFeature) Recompute(in Input) (Output, error) {
	switch f.def.Operation {
	case types.DirectEditMoveOperation:
		return retopoFacesBody(in, f.def.FaceKeys, "directEdit move", func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
			return transform.MoveFaces(b, keys, f.def.Translation)
		})
	case types.DirectEditSizeOperation:
		return f.recomputeSize(in)
	case types.DirectEditRotateOperation:
		return f.recomputeRotate(in)
	case types.DirectEditDeleteOperation:
		return retopoFacesBody(in, f.def.FaceKeys, "directEdit delete", ops.DeleteFaces)
	case types.DirectEditScaleOperation:
		return f.recomputeScale(in)
	default:
		return Output{}, fmt.Errorf("directEdit: unsupported operation %v (want move/size/rotate/delete/scale)", f.def.Operation)
	}
}

// recomputeSize pushes the picked faces along the direction by distance (the push/pull
// size edit — MoveFaces with the vector direction·distance).
func (f *DirectEditFeature) recomputeSize(in Input) (Output, error) {
	dir, err := math.UnitVector3FromVector(f.def.Direction)
	if err != nil {
		return Output{}, fmt.Errorf("directEdit size: direction %v is degenerate", f.def.Direction)
	}
	delta := dir.AsVector().Scale(callOrZero(f.def.Distance))
	return retopoFacesBody(in, f.def.FaceKeys, "directEdit size", func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		return transform.MoveFaces(b, keys, delta)
	})
}

// recomputeRotate rotates the picked faces about the axis.
func (f *DirectEditFeature) recomputeRotate(in Input) (Output, error) {
	dir, err := math.UnitVector3FromVector(f.def.AxisDir)
	if err != nil {
		return Output{}, fmt.Errorf("directEdit rotate: axis %v is degenerate", f.def.AxisDir)
	}
	return retopoFacesBody(in, f.def.FaceKeys, "directEdit rotate", func(b *topo.Body, keys [][]byte) (*topo.Body, error) {
		return transform.RotateFaces(b, keys, f.def.AxisPoint, dir, callOrZero(f.def.Angle))
	})
}

// recomputeScale scales the running body uniformly about the base point. Identity lineage
// derive keeps every reference key, so downstream picks survive the edit.
func (f *DirectEditFeature) recomputeScale(in Input) (Output, error) {
	k := callOrZero(f.def.ScaleFactor)
	if k <= 0 {
		return Output{}, fmt.Errorf("directEdit scale: factor %g must be > 0", k)
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	m := scaleAbout(f.def.BasePoint, k)
	scaled, err := transform.TransformBody(body, m, func(l topo.Lineage) topo.Lineage { return l })
	if err != nil {
		return Output{}, fmt.Errorf("directEdit scale: %w", err)
	}
	return Output{Bodies: replaceBody(in.Bodies, body, scaled)}, nil
}

// scaleAbout is the uniform similarity T(p)·S(k)·T(−p).
func scaleAbout(p math.Point3, k float64) math.Matrix4 {
	toOrigin := math.Translation4(p.AsVector().Scale(-1))
	back := math.Translation4(p.AsVector())
	return back.Mul(math.Scale4(k, k, k)).Mul(toOrigin)
}

// AddDirectEdit adds a consolidated direct-edit operation (#332).
func (c *ModifyFeatures) AddDirectEdit(def *DirectEditDefinition) *PartFeature {
	pf := c.engine.Add(&DirectEditFeature{def: def})
	pf.SetName(c.engine.UniqueName("DirectEdit"))
	return pf
}
