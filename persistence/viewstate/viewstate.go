// SPDX-License-Identifier: GPL-2.0-only

// Package viewstate persists per-document view configuration (each view's camera, the
// active view, the tiling layout, and the splitter positions) to a single per-user file
// in the OS config directory, keyed by the document's full path.
//
// Why not in the .obk: camera/view state is workstation/user presentation, not shared
// document content — saving it inside the document would dirty it in git every time
// someone merely moved the camera. So it lives outside the document and is written only
// when the user saves the document (the host calls Save then), and read back on open.
package viewstate

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati/persistence/yamlcodec"
	"oblikovati/userconfig"
)

// ViewFrame is one view's persisted camera: a name and a look-at frame (eye, target, up in
// model units; fov in radians).
type ViewFrame struct {
	Name       string     `yaml:"name,omitempty"`
	Eye        [3]float64 `yaml:"eye"`
	Target     [3]float64 `yaml:"target"`
	Up         [3]float64 `yaml:"up"`
	FOV        float64    `yaml:"fov"`
	Projection int        `yaml:"projection,omitempty"` // doc.ProjectionMode (0=perspective)
	Home       *HomeFrame `yaml:"home,omitempty"`       // custom Home; nil ⇒ default iso Home
}

// HomeFrame is a view's persisted custom Home camera (ViewCube "Set Current View as Home").
type HomeFrame struct {
	Eye       [3]float64 `yaml:"eye"`
	Target    [3]float64 `yaml:"target"`
	Up        [3]float64 `yaml:"up"`
	FOV       float64    `yaml:"fov"`
	FitToView bool       `yaml:"fitToView,omitempty"`
}

// ViewState is a document's full view configuration: its views' cameras, which is active,
// how they tile, and the divider positions.
type ViewState struct {
	Views  []ViewFrame  `yaml:"views"`
	Active int          `yaml:"active,omitempty"`
	Layout int32        `yaml:"layout,omitempty"`
	SplitX float32      `yaml:"splitX,omitempty"`
	SplitY float32      `yaml:"splitY,omitempty"`
	Front  *OrientFrame `yaml:"front,omitempty"` // ViewCube front redefinition; nil ⇒ default
}

// OrientFrame is a persisted ViewCube orientation: the three column vectors of the
// rotation (cube-local +X/+Y/+Z mapped to world).
type OrientFrame struct {
	X [3]float64 `yaml:"x"`
	Y [3]float64 `yaml:"y"`
	Z [3]float64 `yaml:"z"`
}

// Store reads and writes per-document view state, keyed by the document's full file path.
type Store interface {
	Load(docKey string) (ViewState, bool, error)
	Save(docKey string, st ViewState) error
}

// file is the on-disk shape: document path → view state.
type file struct {
	Documents map[string]ViewState `yaml:"documents"`
}

// FileStore persists view state to one YAML file under the user config directory.
type FileStore struct{ path string }

// DefaultPath is the per-user view-state file in the shared config dir (userconfig):
// ~/.oblikovati/view-state.yaml on Linux/macOS, %AppData%\oblikovati\view-state.yaml on Windows.
func DefaultPath() (string, error) {
	return userconfig.File("view-state.yaml")
}

// NewFileStore returns a store backed by the file at path.
func NewFileStore(path string) *FileStore { return &FileStore{path: path} }

func (s *FileStore) read() (file, error) {
	f := file{Documents: map[string]ViewState{}}
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return f, nil
	}
	if err != nil {
		return f, fmt.Errorf("viewstate: read %q: %w", s.path, err)
	}
	if err := yamlcodec.Unmarshal(raw, &f); err != nil {
		return f, fmt.Errorf("viewstate: parse %q: %w", s.path, err)
	}
	if f.Documents == nil {
		f.Documents = map[string]ViewState{}
	}
	return f, nil
}

// Load returns the view state stored for docKey, or ok=false if there is none.
func (s *FileStore) Load(docKey string) (ViewState, bool, error) {
	f, err := s.read()
	if err != nil {
		return ViewState{}, false, err
	}
	st, ok := f.Documents[docKey]
	return st, ok, nil
}

// Save writes (or replaces) docKey's view state, creating the file/dir as needed. It
// reads the existing file first so other documents' entries are preserved.
func (s *FileStore) Save(docKey string, st ViewState) error {
	f, err := s.read()
	if err != nil {
		return err
	}
	f.Documents[docKey] = st
	raw, err := yamlcodec.Marshal(f)
	if err != nil {
		return fmt.Errorf("viewstate: marshal: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("viewstate: create config dir: %w", err)
	}
	return os.WriteFile(s.path, raw, 0o644)
}

// MemStore is an in-memory Store for tests.
type MemStore struct{ m map[string]ViewState }

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{m: map[string]ViewState{}} }

// Load implements Store.
func (s *MemStore) Load(docKey string) (ViewState, bool, error) {
	st, ok := s.m[docKey]
	return st, ok, nil
}

// Save implements Store.
func (s *MemStore) Save(docKey string, st ViewState) error {
	s.m[docKey] = st
	return nil
}
