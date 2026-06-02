// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileLocations holds a project's configured directories. It is the portable
// replacement for the COM design-data/template path registry — plain paths, no
// Windows registry (architecture core/05).
type FileLocations struct {
	Templates  string // directory holding document templates, named "<type>.obk"
	DesignData string // shared design data (materials, styles, …)
}

// DesignProject is a named set of search roots that locate referenced files: the
// editable workspace first, then read-only library roots. It replaces COM's .ipj
// project, modeled as portable config rather than registry/monikers.
type DesignProject struct {
	Name          string
	WorkspacePath string   // primary editable root for this project's documents
	LibraryPaths  []string // shared library roots, searched after the workspace
	Locations     FileLocations
}

// SearchPaths returns the resolution roots in priority order: workspace, then
// libraries.
func (p *DesignProject) SearchPaths() []string {
	paths := make([]string, 0, 1+len(p.LibraryPaths))
	if p.WorkspacePath != "" {
		paths = append(paths, p.WorkspacePath)
	}
	return append(paths, p.LibraryPaths...)
}

// DesignProjectManager holds the available projects and tracks the active one. The
// first project added becomes active. Modernizes COM DesignProjectManager.
type DesignProjectManager struct {
	projects []*DesignProject
	active   *DesignProject
}

// NewDesignProjectManager returns an empty project manager.
func NewDesignProjectManager() *DesignProjectManager {
	return &DesignProjectManager{}
}

// Add registers a project; the first one registered becomes the active project.
func (m *DesignProjectManager) Add(p *DesignProject) {
	m.projects = append(m.projects, p)
	if m.active == nil {
		m.active = p
	}
}

// Projects returns a snapshot of the registered projects.
func (m *DesignProjectManager) Projects() []*DesignProject {
	out := make([]*DesignProject, len(m.projects))
	copy(out, m.projects)
	return out
}

// ActiveProject returns the active project, or nil if none are registered.
func (m *DesignProjectManager) ActiveProject() *DesignProject { return m.active }

// SetActiveProject makes a registered project active, erroring if it is unknown.
func (m *DesignProjectManager) SetActiveProject(p *DesignProject) error {
	for _, known := range m.projects {
		if known == p {
			m.active = p
			return nil
		}
	}
	return fmt.Errorf("doc: project %q is not registered", p.Name)
}

// FileManager resolves document file names against a project's search paths and
// locates templates by document type. Modernizes COM FileManager.
type FileManager struct {
	project *DesignProject
}

// NewFileManager binds a file manager to a project.
func NewFileManager(project *DesignProject) *FileManager {
	return &FileManager{project: project}
}

// Resolve turns a (possibly relative) document name into an absolute path by
// searching the project's roots in order. An absolute name is returned as-is if it
// exists. It returns false if no existing file is found (a broken reference).
func (fm *FileManager) Resolve(name string) (string, bool) {
	if filepath.IsAbs(name) {
		return name, fileExists(name)
	}
	for _, root := range fm.project.SearchPaths() {
		candidate := filepath.Join(root, name)
		if fileExists(candidate) {
			return candidate, true
		}
	}
	return "", false
}

// Relativize expresses an absolute path relative to the project workspace, so
// references can be stored portably. It returns false if the path is outside the
// workspace tree.
func (fm *FileManager) Relativize(fullPath string) (string, bool) {
	root := fm.project.WorkspacePath
	if root == "" {
		return "", false
	}
	rel, err := filepath.Rel(root, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return rel, true
}

// TemplateFile returns the template package path for a document kind, or false if
// no template exists for it. Templates are named "<type>.obk" in the project's
// templates directory.
func (fm *FileManager) TemplateFile(t DocumentType) (string, bool) {
	if !t.IsValid() {
		return "", false
	}
	path := filepath.Join(fm.project.Locations.Templates, t.String()+PackageExtension)
	return path, fileExists(path)
}

// fileExists reports whether path names an existing regular file (not a directory).
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
