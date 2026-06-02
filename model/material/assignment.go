// SPDX-License-Identifier: GPL-2.0-only

package material

import "encoding/hex"

// AssignmentStore records which material/appearance is assigned where in a part, keyed by
// the persistent hex reference key of a body or face — stable across Recompute, unlike the
// transient body id. It holds a part-default material and appearance plus per-body and
// per-face overrides.
//
// Resolving a surface's effective appearance follows Inventor's AppearanceSourceType
// precedence: face override → body appearance override → body material's appearance →
// part material's appearance → part default appearance → the neutral default.
type AssignmentStore struct {
	partMaterial   string
	partAppearance string
	bodyMaterial   map[string]string // bodyKey → material id
	bodyAppearance map[string]string // bodyKey → appearance id
	faceAppearance map[string]string // faceKey → appearance id
}

// NewAssignmentStore returns an empty store (no assignments — everything resolves to the
// default appearance).
func NewAssignmentStore() *AssignmentStore {
	return &AssignmentStore{
		bodyMaterial:   map[string]string{},
		bodyAppearance: map[string]string{},
		faceAppearance: map[string]string{},
	}
}

// RefKey is the canonical hex encoding of a topology reference key ([]byte) used as the
// map key, so callers in the head/app convert a body/face key consistently.
func RefKey(key []byte) string { return hex.EncodeToString(key) }

// SetPartMaterial / SetPartAppearance set the part-level defaults ("" clears).
func (s *AssignmentStore) SetPartMaterial(id string)   { s.partMaterial = id }
func (s *AssignmentStore) SetPartAppearance(id string) { s.partAppearance = id }

// SetBodyMaterial / SetBodyAppearance / SetFaceAppearance set an override for one
// body/face key; an empty id removes the override.
func (s *AssignmentStore) SetBodyMaterial(key, id string)   { setOrClear(s.bodyMaterial, key, id) }
func (s *AssignmentStore) SetBodyAppearance(key, id string) { setOrClear(s.bodyAppearance, key, id) }
func (s *AssignmentStore) SetFaceAppearance(key, id string) { setOrClear(s.faceAppearance, key, id) }

// PartMaterial / PartAppearance / BodyMaterials / BodyAppearances / FaceAppearances expose
// the raw assignments for persistence (the maps are copies).
func (s *AssignmentStore) PartMaterial() string             { return s.partMaterial }
func (s *AssignmentStore) PartAppearance() string           { return s.partAppearance }
func (s *AssignmentStore) BodyMaterials() map[string]string { return copyMap(s.bodyMaterial) }
func (s *AssignmentStore) BodyAppearances() map[string]string {
	return copyMap(s.bodyAppearance)
}
func (s *AssignmentStore) FaceAppearances() map[string]string { return copyMap(s.faceAppearance) }

// EffectiveMaterial returns the material governing a body (its override, else the part
// default), or false when none is assigned.
func (s *AssignmentStore) EffectiveMaterial(lib *Library, bodyKey string) (*Material, bool) {
	id := s.bodyMaterial[bodyKey]
	if id == "" {
		id = s.partMaterial
	}
	if id == "" {
		return nil, false
	}
	return lib.Material(id)
}

// EffectiveAppearance resolves the appearance shown for a body/face along the precedence
// chain. faceKey may be "" to resolve at body level. It always returns a non-nil
// appearance (the neutral default when nothing else applies).
func (s *AssignmentStore) EffectiveAppearance(lib *Library, bodyKey, faceKey string) *Appearance {
	if faceKey != "" {
		if a := lib.appearanceOrNil(s.faceAppearance[faceKey]); a != nil {
			return a
		}
	}
	if a := lib.appearanceOrNil(s.bodyAppearance[bodyKey]); a != nil {
		return a
	}
	if m, ok := s.EffectiveMaterial(lib, bodyKey); ok {
		if a := lib.appearanceOrNil(m.AppearanceID()); a != nil {
			return a
		}
	}
	if a := lib.appearanceOrNil(s.partAppearance); a != nil {
		return a
	}
	return lib.defaultAppearance()
}

// setOrClear sets key→id, or deletes key when id is empty.
func setOrClear(m map[string]string, key, id string) {
	if id == "" {
		delete(m, key)
		return
	}
	m[key] = id
}

// copyMap returns an independent copy of a string map (defensive, for persistence reads).
func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
