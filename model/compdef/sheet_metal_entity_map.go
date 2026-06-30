// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
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
	fp, err := d.Unfold()
	if err != nil {
		return nil, false, err
	}
	mapped, found := mapFace(d.foldedBody(), fp.Body, fp.Plane.Normal().AsVector(), key)
	return mapped, found, nil
}

// MapFlatToFolded maps a flat face (by reference key) back to the folded model's corresponding
// front/back face (the base top/bottom face).
func (d *PartComponentDefinition) MapFlatToFolded(key []byte) ([]byte, bool, error) {
	fp, err := d.Unfold()
	if err != nil {
		return nil, false, err
	}
	mapped, found := mapFace(fp.Body, d.foldedBody(), fp.Plane.Normal().AsVector(), key)
	return mapped, found, nil
}

// mapFace resolves key to a face of src, classifies which side of the base plane it faces, and
// returns the reference key of dst's face most strongly facing that side — the front↔front,
// back↔back correspondence. found is false when key is not a face of src.
func mapFace(src, dst *topo.Body, baseNormal math.Vector3, key []byte) ([]byte, bool) {
	// Exact/first-match on purpose (NOT FindOrRecoverFace): a flat pattern's front and back faces
	// deliberately share one reference key, and this map disambiguates them by normal direction
	// below — so the ADR-0043 P0 collision guard (which refuses a >1-match) does not fit here.
	face, ok := src.FindFaceByKey(key)
	if !ok {
		return nil, false
	}
	dir := baseNormal
	if dot, ok := faceNormalAlong(face, baseNormal); ok && dot < 0 {
		dir = baseNormal.Negate()
	}
	return dominantFace(dst, dir).ReferenceKey(), true
}

// foldedBody returns the part's current folded solid (non-nil whenever a flat develops, since
// the flat needs the base Face that produced it).
func (d *PartComponentDefinition) foldedBody() *topo.Body {
	return d.features.Result()[0]
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
