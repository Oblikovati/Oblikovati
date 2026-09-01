// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/drawing"
)

// TestExportActiveDrawingDXF checks the session writes the active sheet (with a view) to a DXF
// file — the engine the GUI Export DXF command and the drawing.exportDXF wire method share.
func TestExportActiveDrawingDXF(t *testing.T) {
	t.Parallel()
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	if _, err := c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{
		Orientation: types.BaseViewFront, Scale: 2, CenterX: 120, CenterY: 100,
	}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}

	path := filepath.Join(t.TempDir(), "sheet.dxf")
	n, err := s.ExportActiveDrawingDXF(path, types.DXFR2018)
	if err != nil || n == 0 {
		t.Fatalf("ExportActiveDrawingDXF = (%d, %v), want entities written", n, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "LINE") {
		t.Fatalf("exported DXF unreadable or empty: err=%v", err)
	}
}

// TestExportActiveDrawingDXFRejectsBadInput covers the guard paths: an empty path and a
// non-drawing active document both error rather than writing a file.
func TestExportActiveDrawingDXFRejectsBadInput(t *testing.T) {
	t.Parallel()
	s := drawingWithModelSession(t)
	if _, err := s.ExportActiveDrawingDXF("", types.DXFR2018); err == nil {
		t.Error("empty path = ok, want error")
	}

	bare := NewSession() // no active document
	if _, err := bare.ExportActiveDrawingDXF("/tmp/x.dxf", types.DXFR2018); err == nil {
		t.Error("export with no active drawing = ok, want error")
	}
}

// TestDrawingExportDXFCommandFlow drives the GUI path end to end: the Export DXF command arms
// the host file dialog, and answering it with a path writes the file and reports the outcome.
func TestDrawingExportDXFCommandFlow(t *testing.T) {
	t.Parallel()
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	if _, err := c.Sheets().Active().Views().AddBase(drawing.BaseViewSpec{
		Orientation: types.BaseViewIso, Scale: 1, CenterX: 150, CenterY: 100,
	}); err != nil {
		t.Fatalf("AddBase: %v", err)
	}

	if err := s.Execute("Drawing.ExportDXF"); err != nil {
		t.Fatalf("Execute Export DXF: %v", err)
	}
	req, ok := s.PendingFileDialog()
	if !ok || req.ID != drawingDXFDialogID || !req.Save {
		t.Fatalf("pending file dialog = %+v (ok=%v), want a save request keyed %q", req, ok, drawingDXFDialogID)
	}

	path := filepath.Join(t.TempDir(), "out.dxf")
	if err := s.ResolveFileDialog(drawingDXFDialogID, []string{path}, false); err != nil {
		t.Fatalf("ResolveFileDialog: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Export DXF wrote no file at %q: %v", path, err)
	}
	if !strings.HasPrefix(s.Notice(), "Exported ") {
		t.Errorf("notice = %q, want an export confirmation", s.Notice())
	}
}

// TestDrawingExportDXFCancelWritesNothing checks a cancelled dialog leaves no file and the
// handler ignores it.
func TestDrawingExportDXFCancelWritesNothing(t *testing.T) {
	t.Parallel()
	s := drawingWithModelSession(t)
	if err := requestDrawingDXFExport(s); err != nil {
		t.Fatalf("requestDrawingDXFExport: %v", err)
	}
	if err := s.ResolveFileDialog(drawingDXFDialogID, nil, true); err != nil {
		t.Fatalf("ResolveFileDialog (cancel): %v", err)
	}
	if strings.HasPrefix(s.Notice(), "Exported ") {
		t.Errorf("notice = %q, want no export on cancel", s.Notice())
	}
}

func TestEnsureDXFExtension(t *testing.T) {
	t.Parallel()
	if got := ensureDXFExtension("/tmp/sheet"); got != "/tmp/sheet.dxf" {
		t.Errorf("ensureDXFExtension(bare) = %q, want /tmp/sheet.dxf", got)
	}
	if got := ensureDXFExtension("/tmp/sheet.dxf"); got != "/tmp/sheet.dxf" {
		t.Errorf("ensureDXFExtension kept = %q, want unchanged", got)
	}
}
