// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// fileDialogMode is which file operation the path modal will perform on confirm. The
// zero value (dialogClosed) means the modal is not showing.
type fileDialogMode int

const (
	dialogClosed         fileDialogMode = iota
	dialogOpen                          // File ▸ Open
	dialogSaveAs                        // File ▸ Save As
	dialogLoadHDR                       // View ▸ Load HDR (environment image)
	dialogImport                        // File ▸ Import (STL/OBJ/3MF/STEP → imported body)
	dialogExport                        // File ▸ Export (part bodies → STL/OBJ/3MF/STEP)
	dialogAddIn                         // an add-in's dialogs.showFileDialog request (M05-F08)
	dialogMeshRef                       // Mesh ▸ Place Mesh (ASCII STL → mesh reference geometry, #700)
	dialogPlaceComponent                // Assemble ▸ Place: choose the component document to instance (#763)
	dialogExportBOM                     // Assemble ▸ Bill of Materials ▸ Export CSV (#768)
	dialogPointCloud                    // 3D Model ▸ Import Point Cloud (ASCII scan → referenced cloud, #645)
)

// pathBufferLen bounds the path the user can type. ImGui edits the buffer in place,
// so it is a fixed array, not a growable slice; 1024 covers long Windows paths and
// deep Unix project trees without making each dialog state large.
const pathBufferLen = 1024

const fileSearchBufferLen = 128

// fileEntry is one row in the dialog's cross-platform directory browser.
type fileEntry struct {
	Name    string
	Path    string
	Dir     bool
	Size    int64
	ModTime time.Time
}

// fileDialog is the head's path-entry modal state: which operation is pending and the
// path being typed. It is deliberately free of any native/cgo dependency so the
// open→type→confirm/cancel transitions are unit-testable; the draw method that renders
// it lives in the cgo file file_dialog_draw.go (the navigate.go split, ADR-0014).
type fileDialog struct {
	mode       fileDialogMode
	path       [pathBufferLen]byte
	search     [fileSearchBufferLen]byte
	cwd        string
	entries    []fileEntry
	roots      []string
	errorText  string
	resolution int                   // export mesh resolution: 0 low, 1 medium, 2 high
	planeIndex int                   // DWG/DXF import: index into the active part's work-plane choices
	dxfVersion int                   // DXF export: 0 r2000, 1 r2018
	request    app.FileDialogRequest // the add-in ask behind dialogAddIn mode
	defaultExt string                // Save As: extension appended to a bare name, per the active document's kind (ADR-0034)
}

// fileAction is what a confirmed dialog asks the chrome to perform. Kind ==
// dialogClosed means "nothing this frame" (a cancel, or OK on an empty path).
type fileAction struct {
	Kind       fileDialogMode
	Path       string
	Resolution string // export mesh resolution ("low"|"medium"|"high")
	PlaneIndex int    // DWG/DXF import: chosen work-plane index (see DWGPlaneChoices)
	DXFVersion string // DXF export: target version ("r2000"|"r2018")
}

// exportResolutionNames maps the resolution index to its api/types.MeshResolution string.
var exportResolutionNames = []string{"low", "medium", "high"}

// dxfVersionNames maps the DXF-export version index to its api/types.DXFVersion string.
var dxfVersionNames = []string{"r2000", "r2018"}

// openFor arms the dialog for mode with an empty path (export defaults to medium resolution).
// Calling it again re-arms the dialog (e.g. switching from Open to Save As) and clears prior text.
func (d *fileDialog) openFor(mode fileDialogMode) {
	d.mode = mode
	d.path = [pathBufferLen]byte{}
	d.search = [fileSearchBufferLen]byte{}
	d.resolution = 1
	d.defaultExt = doc.Part.Extension() // the common case; armSaveAs refines it to the active document's kind
	d.roots = explorerRoots()
	d.openDir(initialExplorerDir())
}

// armSaveAs opens the Save As modal with its appended extension defaulting to the
// active document's kind (ADR-0034): a bare typed name becomes "<name>.opd" for a
// part, "<name>.oad" for an assembly, and so on.
func armSaveAs(s *app.Session) {
	fileModal.openFor(dialogSaveAs)
	if d := s.ActiveDocument(); d != nil {
		if ext := d.DocumentType().Extension(); ext != "" {
			fileModal.defaultExt = ext
		}
	}
}

