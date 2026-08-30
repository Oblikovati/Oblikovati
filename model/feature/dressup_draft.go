// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Dress-up features — the FACE DRAFT definition (M48 #2233 split of dressup.go). The taper-selected-
// faces feature (angle about a pull direction, optional neutral parting plane) and its Recompute. The
// adder collection stays in dressup.go.

// FaceDraftDefinition tapers selected faces by an angle about a pull direction. Neutral, when set, is
// the fixed parting plane each face pivots on (#1801); nil ⇒ the implicit lowest-vertex hinge.
type FaceDraftDefinition struct {
	FaceKeys  [][]byte
	PullDir   math.Vector3
	Neutral   *geom.Plane
	Angle     func() float64
	GeomFaces []topo.GeometricFaceRef // externally-authored drafted faces by geometric descriptor (ADR-0040)
}

// FaceDraftFeature applies draft to faces.
type FaceDraftFeature struct{ def *FaceDraftDefinition }

func (d *FaceDraftFeature) Definition() *FaceDraftDefinition { return d.def }
func (d *FaceDraftFeature) Kind() string                     { return "draft" }

// Recompute tapers the picked faces about the pull direction by the angle (see draft.go).
func (d *FaceDraftFeature) Recompute(in Input) (Output, error) {
	keys, err := bindGeomFaces(in, d.def.FaceKeys, d.def.GeomFaces, "draft")
	if err != nil {
		return Output{}, err
	}
	return draftBody(in, keys, d.def.PullDir, d.def.Neutral, callOrZero(d.def.Angle), "draft")
}
