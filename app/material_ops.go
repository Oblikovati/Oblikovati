// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"github.com/Oblikovati/api/types"

	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/model/compdef"
	"github.com/Oblikovati/oblikovati/model/material"
)

// AppearanceScope names where an appearance override applies (the wire "scope" values).
const (
	ScopePart = "part"
	ScopeBody = "body"
	ScopeFace = "face"
)

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
	look := material.MergedLookup{Embedded: part.Assets(), Catalog: s.Materials()}
	assign := part.Assignments()
	var volume, area, mass float64
	var cx, cy, cz float64
	for _, b := range part.SurfaceBodies().All() {
		gp := ops.BodyGeometryProperties(b, ops.DefaultQuality())
		density := 0.0
		if m, ok := assign.EffectiveMaterial(look, material.RefKey(b.ReferenceKey())); ok {
			density = m.Density()
		}
		volume += gp.Volume
		area += gp.Area
		mass += density * gp.Volume
		cx += float64(gp.Centroid.X) * gp.Volume
		cy += float64(gp.Centroid.Y) * gp.Volume
		cz += float64(gp.Centroid.Z) * gp.Volume
	}
	props := types.PhysicalProperties{Mass: mass, Volume: volume, Area: area}
	if volume > 0 {
		props.Density = mass / volume
		props.Centroid = [3]float64{cx / volume, cy / volume, cz / volume}
	}
	return props, true
}