// openForRequest arms the dialog for an add-in's ask (M05-F08): the request's
// title heads the window and its initial directory seeds the browser.
func (d *fileDialog) openForRequest(req app.FileDialogRequest) {
	d.openFor(dialogAddIn)
	d.request = req
	if req.InitialDir != "" {
		d.openDir(req.InitialDir)
	}
}

// isOpen reports whether the modal should render this frame.
func (d *fileDialog) isOpen() bool { return d.mode != dialogClosed }

// staticDialogTitles maps each mode whose heading is a constant to that heading; the
// dynamic ones (add-in titles, the open fallback) are handled in title().
var staticDialogTitles = map[fileDialogMode]string{
	dialogSaveAs:         "Save As",
	dialogLoadHDR:        "Load HDR",
	dialogMeshRef:        "Place Mesh (.stl)",
	dialogPointCloud:     "Import Point Cloud (.xyz/.pts)",
	dialogPlaceComponent: "Place Component",
	dialogImport:         "Import (.stl/.obj/.3mf/.step/.dwg/.dxf · scans .ply/.e57/.xyz/.pts)",
	dialogExport:         "Export (.stl/.obj/.3mf/.step/.dxf)",
	dialogExportBOM:      "Export BOM (.csv)",
}

// title is the window heading for the current mode.
func (d *fileDialog) title() string {
	if d.mode == dialogAddIn {
		return d.addInTitle()
	}
	if t, ok := staticDialogTitles[d.mode]; ok {
		return t
	}
	return "Open"
}

// addInTitle is the window heading for an add-in's file dialog request: its custom title when
// set, otherwise Save As / Open by the request's direction.
func (d *fileDialog) addInTitle() string {
	if d.request.Title != "" {
		return d.request.Title
	}
	if d.request.Save {
		return "Save As"
	}
	return "Open"
}

// filterHint names the file family surfaced by the browser for this operation.
func (d *fileDialog) filterHint() string {
	return strings.Join(d.allowedExts(), ", ")
}

// text returns the NUL-trimmed path the user has typed into the buffer.
func (d *fileDialog) text() string {
	if i := bytes.IndexByte(d.path[:], 0); i >= 0 {
		return string(d.path[:i])
	}
	return string(d.path[:])
}

// searchText returns the NUL-trimmed explorer search text.
func (d *fileDialog) searchText() string { return bufString(d.search[:]) }

// targetPath resolves the typed/selected target against the current directory.
func (d *fileDialog) targetPath() string {
	text := strings.TrimSpace(d.text())
	if text == "" {
		return ""
	}
	return d.withDefaultExt(resolveExplorerPath(d.cwd, text))
}

// confirm dismisses the dialog and returns the action for its mode and typed path. A
// blank path yields a dialogClosed action so the caller never acts on "".
func (d *fileDialog) confirm() fileAction {
	path := d.targetPath()
	mode := d.mode
	res := exportResolutionNames[d.resolution]
	d.cancel()
	if path == "" {
		return fileAction{Kind: dialogClosed}
	}
	return fileAction{Kind: mode, Path: path, Resolution: res, PlaneIndex: d.planeIndex, DXFVersion: dxfVersionNames[d.dxfVersion]}
}

// cancel dismisses the dialog without an action and clears the typed path.
func (d *fileDialog) cancel() {
	d.mode = dialogClosed
	d.path = [pathBufferLen]byte{}
	d.search = [fileSearchBufferLen]byte{}
	d.entries = nil
	d.errorText = ""
}

// openDir changes the browser directory and refreshes its rows.
func (d *fileDialog) openDir(dir string) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		d.errorText = fmt.Sprintf("open %q: %v", dir, err)
		return
	}
	d.cwd = filepath.Clean(abs)
	d.refresh()
}

// refresh reloads the current directory snapshot, preserving the current target path.
func (d *fileDialog) refresh() {
	entries, err := readExplorerDir(d.cwd)
	if err != nil {
		d.entries = nil
		d.errorText = fmt.Sprintf("read %q: %v", d.cwd, err)
		return
	}
	d.entries = entries
	d.errorText = ""
}

// goParent moves to the containing directory when the platform path has one.
func (d *fileDialog) goParent() {
	parent := filepath.Dir(d.cwd)
	if parent != d.cwd {
		d.openDir(parent)
	}
}

// chooseEntry selects a file target or enters a directory row.
func (d *fileDialog) chooseEntry(e fileEntry) {
	if e.Dir {
		d.openDir(e.Path)
		return
	}
	d.setTarget(e.Path)
}

