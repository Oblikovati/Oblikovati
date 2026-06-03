// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// This file holds the YAML codec for the direct face-edit features — split, move-face,
// face-offset, delete-face, replace-face and thicken. They share one shape (a set of
// picked faces by reference key) plus, for the geometric edits, a parameter: move-face
// carries a translation vector, face-offset a distance. The keys re-bind to the
// regenerated faces on recompute.

// FaceEditData is a direct face edit: the picked faces as reference keys, plus the optional
// parameter of the geometric edits (move-face translation, face-offset distance).
type FaceEditData struct {
	Faces       []string  `yaml:"faces"`
	Translation []float64 `yaml:"translation,omitempty"` // move-face: dx, dy, dz
	Distance    float64   `yaml:"distance,omitempty"`    // face-offset
	Target      string    `yaml:"target,omitempty"`      // replace-face: source face key
}

// ThickenData is a thicken feature: the wall thickness applied to the running surface body.
type ThickenData struct {
	Value float64 `yaml:"value"`
}

// faceEditor is satisfied by every direct face-edit feature (they embed faceEditFeature),
// exposing the picked face keys for serialization.
type faceEditor interface {
	Feature
	FaceKeys() [][]byte
}

// restoreFaceEdit rebuilds a direct face edit of the given kind from its face keys.
func restoreFaceEdit(fs *PartFeatures, kind string, d *FaceEditData) (*PartFeature, error) {
	if d == nil {
		return nil, fmt.Errorf("%s feature is missing its payload", kind)
	}
	keys, err := decodeKeys(d.Faces)
	if err != nil {
		return nil, err
	}
	m := NewModifyFeatures(fs)
	switch kind {
	case "split":
		return m.AddSplit(keys), nil
	case "move-face":
		return m.AddMoveFace(keys, decodeVec3(d.Translation)), nil
	case "face-offset":
		return m.AddFaceOffset(keys, d.Distance), nil
	case "delete-face":
		return m.AddDeleteFace(keys), nil
	case "replace-face":
		target, err := decodeKey(d.Target)
		if err != nil {
			return nil, err
		}
		return m.AddReplaceFace(keys, target), nil
	default:
		return nil, fmt.Errorf("unknown face-edit kind %q", kind)
	}
}
