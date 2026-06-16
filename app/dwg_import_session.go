// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

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

// ImportDWGOnPlane imports a .dwg onto the named work plane (case-insensitive; e.g.
// "XY Plane"), defaulting to the first origin plane when planeName is empty or
// unknown. It is the name-keyed entry the RPC/CLI layer calls; the GUI uses
// ImportDWGFile with a plane chosen from DWGPlaneChoices.
func (s *Session) ImportDWGOnPlane(path, planeName string) (exchange.DWGImportResult, error) {
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		return exchange.DWGImportResult{}, err
	}
	if len(choices) == 0 {
		return exchange.DWGImportResult{}, fmt.Errorf("import dwg: active part has no work planes")
	}
	plane := choices[0].Plane
	if planeName != "" {
		match, ok := planeByName(choices, planeName)
		if !ok {
			return exchange.DWGImportResult{}, fmt.Errorf("import dwg: no work plane named %q (have %s)", planeName, planeNames(choices))
		}
		plane = match
	}
	return s.ImportDWGFile(path, plane)
}

// planeByName finds a plane choice by case-insensitive name.
func planeByName(choices []WorkPlaneChoice, name string) (sketch.Plane, bool) {
	for _, c := range choices {
		if strings.EqualFold(c.Name, name) {
			return c.Plane, true
		}
	}
	return sketch.Plane{}, false
}

// planeNames lists the choice names for an error message.
func planeNames(choices []WorkPlaneChoice) string {
	names := make([]string, len(choices))
	for i, c := range choices {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}
