// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/model/exchange"
)

// ImportFile imports a foreign file (STL/OBJ/3MF mesh, or STEP B-rep) into the active part as
// imported-body features, inferring the format from the file extension. The bodies become
// downstream-editable B-rep bodies (a watertight one is a solid); the import is an undoable edit.
// This is the in-process seam the head's File ▸ Import menu calls.
func (s *Session) ImportFile(path string) (exchange.ImportResult, error) {
	part, err := activePart(s)
	if err != nil {
		return exchange.ImportResult{}, err
	}
	format, ok := exchange.FormatFromPath(path)
	if !ok {
		return exchange.ImportResult{}, fmt.Errorf("import: unrecognized file type %q (want .stl/.obj/.3mf/.step)", path)
	}
	res, err := exchange.Import(part, path, format)
	if err != nil {
		return res, err
	}
	s.recordEdit(part, "Import")
	return res, nil
}

// ExportFile writes the active part's bodies to path (STL/OBJ/3MF mesh at the given resolution,
// or STEP B-rep), inferring the format from the file extension. The seam File ▸ Export calls.
func (s *Session) ExportFile(path string, res types.MeshResolution) (exchange.ExportResult, error) {
	part, err := activePart(s)
	if err != nil {
		return exchange.ExportResult{}, err
	}
	format, ok := exchange.FormatFromPath(path)
	if !ok {
		return exchange.ExportResult{}, fmt.Errorf("export: unrecognized file type %q (want .stl/.obj/.3mf/.step)", path)
	}
	return exchange.Export(part, path, format, res)
}
