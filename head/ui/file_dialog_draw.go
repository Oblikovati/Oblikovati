//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/pointcloud"
)

// fileModal is the chrome's single file explorer (UI state, not model state, so it lives
// here in the head). File-menu items arm it; drawFileDialog renders it.
var fileModal fileDialog

// drawFileDialog renders the file explorer and applies a confirmed file operation.
// A pending add-in ask (dialogs.showFileDialog) arms the modal when it is free.
func drawFileDialog(s *app.Session) {
	if !fileModal.isOpen() {
		if req, ok := s.PendingFileDialog(); ok {
			fileModal.openForRequest(req)
		} else {
			return
		}
	}
	wasAddIn := fileModal.mode == dialogAddIn
	request := fileModal.request
	var act fileAction
	native.SetNextWindowSizeOnce(760, 520)
	if native.Begin(fileModal.title()) {
		drawExplorerHeader()
		act = drawExplorerTable()
		drawExplorerTarget(s)
		act = drawExplorerActions(act)
	}
	native.End()
	applyFileAction(s, act)
	resolveAddInDialog(s, wasAddIn, request, act)
}

// resolveAddInDialog answers the add-in's ask: a confirmed path resolves it; the
// modal closing any other way (Cancel, X, empty confirm) is a cancel.
func resolveAddInDialog(s *app.Session, wasAddIn bool, request app.FileDialogRequest, act fileAction) {
	if !wasAddIn || fileModal.isOpen() {
		return
	}
	if act.Kind == dialogAddIn && act.Path != "" {
		_ = s.ResolveFileDialog(request.ID, []string{act.Path}, false)
		return
	}
	_ = s.ResolveFileDialog(request.ID, nil, true)
}

// drawExplorerHeader renders navigation, root selection, and search controls.
func drawExplorerHeader() {
	native.Text("Folder: " + fileModal.cwd)
	if native.Button("Up") {
		fileModal.goParent()
	}
	native.SameLine()
	if native.Button("Home") {
		fileModal.openDir(homeDir())
	}
	native.SameLine()
	if native.Button("Refresh") {
		fileModal.refresh()
	}
	drawRootChooser()
	native.SetNextItemWidth(-1)
	native.InputText("Search##file-search", fileModal.search[:])
}

// drawRootChooser lets Windows users jump between drive roots and Unix users jump home/root.
func drawRootChooser() {
	if len(fileModal.roots) == 0 || !native.BeginCombo("Location", fileModal.cwd) {
		return
	}
	for _, root := range fileModal.roots {
		if native.Selectable(root, root == fileModal.cwd) {
			fileModal.openDir(root)
		}
	}
	native.EndCombo()
}

// drawExplorerTable renders the current directory rows and returns a double-click action.
func drawExplorerTable() fileAction {
	native.SeparatorText("Files")
	if fileModal.errorText != "" {
		native.Text(fileModal.errorText)
	}
	if !native.BeginTable("##file-browser", 4, 0, 280) {
		return fileAction{}
	}
	drawExplorerTableHeader()
	act := drawExplorerRows(fileModal.visibleEntries())
	native.EndTable()
	return act
}

func drawExplorerTableHeader() {
	for _, col := range []string{"Name", "Type", "Size", "Modified"} {
		native.TableSetupColumn(col)
	}
	native.TableSetupScrollFreeze(0, 1)
	native.TableHeadersRow()
}

func drawExplorerRows(entries []fileEntry) fileAction {
	var act fileAction
	for _, entry := range entries {
		native.TableNextRow()
		if rowAct := drawExplorerRow(entry); rowAct.Kind != dialogClosed {
			act = rowAct
		}
	}
	return act
}

func drawExplorerRow(entry fileEntry) fileAction {
	if native.TableNextColumn() && native.Selectable(entryLabel(entry), fileModal.text() == entry.Path) {
		fileModal.chooseEntry(entry)
		return confirmOnDoubleClick(entry)
	}
	if native.TableNextColumn() {
		native.Text(entryKind(entry))
	}
	if native.TableNextColumn() {
		native.Text(entrySize(entry))
	}
	if native.TableNextColumn() {
		native.Text(entryModified(entry))
	}
	return fileAction{}
}

func confirmOnDoubleClick(entry fileEntry) fileAction {
	if entry.Dir || !native.IsMouseDoubleClicked(native.MouseLeft) {
		return fileAction{}
	}
	return fileModal.confirm()
}

