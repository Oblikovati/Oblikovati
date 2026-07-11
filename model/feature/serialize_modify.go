// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
)

// This file holds the YAML codec for the direct face-edit features — split, move-face,
// face-offset, delete-face, replace-face and thicken. They share one shape (a set of
// picked faces by reference key) plus, for the geometric edits, a parameter: move-face
// carries a translation vector or a rotation, face-offset a distance. The keys re-bind to
// the regenerated faces on recompute.

// FaceEditData is a direct face edit: the picked faces as reference keys, plus the optional
// parameter of the geometric edits (move-face translation or rotation, face-offset
// distance + approximation).
type FaceEditData struct {
	Faces         []string  `yaml:"faces"`
	Translation   []float64 `yaml:"translation,omitempty"`   // move-face: dx, dy, dz
	AxisPoint     []float64 `yaml:"axisPoint,omitempty"`     // move-face rotate (#331)
	AxisDir       []float64 `yaml:"axisDir,omitempty"`       // move-face rotate
	Angle         float64   `yaml:"angle,omitempty"`         // move-face rotate, radians
	Distance      float64   `yaml:"distance,omitempty"`      // face-offset
	Approximation string    `yaml:"approximation,omitempty"` // face-offset (#331), wire spelling
	Target        string    `yaml:"target,omitempty"`        // replace-face: source face key
	// Open is delete-face's inverse-heal flag (#1884): stored as the negation so the zero value
	// (absent field) restores the legacy healed behaviour — pre-#1884 recipes had no field and
	// were always healed. Open=true means the delete left the body open (heal=false).
	Open bool `yaml:"open,omitempty"`
}

// ThickenData is a thicken feature: the wall thickness applied to the running surface body,
// plus the #331 approximation request (wire spelling; absent = none/exact) and the #1876 options.
// The zero value of each new field restores the pre-#1876 behaviour: Direction "" = symmetric,
// Operation "" = join, and NoWalls false = vertical surfaces on (stored negated so a legacy recipe,
// which always built walls, restores unchanged).
type ThickenData struct {
	Value         float64  `yaml:"value"`
	Approximation string   `yaml:"approximation,omitempty"`
	Direction     string   `yaml:"direction,omitempty"` // "" = symmetric (legacy)
	Operation     string   `yaml:"operation,omitempty"` // "" = join; "surface" = offset surface
	Faces         []string `yaml:"faces,omitempty"`     // face subset (empty = whole body)
	NoWalls       bool     `yaml:"noWalls,omitempty"`   // inverse of createVerticalSurfaces
	AutoChain     bool     `yaml:"autoChain,omitempty"`
	AutoBlend     bool     `yaml:"autoBlend,omitempty"`
}

// thickenDirectionName / thickenDirectionOf map the direction enum to its serialized spelling;
// symmetric encodes as "" so a legacy recipe (no field) and a new symmetric thicken both restore
// symmetric.
func thickenDirectionName(d ops.ThickenDirection) string {
	if d == ops.ThickenSymmetric {
		return ""
	}
	return d.String()
}

func thickenDirectionOf(s string) ops.ThickenDirection {
	switch s {
	case "positive":
		return ops.ThickenPositive
	case "negative":
		return ops.ThickenNegative
	default:
		return ops.ThickenSymmetric // "" legacy / symmetric
	}
}

// thickenOperationName encodes the thicken output mode: "surface" for the offset-surface path,
// "cut"/"intersect" for the booleans, and "" for join (the legacy default, so it stays omitted).
func thickenOperationName(op ops.PartFeatureOperation, asSurface bool) string {
	if asSurface {
		return "surface"
	}
	if op == ops.Join {
		return ""
	}
	return op.String()
}

// thickenOperationOf decodes the mode back to (operation, asSurface). "" = join.
func thickenOperationOf(s string) (ops.PartFeatureOperation, bool) {
	switch s {
	case "surface":
		return ops.Join, true
	case "cut":
		return ops.Cut, false
	case "intersect":
		return ops.Intersect, false
	default:
		return ops.Join, false
	}
}

// approximationName / approximationOf map the enum to its wire spelling (empty for the zero
// value so older recipes stay byte-identical).
func approximationName(a types.FeatureApproximationType) string {
	if a == 0 {
		return ""
	}
	return a.String()
}

func approximationOf(s string) (types.FeatureApproximationType, error) {
	if s == "" {
		return 0, nil
	}
	a, ok := types.ParseFeatureApproximationType(s)
	if !ok {
		return 0, fmt.Errorf("unknown approximation %q (want none/mean/neverTooThick/neverTooThin)", s)
	}
	return a, nil
}

// serializeMoveFace encodes a move-face in whichever mode it carries.
func serializeMoveFace(f *MoveFaceFeature) *FaceEditData {
	if p, dir, angle, rotating := f.Rotation(); rotating {
		return &FaceEditData{Faces: encodeKeys(f.FaceKeys()),
			AxisPoint: []float64{p.X, p.Y, p.Z}, AxisDir: []float64{dir.X, dir.Y, dir.Z}, Angle: angle}
	}
	t := f.Translation()
	return &FaceEditData{Faces: encodeKeys(f.FaceKeys()), Translation: []float64{t.X, t.Y, t.Z}}
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
		if len(d.AxisDir) == 3 {
			return m.AddMoveFaceRotate(keys, decodePoint3(d.AxisPoint), decodeVec3(d.AxisDir), constFloat(d.Angle)), nil
		}
		return m.AddMoveFace(keys, decodeVec3(d.Translation)), nil
	case "face-offset":
		approx, err := approximationOf(d.Approximation)
		if err != nil {
			return nil, err
		}
		return m.AddFaceOffsetApprox(keys, constFloat(d.Distance), approx), nil
	case "delete-face":
		return m.AddDeleteFace(keys, !d.Open), nil // Open is stored negated (see FaceEditData.Open)
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
