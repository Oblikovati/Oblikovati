// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	"os"

	"oblikovati.org/model/feature"
)

// Mesh reference geometry placement (M10-F04 PBI-115, #700): an ASCII STL becomes a
// MeshFeature — selectable facet topology that passes the running solid through, distinct
// from File ▸ Import which converts meshes into bodies (ImportedBodyFeature).

// ImportMeshFile parses an ASCII STL at path and places it as a mesh reference feature
// on the active part.
//
//	pf, err := s.ImportMeshFile("scan.stl")
func (s *Session) ImportMeshFile(path string) (*feature.PartFeature, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("mesh: open %q: %w", path, err)
	}
	defer f.Close()
	g, err := feature.ParseSTL(f)
	if err != nil {
		return nil, fmt.Errorf("mesh: parse %q: %w", path, err)
	}
	pf := feature.NewMeshFeatures(part.Features()).Add(g)
	part.Recompute()
	s.recordEdit(part, "Mesh")
	if !pf.Health().OK() {
		return pf, errors.New("mesh: " + pf.Health().Reason)
	}
	s.RequestFitView() // #1645: a placed mesh is visible geometry; fit it into view
	return pf, nil
}

// RequestImportMesh flags that the user asked to place a mesh; the head opens its file
// dialog and TakeImportMeshRequest consumes the flag (one-shot, so the dialog opens once).
func (s *Session) RequestImportMesh() { s.meshImportRequested = true }

// TakeImportMeshRequest returns and clears the pending place-mesh request.
func (s *Session) TakeImportMeshRequest() bool {
	req := s.meshImportRequested
	s.meshImportRequested = false
	return req
}