func entryLabel(entry fileEntry) string {
	name := entry.Name
	if entry.Dir {
		name += string(os.PathSeparator)
	}
	return name + "##" + entry.Path
}

func entryKind(entry fileEntry) string {
	if entry.Dir {
		return "Folder"
	}
	return "File"
}

func entrySize(entry fileEntry) string {
	if entry.Dir {
		return ""
	}
	return fmt.Sprintf("%d", entry.Size)
}

func entryModified(entry fileEntry) string {
	if entry.ModTime.IsZero() {
		return ""
	}
	return entry.ModTime.Format("2006-01-02 15:04")
}

// drawExplorerTarget renders the final path field and mode-specific controls.
func drawExplorerTarget(s *app.Session) {
	if hint := fileModal.filterHint(); hint != "" {
		native.Text("Files: " + hint)
	}
	native.SetNextItemWidth(-1)
	native.InputText("File name or path##file-path", fileModal.path[:])
	if fileModal.mode == dialogExport {
		if isDXFTarget() {
			drawExportDXFVersion()
		} else {
			drawExportResolution()
		}
	}
	if fileModal.mode == dialogImport && isSketchImportTarget() {
		drawImportPlane(s)
	}
}

// isSketchImportTarget reports whether the import target is a drawing file (.dwg/.dxf) — the
// sketch-importing formats, for which the plane picker is shown.
func isSketchImportTarget() bool {
	return isSketchPath(fileModal.targetPath())
}

// isDXFTarget reports whether the current target path is a .dxf file.
func isDXFTarget() bool {
	return strings.EqualFold(filepath.Ext(fileModal.targetPath()), ".dxf")
}

// drawImportPlane renders the work-plane picker for a drawing import: a 2D drawing lands
// on the chosen plane (its origin at the plane origin); a 3D drawing ignores it.
func drawImportPlane(s *app.Session) {
	choices, err := s.DWGPlaneChoices()
	if err != nil || len(choices) == 0 {
		native.Text("Open a part to import a drawing into.")
		return
	}
	if fileModal.planeIndex >= len(choices) {
		fileModal.planeIndex = 0
	}
	if native.BeginCombo("Sketch plane", choices[fileModal.planeIndex].Name) {
		for i, c := range choices {
			if native.Selectable(c.Name, i == fileModal.planeIndex) {
				fileModal.planeIndex = i
			}
		}
		native.EndCombo()
	}
}

func drawExplorerActions(act fileAction) fileAction {
	native.BeginDisabled(fileModal.targetPath() == "")
	if native.Button(fileConfirmLabel()) {
		act = fileModal.confirm()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		fileModal.cancel()
	}
	return act
}

func fileConfirmLabel() string {
	switch fileModal.mode {
	case dialogAddIn:
		if fileModal.request.Save {
			return "Save"
		}
		return "Open"
	case dialogSaveAs:
		return "Save"
	case dialogImport:
		return "Import"
	case dialogExport, dialogExportBOM:
		return "Export"
	default:
		return "Open"
	}
}

// drawExportResolution renders the mesh-export resolution selector (low/medium/high).
func drawExportResolution() {
	if native.BeginCombo("Mesh resolution", exportResolutionNames[fileModal.resolution]) {
		for i, name := range exportResolutionNames {
			if native.Selectable(name, i == fileModal.resolution) {
				fileModal.resolution = i
			}
		}
		native.EndCombo()
	}
}

// drawExportDXFVersion renders the DXF-export version selector (r2000/r2018), shown when the
// export target is a .dxf file.
func drawExportDXFVersion() {
	if native.BeginCombo("DXF version", dxfVersionNames[fileModal.dxfVersion]) {
		for i, name := range dxfVersionNames {
			if native.Selectable(name, i == fileModal.dxfVersion) {
				fileModal.dxfVersion = i
			}
		}
		native.EndCombo()
	}
}

