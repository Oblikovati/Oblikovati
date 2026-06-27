// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/material"
)

// AppearanceScope names where an appearance override applies (the wire "scope" values).
const (
	ScopePart = "part"
	ScopeBody = "body"
	ScopeFace = "face"
)

// ActivePartMaterialID returns the active part's assigned part-level material id, or "" when
// there is no active part (e.g. an assembly is active) or it carries no part-level material.
// The Materials UI reads this to keep its selector in sync with the active document.
func (s *Session) ActivePartMaterialID() string {
	part, err := activePart(s)
	if err != nil {
		return ""
	}
	return part.Assignments().PartMaterial()
}

// BodyMaterialID returns the effective material id assigned to a body (its own override, else
// the part default), or "" when none is assigned and there is no active part. The body.list
// router reports it so an analysis add-in can resolve each body's material (#1078 read-back of
// the write-only AssignMaterial).
func (s *Session) BodyMaterialID(bodyKey string) string {
	part, err := activePart(s)
	if err != nil {
		return ""
	}
	return part.Assignments().EffectiveMaterialID(bodyKey)
}

// ActivePartAppearanceID returns the active part's assigned part-level appearance id, or "".
func (s *Session) ActivePartAppearanceID() string {
	part, err := activePart(s)
	if err != nil {
		return ""
	}
	return part.Assignments().PartAppearance()
}

// AssignMaterial sets the material for a body (bodyKey hex) or the part default (empty
// bodyKey), embedding a portable copy of the material and its appearance in the document
// so the assignment survives a round-trip even without the project library.
func (s *Session) AssignMaterial(bodyKey, materialID string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if _, ok := s.Materials().Material(materialID); !ok {
		return fmt.Errorf("app: unknown material %q", materialID)
	}
	s.embedMaterial(part, materialID)
	if bodyKey == "" {
		part.Assignments().SetPartMaterial(materialID)
	} else {
		part.Assignments().SetBodyMaterial(bodyKey, materialID)
	}
	part.MarkChanged()
	s.recordEdit(part, "Assign Material")
	return nil
}

// AssignAppearance overrides the appearance at part/body/face scope, embedding a portable
// copy of the appearance in the document.
func (s *Session) AssignAppearance(scope, key, appearanceID string) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	if _, ok := s.Materials().Appearance(appearanceID); !ok {
		return fmt.Errorf("app: unknown appearance %q", appearanceID)
	}
	s.embedAppearance(part, appearanceID)
	switch scope {
	case ScopePart:
		part.Assignments().SetPartAppearance(appearanceID)
	case ScopeBody:
		part.Assignments().SetBodyAppearance(key, appearanceID)
	case ScopeFace:
		part.Assignments().SetFaceAppearance(key, appearanceID)
	default:
		return fmt.Errorf("app: unknown appearance scope %q", scope)
	}
	part.MarkChanged()
	s.recordEdit(part, "Assign Appearance")
	return nil
}

// embedAppearance copies a non-built-in appearance into the document's asset set, so the
// .obk carries its own copy (built-ins are reproducible and not embedded).
func (s *Session) embedAppearance(part *compdef.PartComponentDefinition, id string) {
	a, ok := s.Materials().Appearance(id)
	if !ok || a.Source() == material.SourceBuiltin {
		return
	}
	part.Assets().PutAppearance(material.NewAppearance(a.ID(), material.SourceDocument, a.Spec()))
}

// embedMaterial copies a non-built-in material (and its appearance) into the document.
func (s *Session) embedMaterial(part *compdef.PartComponentDefinition, id string) {
	m, ok := s.Materials().Material(id)
	if !ok {
		return
	}
	s.embedAppearance(part, m.AppearanceID())
	if m.Source() == material.SourceBuiltin {
		return
	}
	part.Assets().PutMaterial(material.NewMaterial(m.ID(), material.SourceDocument, m.Spec()))
}

// PhysicalProperties computes the active part's mass/volume/area/centroid, summed over its
// bodies, each body using its effective material's density. It returns false when there is
// no active part.
func (s *Session) PhysicalProperties() (types.PhysicalProperties, bool) {
	part, err := activePart(s)
	if err != nil {
		return types.PhysicalProperties{}, false
	}
	return s.sumBodyProperties(part), true
}

// massAccum accumulates volume/area/mass and the volume-weighted centroid numerator across
// a part's bodies.
type massAccum struct {
	volume, area, mass float64
	cx, cy, cz         float64
}

// sumBodyProperties sums the per-body geometry properties (each body weighted by its
// effective material's density) into the part's physical properties.
func (s *Session) sumBodyProperties(part *compdef.PartComponentDefinition) types.PhysicalProperties {
	look := material.MergedLookup{Embedded: part.Assets(), Catalog: s.Materials()}
	assign := part.Assignments()
	var a massAccum
	for _, b := range part.SurfaceBodies().All() {
		gp := ops.BodyGeometryProperties(b, ops.DefaultQuality())
		density := 0.0
		if m, ok := assign.EffectiveMaterial(look, material.RefKey(b.ReferenceKey())); ok {
			density = m.Density()
		}
		a.add(gp, density)
	}
	return a.result()
}

// add folds one body's geometry properties (at the given density) into the accumulator.
func (a *massAccum) add(gp ops.GeometryProperties, density float64) {
	a.volume += gp.Volume
	a.area += gp.Area
	a.mass += density * gp.Volume
	a.cx += float64(gp.Centroid.X) * gp.Volume
	a.cy += float64(gp.Centroid.Y) * gp.Volume
	a.cz += float64(gp.Centroid.Z) * gp.Volume
}

// result finalizes the accumulator, dividing the centroid numerator by total volume.
func (a *massAccum) result() types.PhysicalProperties {
	props := types.PhysicalProperties{Mass: a.mass, Volume: a.volume, Area: a.area}
	if a.volume > 0 {
		props.Density = a.mass / a.volume
		props.Centroid = [3]float64{a.cx / a.volume, a.cy / a.volume, a.cz / a.volume}
	}
	return props
}
