// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Folded↔flat entity map (M13-F05, #635). Drawing dimensions and selections on the flat
// pattern must survive recompute, so the flat-pattern container maps topology entities between
// the folded model and the developed flat by reference key. This first increment maps at the
// face level: a folded face that faces the base normal (a top face) maps to the flat's front
// face, and a back face to the flat's back face — the two faces a flat-pattern drawing or DXF
// presents. (Edge/vertex mapping is a follow-up.)

// MapFoldedToFlat maps a folded face (by reference key) to the developed flat's corresponding
// front/back face. ok is false when the key is not a face of the folded body.
func (d *PartComponentDefinition) MapFoldedToFlat(key []byte) ([]byte, bool, error) {
	folded := d.foldedBody()
	if folded == nil {
		return nil, false, fmt.Errorf("flat-pattern map: the part has no body")
	}
	face, ok := folded.FindFaceByKey(key)
	if !ok {
		return nil, false, nil
	}
	dir, err := d.faceSide(face)
	if err != nil {
		return nil, false, err
	}
	fp, err := d.Unfold()
	if err != nil {
		return nil, false, err
	}
	return mappedFaceKey(fp.Body, dir)
}

// MapFlatToFolded maps a flat face (by reference key) back to the folded model's corresponding
// front/back face (the base top/bottom face).
func (d *PartComponentDefinition) MapFlatToFolded(key []byte) ([]byte, bool, error) {
	fp, err := d.Unfold()
	if err != nil {
		return nil, false, err
	}
	face, ok := fp.Body.FindFaceByKey(key)
	if !ok {
		return nil, false, nil
	}
	dir, err := d.faceSide(face)
	if err != nil {
		return nil, false, err
	}
	folded := d.foldedBody()
	if folded == nil {
		return nil, false, fmt.Errorf("flat-pattern map: the part has no body")
	}
	return mappedFaceKey(folded, dir)
}

// faceSide returns the base-normal direction the face points toward (front: +base normal,
// back: −base normal), erroring when the part has no base or the face is not planar.
func (d *PartComponentDefinition) faceSide(face *topo.Face) (math.Vector3, error) {
	base, ok := d.baseFace()
	if !ok {
		return math.Vector3{}, fmt.Errorf("flat-pattern map: the part has no base Face")
	}
	normal := base.Definition().Sketch.Plane().Normal().AsVector()
	dot, ok := faceNormalAlong(face, normal)
	if !ok {
		return math.Vector3{}, fmt.Errorf("flat-pattern map: entity is not a planar face")
	}
	if dot < 0 {
		return normal.Negate(), nil
	}
	return normal, nil
}

// mappedFaceKey returns the reference key of body's face that most strongly faces dir (the
// front or back plate face), or ok=false when the body has no such face.
func mappedFaceKey(body *topo.Body, dir math.Vector3) ([]byte, bool, error) {
	if f := dominantFace(body, dir); f != nil {
		return f.ReferenceKey(), true, nil
	}
	return nil, false, nil
}

// foldedBody returns the part's current folded solid, or nil when it has none.
func (d *PartComponentDefinition) foldedBody() *topo.Body {
	bodies := d.features.Result()
	if len(bodies) == 0 {
		return nil
	}
	return bodies[0]
}

// faceNormalAlong returns the planar face's outward normal projected onto dir, ok=false when
// the face is not planar.
func faceNormalAlong(f *topo.Face, dir math.Vector3) (float64, bool) {
	pl, ok := f.Geometry().(geom.Plane)
	if !ok {
		return 0, false
	}
	n := pl.Normal()
	if f.Reversed() {
		n = n.Negate()
	}
	return n.Dot(dir), true
}

// dominantFace returns the body's planar face whose outward normal most strongly aligns with
// dir — for the flat plate or the folded sheet, the single front (or back) face.
func dominantFace(body *topo.Body, dir math.Vector3) *topo.Face {
	var best *topo.Face
	bestDot := stdmath.Inf(-1)
	for _, f := range body.Faces() {
		if dot, ok := faceNormalAlong(f, dir); ok && dot > bestDot {
			best, bestDot = f, dot
		}
	}
	return best
}