// applyFileAction performs the confirmed file operation and reports the outcome — success
// or the underlying (kernel) error — in the status bar so the user always knows what
// happened, instead of an import/export silently doing nothing. A successful Save As
// becomes the document's path, so a later File ▸ Save writes straight through.
func applyFileAction(s *app.Session, act fileAction) {
	name := filepath.Base(act.Path)
	switch act.Kind {
	case dialogOpen:
		openDocumentFromFile(s, act.Path, name)
	case dialogPlaceComponent:
		placeComponentFromFile(s, act.Path, name)
	case dialogSaveAs:
		saveActiveDocumentAs(s, act.Path, name)
	case dialogLoadHDR:
		s.LoadEnvironmentFile(act.Path) // the decode happens lazily on the next viewport frame
	case dialogMeshRef:
		placeMeshFromFile(s, act.Path, name)
	case dialogPointCloud:
		attachPointCloudFromFile(s, act.Path, name)
	case dialogImport:
		importFromFile(s, act, name)
	case dialogExport:
		exportToFile(s, act, name)
	case dialogExportBOM:
		exportBOMToFile(s, act.Path, name)
	}
}

// saveActiveDocumentAs writes the active document to path and, on success, queues its thumbnail
// and remembers the path so a later File ▸ Save writes straight through (File ▸ Save As).
func saveActiveDocumentAs(s *app.Session, path, name string) {
	if err := s.SaveActiveDocumentAs(path); err != nil {
		fileNotice(s, "Save failed: %v", err)
		return
	}
	fileNotice(s, "Saved %s", name)
	queueSaveThumbnail(s, path)
}

// placeMeshFromFile imports an ASCII STL as mesh reference geometry (Mesh ▸ Place Mesh).
func placeMeshFromFile(s *app.Session, path, name string) {
	if _, err := s.ImportMeshFile(path); err != nil {
		fileNotice(s, "Place Mesh failed: %v", err)
		return
	}
	fileNotice(s, "Placed mesh %s", name)
}

// attachPointCloudFromFile attaches an ASCII scan as a referenced point cloud (3D Model ▸ Import
// Point Cloud, #645).
func attachPointCloudFromFile(s *app.Session, path, name string) {
	if _, err := s.AttachPointCloud("", path); err != nil {
		fileNotice(s, "Import Point Cloud failed: %v", err)
		return
	}
	fileNotice(s, "Attached point cloud %s", name)
}

// importBodyFromFile imports a CAD file into the active part as an imported body (File ▸ Import).
func importBodyFromFile(s *app.Session, path, name string) {
	res, err := s.ImportFile(path)
	if err != nil {
		fileNotice(s, importFailedFmt, err)
		return
	}
	fileNotice(s, "Imported %s (%d %s)", name, res.BodyCount, plural(res.BodyCount, "body", "bodies"))
}

// isDWGPath reports whether path is a .dwg file (the sketch-importing branch of File ▸ Import).
func isDWGPath(path string) bool { return strings.EqualFold(filepath.Ext(path), ".dwg") }

// isDXFPath reports whether path is a .dxf file (the other sketch-importing branch).
func isDXFPath(path string) bool { return strings.EqualFold(filepath.Ext(path), ".dxf") }

// isSketchPath reports whether path is a drawing file (.dwg/.dxf) that imports into a sketch.
func isSketchPath(path string) bool { return isDWGPath(path) || isDXFPath(path) }

// importDWGFromFile imports a .dwg into the active part on the chosen work plane
// (2D drawing → a sketch on that plane; 3D drawing → a Sketch3D), reporting the
// outcome (File ▸ Import of a .dwg).
func importDWGFromFile(s *app.Session, act fileAction, name string) {
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		fileNotice(s, importFailedFmt, err)
		return
	}
	plane := choices[0].Plane
	if act.PlaneIndex >= 0 && act.PlaneIndex < len(choices) {
		plane = choices[act.PlaneIndex].Plane
	}
	res, err := s.ImportDWGFile(act.Path, plane)
	if err != nil {
		fileNotice(s, importFailedFmt, err)
		return
	}
	kind := "2D sketch"
	if res.Is3D {
		kind = "3D sketch"
	}
	fileNotice(s, "Imported %s into a %s (%d entities)", name, kind, res.EntityCount)
}

// importFromFile routes a File ▸ Import to the right reader by extension: .dwg/.dxf import
// into a sketch (on the chosen work plane), the mesh/B-rep formats into bodies.
func importFromFile(s *app.Session, act fileAction, name string) {
	switch {
	case isDWGPath(act.Path):
		importDWGFromFile(s, act, name)
	case isDXFPath(act.Path):
		importDXFFromFile(s, act, name)
	case pointcloud.IsScanFile(act.Path):
		attachPointCloudFromFile(s, act.Path, name) // a 3D scan (.ply/.xyz/.pts…) → a point cloud (#645)
	default:
		importBodyFromFile(s, act.Path, name)
	}
}