// visibleEntries returns directory rows plus files matching search/filter state.
func (d *fileDialog) visibleEntries() []fileEntry {
	needle := strings.ToLower(strings.TrimSpace(d.searchText()))
	var out []fileEntry
	for _, e := range d.entries {
		if d.entryVisible(e, needle) {
			out = append(out, e)
		}
	}
	return out
}

func (d *fileDialog) entryVisible(e fileEntry, needle string) bool {
	if needle != "" && !strings.Contains(strings.ToLower(e.Name), needle) {
		return false
	}
	return e.Dir || d.allowsFile(e.Name)
}

func (d *fileDialog) allowsFile(name string) bool {
	exts := d.allowedExts()
	if len(exts) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	return containsString(exts, ext)
}

func (d *fileDialog) allowedExts() []string {
	switch d.mode {
	case dialogOpen, dialogSaveAs, dialogPlaceComponent:
		return doc.DocumentExtensions() // a component is another .obk document (part or sub-assembly)
	case dialogLoadHDR:
		return []string{".hdr"}
	case dialogMeshRef:
		return []string{".stl"}
	case dialogPointCloud:
		return []string{".xyz", ".pts", ".asc", ".txt"}
	case dialogImport:
		// DWG/DXF import into a sketch, scan formats (.ply/.e57/.xyz/.pts/.asc) into a point cloud,
		// the rest into bodies.
		return []string{".stl", ".obj", ".3mf", ".step", ".stp", ".dwg", ".dxf", ".ply", ".e57", ".xyz", ".pts", ".asc"}
	case dialogExport:
		// DXF exports the active sketch; the others export the part's bodies.
		return []string{".stl", ".obj", ".3mf", ".step", ".stp", ".dxf"}
	case dialogExportBOM:
		return []string{".csv"}
	case dialogAddIn:
		return filterExtensions(d.request.Filter)
	default:
		return nil
	}
}

// filterExtensions pulls the extensions out of a display-name|pattern filter like
// "Meshes (*.stl *.obj)|*.stl;*.obj" — the browser narrows to them; an empty or
// pattern-less filter shows everything.
func filterExtensions(filter string) []string {
	parts := strings.Split(filter, "|")
	patterns := parts[len(parts)-1]
	var exts []string
	for _, pattern := range strings.FieldsFunc(patterns, func(r rune) bool { return r == ';' || r == ' ' }) {
		if ext := filepath.Ext(strings.TrimSpace(pattern)); ext != "" && ext != "." {
			exts = append(exts, strings.ToLower(ext))
		}
	}
	return exts
}

func (d *fileDialog) withDefaultExt(path string) string {
	if d.mode == dialogSaveAs && filepath.Ext(path) == "" {
		return path + d.defaultExt
	}
	return path
}

func (d *fileDialog) setTarget(path string) { copyText(d.path[:], path) }

func readExplorerDir(dir string) ([]fileEntry, error) {
	rows, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]fileEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, explorerEntry(dir, row))
	}
	sortExplorerEntries(entries)
	return entries, nil
}

func explorerEntry(dir string, row fs.DirEntry) fileEntry {
	info, _ := row.Info()
	entry := fileEntry{Name: row.Name(), Path: filepath.Join(dir, row.Name()), Dir: row.IsDir()}
	if info != nil {
		entry.Size = info.Size()
		entry.ModTime = info.ModTime()
	}
	return entry
}

func sortExplorerEntries(entries []fileEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
}

func initialExplorerDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return string(filepath.Separator)
}

func explorerRoots() []string {
	if runtime.GOOS == "windows" {
		return windowsRoots()
	}
	return uniquePaths([]string{string(filepath.Separator), homeDir()})
}

func windowsRoots() []string {
	var roots []string
	for drive := 'A'; drive <= 'Z'; drive++ {
		root := string(drive) + `:\`
		if _, err := os.Stat(root); err == nil {
			roots = append(roots, root)
		}
	}
	return roots
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func uniquePaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if p != "" && !seen[p] {
			out = append(out, p)
			seen[p] = true
		}
	}
	return out
}

func resolveExplorerPath(cwd, text string) string {
	if filepath.IsAbs(text) {
		return filepath.Clean(text)
	}
	return filepath.Clean(filepath.Join(cwd, text))
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func copyText(dst []byte, text string) {
	clearBuf(dst)
	copy(dst, text)
}
