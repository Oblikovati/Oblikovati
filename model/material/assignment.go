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

	// OpenPBRAppearance assignments (M45-F05 PBI-350, ADR-0053) — a SEPARATE override
	// chain from the legacy Appearance one above, not a fallback into it: a body assigned
	// an OpenPBRAppearance does not also consult bodyAppearance. When BOTH are set on the
	// same body, the OpenPBR chain wins (app.Session.partSurfaceLookup, #2150) — the
	// newer, richer system a user explicitly opted into by assigning it.
	partOpenPBRAppearance string
	bodyOpenPBRAppearance map[string]string
	faceOpenPBRAppearance map[string]string
}

// NewAssignmentStore returns an empty store (no assignments — everything resolves to the
// default appearance).
func NewAssignmentStore() *AssignmentStore {
	return &AssignmentStore{
		bodyMaterial:          map[string]string{},
		bodyAppearance:        map[string]string{},
		faceAppearance:        map[string]string{},
		bodyOpenPBRAppearance: map[string]string{},
		faceOpenPBRAppearance: map[string]string{},
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
func (s *AssignmentStore) SetBodyMaterial(key, id string) { setOrClear(s.bodyMaterial, key, id) }

func (s *AssignmentStore) SetBodyAppearance(key, id string) { setOrClear(s.bodyAppearance, key, id) }

func (s *AssignmentStore) SetFaceAppearance(key, id string) { setOrClear(s.faceAppearance, key, id) }

// SetPartOpenPBRAppearance / SetBodyOpenPBRAppearance / SetFaceOpenPBRAppearance mirror
// the legacy Appearance setters above, for the separate OpenPBRAppearance chain
// (M45-F05 PBI-350, ADR-0053).
func (s *AssignmentStore) SetPartOpenPBRAppearance(id string) { s.partOpenPBRAppearance = id }

func (s *AssignmentStore) SetBodyOpenPBRAppearance(key, id string) {
	setOrClear(s.bodyOpenPBRAppearance, key, id)
}

func (s *AssignmentStore) SetFaceOpenPBRAppearance(key, id string) {
	setOrClear(s.faceOpenPBRAppearance, key, id)
}

// PartMaterial / PartAppearance / BodyMaterials / BodyAppearances / FaceAppearances expose
// the raw assignments for persistence (the maps are copies).
func (s *AssignmentStore) PartMaterial() string             { return s.partMaterial }
func (s *AssignmentStore) PartAppearance() string           { return s.partAppearance }
func (s *AssignmentStore) BodyMaterials() map[string]string { return copyMap(s.bodyMaterial) }
func (s *AssignmentStore) BodyAppearances() map[string]string {
	return copyMap(s.bodyAppearance)
}

// PartOpenPBRAppearance / BodyOpenPBRAppearances / FaceOpenPBRAppearances expose the raw
// OpenPBRAppearance assignments for persistence (the maps are copies).
func (s *AssignmentStore) PartOpenPBRAppearance() string { return s.partOpenPBRAppearance }
func (s *AssignmentStore) BodyOpenPBRAppearances() map[string]string {
	return copyMap(s.bodyOpenPBRAppearance)
}
func (s *AssignmentStore) FaceOpenPBRAppearances() map[string]string {
	return copyMap(s.faceOpenPBRAppearance)
}
func (s *AssignmentStore) FaceAppearances() map[string]string { return copyMap(s.faceAppearance) }

// AssetLookup resolves asset ids to assets. Both [Library] (the session catalog) and a
// [MergedLookup] (document-embedded assets over the catalog) satisfy it, so the same
// precedence logic serves the browser and the renderer regardless of where an asset lives.
type AssetLookup interface {
	Appearance(id string) (*Appearance, bool)
	Material(id string) (*Material, bool)
	DefaultAppearance() *Appearance
	OpenPBRAppearance(id string) (*OpenPBRAppearance, bool)
}

// EffectiveMaterialID returns the id of the material governing a body — its own override if
// set, else the part default — or "" when none is assigned. It is the id-only form of
// EffectiveMaterial, for callers (e.g. the body.list router) that report the assignment
// without resolving the material's properties.
func (s *AssignmentStore) EffectiveMaterialID(bodyKey string) string {
	if id := s.bodyMaterial[bodyKey]; id != "" {
		return id
	}
	return s.partMaterial
}

// EffectiveMaterial returns the material governing a body (its override, else the part
// default), or false when none is assigned.
func (s *AssignmentStore) EffectiveMaterial(look AssetLookup, bodyKey string) (*Material, bool) {
	id := s.EffectiveMaterialID(bodyKey)
	if id == "" {
		return nil, false
	}
	return look.Material(id)
}

// EffectiveAppearance resolves the appearance shown for a body/face along the precedence
// chain. faceKey may be "" to resolve at body level. It always returns a non-nil
// appearance (the neutral default when nothing else applies).
func (s *AssignmentStore) EffectiveAppearance(look AssetLookup, bodyKey, faceKey string) *Appearance {
	if faceKey != "" {
		if a := apprOrNil(look, s.faceAppearance[faceKey]); a != nil {
			return a
		}
	}
	if a := apprOrNil(look, s.bodyAppearance[bodyKey]); a != nil {
		return a
	}
	// An explicit PART-level appearance override wins over the assigned material's own
	// appearance — otherwise assigning a material (e.g. via an add-in) would make a later
	// "set appearance" no-op (the grey-appearance bug, #1103). Material appearance is the
	// fallback when no explicit override is set.
	if a := apprOrNil(look, s.partAppearance); a != nil {
		return a
	}
	if m, ok := s.EffectiveMaterial(look, bodyKey); ok {
		if a := apprOrNil(look, m.AppearanceID()); a != nil {
			return a
		}
	}
	return look.DefaultAppearance()
}

// EffectiveOpenPBRAppearance resolves the OpenPBR appearance overriding a body/face, along
// the same face → body → part precedence [EffectiveAppearance] uses. Unlike
// EffectiveAppearance it returns ok=false (not the neutral default) when nothing in this
// chain is assigned, so a caller can fall back to the legacy Appearance chain instead —
// the two chains are separate (see the AssignmentStore doc comment), and a renderer must
// tell "explicitly assigned OpenPBR appearance" apart from "nothing assigned here."
func (s *AssignmentStore) EffectiveOpenPBRAppearance(look AssetLookup, bodyKey, faceKey string) (*OpenPBRAppearance, bool) {
	if faceKey != "" {
		if a := openPBRApprOrNil(look, s.faceOpenPBRAppearance[faceKey]); a != nil {
			return a, true
		}
	}
	if a := openPBRApprOrNil(look, s.bodyOpenPBRAppearance[bodyKey]); a != nil {
		return a, true
	}
	if a := openPBRApprOrNil(look, s.partOpenPBRAppearance); a != nil {
		return a, true
	}
	return nil, false
}

// openPBRApprOrNil returns the OpenPBR appearance for id via look, or nil for an
// empty/unknown id.
func openPBRApprOrNil(look AssetLookup, id string) *OpenPBRAppearance {
	if id == "" {
		return nil
	}
	a, _ := look.OpenPBRAppearance(id)
	return a
}

// apprOrNil returns the appearance for id via look, or nil for an empty/unknown id.
func apprOrNil(look AssetLookup, id string) *Appearance {
	if id == "" {
		return nil
	}
	a, _ := look.Appearance(id)
	return a
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
