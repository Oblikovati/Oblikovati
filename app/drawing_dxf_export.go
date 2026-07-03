// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"path/filepath"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/event"
	"oblikovati.org/model/exchange"
)

// Drawing → DXF export from the GUI (M14-F05 PBI-145, #392): the Drawing tab's Export DXF
// command arms the host file-save dialog through the RequestFileDialog seam; when the user
// picks a path, watchDrawingExport writes the active sheet to DXF. The headless wire path
// (drawing.exportDXF) and this GUI path share ExportActiveDrawingDXF, so both behave
// identically.

// drawingDXFDialogID keys the Export DXF file-dialog request and its answer event. It mirrors
// the wire method name by design — the GUI and headless (wire.MethodDrawingExportDXF) paths
// share ExportActiveDrawingDXF — so it references the constant rather than re-declaring the
// string (a rename must not leave a stale copy; #1618).
const drawingDXFDialogID = wire.MethodDrawingExportDXF

// ExportActiveDrawingDXF re-projects the active drawing's views (so the export reflects the
// current model) and writes the active sheet — its visible/hidden view edges, border and
// title block — to a DXF file, returning the entity count.
//
//	n, err := s.ExportActiveDrawingDXF("/tmp/sheet.dxf", types.DXFR2018)
func (s *Session) ExportActiveDrawingDXF(path string, version types.DXFVersion) (int, error) {
	if path == "" {
		return 0, fmt.Errorf("drawing: export path is required")
	}
	c, err := ActiveDrawing(s)
	if err != nil {
		return 0, err
	}
	c.RecomputeViews()
	sheet := c.Sheets().Active()
	if sheet == nil {
		return 0, fmt.Errorf("drawing: no active sheet to export")
	}
	return exchange.ExportDrawingDXFFile(sheet, path, exchange.DefaultDrawingExportLayers(), version.Normalized())
}

// requestDrawingDXFExport arms the host file-save dialog for a DXF target (the Export DXF
// command action); the chosen path arrives as a FileDialogChosen event watchDrawingExport handles.
func requestDrawingDXFExport(s *Session) error {
	return s.RequestFileDialog(FileDialogRequest{
		ID:     drawingDXFDialogID,
		Title:  "Export Drawing to DXF",
		Save:   true,
		Filter: "DXF (*.dxf)|*.dxf",
	})
}

// watchDrawingExport writes the active sheet to DXF when the Export DXF file dialog is
// answered with a path (the GUI counterpart of the drawing.exportDXF wire method). The
// version is the modern default; the wire/CLI path exposes the version choice.
func (s *Session) watchDrawingExport() {
	event.Subscribe(s.bus, event.After, func(_ event.Context, e FileDialogChosen) event.Outcome {
		if e.ID != drawingDXFDialogID || e.Cancelled || len(e.Paths) == 0 {
			return event.Continue()
		}
		path := ensureDXFExtension(e.Paths[0])
		n, err := s.ExportActiveDrawingDXF(path, types.DXFR2018)
		if err != nil {
			s.notice = "Export DXF failed: " + err.Error()
			return event.Continue()
		}
		s.notice = fmt.Sprintf("Exported %s (%d entities)", path, n)
		return event.Continue()
	})
}

// ensureDXFExtension appends ".dxf" to a bare typed name so the written file carries the
// extension the user expects (the host save dialog does not enforce one for add-in asks).
func ensureDXFExtension(path string) string {
	if filepath.Ext(path) == "" {
		return path + ".dxf"
	}
	return path
}
