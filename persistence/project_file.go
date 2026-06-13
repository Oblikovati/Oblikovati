// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
	"oblikovati.org/persistence/yamlcodec"
)

// Design-project file (.opj) read/write — the basic, forward-looking handling for
// the project format introduced by ADR-0034. A project is NOT a document: it is
// the portable search-path config that resolves a document's referenced files
// (architecture core/05). Like a document it is one readable YAML text file
// (ADR-0020): no binary, git-diffable line by line.
//
// This codec round-trips the essentials — name, workspace, library roots, and the
// template/design-data locations. Richer settings are deferred behind explicit
// TODO markers:
//
//	TODO(M11): assemblies resolve their component references through the active
//	project, so assembly modeling is the first consumer that loads an .opj from
//	disk and binds it as the active project; until then this is exercised only by
//	its own tests.
//	TODO(M14): per-project library/style configurations and version-control
//	bindings (the reference API's project options) are out of scope here.

// projectSchemaVersion is the .opj schema version; bump it on a breaking shape change.
const projectSchemaVersion = 1

// projectRecord is the on-disk YAML shape of a [doc.DesignProject]. It is a flat,
// human-readable record so a project file reviews cleanly in a diff.
type projectRecord struct {
	SchemaVersion int      `yaml:"schemaVersion"`
	Name          string   `yaml:"name"`
	Workspace     string   `yaml:"workspace"`
	Libraries     []string `yaml:"libraries,omitempty"`
	Templates     string   `yaml:"templates,omitempty"`
	DesignData    string   `yaml:"designData,omitempty"`
}

// WriteProjectFile writes p to path atomically as a YAML .opj project file. It
// errors if path does not carry the project extension, so a caller cannot
// silently write a project under a document extension.
//
//	persistence.WriteProjectFile(proj, "/work/Acme.opj")
func WriteProjectFile(p *doc.DesignProject, path string) error {
	if err := requireProjectExt(path); err != nil {
		return err
	}
	out, err := yamlcodec.Marshal(projectRecord{
		SchemaVersion: projectSchemaVersion,
		Name:          p.Name,
		Workspace:     p.WorkspacePath,
		Libraries:     p.LibraryPaths,
		Templates:     p.Locations.Templates,
		DesignData:    p.Locations.DesignData,
	})
	if err != nil {
		return fmt.Errorf("persistence: marshal project %q: %w", path, err)
	}
	return atomicWriteFile(path, out)
}

// ReadProjectFile reads a YAML .opj project file at path back into a
// [doc.DesignProject]. It errors on a wrong extension or an unrecognized schema
// version (no silent data loss — ADR-0020).
func ReadProjectFile(path string) (*doc.DesignProject, error) {
	if err := requireProjectExt(path); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persistence: open project %q: %w", path, err)
	}
	var rec projectRecord
	if err := yamlcodec.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("persistence: decode project %q: %w", path, err)
	}
	if rec.SchemaVersion != projectSchemaVersion {
		return nil, fmt.Errorf("persistence: project %q schema version %d, want %d",
			path, rec.SchemaVersion, projectSchemaVersion)
	}
	return &doc.DesignProject{
		Name:          rec.Name,
		WorkspacePath: rec.Workspace,
		LibraryPaths:  rec.Libraries,
		Locations:     doc.FileLocations{Templates: rec.Templates, DesignData: rec.DesignData},
	}, nil
}

// requireProjectExt rejects a path whose extension is not the project extension,
// naming the offending value and the expected extension.
func requireProjectExt(path string) error {
	if ext := strings.ToLower(filepath.Ext(path)); ext != types.ProjectFileExtension {
		return fmt.Errorf("persistence: project file %q must use the %q extension, got %q",
			path, types.ProjectFileExtension, ext)
	}
	return nil
}
