// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/material"

// DuplicateOpenPBRAppearance creates a project-scoped editable copy of an OpenPBR
// appearance and persists the project library. Mirrors [Session.DuplicateAppearance] for
// the full OpenPBR lobe set (M45, ADR-0053).
func (s *Session) DuplicateOpenPBRAppearance(baseID, name string) (*material.OpenPBRAppearance, error) {
	a, err := s.Materials().DuplicateOpenPBRAppearance(baseID, name, material.SourceProject)
	if err != nil {
		return nil, err
	}
	s.saveProjectMaterials()
	return a, nil
}

// UpdateOpenPBRAppearance edits an OpenPBR appearance's spec (a no-op for built-ins) and
// persists the project library.
func (s *Session) UpdateOpenPBRAppearance(id string, spec material.OpenPBRAppearanceSpec) {
	s.Materials().EditOpenPBRAppearance(id, spec)
	s.saveProjectMaterials()
}
