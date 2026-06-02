// SPDX-License-Identifier: GPL-2.0-only

package feature

import "fmt"

// This file holds the YAML codec for the direct face-edit features — split, move-face,
// face-offset, delete-face, replace-face and thicken. They share one shape (a set of
// picked faces by reference key), so they serialize uniformly via the faceEditor
// interface and restore by kind. The keys re-bind to the regenerated faces on recompute.

// FaceEditData is a direct face edit: the picked faces as reference keys.
type FaceEditData struct {
	Faces []string `yaml:"faces"`
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
		return m.AddMoveFace(keys), nil
	case "face-offset":
		return m.AddFaceOffset(keys), nil
	case "delete-face":
		return m.AddDeleteFace(keys), nil
	case "replace-face":
		return m.AddReplaceFace(keys), nil
	case "thicken":
		return m.AddThicken(keys), nil
	default:
		return nil, fmt.Errorf("unknown face-edit kind %q", kind)
	}
}