// importDXFFromFile imports a .dxf into the active part on the chosen work plane (2D drawing
// → a sketch on that plane; 3D drawing → a Sketch3D), reporting the outcome.
func importDXFFromFile(s *app.Session, act fileAction, name string) {
	choices, err := s.DWGPlaneChoices()
	if err != nil {
		fileNotice(s, importFailedFmt, err)
		return
	}
	plane := choices[0].Plane
	if act.PlaneIndex >= 0 && act.PlaneIndex < len(choices) {
		plane = choices[act.PlaneIndex].Plane
	}
	res, err := s.ImportDXFFile(act.Path, plane)
	if err != nil {
		fileNotice(s, importFailedFmt, err)
		return
	}
	kind := "2D sketch"
	if res.Is3D {
		kind = "3D sketch"
	}
	fileNotice(s, "Imported %s into a %s (%d entities)", name, kind, res.EntityCount)
}

// exportToFile writes the active part to the chosen file: a .dxf exports the active sketch's
// curves at the chosen version; the mesh/B-rep formats export the part's bodies.
func exportToFile(s *app.Session, act fileAction, name string) {
	if isDXFPath(act.Path) {
		n, err := s.ExportActiveSketchDXF(act.Path, types.DXFVersion(act.DXFVersion))
		if err != nil {
			fileNotice(s, "Export failed: %v", err)
			return
		}
		fileNotice(s, "Exported %s (%d curves)", name, n)
		return
	}
	if _, err := s.ExportFile(act.Path, types.MeshResolution(act.Resolution)); err != nil {
		fileNotice(s, "Export failed: %v", err)
		return
	}
	fileNotice(s, "Exported %s", name)
}

// exportBOMToFile writes the active assembly's current BOM view to a CSV file (Assemble ▸ Bill of
// Materials ▸ Export CSV, #768).
func exportBOMToFile(s *app.Session, path, name string) {
	if err := s.ExportBOMCSV(path); err != nil {
		fileNotice(s, "Export BOM failed: %v", err)
		return
	}
	fileNotice(s, "Exported BOM %s", name)
}

// openDocumentFromFile opens the chosen document and reports the outcome (File ▸ Open).
func openDocumentFromFile(s *app.Session, path, name string) {
	if _, err := s.OpenDocument(path); err != nil {
		fileNotice(s, "Open failed: %v", err)
		return
	}
	fileNotice(s, "Opened %s", name)
}

// placeComponentFromFile loads the chosen component in the background and hands it to the
// running Place tool (#763); the tool then drops instances on ground-plane clicks. The
// component is loaded (not just named) so the live content is shared and edits propagate to
// every placement — but it is NOT made visible or active, so placing a part never switches the
// tab away from the assembly. The user opens the part in a tab later via Edit on the occurrence.
func placeComponentFromFile(s *app.Session, path, name string) {
	d, err := s.OpenComponentForPlacement(path)
	if err != nil {
		fileNotice(s, "Place Component failed: %v", err)
		return
	}
	s.SetPlaceComponentDocument(d)
	fileNotice(s, "Placing %s — click to drop instances", name)
}

// fileNotice shows a file-operation outcome in the status bar (s.Notice) and mirrors it to
// stderr for the logs. The status bar is the user-facing channel — a kernel import error
// (e.g. an unsupported STEP entity) now surfaces here rather than vanishing to stderr.
func fileNotice(s *app.Session, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.SetNotice(msg)
	fmt.Fprintln(os.Stderr, msg)
}

// plural picks the singular or plural word for n.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// saveActive saves the active document in place, falling back to the Save As modal when
// it has never been saved (a freshly created "PartN" has no path yet — app.ErrNeedsPath).
func saveActive(s *app.Session) {
	err := s.SaveActiveDocument()
	if errors.Is(err, app.ErrNeedsPath) {
		armSaveAs(s)
		return
	}
	if err != nil {
		fileNotice(s, "Save failed: %v", err)
		return
	}
	fileNotice(s, "Saved")
	if d := s.Workspace().ActiveDocument(); d != nil {
		queueSaveThumbnail(s, d.FullFileName())
	}
}
