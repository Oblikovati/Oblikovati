// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/exchange"
	"oblikovati.org/model/sketch"
)

// WorkPlaneChoice is a target plane a 2D DWG import can be placed on: a display
// name plus the geometric plane. The File ▸ Import plane picker offers one per
// origin plane and per user work plane of the active part.
type WorkPlaneChoice struct {
	Name  string
	Plane sketch.Plane
}

// DWGPlaneChoices lists the planes a 2D DWG import may target in the active part —
// the three origin planes followed by any user-created work planes. The DWG world
// origin maps onto the chosen plane's origin.
func (s *Session) DWGPlaneChoices() ([]WorkPlaneChoice, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	var choices []WorkPlaneChoice
	for _, wp := range part.OriginPlanes() {
		choices = append(choices, WorkPlaneChoice{Name: wp.Name(), Plane: wp.Plane()})
	}
	planes := part.WorkPlanes()
	for i := 0; i < planes.Count(); i++ {
		wp := planes.Item(i)
		choices = append(choices, WorkPlaneChoice{Name: wp.Name(), Plane: wp.Plane()})
	}
	return choices, nil
}

// ImportDWGFile imports a .dwg into the active part: a planar drawing becomes a 2D
// sketch on plane (the user's chosen work plane), a non-planar drawing a Sketch3D.
// The import is an undoable edit.
//
// Example:
//
//	choices, _ := s.DWGPlaneChoices()
//	res, err := s.ImportDWGFile("floor.dwg", choices[0].Plane) // onto XY
func (s *Session) ImportDWGFile(path string, plane sketch.Plane) (exchange.DWGImportResult, error) {
	part, err := activePart(s)
	if err != nil {
		return exchange.DWGImportResult{}, err
	}
	res, err := exchange.ImportDWGFile(part, path, plane)
	if err != nil {
		return res, err
	}
	s.recordEdit(part, fmt.Sprintf("Import DWG (%d entities)", res.EntityCount))
	return res, nil
}
